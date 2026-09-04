package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestLinkLifecycle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := t.Context()
	if _, err := store.Get(ctx, "42"); !errors.Is(err, ErrNoLink) {
		t.Fatalf("want ErrNoLink, got %v", err)
	}

	link := Link{DiscordID: "42", BaseURL: "https://korus.test", Username: "aun", Role: "admin", Access: "a1", Refresh: "r1"}
	if err := store.Save(ctx, link); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, "42")
	if err != nil || got != link {
		t.Fatalf("got %+v err=%v", got, err)
	}

	if err := store.SaveTokens(ctx, "42", "a2", "r2"); err != nil {
		t.Fatal(err)
	}
	got, err = store.Get(ctx, "42")
	if err != nil {
		t.Fatal(err)
	}
	if got.Access != "a2" || got.Refresh != "r2" || got.Username != "aun" {
		t.Fatalf("rotation clobbered the link: %+v", got)
	}

	relinked := Link{DiscordID: "42", BaseURL: "https://other.test", Username: "other", Role: "user", Access: "a3", Refresh: "r3"}
	if err := store.Save(ctx, relinked); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.Get(ctx, "42"); got != relinked {
		t.Fatalf("relink did not replace the row: %+v", got)
	}

	if err := store.Delete(ctx, "42"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "42"); !errors.Is(err, ErrNoLink) {
		t.Fatalf("want ErrNoLink after delete, got %v", err)
	}
}
