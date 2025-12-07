package services

import (
	"context"
	"fmt"
	"time"

	"korus/internal/database"
	"korus/internal/models"
)

type HistoryService struct {
	db *database.DB
}

func NewHistoryService(db *database.DB) *HistoryService {
	return &HistoryService{db: db}
}

// RecordPlay records a song play event
func (hs *HistoryService) RecordPlay(ctx context.Context, userID, songID int, playedAt time.Time, playDuration *int, ipAddress *string) error {
	query := `
		INSERT INTO play_history (user_id, song_id, played_at, play_duration, ip_address)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := hs.db.ExecContext(ctx, query, userID, songID, playedAt, playDuration, ipAddress)
	if err != nil {
		return fmt.Errorf("failed to record play: %w", err)
	}

	return nil
}

// GetRecentHistory returns user's recent listening history
func (hs *HistoryService) GetRecentHistory(ctx context.Context, userID int, limit int) ([]models.PlayHistory, error) {
	query := `
		SELECT ph.id, ph.user_id, ph.song_id, ph.played_at, ph.play_duration, ph.ip_address,
			   s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified,
			   s.bitrate, s.format, s.cover_path, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name
		FROM play_history ph
		JOIN songs s ON ph.song_id = s.id
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE ph.user_id = $1
		ORDER BY ph.played_at DESC
		LIMIT $2
	`

	rows, err := hs.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent history: %w", err)
	}
	defer rows.Close()

	history := make([]models.PlayHistory, 0)
	for rows.Next() {
		var play models.PlayHistory
		var song models.Song
		var artistName, albumName *string

		err := rows.Scan(&play.ID, &play.UserID, &play.SongID, &play.PlayedAt, &play.PlayDuration, &play.IPAddress,
			&song.ID, &song.Title, &song.AlbumID, &song.ArtistID, &song.TrackNumber, &song.DiscNumber,
			&song.Duration, &song.FilePath, &song.FileSize, &song.FileModified,
			&song.Bitrate, &song.Format, &song.CoverPath, &song.DateAdded,
			&artistName, &albumName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan play history: %w", err)
		}

		if song.ArtistID != nil && artistName != nil {
			song.Artist = &models.Artist{
				ID:   *song.ArtistID,
				Name: *artistName,
			}
		}

		if song.AlbumID != nil && albumName != nil {
			song.Album = &models.Album{
				ID:   *song.AlbumID,
				Name: *albumName,
			}
		}

		play.Song = &song
		history = append(history, play)
	}

	return history, rows.Err()
}

// GetUserStats returns comprehensive user listening statistics
func (hs *HistoryService) GetUserStats(ctx context.Context, userID int) (*models.UserStats, error) {
	stats := &models.UserStats{}

	// Get total plays and total time listened
	basicQuery := `
		SELECT COUNT(*), COALESCE(SUM(play_duration), 0)
		FROM play_history
		WHERE user_id = $1
	`
	err := hs.db.QueryRowContext(ctx, basicQuery, userID).Scan(&stats.TotalPlays, &stats.TotalTimeListened)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic stats: %w", err)
	}

	// Get most played artist
	artistQuery := `
		SELECT ar.id, ar.name, ar.sort_name, ar.musicbrainz_id, COUNT(*) as play_count
		FROM play_history ph
		JOIN songs s ON ph.song_id = s.id
		JOIN artists ar ON s.artist_id = ar.id
		WHERE ph.user_id = $1
		GROUP BY ar.id, ar.name, ar.sort_name, ar.musicbrainz_id
		ORDER BY play_count DESC
		LIMIT 1
	`
	var playCount int
	var artist models.Artist
	err = hs.db.QueryRowContext(ctx, artistQuery, userID).
		Scan(&artist.ID, &artist.Name, &artist.SortName, &artist.MusicBrainzID, &playCount)
	if err == nil {
		stats.MostPlayedArtist = &artist
	} // If no result, leave it nil

	// Get most played song
	songQuery := `
		SELECT s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified,
			   s.bitrate, s.format, s.cover_path, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name,
			   COUNT(*) as play_count
		FROM play_history ph
		JOIN songs s ON ph.song_id = s.id
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE ph.user_id = $1
		GROUP BY s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
				 s.duration, s.file_path, s.file_size, s.file_modified,
				 s.bitrate, s.format, s.cover_path, s.date_added, ar.name, a.name
		ORDER BY play_count DESC
		LIMIT 1
	`
	rows, err := hs.db.QueryContext(ctx, songQuery, userID)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			var song models.Song
			var artistName, albumName *string
			var songPlayCount int

			err := rows.Scan(&song.ID, &song.Title, &song.AlbumID, &song.ArtistID,
				&song.TrackNumber, &song.DiscNumber, &song.Duration,
				&song.FilePath, &song.FileSize, &song.FileModified,
				&song.Bitrate, &song.Format, &song.CoverPath, &song.DateAdded,
				&artistName, &albumName, &songPlayCount)
			if err == nil {
				if song.ArtistID != nil && artistName != nil {
					song.Artist = &models.Artist{
						ID:   *song.ArtistID,
						Name: *artistName,
					}
				}

				if song.AlbumID != nil && albumName != nil {
					song.Album = &models.Album{
						ID:   *song.AlbumID,
						Name: *albumName,
					}
				}

				stats.MostPlayedSong = &song
			}
		}
	}

	// For now, we'll leave TopGenres empty as we don't have genre information in our schema
	// This could be added later when genre support is implemented
	stats.TopGenres = make([]string, 0)

	return stats, nil
}

// GetHomeData returns personalized home page data
func (hs *HistoryService) GetHomeData(ctx context.Context, userID int, limit int) (*models.HomeData, error) {
	homeData := &models.HomeData{
		RecentlyAdded:     make([]models.Album, 0),
		RecentlyPlayed:    make([]models.Song, 0),
		MostPlayed:        make([]models.Song, 0),
		RecommendedAlbums: make([]models.Album, 0),
	}

	// Get recently added albums
	recentAlbumsQuery := `
		SELECT a.id, a.name, a.artist_id, a.album_artist_id, a.year,
			   a.musicbrainz_id, a.cover_path, a.date_added,
			   ar.name as artist_name,
			   COUNT(DISTINCT s.id) as song_count,
			   COALESCE(SUM(s.duration), 0) as duration
		FROM albums a
		LEFT JOIN artists ar ON a.artist_id = ar.id
		LEFT JOIN songs s ON a.id = s.album_id
		GROUP BY a.id, a.name, a.artist_id, a.album_artist_id, a.year,
				 a.musicbrainz_id, a.cover_path, a.date_added, ar.name
		ORDER BY a.date_added DESC
		LIMIT $1
	`

	rows, err := hs.db.QueryContext(ctx, recentAlbumsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent albums: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var album models.Album
		var artistName *string

		err := rows.Scan(&album.ID, &album.Name, &album.ArtistID, &album.AlbumArtistID,
			&album.Year, &album.MusicBrainzID, &album.CoverPath, &album.DateAdded,
			&artistName, &album.SongCount, &album.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recent album: %w", err)
		}

		if album.ArtistID != nil && artistName != nil {
			album.Artist = &models.Artist{
				ID:   *album.ArtistID,
				Name: *artistName,
			}
		}

		homeData.RecentlyAdded = append(homeData.RecentlyAdded, album)
	}

	// Get recently played songs (distinct)
	recentSongsQuery := `
		SELECT s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified,
			   s.bitrate, s.format, s.cover_path, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name,
			   MAX(ph.played_at) as last_played
		FROM play_history ph
		JOIN songs s ON ph.song_id = s.id
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE ph.user_id = $1
		GROUP BY s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
				 s.duration, s.file_path, s.file_size, s.file_modified,
				 s.bitrate, s.format, s.cover_path, s.date_added, ar.name, a.name
		ORDER BY last_played DESC
		LIMIT $2
	`

	rows2, err := hs.db.QueryContext(ctx, recentSongsQuery, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent songs: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var song models.Song
		var artistName, albumName *string
		var lastPlayed time.Time

		err := rows2.Scan(&song.ID, &song.Title, &song.AlbumID, &song.ArtistID,
			&song.TrackNumber, &song.DiscNumber, &song.Duration,
			&song.FilePath, &song.FileSize, &song.FileModified,
			&song.Bitrate, &song.Format, &song.CoverPath, &song.DateAdded,
			&artistName, &albumName, &lastPlayed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recent song: %w", err)
		}

		if song.ArtistID != nil && artistName != nil {
			song.Artist = &models.Artist{
				ID:   *song.ArtistID,
				Name: *artistName,
			}
		}

		if song.AlbumID != nil && albumName != nil {
			song.Album = &models.Album{
				ID:   *song.AlbumID,
				Name: *albumName,
			}
		}

		homeData.RecentlyPlayed = append(homeData.RecentlyPlayed, song)
	}

	// Get most played songs for this user
	mostPlayedQuery := `
		SELECT s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
			   s.duration, s.file_path, s.file_size, s.file_modified,
			   s.bitrate, s.format, s.cover_path, s.date_added,
			   ar.name as artist_name,
			   a.name as album_name,
			   COUNT(*) as play_count
		FROM play_history ph
		JOIN songs s ON ph.song_id = s.id
		LEFT JOIN artists ar ON s.artist_id = ar.id
		LEFT JOIN albums a ON s.album_id = a.id
		WHERE ph.user_id = $1
		GROUP BY s.id, s.title, s.album_id, s.artist_id, s.track_number, s.disc_number,
				 s.duration, s.file_path, s.file_size, s.file_modified,
				 s.bitrate, s.format, s.cover_path, s.date_added, ar.name, a.name
		ORDER BY play_count DESC
		LIMIT $2
	`

	rows3, err := hs.db.QueryContext(ctx, mostPlayedQuery, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query most played songs: %w", err)
	}
	defer rows3.Close()

	for rows3.Next() {
		var song models.Song
		var artistName, albumName *string
		var playCount int

		err := rows3.Scan(&song.ID, &song.Title, &song.AlbumID, &song.ArtistID,
			&song.TrackNumber, &song.DiscNumber, &song.Duration,
			&song.FilePath, &song.FileSize, &song.FileModified,
			&song.Bitrate, &song.Format, &song.CoverPath, &song.DateAdded,
			&artistName, &albumName, &playCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan most played song: %w", err)
		}

		if song.ArtistID != nil && artistName != nil {
			song.Artist = &models.Artist{
				ID:   *song.ArtistID,
				Name: *artistName,
			}
		}

		if song.AlbumID != nil && albumName != nil {
			song.Album = &models.Album{
				ID:   *song.AlbumID,
				Name: *albumName,
			}
		}

		homeData.MostPlayed = append(homeData.MostPlayed, song)
	}

	// For recommended albums, we'll use a simple algorithm: albums from artists the user has played recently
	// but albums they haven't played much themselves
	recommendedQuery := `
		SELECT a.id, a.name, a.artist_id, a.album_artist_id, a.year,
			   a.musicbrainz_id, a.cover_path, a.date_added,
			   ar.name as artist_name,
			   COUNT(DISTINCT s.id) as song_count,
			   COALESCE(SUM(s.duration), 0) as duration
		FROM albums a
		LEFT JOIN artists ar ON a.artist_id = ar.id
		LEFT JOIN songs s ON a.id = s.album_id
		WHERE a.artist_id IN (
			SELECT s.artist_id
			FROM play_history ph
			JOIN songs s ON ph.song_id = s.id
			WHERE ph.user_id = $1 AND s.artist_id IS NOT NULL
			GROUP BY s.artist_id
			ORDER BY MAX(ph.played_at) DESC
			LIMIT 10
		)
		AND a.id NOT IN (
			SELECT s.album_id
			FROM play_history ph
			JOIN songs s ON ph.song_id = s.id
			WHERE ph.user_id = $1 AND s.album_id IS NOT NULL
			GROUP BY s.album_id
		)
		GROUP BY a.id, a.name, a.artist_id, a.album_artist_id, a.year,
				 a.musicbrainz_id, a.cover_path, a.date_added, ar.name
		ORDER BY a.date_added DESC
		LIMIT $2
	`

	rows4, err := hs.db.QueryContext(ctx, recommendedQuery, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recommended albums: %w", err)
	}
	defer rows4.Close()

	for rows4.Next() {
		var album models.Album
		var artistName *string

		err := rows4.Scan(&album.ID, &album.Name, &album.ArtistID, &album.AlbumArtistID,
			&album.Year, &album.MusicBrainzID, &album.CoverPath, &album.DateAdded,
			&artistName, &album.SongCount, &album.Duration)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recommended album: %w", err)
		}

		if album.ArtistID != nil && artistName != nil {
			album.Artist = &models.Artist{
				ID:   *album.ArtistID,
				Name: *artistName,
			}
		}

		homeData.RecommendedAlbums = append(homeData.RecommendedAlbums, album)
	}

	return homeData, nil
}
