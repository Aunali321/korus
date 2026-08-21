package tests

import (
	"context"
	"testing"

	"github.com/Aunali321/korus/internal/services"
)

func TestSearchAcrossKinds(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	search := services.NewSearchService(database)
	playlists := services.NewPlaylistService(database)

	if _, err := playlists.Create(ctx, userID, "Late Night Alpha", "", false, songIDs[:1]); err != nil {
		t.Fatalf("create playlist: %v", err)
	}

	res, err := search.Search(ctx, userID, "alpha", services.AllSearchKinds, 25, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res.Songs) != 1 || res.Songs[0].ID != songIDs[0] {
		t.Fatalf("songs = %v, want only Alpha", songIDList(res.Songs))
	}
	if len(res.Playlists) != 1 || res.Playlists[0].Name != "Late Night Alpha" {
		t.Fatalf("playlists = %d, want the matching playlist", len(res.Playlists))
	}

	res, err = search.Search(ctx, userID, "Artist Two", services.AllSearchKinds, 25, 0)
	if err != nil {
		t.Fatalf("search artist: %v", err)
	}
	if len(res.Artists) != 1 || res.Artists[0].Name != "Artist Two" {
		t.Fatalf("artists = %d, want Artist Two", len(res.Artists))
	}
	if len(res.Songs) != 2 {
		t.Fatalf("songs for artist = %v, want both of that artist's tracks", songIDList(res.Songs))
	}
}

// Punctuation is FTS5 query syntax. A raw query would either error or match
// nothing, so the service strips it before building the expression.
func TestSearchToleratesPunctuation(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	search := services.NewSearchService(database)

	for _, query := range []string{`"alpha"`, `alpha!`, `alpha - first album`, `(alpha)`, `alpha OR`, `alpha AND NOT beta`} {
		res, err := search.Search(ctx, userID, query, []services.SearchKind{services.KindSong}, 25, 0)
		if err != nil {
			t.Fatalf("search %q: %v", query, err)
		}
		if len(res.Songs) == 0 || res.Songs[0].ID != songIDs[0] {
			t.Fatalf("search %q = %v, want Alpha first", query, songIDList(res.Songs))
		}
	}
}

// A query where no song matches every term must fall back to the best partial
// matches rather than returning nothing.
func TestSearchFallsBackToPartialMatches(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	search := services.NewSearchService(database)

	res, err := search.Search(ctx, userID, "alpha nonexistentword", []services.SearchKind{services.KindSong}, 25, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res.Songs) == 0 {
		t.Fatalf("expected the partial-match fallback to return Alpha")
	}
	if res.Songs[0].ID != songIDs[0] {
		t.Fatalf("top result = %d, want %d", res.Songs[0].ID, songIDs[0])
	}
}

func TestSearchPrefixMatching(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	search := services.NewSearchService(database)

	res, err := search.Search(ctx, userID, "alph", []services.SearchKind{services.KindSong}, 25, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res.Songs) != 1 || res.Songs[0].ID != songIDs[0] {
		t.Fatalf("prefix search = %v, want Alpha", songIDList(res.Songs))
	}
}

func TestSearchOnlyRequestedKinds(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, _ := seedLibrary(t, database)
	search := services.NewSearchService(database)

	res, err := search.Search(ctx, userID, "first", []services.SearchKind{services.KindAlbum}, 25, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res.Albums) != 1 {
		t.Fatalf("albums = %d, want 1", len(res.Albums))
	}
	if len(res.Songs) != 0 || len(res.Artists) != 0 || len(res.Playlists) != 0 {
		t.Fatalf("expected only albums to be queried")
	}
}

func TestSearchExcludesOtherUsersPrivatePlaylists(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	search := services.NewSearchService(database)
	playlists := services.NewPlaylistService(database)

	res, err := database.ExecContext(ctx,
		`INSERT INTO users(username, password_hash, email) VALUES('other', 'x', 'other@example.com')`)
	if err != nil {
		t.Fatalf("seed other user: %v", err)
	}
	otherID, _ := res.LastInsertId()

	if _, err := playlists.Create(ctx, userID, "Secret Alpha", "", false, songIDs[:1]); err != nil {
		t.Fatalf("create private: %v", err)
	}
	if _, err := playlists.Create(ctx, userID, "Shared Alpha", "", true, songIDs[:1]); err != nil {
		t.Fatalf("create public: %v", err)
	}

	found, err := search.Search(ctx, otherID, "alpha", []services.SearchKind{services.KindPlaylist}, 25, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(found.Playlists) != 1 || found.Playlists[0].Name != "Shared Alpha" {
		t.Fatalf("playlists = %d, want only the public one", len(found.Playlists))
	}
}
