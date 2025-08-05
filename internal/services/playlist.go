package services

import (
	"context"
	"fmt"

	"korus/internal/database"
	"korus/internal/models"
)

type PlaylistService struct {
	db *database.DB
}

func NewPlaylistService(db *database.DB) *PlaylistService {
	return &PlaylistService{db: db}
}

// GetUserPlaylists returns all playlists owned by the user
func (ps *PlaylistService) GetUserPlaylists(ctx context.Context, userID int) ([]models.Playlist, error) {
	query := `
		SELECT p.id, p.user_id, p.name, p.description, p.visibility,
			   p.created_at, p.updated_at,
			   COUNT(ps.song_id) as song_count,
			   COALESCE(SUM(s.duration), 0) as duration
		FROM playlists p
		LEFT JOIN playlist_songs ps ON p.id = ps.playlist_id
		LEFT JOIN songs s ON ps.song_id = s.id
		WHERE p.user_id = $1
		GROUP BY p.id, p.user_id, p.name, p.description, p.visibility, p.created_at, p.updated_at
		ORDER BY p.updated_at DESC
	`

	rows, err := ps.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user playlists: %w", err)
	}
	defer rows.Close()

	playlists := make([]models.Playlist, 0)
	for rows.Next() {
		var playlist models.Playlist
		err := rows.Scan(&playlist.ID, &playlist.UserID, &playlist.Name, &playlist.Description,
			&playlist.Visibility, &playlist.CreatedAt, &playlist.UpdatedAt,
			&playlist.SongCount, &playlist.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed to scan playlist: %w", err)
		}
		playlists = append(playlists, playlist)
	}

	return playlists, rows.Err()
}

// CreatePlaylist creates a new playlist for the user
func (ps *PlaylistService) CreatePlaylist(ctx context.Context, userID int, name, description, visibility string) (*models.Playlist, error) {
	query := `
		INSERT INTO playlists (user_id, name, description, visibility, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, user_id, name, description, visibility, created_at, updated_at
	`

	var playlist models.Playlist
	err := ps.db.QueryRowContext(ctx, query, userID, name, description, visibility).
		Scan(&playlist.ID, &playlist.UserID, &playlist.Name, &playlist.Description,
			&playlist.Visibility, &playlist.CreatedAt, &playlist.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}

	playlist.SongCount = 0
	playlist.Duration = 0

	return &playlist, nil
}

// GetPlaylist returns a playlist by ID, including its songs
func (ps *PlaylistService) GetPlaylist(ctx context.Context, playlistID, userID int) (*models.Playlist, []models.PlaylistSong, error) {
	// First get the playlist metadata
	playlistQuery := `
		SELECT p.id, p.user_id, p.name, p.description, p.visibility,
			   p.created_at, p.updated_at,
			   COUNT(ps.song_id) as song_count,
			   COALESCE(SUM(s.duration), 0) as duration
		FROM playlists p
		LEFT JOIN playlist_songs ps ON p.id = ps.playlist_id
		LEFT JOIN songs s ON ps.song_id = s.id
		WHERE p.id = $1 AND p.user_id = $2
		GROUP BY p.id, p.user_id, p.name, p.description, p.visibility, p.created_at, p.updated_at
	`

	var playlist models.Playlist
	err := ps.db.QueryRowContext(ctx, playlistQuery, playlistID, userID).
		Scan(&playlist.ID, &playlist.UserID, &playlist.Name, &playlist.Description,
			&playlist.Visibility, &playlist.CreatedAt, &playlist.UpdatedAt,
			&playlist.SongCount, &playlist.Duration)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	// Get playlist songs
	songsQuery := `
		SELECT ps.id, ps.playlist_id, ps.song_id, ps.position, ps.added_at,
			   s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified,
			   s.bitrate, s.format, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name
		FROM playlist_songs ps
		JOIN songs s ON ps.song_id = s.id
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE ps.playlist_id = $1
		ORDER BY ps.position
	`

	rows, err := ps.db.QueryContext(ctx, songsQuery, playlistID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query playlist songs: %w", err)
	}
	defer rows.Close()

	playlistSongs := make([]models.PlaylistSong, 0)
	for rows.Next() {
		var ps models.PlaylistSong
		var song models.Song
		var artistName, albumName *string

		err := rows.Scan(&ps.ID, &ps.PlaylistID, &ps.SongID, &ps.Position, &ps.AddedAt,
			&song.ID, &song.Title, &song.AlbumID, &song.ArtistID, &song.TrackNumber, &song.DiscNumber,
			&song.Duration, &song.FilePath, &song.FileSize, &song.FileModified,
			&song.Bitrate, &song.Format, &song.DateAdded,
			&artistName, &albumName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan playlist song: %w", err)
		}

		// Set artist info
		if song.ArtistID != nil && artistName != nil {
			song.Artist = &models.Artist{
				ID:   *song.ArtistID,
				Name: *artistName,
			}
		}

		// Set album info
		if song.AlbumID != nil && albumName != nil {
			song.Album = &models.Album{
				ID:   *song.AlbumID,
				Name: *albumName,
			}
		}

		ps.Song = &song
		playlistSongs = append(playlistSongs, ps)
	}

	return &playlist, playlistSongs, rows.Err()
}

