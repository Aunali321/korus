package bot

import (
	"context"
	"sync"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/store"
)

// Account is one Discord user's linked Korus account, with a client that keeps
// its own tokens fresh.
type Account struct {
	DiscordID string
	Username  string
	Role      string
	BaseURL   string
	Client    *korus.Client
}

// accounts maps Discord users to Korus clients. Every user links their own
// account, so nothing a user does can write to someone else's library or stats.
type accounts struct {
	store *store.Store

	mu     sync.Mutex
	cached map[string]*korus.Client
}

func newAccounts(s *store.Store) *accounts {
	return &accounts{store: s, cached: map[string]*korus.Client{}}
}

// Get returns the caller's account, or store.ErrNoLink when they never linked.
func (a *accounts) Get(ctx context.Context, discordID string) (Account, error) {
	link, err := a.store.Get(ctx, discordID)
	if err != nil {
		return Account{}, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	client, ok := a.cached[discordID]
	if !ok {
		client = korus.New(link.BaseURL, korus.Tokens{Access: link.Access, Refresh: link.Refresh}, a.sink(discordID))
		a.cached[discordID] = client
	}
	return Account{
		DiscordID: discordID,
		Username:  link.Username,
		Role:      link.Role,
		BaseURL:   link.BaseURL,
		Client:    client,
	}, nil
}

// Link replaces any existing link for a Discord user.
func (a *accounts) Link(ctx context.Context, discordID, baseURL string, user korus.User, tokens korus.Tokens) error {
	a.forget(discordID)
	return a.store.Save(ctx, store.Link{
		DiscordID: discordID,
		BaseURL:   baseURL,
		Username:  user.Username,
		Role:      user.Role,
		Access:    tokens.Access,
		Refresh:   tokens.Refresh,
	})
}

func (a *accounts) Unlink(ctx context.Context, discordID string) error {
	a.forget(discordID)
	return a.store.Delete(ctx, discordID)
}

func (a *accounts) forget(discordID string) {
	a.mu.Lock()
	if client, ok := a.cached[discordID]; ok {
		client.Revoke()
		delete(a.cached, discordID)
	}
	a.mu.Unlock()
}

// sink persists rotated tokens, and drops the link when the session is dead.
func (a *accounts) sink(discordID string) korus.TokenSink {
	return func(ctx context.Context, tokens korus.Tokens) error {
		if tokens.Access == "" {
			a.forget(discordID)
			return a.store.Delete(ctx, discordID)
		}
		return a.store.SaveTokens(ctx, discordID, tokens.Access, tokens.Refresh)
	}
}
