package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/models"
)

// LibraryService owns every read of the music catalogue and the user's
// relationship to it (favorites, follows, listening history). It is the single
// source of these queries for both the HTTP handlers and the AI tools.
type LibraryService struct {
	db *sql.DB
}

func NewLibraryService(database *sql.DB) *LibraryService {
	return &LibraryService{db: database}
}

// Slice names a user-scoped view of the library.
type Slice string

const (
	SliceTop       Slice = "top"
	SliceRecent    Slice = "recent"
	SliceFavorites Slice = "favorites"
	SliceSkipped   Slice = "skipped"
	SliceUnplayed  Slice = "unplayed"
)

// EntityKind names a favouritable entity.
type EntityKind string

const (
	EntitySong   EntityKind = "song"
	EntityAlbum  EntityKind = "album"
	EntityArtist EntityKind = "artist"
)

// Service-layer errors the HTTP and AI callers translate into their own
// vocabulary.
var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
	ErrInvalid   = errors.New("invalid request")
)

// Song returns one song with its lyrics, album and credited performers.
func (s *LibraryService) Song(ctx context.Context, id int64) (models.Song, error) {
	var song models.Song
	var track, duration sql.NullInt64
	var lyrics, lyricsSynced, mbid sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, album_id, title, track_number, duration_ms / 1000, file_path, lyrics, lyrics_synced, mbid
		FROM songs WHERE id = ?
	`, id).Scan(&song.ID, &song.AlbumID, &song.Title, &track, &duration, &song.FilePath,
		&lyrics, &lyricsSynced, &mbid)
	if err != nil {
		return song, ErrNotFound
	}

	if track.Valid {
		t := int(track.Int64)
		song.TrackNumber = &t
	}
	if duration.Valid {
		d := int(duration.Int64)
		song.Duration = &d
	}
	song.Lyrics = lyrics.String
	song.LyricsSynced = lyricsSynced.String
	if mbid.Valid {
		song.MBID = &mbid.String
	}

	if album, err := s.Album(ctx, song.AlbumID); err == nil {
		song.Album = &album
	}
	song.Artists, _ = db.ArtistsForSong(ctx, s.db, song.ID)
	return song, nil
}

// Songs returns the given songs in the order the ids were passed. Missing ids
// are skipped rather than erroring, since callers resolve ids that a scan may
// have removed since.
func (s *LibraryService) Songs(ctx context.Context, ids []int64) ([]models.Song, error) {
	if len(ids) == 0 {
		return []models.Song{}, nil
	}
	byID := make(map[int64]models.Song, len(ids))
	for _, batch := range db.Chunks(ids) {
		found, err := s.querySongs(ctx, `
			SELECT `+db.SongColumns+`
			FROM songs s
			`+db.SongJoins+`
			WHERE s.id IN (`+db.Placeholders(len(batch))+`)
		`, db.Args(batch)...)
		if err != nil {
			return nil, err
		}
		for _, song := range found {
			byID[song.ID] = song
		}
	}
	ordered := make([]models.Song, 0, len(ids))
	for _, id := range ids {
		if song, ok := byID[id]; ok {
			ordered = append(ordered, song)
		}
	}
	return ordered, nil
}

// ArtistsByIDs returns the given artists in the order the ids were passed.
func (s *LibraryService) ArtistsByIDs(ctx context.Context, ids []int64) ([]models.Artist, error) {
	found := map[int64]models.Artist{}
	for _, batch := range db.Chunks(ids) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+db.ArtistColumns+` FROM artists a WHERE a.id IN (`+db.Placeholders(len(batch))+`)`,
			db.Args(batch)...)
		if err != nil {
			return nil, err
		}
		artists, err := db.ScanArtists(rows)
		rows.Close()
		if err != nil {
			return nil, err
		}
		for _, a := range artists {
			found[a.ID] = a
		}
	}
	return inRequestedOrder(ids, found), nil
}

// AlbumsByIDs returns the given albums in the order the ids were passed.
func (s *LibraryService) AlbumsByIDs(ctx context.Context, ids []int64) ([]models.Album, error) {
	found := map[int64]models.Album{}
	for _, batch := range db.Chunks(ids) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+db.AlbumColumns+` FROM albums al `+db.AlbumJoins+
				` WHERE al.id IN (`+db.Placeholders(len(batch))+`)`,
			db.Args(batch)...)
		if err != nil {
			return nil, err
		}
		albums, err := db.ScanAlbums(rows)
		rows.Close()
		if err != nil {
			return nil, err
		}
		for _, al := range albums {
			found[al.ID] = al
		}
	}
	return inRequestedOrder(ids, found), nil
}

func inRequestedOrder[T any](ids []int64, found map[int64]T) []T {
	ordered := make([]T, 0, len(ids))
	for _, id := range ids {
		if item, ok := found[id]; ok {
			ordered = append(ordered, item)
		}
	}
	return ordered
}

// Album returns one album with its credited artist.
func (s *LibraryService) Album(ctx context.Context, id int64) (models.Album, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+db.AlbumColumns+`
		FROM albums al
		`+db.AlbumJoins+`
		WHERE al.id = ?
	`, id)
	album, err := db.ScanAlbum(row)
	if err != nil {
		return album, ErrNotFound
	}
	return album, nil
}

// Albums returns the most recently added albums.
func (s *LibraryService) Albums(ctx context.Context, limit int) ([]models.Album, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+db.AlbumColumns+`
		FROM albums al
		`+db.AlbumJoins+`
		ORDER BY al.created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return db.ScanAlbums(rows)
}