// UpdatePlaylist updates playlist metadata
func (ps *PlaylistService) UpdatePlaylist(ctx context.Context, playlistID, userID int, name, description, visibility string) (*models.Playlist, error) {
	query := `
		UPDATE playlists
		SET name = $3, description = $4, visibility = $5, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, description, visibility, created_at, updated_at
	`

	var playlist models.Playlist
	err := ps.db.QueryRowContext(ctx, query, playlistID, userID, name, description, visibility).
		Scan(&playlist.ID, &playlist.UserID, &playlist.Name, &playlist.Description,
			&playlist.Visibility, &playlist.CreatedAt, &playlist.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update playlist: %w", err)
	}

	return &playlist, nil
}

// DeletePlaylist deletes a playlist and all its songs
func (ps *PlaylistService) DeletePlaylist(ctx context.Context, playlistID, userID int) error {
	query := `DELETE FROM playlists WHERE id = $1 AND user_id = $2`

	result, err := ps.db.ExecContext(ctx, query, playlistID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete playlist: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("playlist not found or not owned by user")
	}

	return nil
}

// AddSongsToPlaylist adds songs to a playlist at specified positions
func (ps *PlaylistService) AddSongsToPlaylist(ctx context.Context, playlistID, userID int, songIDs []int, position *int) error {
	// First verify the playlist belongs to the user
	ownerQuery := `SELECT user_id FROM playlists WHERE id = $1`
	var ownerID int
	err := ps.db.QueryRowContext(ctx, ownerQuery, playlistID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("failed to verify playlist ownership: %w", err)
	}
	if ownerID != userID {
		return fmt.Errorf("playlist not found or not owned by user")
	}

	// Get current max position if no position specified
	currentPosition := 0
	if position == nil {
		posQuery := `SELECT COALESCE(MAX(position), 0) FROM playlist_songs WHERE playlist_id = $1`
		err := ps.db.QueryRowContext(ctx, posQuery, playlistID).Scan(&currentPosition)
		if err != nil {
			return fmt.Errorf("failed to get current max position: %w", err)
		}
		currentPosition++ // Start after max position
	} else {
		currentPosition = *position
		// Shift existing songs down to make room
		shiftQuery := `UPDATE playlist_songs SET position = position + $1 WHERE playlist_id = $2 AND position >= $3`
		_, err := ps.db.ExecContext(ctx, shiftQuery, len(songIDs), playlistID, currentPosition)
		if err != nil {
			return fmt.Errorf("failed to shift existing songs: %w", err)
		}
	}

	// Insert new songs
	for i, songID := range songIDs {
		insertQuery := `
			INSERT INTO playlist_songs (playlist_id, song_id, position, added_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT DO NOTHING
		`
		_, err := ps.db.ExecContext(ctx, insertQuery, playlistID, songID, currentPosition+i)
		if err != nil {
			return fmt.Errorf("failed to add song to playlist: %w", err)
		}
	}

	// Update playlist modified time
	_, err = ps.db.ExecContext(ctx, `UPDATE playlists SET updated_at = NOW() WHERE id = $1`, playlistID)
	if err != nil {
		return fmt.Errorf("failed to update playlist timestamp: %w", err)
	}

	return nil
}

// RemoveSongsFromPlaylist removes songs from a playlist
func (ps *PlaylistService) RemoveSongsFromPlaylist(ctx context.Context, playlistID, userID int, songIDs []int) error {
	// First verify the playlist belongs to the user
	ownerQuery := `SELECT user_id FROM playlists WHERE id = $1`
	var ownerID int
	err := ps.db.QueryRowContext(ctx, ownerQuery, playlistID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("failed to verify playlist ownership: %w", err)
	}
	if ownerID != userID {
		return fmt.Errorf("playlist not found or not owned by user")
	}

	// Remove songs
	deleteQuery := `DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = ANY($2)`
	_, err = ps.db.ExecContext(ctx, deleteQuery, playlistID, songIDs)
	if err != nil {
		return fmt.Errorf("failed to remove songs from playlist: %w", err)
	}

	// Reorder remaining songs to fill gaps
	reorderQuery := `
		WITH ranked_songs AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY position) as new_position
			FROM playlist_songs
			WHERE playlist_id = $1
		)
		UPDATE playlist_songs
		SET position = ranked_songs.new_position
		FROM ranked_songs
		WHERE playlist_songs.id = ranked_songs.id
	`
	_, err = ps.db.ExecContext(ctx, reorderQuery, playlistID)
	if err != nil {
		return fmt.Errorf("failed to reorder playlist songs: %w", err)
	}

	// Update playlist modified time
	_, err = ps.db.ExecContext(ctx, `UPDATE playlists SET updated_at = NOW() WHERE id = $1`, playlistID)
	if err != nil {
		return fmt.Errorf("failed to update playlist timestamp: %w", err)
	}

	return nil
}

