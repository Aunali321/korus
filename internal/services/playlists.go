package services

import (
	"context"
	"database/sql"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/models"
)

// PlaylistService owns playlists and their track order. Positions are always
// dense and zero-based: every write rewrites the whole ordering, so no caller
// invents its own numbering scheme.
type PlaylistService struct {
	db *sql.DB
}

func NewPlaylistService(database *sql.DB) *PlaylistService {
	return &PlaylistService{db: database}
}

// WriteMode says whether SetSongs appends to the playlist or replaces it.
type WriteMode string

const (
	ModeAppend  WriteMode = "append"
	ModeReplace WriteMode = "replace"
)

const playlistColumns = `p.id, p.user_id, p.name, p.description, p.cover_path, p.public, p.created_at, u.username,
	(SELECT COUNT(*) FROM playlist_songs ps WHERE ps.playlist_id = p.id),
	(SELECT ps.song_id FROM playlist_songs ps WHERE ps.playlist_id = p.id ORDER BY ps.position LIMIT 1)`

func scanPlaylist(row interface{ Scan(...any) error }) (models.Playlist, error) {
	var p models.Playlist
	var description, coverPath sql.NullString
	var ownerName string
	var firstSongID sql.NullInt64

	err := row.Scan(&p.ID, &p.UserID, &p.Name, &description, &coverPath, &p.Public, &p.CreatedAt,
		&ownerName, &p.SongCount, &firstSongID)
	if err != nil {
		return p, err
	}

	p.Description = description.String
	p.CoverPath = coverPath.String
	p.Owner = &models.PlaylistOwner{ID: p.UserID, Username: ownerName}
	if firstSongID.Valid {
		p.FirstSongID = &firstSongID.Int64
	}
	return p, nil
}

// List returns the playlists the user can see: their own plus public ones.
func (s *PlaylistService) List(ctx context.Context, userID int64, limit, offset int) ([]models.Playlist, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+playlistColumns+`
		FROM playlists p
		JOIN users u ON u.id = p.user_id
		WHERE p.public = 1 OR p.user_id = ?
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playlists := []models.Playlist{}
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			continue
		}
		playlists = append(playlists, p)
	}
	return playlists, rows.Err()
}

// Mine returns only the playlists the user owns.
func (s *PlaylistService) Mine(ctx context.Context, userID int64) ([]models.Playlist, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+playlistColumns+`
		FROM playlists p
		JOIN users u ON u.id = p.user_id
		WHERE p.user_id = ?
		ORDER BY p.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playlists := []models.Playlist{}
	for rows.Next() {
		p, err := scanPlaylist(rows)
		if err != nil {
			continue
		}
		playlists = append(playlists, p)
	}
	return playlists, rows.Err()
}

// Get returns a playlist without its songs, enforcing visibility.
func (s *PlaylistService) Get(ctx context.Context, userID, id int64) (models.Playlist, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+playlistColumns+`
		FROM playlists p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = ?
	`, id)
	p, err := scanPlaylist(row)
	if err != nil {
		return p, ErrNotFound
	}
	if !p.Public && p.UserID != userID {
		return p, ErrForbidden
	}
	return p, nil
}

// GetWithSongs returns a playlist and its tracks in playlist order.
func (s *PlaylistService) GetWithSongs(ctx context.Context, userID, id int64) (models.Playlist, error) {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return p, err
	}
	p.Songs, err = s.Songs(ctx, id)
	return p, err
}

// Songs returns a playlist's tracks in playlist order.
func (s *PlaylistService) Songs(ctx context.Context, playlistID int64) ([]models.Song, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+db.SongColumns+`
		FROM playlist_songs ps
		JOIN songs s ON s.id = ps.song_id
		`+db.SongJoins+`
		WHERE ps.playlist_id = ?
		ORDER BY ps.position
	`, playlistID)
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

// Create makes a playlist and fills it with songIDs in the given order.
func (s *PlaylistService) Create(ctx context.Context, userID int64, name, description string, public bool, songIDs []int64) (models.Playlist, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO playlists(user_id, name, description, public) VALUES(?, ?, ?, ?)`,
		userID, name, description, public)
	if err != nil {
		return models.Playlist{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return models.Playlist{}, err
	}
	if len(songIDs) > 0 {
		if err := s.SetSongs(ctx, userID, id, songIDs, ModeAppend); err != nil {
			return models.Playlist{}, err
		}
	}
	return s.Get(ctx, userID, id)
}

// UpdateMeta renames a playlist and updates its description and visibility.
func (s *PlaylistService) UpdateMeta(ctx context.Context, userID, id int64, name, description string, public bool) (models.Playlist, error) {
	if err := s.requireOwner(ctx, userID, id); err != nil {
		return models.Playlist{}, err
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE playlists SET name = ?, description = ?, public = ? WHERE id = ?`,
		name, description, public, id)
	if err != nil {
		return models.Playlist{}, err
	}
	return s.Get(ctx, userID, id)
}