// AlbumsByArtist returns albums credited to the artist plus compilations where
// they perform on at least one track. Without the second half, an artist's
// page loses every compilation they appear on.
func (s *LibraryService) AlbumsByArtist(ctx context.Context, artistID int64) ([]models.Album, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+db.AlbumColumns+`
		FROM albums al
		`+db.AlbumJoins+`
		WHERE al.artist_id = ?
		   OR EXISTS (
		        SELECT 1 FROM songs s
		        JOIN song_artists sa ON sa.song_id = s.id AND sa.role = 'primary'
		        WHERE s.album_id = al.id AND sa.artist_id = ?
		      )
		ORDER BY al.year IS NULL, al.year DESC, al.title
	`, artistID, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return db.ScanAlbums(rows)
}

// Artist returns one artist.
func (s *LibraryService) Artist(ctx context.Context, id int64) (models.Artist, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+db.ArtistColumns+` FROM artists a WHERE a.id = ?`, id)
	artist, err := db.ScanArtist(row)
	if err != nil {
		return artist, ErrNotFound
	}
	return artist, nil
}

// Artists returns the most recently added artists.
func (s *LibraryService) Artists(ctx context.Context, limit int) ([]models.Artist, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+db.ArtistColumns+` FROM artists a ORDER BY a.created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return db.ScanArtists(rows)
}

// SongsByAlbum returns an album's tracks in track order.
func (s *LibraryService) SongsByAlbum(ctx context.Context, albumID int64) ([]models.Song, error) {
	return s.querySongs(ctx, `
		SELECT `+db.SongColumns+`
		FROM songs s
		`+db.SongJoins+`
		WHERE s.album_id = ?
		ORDER BY s.track_number
	`, albumID)
}

// SongsByArtist returns songs where the artist is the album artist or a
// credited performer.
func (s *LibraryService) SongsByArtist(ctx context.Context, artistID int64) ([]models.Song, error) {
	return s.querySongs(ctx, `
		SELECT DISTINCT `+db.SongColumns+`
		FROM songs s
		`+db.SongJoins+`
		WHERE al.artist_id = ?
		   OR EXISTS (SELECT 1 FROM song_artists sa WHERE sa.song_id = s.id AND sa.artist_id = ?)
		ORDER BY s.id
	`, artistID, artistID)
}

// RecentlyAdded returns the newest songs in the library.
func (s *LibraryService) RecentlyAdded(ctx context.Context, limit int) ([]models.Song, error) {
	return s.querySongs(ctx, `
		SELECT `+db.SongColumns+`
		FROM songs s
		`+db.SongJoins+`
		ORDER BY s.id DESC
		LIMIT ?
	`, limit)
}

// UserSongs returns one user-scoped slice of the library. days limits the
// listening window for the history-backed slices; 0 means all time.
func (s *LibraryService) UserSongs(ctx context.Context, userID int64, slice Slice, days, limit int) ([]models.Song, error) {
	window := ""
	args := []any{userID}
	if days > 0 {
		window = ` AND ph.played_at >= datetime('now', ?)`
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	switch slice {
	case SliceTop:
		return s.querySongs(ctx, `
			SELECT `+db.SongColumns+`
			FROM play_history ph
			JOIN songs s ON s.id = ph.song_id
			`+db.SongJoins+`
			WHERE ph.user_id = ?`+window+`
			GROUP BY s.id
			ORDER BY COUNT(*) DESC
			LIMIT ?
		`, append(args, limit)...)

	case SliceRecent:
		return s.querySongs(ctx, `
			SELECT `+db.SongColumns+`
			FROM play_history ph
			JOIN songs s ON s.id = ph.song_id
			`+db.SongJoins+`
			WHERE ph.user_id = ?`+window+`
			GROUP BY s.id
			ORDER BY MAX(ph.played_at) DESC
			LIMIT ?
		`, append(args, limit)...)

	case SliceFavorites:
		return s.querySongs(ctx, `
			SELECT `+db.SongColumns+`
			FROM favorites_songs f
			JOIN songs s ON s.id = f.song_id
			`+db.SongJoins+`
			WHERE f.user_id = ?
			ORDER BY f.created_at DESC
			LIMIT ?
		`, userID, limit)

	// Two plays is the threshold for a skip being a preference rather than an
	// interruption; below that a single abandoned play says nothing.
	case SliceSkipped:
		return s.querySongs(ctx, `
			SELECT `+db.SongColumns+`
			FROM play_history ph
			JOIN songs s ON s.id = ph.song_id
			`+db.SongJoins+`
			WHERE ph.user_id = ?`+window+`
			GROUP BY s.id
			HAVING COUNT(*) >= 2 AND AVG(ph.completion_rate) < 0.5
			ORDER BY COUNT(*) DESC
			LIMIT ?
		`, append(args, limit)...)

	// Randomised because any stable ordering would surface the same forgotten
	// corner of the library every time it is asked.
	case SliceUnplayed:
		return s.querySongs(ctx, `
			SELECT `+db.SongColumns+`
			FROM songs s
			`+db.SongJoins+`
			WHERE NOT EXISTS (SELECT 1 FROM play_history ph WHERE ph.song_id = s.id AND ph.user_id = ?)
			ORDER BY RANDOM()
			LIMIT ?
		`, userID, limit)
	}
	return nil, fmt.Errorf("unknown library slice %q", slice)
}

// SimilarSongs is the metadata-only radio: songs scored by how much they share
// with the seed, strongest first. The artist match runs over song_artists so a
// compilation track can match through its actual performer rather than the
// album's credited artist.
func (s *LibraryService) SimilarSongs(ctx context.Context, seedID int64, limit int) ([]models.Song, error) {
	var albumID int64
	var artistID, year sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT s.album_id, al.year,
		       COALESCE(
		         (SELECT sa.artist_id FROM song_artists sa
		          WHERE sa.song_id = s.id AND sa.role = 'primary'
		          ORDER BY sa.position LIMIT 1),
		         al.artist_id)
		FROM songs s
		JOIN albums al ON al.id = s.album_id
		WHERE s.id = ?
	`, seedID).Scan(&albumID, &year, &artistID)
	if err != nil {
		return nil, ErrNotFound
	}

	return s.querySongs(ctx, `
		SELECT `+db.SongColumns+`
		FROM songs s
		`+db.SongJoins+`
		WHERE s.id != ?
		ORDER BY
			(CASE WHEN s.album_id = ? THEN 3 ELSE 0 END) +
			(CASE WHEN EXISTS (SELECT 1 FROM song_artists sa
			                   WHERE sa.song_id = s.id AND sa.artist_id = ?) THEN 2 ELSE 0 END) +
			(CASE WHEN al.year = ? THEN 1 ELSE 0 END) DESC,
			RANDOM()
		LIMIT ?
	`, seedID, albumID, artistID, year, limit)
}

