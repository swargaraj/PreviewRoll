package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/config"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	db  *database.DB
	cfg *config.Config
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(db *database.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserResponse represents the user response
type UserResponse struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Username == "" || req.Password == "" {
		http.Error(w, "Email, username, and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Hash password with Argon2id
	hash, err := hashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Insert user into database
	var userID int64
	err = h.db.QueryRow(`
		INSERT INTO users (email, username, password_hash)
		VALUES (?, ?, ?)
		RETURNING id
	`, req.Email, req.Username, hash).Scan(&userID)
	if err != nil {
		http.Error(w, "Email or username already exists", http.StatusConflict)
		return
	}

	// Create response
	user := UserResponse{
		ID:        userID,
		Email:     req.Email,
		Username:  req.Username,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find user by email
	var user struct {
		ID           int64
		PasswordHash string
	}
	err := h.db.QueryRow(`
		SELECT id, password_hash FROM users WHERE email = ?
	`, req.Email).Scan(&user.ID, &user.PasswordHash)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Verify password
	if !verifyPassword(req.Password, user.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate session token
	sessionToken, err := generateToken(32)
	if err != nil {
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	// Generate session ID
	sessionID, err := generateToken(16)
	if err != nil {
		http.Error(w, "Failed to generate session ID", http.StatusInternalServerError)
		return
	}

	// Calculate expiry time
	expiresAt := time.Now().Add(time.Duration(h.cfg.SessionMaxAge) * time.Second)

	// Insert session into database
	_, err = h.db.Exec(`
		INSERT INTO sessions (id, user_id, token, expires_at)
		VALUES (?, ?, ?, ?)
	`, sessionID, user.ID, sessionToken, expiresAt)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-sid",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   h.cfg.SessionMaxAge,
	})

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie
	cookie, err := r.Cookie("__Host-sid")
	if err != nil {
		http.Error(w, "No active session", http.StatusUnauthorized)
		return
	}

	// Delete session from database
	_, err = h.db.Exec(`
		DELETE FROM sessions WHERE token = ?
	`, cookie.Value)
	if err != nil {
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-sid",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

// Me returns the current user
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie
	cookie, err := r.Cookie("__Host-sid")
	if err != nil {
		http.Error(w, "No active session", http.StatusUnauthorized)
		return
	}

	// Find session and user
	var user UserResponse
	err = h.db.QueryRow(`
		SELECT u.id, u.email, u.username, u.created_at
		FROM users u
		INNER JOIN sessions s ON u.id = s.user_id
		WHERE s.token = ? AND s.expires_at > datetime('now')
	`, cookie.Value).Scan(&user.ID, &user.Email, &user.Username, &user.CreatedAt)
	if err != nil {
		http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	// Return user
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// hashPassword hashes a password using Argon2id
func hashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash password with Argon2id (OWASP recommended parameters)
	// time=3, memory=64MB, parallelism=4, keyLen=32
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	// Encode as PHC string
	return encodeArgon2Hash(hash, salt), nil
}

// verifyPassword verifies a password against a hash
func verifyPassword(password, encodedHash string) bool {
	hash, salt, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return false
	}

	// Hash the provided password with the same parameters
	testHash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	// Compare hashes in constant time
	return subtle.ConstantTimeCompare(hash, testHash) == 1
}

// encodeArgon2Hash encodes an Argon2 hash as a PHC string
func encodeArgon2Hash(hash, salt []byte) string {
	// PHC format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		64*1024, // memory in KiB
		3,       // time
		4,       // parallelism
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}

// decodeArgon2Hash decodes a PHC string into hash and salt
func decodeArgon2Hash(encoded string) (hash, salt []byte, err error) {
	// Parse PHC string format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return nil, nil, fmt.Errorf("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return nil, nil, fmt.Errorf("unsupported hash algorithm")
	}

	// Decode salt
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	// Decode hash
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}

	return hash, salt, nil
}

// generateToken generates a random token
func generateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
