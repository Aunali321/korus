package handlers

import (
	"database/sql"
	"net/http"

	"github.com/Aunali321/korus/internal/models"
	"github.com/Aunali321/korus/internal/services"
	"github.com/labstack/echo/v4"
)

// Radio godoc
// @Summary Get similar songs for radio playback
// @Tags Radio
// @Produce json
// @Param id path int true "Song ID to seed radio from"
// @Param limit query int false "Number of songs to return" default(20)
// @Param mode query string false "Radio mode: curator or mainstream" default(curator)
// @Success 200 {object} map[string][]models.Song
// @Failure 404 {object} map[string]string
// @Router /radio/{id} [get]
func (h *Handler) Radio(c echo.Context) error {
	var songID int64
	if err := echo.PathParamsBinder(c).Int64("id", &songID).BindError(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid id", "code": "BAD_REQUEST"})
	}

	limit := h.radioDefaultLimit
	if l := c.QueryParam("limit"); l != "" {
		if err := echo.QueryParamsBinder(c).Int("limit", &limit).BindError(); err == nil && limit > 0 && limit <= 100 {
			// valid
		} else {
			limit = h.radioDefaultLimit
		}
	}

	mode := c.QueryParam("mode")

	ctx := c.Request().Context()

	// Try LLM-based recommendations if available
	if h.radio != nil {
		radioMode := services.RadioModeCurator
		if mode == "mainstream" {
			radioMode = services.RadioModeMainstream
		}
		ids, err := h.radio.GetRecommendations(ctx, songID, limit, radioMode)
		if err == nil && len(ids) > 0 {
			return h.getSongsByIDs(c, ids)
		}
		// Fall through to metadata-based if LLM fails
	}

	// Fallback to metadata-based recommendations
	var artistID, albumID int64
	var year sql.NullInt64
	err := h.db.QueryRowContext(ctx, `
		SELECT al.artist_id, s.album_id, al.year 
		FROM songs s 
		JOIN albums al ON s.album_id = al.id 
		WHERE s.id = ?
	`, songID).Scan(&artistID, &albumID, &year)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "song not found", "code": "NOT_FOUND"})
	}

	return h.radioByMetadata(c, songID, artistID, albumID, year, limit)
}

func (h *Handler) getSongsByIDs(c echo.Context, ids []int64) error {
	ctx := c.Request().Context()

	songs := make([]models.Song, 0, len(ids))
	for _, id := range ids {
		var s models.Song
		var ar models.Artist
		var al models.Album
		var trackNum, duration sql.NullInt64
		var lyrics, lyricsSynced, mbid, coverPath sql.NullString
		var year sql.NullInt64
		err := h.db.QueryRowContext(ctx, `
			SELECT s.id, s.album_id, s.title, s.track_number, s.duration_ms / 1000,
				s.file_path, s.lyrics, s.lyrics_synced, s.mbid,
				ar.id, ar.name, al.id, al.title, al.year, al.cover_path
			FROM songs s
			JOIN albums al ON s.album_id = al.id
			JOIN artists ar ON al.artist_id = ar.id
			WHERE s.id = ?
		`, id).Scan(
			&s.ID, &s.AlbumID, &s.Title, &trackNum, &duration,
			&s.FilePath, &lyrics, &lyricsSynced, &mbid,
			&ar.ID, &ar.Name, &al.ID, &al.Title, &year, &coverPath,
		)
		if err != nil {
			continue
		}
		if trackNum.Valid {
			t := int(trackNum.Int64)
			s.TrackNumber = &t
		}
		if duration.Valid {
			d := int(duration.Int64)
			s.Duration = &d
		}
		if lyrics.Valid {
			s.Lyrics = lyrics.String
		}
		if lyricsSynced.Valid {
			s.LyricsSynced = lyricsSynced.String
		}
		if mbid.Valid {
			s.MBID = &mbid.String
		}
		if year.Valid {
			y := int(year.Int64)
			al.Year = &y
		}
		if coverPath.Valid {
			al.CoverPath = coverPath.String
		}
		al.ArtistID = ar.ID
		al.Artist = &ar
		s.Album = &al
		s.Artist = &ar
		songs = append(songs, s)
	}

	return c.JSON(http.StatusOK, map[string]any{"songs": songs})
}

func (h *Handler) radioByMetadata(c echo.Context, seedID, artistID, albumID int64, year sql.NullInt64, limit int) error {
	ctx := c.Request().Context()

	query := `
		SELECT s.id, s.album_id, s.title, s.track_number, s.duration_ms / 1000,
			s.file_path, s.lyrics, s.lyrics_synced, s.mbid,
			ar.id, ar.name, al.id, al.title, al.year, al.cover_path,
			(CASE WHEN s.album_id = ? THEN 3 ELSE 0 END) +
			(CASE WHEN al.artist_id = ? THEN 2 ELSE 0 END) +
			(CASE WHEN al.year = ? THEN 1 ELSE 0 END) AS score
		FROM songs s
		JOIN albums al ON s.album_id = al.id
		JOIN artists ar ON al.artist_id = ar.id
		WHERE s.id != ?
		ORDER BY score DESC, RANDOM()
		LIMIT ?
	`

	var yearVal any = nil
	if year.Valid {
		yearVal = year.Int64
	}

	rows, err := h.db.QueryContext(ctx, query, albumID, artistID, yearVal, seedID, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "query failed", "code": "INTERNAL_ERROR"})
	}
	defer rows.Close()

	songs := make([]models.Song, 0)
	for rows.Next() {
		var s models.Song
		var ar models.Artist
		var al models.Album
		var score int
		var trackNum, duration sql.NullInt64
		var lyrics, lyricsSynced, mbid, coverPath sql.NullString
		var albumYear sql.NullInt64
		if err := rows.Scan(
			&s.ID, &s.AlbumID, &s.Title, &trackNum, &duration,
			&s.FilePath, &lyrics, &lyricsSynced, &mbid,
			&ar.ID, &ar.Name, &al.ID, &al.Title, &albumYear, &coverPath,
			&score,
		); err != nil {
			continue
		}
		if trackNum.Valid {
			t := int(trackNum.Int64)
			s.TrackNumber = &t
		}
		if duration.Valid {
			d := int(duration.Int64)
			s.Duration = &d
		}
		if lyrics.Valid {
			s.Lyrics = lyrics.String
		}
		if lyricsSynced.Valid {
			s.LyricsSynced = lyricsSynced.String
		}
		if mbid.Valid {
			s.MBID = &mbid.String
		}
		if albumYear.Valid {
			y := int(albumYear.Int64)
			al.Year = &y
		}
		if coverPath.Valid {
			al.CoverPath = coverPath.String
		}
		al.ArtistID = ar.ID
		al.Artist = &ar
		s.Album = &al
		s.Artist = &ar
		songs = append(songs, s)
	}

	return c.JSON(http.StatusOK, map[string]any{"songs": songs})
}
