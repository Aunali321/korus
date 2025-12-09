package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/models"
)

// Library godoc
// @Summary Get library overview
// @Tags Library
// @Produce json
// @Param limit query int false "max songs (default 50, max 200)"
// @Param offset query int false "song offset"
// @Success 200 {object} map[string]interface{}
// @Router /api/library [get]
func (h *Handler) Library(c echo.Context) error {
	ctx := c.Request().Context()
	limit, offset := parseLimitOffset(c, 50, 200)
	artists, _ := h.fetchArtists(ctx, 20, 0)
	albums, _ := h.fetchAlbums(ctx, 20, 0)
	songs, _ := h.fetchSongs(ctx, limit, offset)
	return c.JSON(http.StatusOK, map[string]interface{}{
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
// @Router /api/artists/{id} [get]
func (h *Handler) Artist(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var a models.Artist
	var mbid sql.NullString
	err := h.db.QueryRowContext(ctx, `SELECT id, name, bio, image_path, mbid, created_at FROM artists WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &a.Bio, &a.ImagePath, &mbid, &a.CreatedAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artist not found", "code": "NOT_FOUND"})
	}
	if mbid.Valid {
		a.MBID = &mbid.String
	}
	albums, _ := h.fetchAlbumsByArtist(ctx, id)
	songs, _ := h.fetchSongsByArtist(ctx, id)
	return c.JSON(http.StatusOK, map[string]interface{}{
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
// @Router /api/albums/{id} [get]
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
	songs, _ := h.fetchSongsByAlbum(ctx, id)
	artist, _ := h.fetchArtist(ctx, al.ArtistID)
	al.Artist = artist
	return c.JSON(http.StatusOK, map[string]interface{}{
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
// @Router /api/songs/{id} [get]
func (h *Handler) Song(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var s models.Song
	var track sql.NullInt64
	var duration sql.NullInt64
	var mbid sql.NullString
	err := h.db.QueryRowContext(ctx, `
		SELECT id, album_id, title, track_number, duration, file_path, lyrics, lyrics_synced, mbid
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
func (h *Handler) fetchArtists(ctx context.Context, limit, offset int) ([]models.Artist, error) {
	rows, err := h.db.QueryContext(ctx, `SELECT id, name, bio, image_path, mbid, created_at FROM artists ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Artist
	for rows.Next() {
		var a models.Artist
		var mbid sql.NullString
		if err := rows.Scan(&a.ID, &a.Name, &a.Bio, &a.ImagePath, &mbid, &a.CreatedAt); err == nil {
			if mbid.Valid {
				a.MBID = &mbid.String
			}
			res = append(res, a)
		}
	}
	return res, nil
}

func (h *Handler) fetchAlbums(ctx context.Context, limit, offset int) ([]models.Album, error) {
	rows, err := h.db.QueryContext(ctx, `SELECT id, artist_id, title, year, cover_path, mbid, created_at FROM albums ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Album
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

func (h *Handler) fetchSongs(ctx context.Context, limit, offset int) ([]models.Song, error) {
	rows, err := h.db.QueryContext(ctx, `SELECT id, album_id, title, track_number, duration, file_path FROM songs ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Song
	for rows.Next() {
		var s models.Song
		var track sql.NullInt64
		var duration sql.NullInt64
		if err := rows.Scan(&s.ID, &s.AlbumID, &s.Title, &track, &duration, &s.FilePath); err == nil {
			if track.Valid {
				t := int(track.Int64)
				s.TrackNumber = &t
			}
			if duration.Valid {
				d := int(duration.Int64)
				s.Duration = &d
			}
			res = append(res, s)
		}
	}
	return res, nil
}

func (h *Handler) fetchAlbumsByArtist(ctx context.Context, artistID int64) ([]models.Album, error) {
	rows, err := h.db.QueryContext(ctx, `SELECT id, artist_id, title, year, cover_path, mbid, created_at FROM albums WHERE artist_id = ?`, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Album
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

func (h *Handler) fetchSongsByArtist(ctx context.Context, artistID int64) ([]models.Song, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT s.id, s.album_id, s.title, s.track_number, s.duration, s.file_path
		FROM songs s
		INNER JOIN albums a ON a.id = s.album_id
		WHERE a.artist_id = ?
	`, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Song
	for rows.Next() {
		var s models.Song
		var track sql.NullInt64
		var duration sql.NullInt64
		if err := rows.Scan(&s.ID, &s.AlbumID, &s.Title, &track, &duration, &s.FilePath); err == nil {
			if track.Valid {
				t := int(track.Int64)
				s.TrackNumber = &t
			}
			if duration.Valid {
				d := int(duration.Int64)
				s.Duration = &d
			}
			res = append(res, s)
		}
	}
	return res, nil
}

func (h *Handler) fetchSongsByAlbum(ctx context.Context, albumID int64) ([]models.Song, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT id, album_id, title, track_number, duration, file_path
		FROM songs WHERE album_id = ?
		ORDER BY track_number
	`, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Song
	for rows.Next() {
		var s models.Song
		var track sql.NullInt64
		var duration sql.NullInt64
		if err := rows.Scan(&s.ID, &s.AlbumID, &s.Title, &track, &duration, &s.FilePath); err == nil {
			if track.Valid {
				t := int(track.Int64)
				s.TrackNumber = &t
			}
			if duration.Valid {
				d := int(duration.Int64)
				s.Duration = &d
			}
			res = append(res, s)
		}
	}
	return res, nil
}

func (h *Handler) fetchArtist(ctx context.Context, id int64) (*models.Artist, error) {
	var a models.Artist
	var mbid sql.NullString
	err := h.db.QueryRowContext(ctx, `SELECT id, name, bio, image_path, mbid, created_at FROM artists WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &a.Bio, &a.ImagePath, &mbid, &a.CreatedAt)
	if err != nil {
		return nil, err
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
