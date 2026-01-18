package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/models"
)

type SearchService struct {
	db *sql.DB
}

func NewSearchService(db *sql.DB) *SearchService {
	return &SearchService{db: db}
}

type SearchResult struct {
	Songs     []models.Song     `json:"songs"`
	Albums    []models.Album    `json:"albums"`
	Artists   []models.Artist   `json:"artists"`
	Playlists []models.Playlist `json:"playlists"`
}

func (s *SearchService) Search(ctx context.Context, q string, limit, offset int) (SearchResult, error) {
	res := SearchResult{
		Songs:     []models.Song{},
		Albums:    []models.Album{},
		Artists:   []models.Artist{},
		Playlists: []models.Playlist{},
	}
	if q == "" {
		return res, nil
	}
	// Songs via FTS - join with actual tables since FTS is contentless
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.album_id, s.title, s.duration_ms / 1000, ar.id, ar.name, al.id, al.title
		FROM songs_fts fts
		JOIN songs s ON s.id = fts.rowid
		JOIN albums al ON al.id = s.album_id
		JOIN artists ar ON ar.id = al.artist_id
		WHERE songs_fts MATCH ?
		LIMIT ? OFFSET ?
	`, q, limit, offset)
	if err != nil {
		return res, fmt.Errorf("search songs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var song models.Song
		var duration sql.NullInt64
		var artistID int64
		var artistName string
		var albumID int64
		var albumTitle string
		if err := rows.Scan(&song.ID, &song.AlbumID, &song.Title, &duration, &artistID, &artistName, &albumID, &albumTitle); err == nil {
			if duration.Valid {
				d := int(duration.Int64)
				song.Duration = &d
			}
			song.Album = &models.Album{ID: albumID, Title: albumTitle}
			res.Songs = append(res.Songs, song)
		}
	}
	// Populate artists for songs
	_ = db.PopulateSongArtists(ctx, s.db, res.Songs)

	// Artists
	artistRows, err := s.db.QueryContext(ctx, `
		SELECT id, name, bio, image_path, mbid, created_at
		FROM artists WHERE name LIKE ? LIMIT ? OFFSET ?
	`, "%"+q+"%", limit, offset)
	if err == nil {
		defer artistRows.Close()
		for artistRows.Next() {
			var mbid, bio, imagePath sql.NullString
			var a models.Artist
			if err := artistRows.Scan(&a.ID, &a.Name, &bio, &imagePath, &mbid, &a.CreatedAt); err == nil {
				if bio.Valid {
					a.Bio = bio.String
				}
				if imagePath.Valid {
					a.ImagePath = imagePath.String
				}
				if mbid.Valid {
					a.MBID = &mbid.String
				}
				res.Artists = append(res.Artists, a)
			}
		}
	}
	// Albums - join with artists
	albumRows, err := s.db.QueryContext(ctx, `
		SELECT al.id, al.artist_id, al.title, al.year, al.cover_path, al.mbid, al.created_at,
		       ar.id, ar.name
		FROM albums al
		LEFT JOIN artists ar ON ar.id = al.artist_id
		WHERE al.title LIKE ? LIMIT ? OFFSET ?
	`, "%"+q+"%", limit, offset)
	if err == nil {
		defer albumRows.Close()
		for albumRows.Next() {
			var mbid sql.NullString
			var year sql.NullInt64
			var artistID sql.NullInt64
			var artistName sql.NullString
			var al models.Album
			if err := albumRows.Scan(&al.ID, &al.ArtistID, &al.Title, &year, &al.CoverPath, &mbid, &al.CreatedAt,
				&artistID, &artistName); err == nil {
				if year.Valid {
					y := int(year.Int64)
					al.Year = &y
				}
				if mbid.Valid {
					al.MBID = &mbid.String
				}
				if artistID.Valid {
					al.Artist = &models.Artist{ID: artistID.Int64, Name: artistName.String}
				}
				res.Albums = append(res.Albums, al)
			}
		}
	}
	return res, nil
}
