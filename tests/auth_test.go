package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/services"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(context.Background(), database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func TestAuthRegisterLoginRefresh(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	auth := services.NewAuthService(database, []byte("secret"), time.Hour, 24*time.Hour)

	user, tokens, err := auth.Register(context.Background(), "tester", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tokens.Access == "" || tokens.Refresh == "" {
		t.Fatalf("expected access and refresh tokens")
	}

	user2, tokens2, err := auth.Login(context.Background(), "tester", "password123")
	if err != nil || user2.ID != user.ID {
		t.Fatalf("login failed: %v", err)
	}
	if tokens2.Access == "" || tokens2.Refresh == "" {
		t.Fatalf("expected tokens on login")
	}

	_, tokens3, err := auth.Refresh(context.Background(), tokens2.Refresh)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if tokens3.Access == tokens2.Access {
		t.Fatalf("access token should rotate on refresh")
	}
}