// Delete removes a playlist the user owns.
func (s *PlaylistService) Delete(ctx context.Context, userID, id int64) error {
	if err := s.requireOwner(ctx, userID, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM playlists WHERE id = ?`, id)
	return err
}

// SetSongs writes the playlist's contents. ModeReplace makes songIDs the whole
// playlist, which is also how a caller removes or reorders tracks. ModeAppend
// adds them to the end, skipping ones already present. Either way the stored
// positions come out dense and zero-based.
func (s *PlaylistService) SetSongs(ctx context.Context, userID, id int64, songIDs []int64, mode WriteMode) error {
	if err := s.requireOwner(ctx, userID, id); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ordered := songIDs
	if mode == ModeAppend {
		existing, err := txSongIDs(ctx, tx, id)
		if err != nil {
			return err
		}
		ordered = append(existing, songIDs...)
	}

	if err := writeSongs(ctx, tx, id, ordered); err != nil {
		return err
	}
	return tx.Commit()
}

// Reorder rewrites the playlist's order. songIDs must be a permutation of what
// the playlist already holds; a partial list is rejected rather than treated as
// a request to delete everything missing from it.
func (s *PlaylistService) Reorder(ctx context.Context, userID, id int64, songIDs []int64) error {
	if err := s.requireOwner(ctx, userID, id); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	existing, err := txSongIDs(ctx, tx, id)
	if err != nil {
		return err
	}
	if len(songIDs) != len(existing) {
		return ErrInvalid
	}
	remaining := make(map[int64]bool, len(existing))
	for _, sid := range existing {
		remaining[sid] = true
	}
	for _, sid := range songIDs {
		if !remaining[sid] {
			return ErrInvalid
		}
		delete(remaining, sid)
	}

	if err := writeSongs(ctx, tx, id, songIDs); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveSongs drops the given tracks and closes the gaps they leave.
func (s *PlaylistService) RemoveSongs(ctx context.Context, userID, id int64, songIDs []int64) error {
	if err := s.requireOwner(ctx, userID, id); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	existing, err := txSongIDs(ctx, tx, id)
	if err != nil {
		return err
	}
	drop := make(map[int64]bool, len(songIDs))
	for _, sid := range songIDs {
		drop[sid] = true
	}
	kept := make([]int64, 0, len(existing))
	for _, sid := range existing {
		if !drop[sid] {
			kept = append(kept, sid)
		}
	}

	if err := writeSongs(ctx, tx, id, kept); err != nil {
		return err
	}
	return tx.Commit()
}

// SetCover records the stored path of a playlist's cover image.
func (s *PlaylistService) SetCover(ctx context.Context, userID, id int64, path string) error {
	if err := s.requireOwner(ctx, userID, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE playlists SET cover_path = ? WHERE id = ?`, path, id)
	return err
}

// Cover returns the playlist's stored cover path.
func (s *PlaylistService) Cover(ctx context.Context, id int64) (string, error) {
	var coverPath sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT cover_path FROM playlists WHERE id = ?`, id).Scan(&coverPath); err != nil {
		return "", ErrNotFound
	}
	if coverPath.String == "" {
		return "", ErrNotFound
	}
	return coverPath.String, nil
}

func (s *PlaylistService) requireOwner(ctx context.Context, userID, id int64) error {
	var owner int64
	if err := s.db.QueryRowContext(ctx, `SELECT user_id FROM playlists WHERE id = ?`, id).Scan(&owner); err != nil {
		return ErrNotFound
	}
	if owner != userID {
		return ErrForbidden
	}
	return nil
}

func txSongIDs(ctx context.Context, tx *sql.Tx, playlistID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT song_id FROM playlist_songs WHERE playlist_id = ? ORDER BY position`, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// writeSongs makes songIDs the playlist's entire contents, keeping only the
// first occurrence of a repeated id since playlist_songs is keyed by song.
func writeSongs(ctx context.Context, tx *sql.Tx, playlistID int64, songIDs []int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM playlist_songs WHERE playlist_id = ?`, playlistID); err != nil {
		return err
	}

	seen := make(map[int64]bool, len(songIDs))
	position := 0
	for _, songID := range songIDs {
		if seen[songID] {
			continue
		}
		seen[songID] = true
		res, err := tx.ExecContext(ctx,
			`INSERT INTO playlist_songs(playlist_id, song_id, position)
			 SELECT ?, ?, ? WHERE EXISTS (SELECT 1 FROM songs WHERE id = ?)`,
			playlistID, songID, position, songID)
		if err != nil {
			return err
		}
		// An id that no longer resolves inserts nothing; holding the position
		// back keeps the stored ordering dense.
		if inserted, err := res.RowsAffected(); err == nil && inserted > 0 {
			position++
		}
	}
	return nil
}
