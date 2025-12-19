package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/models"
)

// Library godoc
// @Summary Get library overview
// @Tags Library
// @Produce json
// @Param limit query int false "max items (default: all)"
// @Success 200 {object} map[string]interface{}
// @Router /library [get]
func (h *Handler) Library(c echo.Context) error {
	ctx := c.Request().Context()
	limit := parseOptionalLimit(c)
	artists, _ := h.fetchArtists(ctx, limit)
	albums, _ := h.fetchAlbums(ctx, limit)
	songs, _ := db.GetSongsRecent(ctx, h.db, limit)
	return c.JSON(http.StatusOK, map[string]any{
		"artists": artists,
		"albums":  albums,
		"songs":   songs,
	})
}

// Artist godoc
// @Summary Get artist by id
// @Tags Library
// @Produce json
// @Param id path int true "Artist ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /artists/{id} [get]
func (h *Handler) Artist(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var a models.Artist
	var mbid, bio, imagePath sql.NullString
	err := h.db.QueryRowContext(ctx, `SELECT id, name, bio, image_path, mbid, created_at FROM artists WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &bio, &imagePath, &mbid, &a.CreatedAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artist not found", "code": "NOT_FOUND"})
	}
	if bio.Valid {
		a.Bio = bio.String
	}
	if imagePath.Valid {
		a.ImagePath = imagePath.String
	}
	if mbid.Valid {
		a.MBID = &mbid.String
	}
	albums, _ := h.fetchAlbumsByArtist(ctx, id)
	songs, _ := db.GetSongsByArtist(ctx, h.db, id)
	return c.JSON(http.StatusOK, map[string]any{
		"id":         a.ID,
		"name":       a.Name,
		"bio":        a.Bio,
		"image_path": a.ImagePath,
		"mbid":       a.MBID,
		"albums":     albums,
		"songs":      songs,
	})
}

// Album godoc
// @Summary Get album by id
// @Tags Library
// @Produce json
// @Param id path int true "Album ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /albums/{id} [get]
func (h *Handler) Album(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var al models.Album
	var mbid sql.NullString
	var year sql.NullInt64
	err := h.db.QueryRowContext(ctx, `
		SELECT id, artist_id, title, year, cover_path, mbid, created_at FROM albums WHERE id = ?
	`, id).Scan(&al.ID, &al.ArtistID, &al.Title, &year, &al.CoverPath, &mbid, &al.CreatedAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "album not found", "code": "NOT_FOUND"})
	}
	if year.Valid {
		y := int(year.Int64)
		al.Year = &y
	}
	if mbid.Valid {
		al.MBID = &mbid.String
	}
	songs, _ := db.GetSongsByAlbum(ctx, h.db, id)
	artist, _ := h.fetchArtist(ctx, al.ArtistID)
	al.Artist = artist
	return c.JSON(http.StatusOK, map[string]any{
		"id":         al.ID,
		"title":      al.Title,
		"year":       al.Year,
		"cover_path": al.CoverPath,
		"mbid":       al.MBID,
		"artist":     artist,
		"songs":      songs,
	})
}

// Song godoc
// @Summary Get song by id
// @Tags Library
// @Produce json
// @Param id path int true "Song ID"
// @Success 200 {object} models.Song
// @Failure 404 {object} map[string]string
// @Router /songs/{id} [get]
func (h *Handler) Song(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var s models.Song
	var track sql.NullInt64
	var duration sql.NullInt64
	var mbid sql.NullString
	err := h.db.QueryRowContext(ctx, `
		SELECT id, album_id, title, track_number, duration_ms / 1000, file_path, lyrics, lyrics_synced, mbid
		FROM songs WHERE id = ?
	`, id).Scan(&s.ID, &s.AlbumID, &s.Title, &track, &duration, &s.FilePath, &s.Lyrics, &s.LyricsSynced, &mbid)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "song not found", "code": "NOT_FOUND"})
	}
	if track.Valid {
		t := int(track.Int64)
		s.TrackNumber = &t
	}
	if duration.Valid {
		d := int(duration.Int64)
		s.Duration = &d
	}
	if mbid.Valid {
		s.MBID = &mbid.String
	}
	album, _ := h.fetchAlbum(ctx, s.AlbumID)
	s.Album = album
	if album != nil {
		artist, _ := h.fetchArtist(ctx, album.ArtistID)
		s.Artist = artist
	}
	return c.JSON(http.StatusOK, s)
}

// helpers
func (h *Handler) fetchArtists(ctx context.Context, limit int) ([]models.Artist, error) {
	rows, err := h.db.QueryContext(ctx, `SELECT id, name, bio, image_path, mbid, created_at FROM artists ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return []models.Artist{}, err
	}
	defer rows.Close()
	res := []models.Artist{}
	for rows.Next() {
		var a models.Artist
		var mbid, bio, imagePath sql.NullString
		if err := rows.Scan(&a.ID, &a.Name, &bio, &imagePath, &mbid, &a.CreatedAt); err == nil {
			if bio.Valid {
				a.Bio = bio.String
			}
			if imagePath.Valid {
				a.ImagePath = imagePath.String
			}
			if mbid.Valid {
				a.MBID = &mbid.String
			}
			res = append(res, a)
		}
	}
	return res, nil
}

