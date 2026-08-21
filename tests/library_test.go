package tests

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	"github.com/Aunali321/korus/internal/models"
	"github.com/Aunali321/korus/internal/services"
)

// seedLibrary creates one user, two artists, two albums and four songs, and
// returns the user id plus the song ids in insertion order.
func seedLibrary(t *testing.T, database *sql.DB) (userID int64, songIDs []int64) {
	t.Helper()
	ctx := context.Background()

	res, err := database.ExecContext(ctx,
		`INSERT INTO users(username, password_hash, email) VALUES('tester', 'x', 'tester@example.com')`)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	userID, _ = res.LastInsertId()

	artistIDs := make([]int64, 0, 2)
	for _, name := range []string{"Artist One", "Artist Two"} {
		res, err := database.ExecContext(ctx, `INSERT INTO artists(name) VALUES(?)`, name)
		if err != nil {
			t.Fatalf("seed artist: %v", err)
		}
		id, _ := res.LastInsertId()
		artistIDs = append(artistIDs, id)
	}

	// The second album deliberately has no cover_path and no year, the shape
	// that used to be dropped by the album scanner.
	albumIDs := make([]int64, 0, 2)
	res, err = database.ExecContext(ctx,
		`INSERT INTO albums(artist_id, title, year, cover_path) VALUES(?, 'First Album', 1999, '/covers/a.jpg')`, artistIDs[0])
	if err != nil {
		t.Fatalf("seed album: %v", err)
	}
	id, _ := res.LastInsertId()
	albumIDs = append(albumIDs, id)

	res, err = database.ExecContext(ctx,
		`INSERT INTO albums(artist_id, title) VALUES(?, 'Second Album')`, artistIDs[1])
	if err != nil {
		t.Fatalf("seed album: %v", err)
	}
	id, _ = res.LastInsertId()
	albumIDs = append(albumIDs, id)

	titles := []struct {
		title    string
		album    int64
		artist   int64
		duration int
	}{
		{"Alpha", albumIDs[0], artistIDs[0], 180000},
		{"Beta", albumIDs[0], artistIDs[0], 240000},
		{"Gamma", albumIDs[1], artistIDs[1], 200000},
		{"Delta", albumIDs[1], artistIDs[1], 300000},
	}
	for i, s := range titles {
		res, err := database.ExecContext(ctx,
			`INSERT INTO songs(album_id, title, track_number, duration_ms, file_path) VALUES(?, ?, ?, ?, ?)`,
			s.album, s.title, i+1, s.duration, "/music/"+s.title+".flac")
		if err != nil {
			t.Fatalf("seed song: %v", err)
		}
		songID, _ := res.LastInsertId()
		songIDs = append(songIDs, songID)
		if _, err := database.ExecContext(ctx,
			`INSERT INTO song_artists(song_id, artist_id, role, position) VALUES(?, ?, 'primary', 0)`,
			songID, s.artist); err != nil {
			t.Fatalf("seed song_artists: %v", err)
		}
		// Mirrors what the scanner writes so search runs against a real index.
		artistName := "Artist One"
		albumTitle := "First Album"
		if s.album == albumIDs[1] {
			artistName, albumTitle = "Artist Two", "Second Album"
		}
		if _, err := database.ExecContext(ctx,
			`INSERT INTO songs_fts(rowid, song_id, title, artist_name, album_title) VALUES(?, ?, ?, ?, ?)`,
			songID, songID, s.title, artistName, albumTitle); err != nil {
			t.Fatalf("seed songs_fts: %v", err)
		}
	}
	return userID, songIDs
}

