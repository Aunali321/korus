// Package store persists the link between a Discord user and their Korus account.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// ErrNoLink reports that a Discord user has not linked a Korus account.
var ErrNoLink = errors.New("no linked korus account")

// Link is one Discord user's Korus session. Every user links their own account,
// so personal data and listening history stay attributed to whoever earned it.
type Link struct {
	DiscordID string
	BaseURL   string
	Username  string
	Role      string
	Access    string
	Refresh   string
}

const schema = `
CREATE TABLE IF NOT EXISTS links (
	discord_id    TEXT PRIMARY KEY,
	base_url      TEXT NOT NULL,
	username      TEXT NOT NULL,
	role          TEXT NOT NULL,
	access_token  TEXT NOT NULL,
	refresh_token TEXT NOT NULL,
	linked_at     TEXT NOT NULL
);`

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open bot db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(time.Hour)
	if _, err := db.Exec(`PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;`); err != nil {
		return nil, fmt.Errorf("set pragmas: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Get returns the link for a Discord user, or ErrNoLink.
func (s *Store) Get(ctx context.Context, discordID string) (Link, error) {
	link := Link{DiscordID: discordID}
	err := s.db.QueryRowContext(ctx, `
		SELECT base_url, username, role, access_token, refresh_token
		FROM links WHERE discord_id = ?
	`, discordID).Scan(&link.BaseURL, &link.Username, &link.Role, &link.Access, &link.Refresh)
	if errors.Is(err, sql.ErrNoRows) {
		return Link{}, ErrNoLink
	}
	if err != nil {
		return Link{}, fmt.Errorf("get link: %w", err)
	}
	return link, nil
}

func (s *Store) Save(ctx context.Context, link Link) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO links (discord_id, base_url, username, role, access_token, refresh_token, linked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(discord_id) DO UPDATE SET
			base_url = excluded.base_url,
			username = excluded.username,
			role = excluded.role,
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			linked_at = excluded.linked_at
	`, link.DiscordID, link.BaseURL, link.Username, link.Role, link.Access, link.Refresh, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save link: %w", err)
	}
	return nil
}

// SaveTokens persists rotated tokens without touching the rest of the link.
func (s *Store) SaveTokens(ctx context.Context, discordID, access, refresh string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE links SET access_token = ?, refresh_token = ? WHERE discord_id = ?
	`, access, refresh, discordID)
	if err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, discordID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM links WHERE discord_id = ?`, discordID); err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	return nil
}
