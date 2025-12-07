package migrations

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var migrationFiles embed.FS

type Migration struct {
	ID          int
	Filename    string
	Description string
	SQL         string
}

type Migrator struct {
	db *pgxpool.Pool
}

func NewMigrator(db *pgxpool.Pool) *Migrator {
	return &Migrator{db: db}
}

func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER PRIMARY KEY,
			filename VARCHAR(255) NOT NULL,
			description TEXT,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	_, err := m.db.Exec(ctx, query)
	return err
}

func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[int]bool, error) {
	applied := make(map[int]bool)

	rows, err := m.db.Query(ctx, "SELECT id FROM schema_migrations ORDER BY id")
	if err != nil {
		return applied, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return applied, err
		}
		applied[id] = true
	}

	return applied, rows.Err()
}

func (m *Migrator) loadMigrations() ([]Migration, error) {
	var migrations []Migration

	entries, err := migrationFiles.ReadDir(".")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Parse migration ID from filename (e.g., "001_initial_schema.sql" -> 1)
		parts := strings.Split(entry.Name(), "_")
		if len(parts) < 2 {
			continue
		}

		id, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		content, err := migrationFiles.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		// Extract description from SQL comment
		description := ""
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "-- Description:") {
				description = strings.TrimSpace(strings.TrimPrefix(line, "-- Description:"))
				break
			}
		}

		migrations = append(migrations, Migration{
			ID:          id,
			Filename:    entry.Name(),
			Description: description,
			SQL:         string(content),
		})
	}

	// Sort migrations by ID
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})

	return migrations, nil
}

func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Execute migration SQL
	if _, err := tx.Exec(ctx, migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration %d: %w", migration.ID, err)
	}

	// Record migration in schema_migrations table
	_, err = tx.Exec(ctx,
		"INSERT INTO schema_migrations (id, filename, description) VALUES ($1, $2, $3)",
		migration.ID, migration.Filename, migration.Description)
	if err != nil {
		return fmt.Errorf("failed to record migration %d: %w", migration.ID, err)
	}

	return tx.Commit(ctx)
}

func (m *Migrator) Migrate(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Load all migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Apply pending migrations
	var appliedCount int
	for _, migration := range migrations {
		if applied[migration.ID] {
			continue // Already applied
		}

		fmt.Printf("Applying migration %03d: %s\n", migration.ID, migration.Description)
		if err := m.applyMigration(ctx, migration); err != nil {
			return err
		}
		appliedCount++
	}

	if appliedCount == 0 {
		fmt.Println("No pending migrations to apply")
	} else {
		fmt.Printf("Successfully applied %d migrations\n", appliedCount)
	}

	return nil
}

func (m *Migrator) Status(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Load all migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	fmt.Println("Migration Status:")
	fmt.Println("================")

	for _, migration := range migrations {
		status := "PENDING"
		if applied[migration.ID] {
			status = "APPLIED"
		}
		fmt.Printf("%03d %-8s %s\n", migration.ID, status, migration.Description)
	}

	return nil
}
