package bot

import (
	"log/slog"
	"testing"
	"time"

	"github.com/disgoorg/disgo/rest"
)

// Every acknowledgement is disposable now that the live panel is what stays on
// screen, so the only thing to pin is that they actually go.
func TestAcknowledgementIsRemoved(t *testing.T) {
	previous := noticeTTL
	noticeTTL = 20 * time.Millisecond
	t.Cleanup(func() { noticeTTL = previous })

	b := &Bot{log: slog.New(slog.DiscardHandler)}
	removed := make(chan struct{}, 1)
	b.expire(func(...rest.RequestOpt) error { removed <- struct{}{}; return nil })

	select {
	case <-removed:
	case <-time.After(2 * time.Second):
		t.Fatal("acknowledgement was never removed")
	}
}
