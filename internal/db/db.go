package db

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite
	db.SetConnMaxLifetime(time.Hour)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;`); err != nil {
		return nil, fmt.Errorf("set pragmas: %w", err)
	}
	return db, nil
}

// Migrate applies the embedded schema.
func Migrate(ctx context.Context, db *sql.DB) error {
	stmts := strings.Split(schemaSQL, ";\n")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec schema: %w", err)
		}
	}
	return nil
}

// SeedAdmin inserts the first admin user if none exist.
func SeedAdmin(ctx context.Context, db *sql.DB, username, email, passwordHash string) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}
	if username == "" || passwordHash == "" || email == "" {
		return errors.New("admin seed requires username, email, password hash")
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, email, role)
		VALUES (?, ?, ?, 'admin')
	`, username, passwordHash, email)
	return err
}