// ReorderPlaylistSongs reorders all songs in a playlist based on provided song IDs
func (ps *PlaylistService) ReorderPlaylistSongs(ctx context.Context, playlistID, userID int, songIDs []int) error {
	// First verify the playlist belongs to the user
	ownerQuery := `SELECT user_id FROM playlists WHERE id = $1`
	var ownerID int
	err := ps.db.QueryRowContext(ctx, ownerQuery, playlistID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("failed to verify playlist ownership: %w", err)
	}
	if ownerID != userID {
		return fmt.Errorf("playlist not found or not owned by user")
	}

	// Update positions for each song
	for i, songID := range songIDs {
		updateQuery := `UPDATE playlist_songs SET position = $1 WHERE playlist_id = $2 AND song_id = $3`
		_, err := ps.db.ExecContext(ctx, updateQuery, i+1, playlistID, songID)
		if err != nil {
			return fmt.Errorf("failed to update song position: %w", err)
		}
	}

	// Update playlist modified time
	_, err = ps.db.ExecContext(ctx, `UPDATE playlists SET updated_at = NOW() WHERE id = $1`, playlistID)
	if err != nil {
		return fmt.Errorf("failed to update playlist timestamp: %w", err)
	}

	return nil
}

// RemovePlaylistSongsByID removes playlist songs by their playlist_song IDs
func (ps *PlaylistService) RemovePlaylistSongsByID(ctx context.Context, playlistID, userID int, playlistSongIDs []int) error {
	// Verify playlist ownership
	ownerQuery := `SELECT user_id FROM playlists WHERE id = $1`
	var ownerID int
	err := ps.db.QueryRowContext(ctx, ownerQuery, playlistID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("failed to verify playlist ownership: %w", err)
	}
	if ownerID != userID {
		return fmt.Errorf("playlist not found or not owned by user")
	}

	// Remove songs by playlist_song IDs
	deleteQuery := `DELETE FROM playlist_songs WHERE id = ANY($1) AND playlist_id = $2`
	_, err = ps.db.ExecContext(ctx, deleteQuery, playlistSongIDs, playlistID)
	if err != nil {
		return fmt.Errorf("failed to remove songs from playlist: %w", err)
	}

	// Update playlist modified time
	_, err = ps.db.ExecContext(ctx, `UPDATE playlists SET updated_at = NOW() WHERE id = $1`, playlistID)
	if err != nil {
		return fmt.Errorf("failed to update playlist timestamp: %w", err)
	}

	return nil
}

// ReorderPlaylistSong moves a specific playlist song to a new position
func (ps *PlaylistService) ReorderPlaylistSong(ctx context.Context, playlistID, userID, playlistSongID, newPosition int) error {
	// Verify playlist ownership
	ownerQuery := `SELECT user_id FROM playlists WHERE id = $1`
	var ownerID int
	err := ps.db.QueryRowContext(ctx, ownerQuery, playlistID).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("failed to verify playlist ownership: %w", err)
	}
	if ownerID != userID {
		return fmt.Errorf("playlist not found or not owned by user")
	}

	// Get current position of the song
	var currentPosition int
	posQuery := `SELECT position FROM playlist_songs WHERE id = $1 AND playlist_id = $2`
	err = ps.db.QueryRowContext(ctx, posQuery, playlistSongID, playlistID).Scan(&currentPosition)
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}

	// Update positions of other songs
	if newPosition > currentPosition {
		// Moving down: shift songs up
		updateQuery := `UPDATE playlist_songs SET position = position - 1 
						WHERE playlist_id = $1 AND position > $2 AND position <= $3`
		_, err = ps.db.ExecContext(ctx, updateQuery, playlistID, currentPosition, newPosition)
	} else if newPosition < currentPosition {
		// Moving up: shift songs down
		updateQuery := `UPDATE playlist_songs SET position = position + 1 
						WHERE playlist_id = $1 AND position >= $2 AND position < $3`
		_, err = ps.db.ExecContext(ctx, updateQuery, playlistID, newPosition, currentPosition)
	}
	if err != nil {
		return fmt.Errorf("failed to update other song positions: %w", err)
	}

	// Update the target song's position
	updateQuery := `UPDATE playlist_songs SET position = $1 WHERE id = $2`
	_, err = ps.db.ExecContext(ctx, updateQuery, newPosition, playlistSongID)
	if err != nil {
		return fmt.Errorf("failed to update song position: %w", err)
	}

	// Update playlist modified time
	_, err = ps.db.ExecContext(ctx, `UPDATE playlists SET updated_at = NOW() WHERE id = $1`, playlistID)
	if err != nil {
		return fmt.Errorf("failed to update playlist timestamp: %w", err)
	}

	return nil
}