func TestPlaylistPositionsStayDense(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	playlists := services.NewPlaylistService(database)

	playlist, err := playlists.Create(ctx, userID, "Mix", "notes", false, songIDs[:2])
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if playlist.SongCount != 2 {
		t.Fatalf("song count = %d, want 2", playlist.SongCount)
	}

	if err := playlists.SetSongs(ctx, userID, playlist.ID, songIDs[2:], services.ModeAppend); err != nil {
		t.Fatalf("append: %v", err)
	}
	assertPositions(t, database, playlist.ID, songIDs)

	// Appending a song already present must not duplicate it or leave a gap.
	if err := playlists.SetSongs(ctx, userID, playlist.ID, []int64{songIDs[0]}, services.ModeAppend); err != nil {
		t.Fatalf("append duplicate: %v", err)
	}
	assertPositions(t, database, playlist.ID, songIDs)

	if err := playlists.RemoveSongs(ctx, userID, playlist.ID, []int64{songIDs[1]}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	assertPositions(t, database, playlist.ID, []int64{songIDs[0], songIDs[2], songIDs[3]})

	reordered := []int64{songIDs[3], songIDs[0], songIDs[2]}
	if err := playlists.Reorder(ctx, userID, playlist.ID, reordered); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	assertPositions(t, database, playlist.ID, reordered)

	songs, err := playlists.Songs(ctx, playlist.ID)
	if err != nil {
		t.Fatalf("songs: %v", err)
	}
	for i, want := range reordered {
		if songs[i].ID != want {
			t.Fatalf("song %d = %d, want %d", i, songs[i].ID, want)
		}
	}
	if len(songs[0].Artists) == 0 {
		t.Fatalf("expected credited performers to be populated")
	}
	if songs[0].Album == nil || songs[0].Album.Artist == nil {
		t.Fatalf("expected album and album artist to be populated")
	}
}

func TestPlaylistReorderRejectsPartialList(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	playlists := services.NewPlaylistService(database)

	playlist, err := playlists.Create(ctx, userID, "Mix", "", false, songIDs)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := playlists.Reorder(ctx, userID, playlist.ID, songIDs[:2]); err != services.ErrInvalid {
		t.Fatalf("reorder with partial list = %v, want ErrInvalid", err)
	}
	assertPositions(t, database, playlist.ID, songIDs)
}

func TestPlaylistOwnership(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	playlists := services.NewPlaylistService(database)

	res, err := database.ExecContext(ctx,
		`INSERT INTO users(username, password_hash, email) VALUES('other', 'x', 'other@example.com')`)
	if err != nil {
		t.Fatalf("seed other user: %v", err)
	}
	otherID, _ := res.LastInsertId()

	playlist, err := playlists.Create(ctx, userID, "Private", "", false, songIDs[:1])
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := playlists.Get(ctx, otherID, playlist.ID); err != services.ErrForbidden {
		t.Fatalf("get private playlist as other user = %v, want ErrForbidden", err)
	}
	if err := playlists.SetSongs(ctx, otherID, playlist.ID, songIDs, services.ModeAppend); err != services.ErrForbidden {
		t.Fatalf("write as other user = %v, want ErrForbidden", err)
	}
	if err := playlists.Delete(ctx, otherID, playlist.ID); err != services.ErrForbidden {
		t.Fatalf("delete as other user = %v, want ErrForbidden", err)
	}
	if _, err := playlists.Get(ctx, otherID, 9999); err != services.ErrNotFound {
		t.Fatalf("get missing playlist = %v, want ErrNotFound", err)
	}
}

func TestLibrarySlices(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	library := services.NewLibraryService(database)

	// songs[0] played three times fully, songs[1] played twice and abandoned
	// both times, songs[2] played once. songs[3] never played.
	plays := []struct {
		song       int64
		count      int
		completion float64
	}{
		{songIDs[0], 3, 1.0},
		{songIDs[1], 2, 0.1},
		{songIDs[2], 1, 0.9},
	}
	for _, p := range plays {
		for i := 0; i < p.count; i++ {
			if _, err := database.ExecContext(ctx,
				`INSERT INTO play_history(user_id, song_id, completion_rate) VALUES(?, ?, ?)`,
				userID, p.song, p.completion); err != nil {
				t.Fatalf("seed play: %v", err)
			}
		}
	}

	top, err := library.UserSongs(ctx, userID, services.SliceTop, 0, 10)
	if err != nil {
		t.Fatalf("top: %v", err)
	}
	if len(top) != 3 || top[0].ID != songIDs[0] {
		t.Fatalf("top = %v, want songs[0] first of 3", songIDList(top))
	}

	skipped, err := library.UserSongs(ctx, userID, services.SliceSkipped, 0, 10)
	if err != nil {
		t.Fatalf("skipped: %v", err)
	}
	if len(skipped) != 1 || skipped[0].ID != songIDs[1] {
		t.Fatalf("skipped = %v, want only songs[1]", songIDList(skipped))
	}

	unplayed, err := library.UserSongs(ctx, userID, services.SliceUnplayed, 0, 10)
	if err != nil {
		t.Fatalf("unplayed: %v", err)
	}
	if len(unplayed) != 1 || unplayed[0].ID != songIDs[3] {
		t.Fatalf("unplayed = %v, want only songs[3]", songIDList(unplayed))
	}

	// A window shorter than the play's age excludes it; the plays above are
	// stamped now, so a one-day window still includes them.
	windowed, err := library.UserSongs(ctx, userID, services.SliceTop, 1, 10)
	if err != nil {
		t.Fatalf("windowed top: %v", err)
	}
	if len(windowed) != 3 {
		t.Fatalf("windowed top = %v, want 3", songIDList(windowed))
	}
}

func TestSetFavorite(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	library := services.NewLibraryService(database)

	if err := library.SetFavorite(ctx, userID, services.EntitySong, songIDs[0], true); err != nil {
		t.Fatalf("favorite: %v", err)
	}
	if !library.IsFavorite(ctx, userID, services.EntitySong, songIDs[0]) {
		t.Fatalf("expected song to be favorited")
	}

	// Favouriting twice is idempotent rather than an error.
	if err := library.SetFavorite(ctx, userID, services.EntitySong, songIDs[0], true); err != nil {
		t.Fatalf("favorite twice: %v", err)
	}

	favorites, err := library.UserSongs(ctx, userID, services.SliceFavorites, 0, 10)
	if err != nil {
		t.Fatalf("favorites: %v", err)
	}
	if len(favorites) != 1 || favorites[0].ID != songIDs[0] {
		t.Fatalf("favorites = %v, want only songs[0]", songIDList(favorites))
	}

	if err := library.SetFavorite(ctx, userID, services.EntitySong, 9999, true); err != services.ErrNotFound {
		t.Fatalf("favorite missing song = %v, want ErrNotFound", err)
	}

	if err := library.SetFavorite(ctx, userID, services.EntitySong, songIDs[0], false); err != nil {
		t.Fatalf("unfavorite: %v", err)
	}
	if library.IsFavorite(ctx, userID, services.EntitySong, songIDs[0]) {
		t.Fatalf("expected song to be unfavorited")
	}
}

func TestAlbumsSurviveMissingCoverAndYear(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	_, _ = seedLibrary(t, database)
	library := services.NewLibraryService(database)

	albums, err := library.Albums(ctx, 10)
	if err != nil {
		t.Fatalf("albums: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("albums = %d, want 2 (the one without cover_path must not be dropped)", len(albums))
	}
}

func TestSongsPreserveRequestedOrder(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	_, songIDs := seedLibrary(t, database)
	library := services.NewLibraryService(database)

	want := []int64{songIDs[2], songIDs[0], 9999, songIDs[3]}
	songs, err := library.Songs(ctx, want)
	if err != nil {
		t.Fatalf("songs: %v", err)
	}
	if len(songs) != 3 {
		t.Fatalf("songs = %v, want the 3 that resolve", songIDList(songs))
	}
	for i, id := range []int64{songIDs[2], songIDs[0], songIDs[3]} {
		if songs[i].ID != id {
			t.Fatalf("song %d = %d, want %d", i, songs[i].ID, id)
		}
	}
}

func TestSongDetailIncludesLyrics(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	_, songIDs := seedLibrary(t, database)
	library := services.NewLibraryService(database)

	if _, err := database.ExecContext(ctx, `UPDATE songs SET lyrics = 'a line' WHERE id = ?`, songIDs[0]); err != nil {
		t.Fatalf("set lyrics: %v", err)
	}

	song, err := library.Song(ctx, songIDs[0])
	if err != nil {
		t.Fatalf("song: %v", err)
	}
	if song.Lyrics != "a line" {
		t.Fatalf("lyrics = %q, want %q", song.Lyrics, "a line")
	}
	if song.Album == nil || len(song.Artists) == 0 {
		t.Fatalf("expected album and artists on song detail")
	}
	if _, err := library.Song(ctx, 9999); err != services.ErrNotFound {
		t.Fatalf("missing song = %v, want ErrNotFound", err)
	}
}

// SQLite rejects a statement binding more than 32766 variables, so a long
// playlist has to be read in batches rather than one IN clause.
func TestSongsHandlesMoreIDsThanSQLiteCanBind(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	library := services.NewLibraryService(database)

	res, err := database.ExecContext(ctx, `INSERT INTO albums(title) VALUES('Bulk')`)
	if err != nil {
		t.Fatalf("seed album: %v", err)
	}
	albumID, _ := res.LastInsertId()

	const count = 40000
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	ids := make([]int64, 0, count)
	for i := 0; i < count; i++ {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO songs(album_id, title, file_path) VALUES(?, ?, ?)`,
			albumID, "Track", "/music/bulk/"+strconv.Itoa(i)+".flac")
		if err != nil {
			t.Fatalf("seed song %d: %v", i, err)
		}
		id, _ := res.LastInsertId()
		ids = append(ids, id)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	songs, err := library.Songs(ctx, ids)
	if err != nil {
		t.Fatalf("songs: %v", err)
	}
	if len(songs) != count {
		t.Fatalf("got %d songs, want %d", len(songs), count)
	}
	if songs[0].ID != ids[0] || songs[count-1].ID != ids[count-1] {
		t.Fatalf("batching lost the requested order")
	}
}

func TestSimilarSongsScoresSeedAlbumFirst(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	_, songIDs := seedLibrary(t, database)
	library := services.NewLibraryService(database)

	// songs[0] and songs[1] share an album, so seeding from songs[0] must rank
	// songs[1] above the other album's tracks.
	songs, err := library.SimilarSongs(ctx, songIDs[0], 10)
	if err != nil {
		t.Fatalf("similar: %v", err)
	}
	if len(songs) != 3 {
		t.Fatalf("similar = %v, want the 3 non-seed songs", songIDList(songs))
	}
	if songs[0].ID != songIDs[1] {
		t.Fatalf("top similar = %d, want %d (same album as seed)", songs[0].ID, songIDs[1])
	}
	for _, s := range songs {
		if s.ID == songIDs[0] {
			t.Fatalf("seed song must be excluded")
		}
	}

	if _, err := library.SimilarSongs(ctx, 9999, 10); err != services.ErrNotFound {
		t.Fatalf("similar for missing seed = %v, want ErrNotFound", err)
	}
}

func assertPositions(t *testing.T, database *sql.DB, playlistID int64, want []int64) {
	t.Helper()
	rows, err := database.QueryContext(context.Background(),
		`SELECT song_id, position FROM playlist_songs WHERE playlist_id = ? ORDER BY position`, playlistID)
	if err != nil {
		t.Fatalf("read positions: %v", err)
	}
	defer rows.Close()

	got := []int64{}
	for i := 0; rows.Next(); i++ {
		var songID int64
		var position int
		if err := rows.Scan(&songID, &position); err != nil {
			t.Fatalf("scan position: %v", err)
		}
		if position != i {
			t.Fatalf("position at index %d = %d, want %d (positions must stay dense)", i, position, i)
		}
		got = append(got, songID)
	}
	if len(got) != len(want) {
		t.Fatalf("playlist has %d songs, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("song at %d = %d, want %d", i, got[i], want[i])
		}
	}
}

func songIDList(songs []models.Song) []int64 {
	ids := make([]int64, len(songs))
	for i, s := range songs {
		ids[i] = s.ID
	}
	return ids
}
