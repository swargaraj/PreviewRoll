package database

import (
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
	HasDown     bool
}

// RunMigrations runs all database migrations from the embedded filesystem
func RunMigrations(db *DB) error {
	db.logger.Info("running database migrations")

	// Create migrations table if it doesn't exist
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Load migrations from embedded files
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Run pending migrations
	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue
		}

		if !m.HasDown {
			db.logger.Warn("migration has no down file", "version", m.Version, "description", m.Description)
		}

		db.logger.Info("applying migration", "version", m.Version, "description", m.Description)
		if err := applyMigration(db, m); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", m.Version, err)
		}
	}

	db.logger.Info("database migrations completed")
	return nil
}

func applyMigration(db *DB, m Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Apply migration
	if _, err := tx.Exec(m.Up); err != nil {
		return fmt.Errorf("migration up failed: %w", err)
	}

	// Record migration
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.Version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// loadMigrations loads migrations from the embedded filesystem
func loadMigrations() ([]Migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	// Group up and down migrations
	upFiles := make(map[int]string)
	downFiles := make(map[int]string)
	descriptions := make(map[int]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse version from filename: 000001_create_users.up.sql
		versionStr, description, ok := strings.Cut(name, "_")
		if !ok {
			continue
		}

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			continue
		}

		// Strip the .up.sql or .down.sql suffix from description
		description = strings.TrimSuffix(description, ".up.sql")
		description = strings.TrimSuffix(description, ".down.sql")

		// Store description (from whichever file we see first)
		if _, exists := descriptions[version]; !exists {
			descriptions[version] = description
		}

		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", name, err)
		}

		if strings.HasSuffix(name, ".up.sql") {
			upFiles[version] = string(content)
		} else if strings.HasSuffix(name, ".down.sql") {
			downFiles[version] = string(content)
		}
	}

	// Create migrations
	var migrations []Migration
	for version, up := range upFiles {
		_, hasDown := downFiles[version]
		migrations = append(migrations, Migration{
			Version:     version,
			Description: descriptions[version],
			Up:          up,
			Down:        downFiles[version],
			HasDown:     hasDown,
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}
