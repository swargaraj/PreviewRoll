package database

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/argon2"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database/sqlc"
)

// SeedAdminIfNeeded creates a default admin user if no users exist.
func (db *DB) SeedAdminIfNeeded() error {
	count, err := db.Q().CountUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count > 0 {
		db.logger.Info("users exist, skipping admin seed")
		return nil
	}

	password := generatePassword(16)
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = db.Q().CreateUser(context.Background(), sqlc.CreateUserParams{
		Username:     "admin",
		PasswordHash: hash,
	})
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	db.logger.Info("created default admin user", "username", "admin", "password", password)
	fmt.Printf("\n  Admin credentials — username: admin  password: %s\n\n", password)
	return nil
}

func generatePassword(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		64*1024,
		3,
		4,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}
