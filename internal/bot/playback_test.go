package bot

import (
	"errors"
	"log/slog"
	"path/filepath"
	"testing"

	disgobot "github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/Aunali321/korus/internal/bot/player"
	"github.com/Aunali321/korus/internal/bot/store"
)

const (
	testGuild   = snowflake.ID(1)
	testChannel = snowflake.ID(100)
	otherRoom   = snowflake.ID(200)
	guest       = snowflake.ID(9)
)

func testBot(t *testing.T, links ...store.Link) *Bot {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	for _, link := range links {
		if err := st.Save(t.Context(), link); err != nil {
			t.Fatal(err)
		}
	}
	return &Bot{
		store:    st,
		accounts: newAccounts(st),
		client:   &disgobot.Client{Caches: cache.New(cache.WithCaches(cache.FlagVoiceStates))},
	}
}

func linkFor(id snowflake.ID, baseURL string) store.Link {
	return store.Link{DiscordID: id.String(), BaseURL: baseURL, Username: "u" + id.String(), Role: "user", Access: "a", Refresh: "r"}
}

func seat(b *Bot, user, channel snowflake.ID) {
	b.client.Caches.AddVoiceState(discord.VoiceState{GuildID: testGuild, UserID: user, ChannelID: &channel})
}

// A guest starting playback borrows a library from someone sitting in the same
// voice channel, which is the only way /play works before anyone linked has run it.
func TestBorrowLibraryPicksLinkedListenerInChannel(t *testing.T) {
	b := testBot(t, linkFor(20, "https://twenty.test"), linkFor(10, "https://ten.test"))
	seat(b, guest, testChannel)
	seat(b, 20, testChannel)
	seat(b, 30, testChannel) // present but never linked
	seat(b, 10, otherRoom)   // linked, wrong channel

	host, source, err := b.borrowLibrary(t.Context(), testGuild, testChannel, guest)
	if err != nil {
		t.Fatal(err)
	}
	if host != 20 {
		t.Fatalf("host = %d, want 20", host)
	}
	if got := source.BaseURL(); got != "https://twenty.test" {
		t.Fatalf("source = %q, want the listener in the channel", got)
	}
}

// Two linked listeners in one channel must resolve to the same library on every
// call, or a session would stream from whichever map iteration won that day.
func TestBorrowLibraryIsStable(t *testing.T) {
	b := testBot(t, linkFor(20, "https://twenty.test"), linkFor(15, "https://fifteen.test"))
	seat(b, guest, testChannel)
	seat(b, 20, testChannel)
	seat(b, 15, testChannel)

	for range 20 {
		host, _, err := b.borrowLibrary(t.Context(), testGuild, testChannel, guest)
		if err != nil || host != 15 {
			t.Fatalf("host = %d err = %v, want a stable 15", host, err)
		}
	}
}

// With nobody else linked the guest gets the same advice as before: link an
// account. Their own link is excluded, since a linked caller never gets here.
func TestBorrowLibraryWithoutAListenerReportsNoLink(t *testing.T) {
	b := testBot(t, linkFor(guest, "https://guest.test"))
	seat(b, guest, testChannel)
	seat(b, 30, testChannel)

	if _, _, err := b.borrowLibrary(t.Context(), testGuild, testChannel, guest); !errors.Is(err, store.ErrNoLink) {
		t.Fatalf("err = %v, want store.ErrNoLink", err)
	}
}

type stubCaller struct {
	guild snowflake.ID
	user  discord.User
}

func (c stubCaller) GuildID() *snowflake.ID          { return &c.guild }
func (c stubCaller) User() discord.User              { return c.user }
func (c stubCaller) Member() *discord.ResolvedMember { return nil }

func callerIn(b *Bot, user snowflake.ID) stubCaller {
	b.players = player.NewManager(b.client, "ffmpeg", slog.New(slog.DiscardHandler))
	return stubCaller{guild: testGuild, user: discord.User{ID: user, Username: "u" + user.String()}}
}

// The whole point of the borrow: a guest can open a session, and their plays are
// still recorded nowhere, so the account owner's history stays untouched.
func TestSessionForGuestBorrowsWithoutAListener(t *testing.T) {
	b := testBot(t, linkFor(20, "https://twenty.test"))
	e := callerIn(b, guest)
	seat(b, guest, testChannel)
	seat(b, 20, testChannel)

	current, err := b.session(t.Context(), e)
	if err != nil {
		t.Fatal(err)
	}
	if current.host != 20 {
		t.Fatalf("host = %d, want the account owner", current.host)
	}
	if got := current.source.BaseURL(); got != "https://twenty.test" {
		t.Fatalf("source = %q, want the borrowed library", got)
	}
	if current.listener != nil {
		t.Fatal("a guest's plays must not be recorded to the borrowed account")
	}
}

// A linked caller still opens their own session, borrowing nothing.
func TestSessionForLinkedCallerUsesOwnLibrary(t *testing.T) {
	b := testBot(t, linkFor(guest, "https://mine.test"), linkFor(20, "https://twenty.test"))
	e := callerIn(b, guest)
	seat(b, guest, testChannel)
	seat(b, 20, testChannel)

	current, err := b.session(t.Context(), e)
	if err != nil {
		t.Fatal(err)
	}
	if current.host != guest {
		t.Fatalf("host = %d, want the caller", current.host)
	}
	if got := current.source.BaseURL(); got != "https://mine.test" {
		t.Fatalf("source = %q, want the caller's own library", got)
	}
	if current.listener == nil {
		t.Fatal("a linked caller's plays must be recorded to their own account")
	}
}
