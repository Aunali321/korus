package db

import (
	"context"
	"database/sql"

	"github.com/Aunali321/korus/internal/models"
)

// SongColumns is the standard SELECT columns for songs with artist and album
// Note: duration_ms is converted to seconds for API compatibility
const SongColumns = `s.id, s.album_id, s.title, s.track_number, s.duration_ms / 1000 as duration, s.file_path,
	ar.id, ar.name, al.id, al.title`

// SongJoins is the standard JOIN clause to get artist and album
const SongJoins = `LEFT JOIN albums al ON al.id = s.album_id
	LEFT JOIN artists ar ON ar.id = al.artist_id`

// ScanSong scans a row into a models.Song with optional artist and album
func ScanSong(row interface{ Scan(...any) error }) (models.Song, error) {
	var song models.Song
	var track, duration sql.NullInt64
	var artistID, albumID sql.NullInt64
	var artistName, albumTitle sql.NullString

	err := row.Scan(
		&song.ID, &song.AlbumID, &song.Title, &track, &duration, &song.FilePath,
		&artistID, &artistName, &albumID, &albumTitle,
	)
	if err != nil {
		return song, err
	}

	if track.Valid {
		t := int(track.Int64)
		song.TrackNumber = &t
	}
	if duration.Valid {
		d := int(duration.Int64)
		song.Duration = &d
	}
	if artistID.Valid {
		song.Artist = &models.Artist{ID: artistID.Int64, Name: artistName.String}
	}
	if albumID.Valid {
		song.Album = &models.Album{ID: albumID.Int64, Title: albumTitle.String}
	}

	return song, nil
}

// GetSongsByPlaylist returns all songs in a playlist with artist and album info
func GetSongsByPlaylist(ctx context.Context, db *sql.DB, playlistID int64) ([]models.Song, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+SongColumns+`
		FROM playlist_songs ps
		JOIN songs s ON s.id = ps.song_id
		`+SongJoins+`
		WHERE ps.playlist_id = ?
		ORDER BY ps.position
	`, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSongs(rows)
}

// GetSongsByAlbum returns all songs in an album with artist info
func GetSongsByAlbum(ctx context.Context, db *sql.DB, albumID int64) ([]models.Song, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+SongColumns+`
		FROM songs s
		`+SongJoins+`
		WHERE s.album_id = ?
		ORDER BY s.track_number
	`, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSongs(rows)
}

// GetSongsByArtist returns all songs by an artist with album info
func GetSongsByArtist(ctx context.Context, db *sql.DB, artistID int64) ([]models.Song, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+SongColumns+`
		FROM songs s
		`+SongJoins+`
		WHERE al.artist_id = ?
		ORDER BY s.id
	`, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSongs(rows)
}

// GetSongsRecent returns the most recent songs with artist and album info
func GetSongsRecent(ctx context.Context, db *sql.DB, limit int) ([]models.Song, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+SongColumns+`
		FROM songs s
		`+SongJoins+`
		ORDER BY s.id DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSongs(rows)
}

// GetSongsByFavorites returns favorite songs for a user
func GetSongsByFavorites(ctx context.Context, db *sql.DB, userID int64) ([]models.Song, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+SongColumns+`
		FROM favorites_songs f
		JOIN songs s ON s.id = f.song_id
		`+SongJoins+`
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSongs(rows)
}

func scanSongs(rows *sql.Rows) ([]models.Song, error) {
	var songs []models.Song
	for rows.Next() {
		song, err := ScanSong(rows)
		if err != nil {
			continue
		}
		songs = append(songs, song)
	}
	return songs, rows.Err()
}

// GetSongsByRecentPlays returns recently played songs for a user with full song data
func GetSongsByRecentPlays(ctx context.Context, db *sql.DB, userID int64, limit int) ([]models.Song, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+SongColumns+`
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		`+SongJoins+`
		WHERE ph.user_id = ?
		GROUP BY s.id
		ORDER BY MAX(ph.played_at) DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSongs(rows)
}

// GetSongsByTopPlayed returns top played songs for a user with full song data
func GetSongsByTopPlayed(ctx context.Context, db *sql.DB, userID int64, limit int) ([]models.Song, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+SongColumns+`
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		`+SongJoins+`
		WHERE ph.user_id = ?
		GROUP BY s.id
		ORDER BY COUNT(*) DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSongs(rows)
}
