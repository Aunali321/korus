package tests

import (
	"context"
	"os"
	"testing"
)

// TestPlaylistPositionMigration runs the shipped migration against the mixed
// numbering the old code produced: unix timestamps from the HTTP add endpoint
// alongside small sequential values from reorder and the AI tools.
func TestPlaylistPositionMigration(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)

	res, err := database.ExecContext(ctx,
		`INSERT INTO playlists(user_id, name) VALUES(?, 'Legacy')`, userID)
	if err != nil {
		t.Fatalf("seed playlist: %v", err)
	}
	playlistID, _ := res.LastInsertId()

	legacy := []struct {
		song     int64
		position int64
	}{
		{songIDs[0], 0},
		{songIDs[1], 1},
		{songIDs[2], 1755300000},
		{songIDs[3], 1755300001},
	}
	for _, row := range legacy {
		if _, err := database.ExecContext(ctx,
			`INSERT INTO playlist_songs(playlist_id, song_id, position) VALUES(?, ?, ?)`,
			playlistID, row.song, row.position); err != nil {
			t.Fatalf("seed playlist_songs: %v", err)
		}
	}

	statement, err := os.ReadFile("../internal/db/migrations/000006_playlist_positions.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := database.ExecContext(ctx, string(statement)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	assertPositions(t, database, playlistID, songIDs)
}
