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

// A queue confirmation carries the player controls, so using one turns it into
// the view the listener is working with. Deleting it then is the bug this guards.
func TestKeepCancelsRemoval(t *testing.T) {
	b := expiringBot(t, 20*time.Millisecond)
	removed := make(chan struct{}, 1)

	b.expire(message, func(...rest.RequestOpt) error { removed <- struct{}{}; return nil }, true)
	b.keep(message)

	select {
	case <-removed:
		t.Fatal("a confirmation the listener acted on was still deleted")
	case <-time.After(100 * time.Millisecond):
	}

	b.expiryMu.Lock()
	defer b.expiryMu.Unlock()
	if len(b.expiry) != 0 {
		t.Fatalf("registry still holds %d entries", len(b.expiry))
	}
}

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

// keep on a message with nothing pending is a no-op, since every control click
// calls it whether the message was transient or not.
func TestKeepUnknownMessage(t *testing.T) {
	expiringBot(t, time.Second).keep(snowflake.ID(1234))
}