func (h *Handler) fetchAlbums(ctx context.Context, limit int) ([]models.Album, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT al.id, al.artist_id, al.title, al.year, al.cover_path, al.mbid, al.created_at,
		       ar.id, ar.name, ar.bio, ar.image_path, ar.mbid
		FROM albums al
		LEFT JOIN artists ar ON ar.id = al.artist_id
		ORDER BY al.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return []models.Album{}, err
	}
	defer rows.Close()
	res := []models.Album{}
	for rows.Next() {
		var al models.Album
		var mbid sql.NullString
		var year sql.NullInt64
		var artistID sql.NullInt64
		var artistName, artistBio, artistImagePath, artistMBID sql.NullString
		if err := rows.Scan(&al.ID, &al.ArtistID, &al.Title, &year, &al.CoverPath, &mbid, &al.CreatedAt,
			&artistID, &artistName, &artistBio, &artistImagePath, &artistMBID); err == nil {
			if year.Valid {
				y := int(year.Int64)
				al.Year = &y
			}
			if mbid.Valid {
				al.MBID = &mbid.String
			}
			if artistID.Valid {
				artist := &models.Artist{ID: artistID.Int64, Name: artistName.String}
				if artistBio.Valid {
					artist.Bio = artistBio.String
				}
				if artistImagePath.Valid {
					artist.ImagePath = artistImagePath.String
				}
				if artistMBID.Valid {
					artist.MBID = &artistMBID.String
				}
				al.Artist = artist
			}
			res = append(res, al)
		}
	}
	return res, nil
}

func (h *Handler) fetchAlbumsByArtist(ctx context.Context, artistID int64) ([]models.Album, error) {
	rows, err := h.db.QueryContext(ctx, `SELECT id, artist_id, title, year, cover_path, mbid, created_at FROM albums WHERE artist_id = ?`, artistID)
	if err != nil {
		return []models.Album{}, err
	}
	defer rows.Close()
	res := []models.Album{}
	for rows.Next() {
		var al models.Album
		var mbid sql.NullString
		var year sql.NullInt64
		if err := rows.Scan(&al.ID, &al.ArtistID, &al.Title, &year, &al.CoverPath, &mbid, &al.CreatedAt); err == nil {
			if year.Valid {
				y := int(year.Int64)
				al.Year = &y
			}
			if mbid.Valid {
				al.MBID = &mbid.String
			}
			res = append(res, al)
		}
	}
	return res, nil
}

func (h *Handler) fetchArtist(ctx context.Context, id int64) (*models.Artist, error) {
	var a models.Artist
	var mbid, bio, imagePath sql.NullString
	err := h.db.QueryRowContext(ctx, `SELECT id, name, bio, image_path, mbid, created_at FROM artists WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &bio, &imagePath, &mbid, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if bio.Valid {
		a.Bio = bio.String
	}
	if imagePath.Valid {
		a.ImagePath = imagePath.String
	}
	if mbid.Valid {
		a.MBID = &mbid.String
	}
	return &a, nil
}

func (h *Handler) fetchAlbum(ctx context.Context, id int64) (*models.Album, error) {
	var al models.Album
	var mbid sql.NullString
	var year sql.NullInt64
	err := h.db.QueryRowContext(ctx, `SELECT id, artist_id, title, year, cover_path, mbid, created_at FROM albums WHERE id = ?`, id).
		Scan(&al.ID, &al.ArtistID, &al.Title, &year, &al.CoverPath, &mbid, &al.CreatedAt)
	if err != nil {
		return nil, err
	}
	if mbid.Valid {
		al.MBID = &mbid.String
	}
	if year.Valid {
		y := int(year.Int64)
		al.Year = &y
	}
	return &al, nil
}
