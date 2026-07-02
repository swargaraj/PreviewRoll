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
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database/sqlc"
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
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserResponse represents the user response
type UserResponse struct {
	ID        int64  `json:"id"`
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

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	hash, err := hashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := h.db.Q().CreateUser(r.Context(), sqlc.CreateUserParams{
		Username:     req.Username,
		PasswordHash: hash,
	})
	if err != nil {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	resp := UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Time.UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	creds, err := h.db.Q().GetUserCredentials(r.Context(), req.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !verifyPassword(req.Password, creds.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	sessionToken, err := generateToken(32)
	if err != nil {
		http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
		return
	}

	sessionID, err := generateToken(16)
	if err != nil {
		http.Error(w, "Failed to generate session ID", http.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(time.Duration(h.cfg.SessionMaxAge) * time.Second)

	err = h.db.Q().CreateSession(r.Context(), sqlc.CreateSessionParams{
		ID:        sessionID,
		UserID:    creds.ID,
		Token:     sessionToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-sid",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   h.cfg.SessionMaxAge,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("__Host-sid")
	if err != nil {
		http.Error(w, "No active session", http.StatusUnauthorized)
		return
	}

	err = h.db.Q().DeleteSession(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-sid",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out successfully",
	})
}

// Me returns the current user
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("__Host-sid")
	if err != nil {
		http.Error(w, "No active session", http.StatusUnauthorized)
		return
	}

	user, err := h.db.Q().GetUserBySessionToken(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	resp := UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Time.UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// hashPassword hashes a password using Argon2id
func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	return encodeArgon2Hash(hash, salt), nil
}

// verifyPassword verifies a password against a hash
func verifyPassword(password, encodedHash string) bool {
	hash, salt, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		return false
	}

	testHash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	return subtle.ConstantTimeCompare(hash, testHash) == 1
}

// encodeArgon2Hash encodes an Argon2 hash as a PHC string
func encodeArgon2Hash(hash, salt []byte) string {
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		64*1024,
		3,
		4,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}

// decodeArgon2Hash decodes a PHC string into hash and salt
func decodeArgon2Hash(encoded string) (hash, salt []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return nil, nil, fmt.Errorf("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return nil, nil, fmt.Errorf("unsupported hash algorithm")
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

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
