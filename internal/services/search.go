package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Aunali321/korus/internal/models"
)

type SearchService struct {
	db *sql.DB
}

func NewSearchService(db *sql.DB) *SearchService {
	return &SearchService{db: db}
}

type SearchResult struct {
	Songs   []models.Song   `json:"songs"`
	Albums  []models.Album  `json:"albums"`
	Artists []models.Artist `json:"artists"`
}

func (s *SearchService) Search(ctx context.Context, q string, limit, offset int) (SearchResult, error) {
	var res SearchResult
	if q == "" {
		return res, nil
	}
	// Songs via FTS
	rows, err := s.db.QueryContext(ctx, `
		SELECT song_id, title, artist_name, album_title
		FROM songs_fts
		WHERE songs_fts MATCH ?
		LIMIT ? OFFSET ?
	`, q, limit, offset)
	if err != nil {
		return res, fmt.Errorf("search songs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var song models.Song
		var artistName, albumTitle string
		if err := rows.Scan(&song.ID, &song.Title, &artistName, &albumTitle); err == nil {
			song.Artist = &models.Artist{Name: artistName}
			song.Album = &models.Album{Title: albumTitle}
			res.Songs = append(res.Songs, song)
		}
	}
	// Artists
	artistRows, err := s.db.QueryContext(ctx, `
		SELECT id, name, bio, image_path, mbid, created_at
		FROM artists WHERE name LIKE ? LIMIT ? OFFSET ?
	`, "%"+q+"%", limit, offset)
	if err == nil {
		defer artistRows.Close()
		for artistRows.Next() {
			var mbid sql.NullString
			var a models.Artist
			if err := artistRows.Scan(&a.ID, &a.Name, &a.Bio, &a.ImagePath, &mbid, &a.CreatedAt); err == nil {
				if mbid.Valid {
					a.MBID = &mbid.String
				}
				res.Artists = append(res.Artists, a)
			}
		}
	}
	// Albums
	albumRows, err := s.db.QueryContext(ctx, `
		SELECT id, artist_id, title, year, cover_path, mbid, created_at
		FROM albums WHERE title LIKE ? LIMIT ? OFFSET ?
	`, "%"+q+"%", limit, offset)
	if err == nil {
		defer albumRows.Close()
		for albumRows.Next() {
			var mbid sql.NullString
			var year sql.NullInt64
			var al models.Album
			if err := albumRows.Scan(&al.ID, &al.ArtistID, &al.Title, &year, &al.CoverPath, &mbid, &al.CreatedAt); err == nil {
				if year.Valid {
					y := int(year.Int64)
					al.Year = &y
				}
				if mbid.Valid {
					al.MBID = &mbid.String
				}
				res.Albums = append(res.Albums, al)
			}
		}
	}
	return res, nil
}
