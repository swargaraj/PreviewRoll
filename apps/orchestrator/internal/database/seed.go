package database

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/swargaraj/previewroll/apps/orchestrator/internal/database/sqlc"
	"github.com/swargaraj/previewroll/apps/orchestrator/internal/password"
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

	pwd := generatePassword(16)
	hash, err := password.Hash(pwd)
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

	db.logger.Info("created default admin user", "username", "admin", "password", pwd)
	fmt.Printf("\n  Admin credentials — username: admin  password: %s\n\n", pwd)
	return nil
}

func generatePassword(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}
