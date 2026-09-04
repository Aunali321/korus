package bot

import (
	"log/slog"
	"testing"
	"time"

	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
)

func expiringBot(t *testing.T, ttl time.Duration) *Bot {
	t.Helper()
	previous := noticeTTL
	noticeTTL = ttl
	t.Cleanup(func() { noticeTTL = previous })
	return &Bot{log: slog.New(slog.DiscardHandler)}
}

const message = snowflake.ID(77)

func TestUntouchedConfirmationIsRemoved(t *testing.T) {
	b := expiringBot(t, 20*time.Millisecond)
	removed := make(chan struct{}, 1)

	b.expire(message, func(...rest.RequestOpt) error { removed <- struct{}{}; return nil }, true)

	select {
	case <-removed:
	case <-time.After(2 * time.Second):
		t.Fatal("confirmation was never removed")
	}
}

// A reply that started playback keeps its controls and must never be scheduled.
func TestLastingReplyIsNeverRemoved(t *testing.T) {
	b := expiringBot(t, 10*time.Millisecond)
	removed := make(chan struct{}, 1)

	b.expire(message, func(...rest.RequestOpt) error { removed <- struct{}{}; return nil }, false)

	select {
	case <-removed:
		t.Fatal("a lasting reply was deleted")
	case <-time.After(80 * time.Millisecond):
	}
}