// FavoriteAlbums returns the user's favourited albums, newest first.
func (s *LibraryService) FavoriteAlbums(ctx context.Context, userID int64) ([]models.Album, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+db.AlbumColumns+`
		FROM favorites_albums f
		JOIN albums al ON al.id = f.album_id
		`+db.AlbumJoins+`
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return db.ScanAlbums(rows)
}

// FollowedArtists returns the artists the user follows, newest first.
func (s *LibraryService) FollowedArtists(ctx context.Context, userID int64) ([]models.Artist, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+db.ArtistColumns+`
		FROM follows_artists f
		JOIN artists a ON a.id = f.artist_id
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return db.ScanArtists(rows)
}

var favoriteTables = map[EntityKind]struct{ table, column, entity string }{
	EntitySong:   {"favorites_songs", "song_id", "songs"},
	EntityAlbum:  {"favorites_albums", "album_id", "albums"},
	EntityArtist: {"follows_artists", "artist_id", "artists"},
}

// SetFavorite adds or removes a favourite song, favourite album, or artist
// follow. It reports ErrNotFound when the target does not exist.
func (s *LibraryService) SetFavorite(ctx context.Context, userID int64, kind EntityKind, id int64, on bool) error {
	t, ok := favoriteTables[kind]
	if !ok {
		return fmt.Errorf("unknown favorite kind %q", kind)
	}
	if !on {
		_, err := s.db.ExecContext(ctx,
			`DELETE FROM `+t.table+` WHERE user_id = ? AND `+t.column+` = ?`, userID, id)
		return err
	}

	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM `+t.entity+` WHERE id = ?`, id).Scan(&exists); err != nil {
		return ErrNotFound
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO `+t.table+`(user_id, `+t.column+`) VALUES(?, ?)`, userID, id)
	return err
}

// IsFavorite reports whether the user has favourited or follows the entity.
func (s *LibraryService) IsFavorite(ctx context.Context, userID int64, kind EntityKind, id int64) bool {
	t, ok := favoriteTables[kind]
	if !ok {
		return false
	}
	var exists int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM `+t.table+` WHERE user_id = ? AND `+t.column+` = ?`, userID, id).Scan(&exists)
	return err == nil
}

// SongExists reports whether a song id resolves, for callers validating input
// before a write.
func (s *LibraryService) SongExists(ctx context.Context, id int64) bool {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM songs WHERE id = ?`, id).Scan(&exists)
	return err == nil
}

func (s *LibraryService) querySongs(ctx context.Context, query string, args ...any) ([]models.Song, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	songs, err := db.ScanSongs(rows)
	if err != nil {
		return nil, err
	}
	if err := db.PopulateSongArtists(ctx, s.db, songs); err != nil {
		return nil, err
	}
	return songs, nil
}
