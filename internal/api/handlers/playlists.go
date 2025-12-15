package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/db"
)

type playlistRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
}

// ListPlaylists godoc
// @Summary List playlists
// @Tags Playlists
// @Produce json
// @Param limit query int false "max items (default 50, max 200)"
// @Param offset query int false "offset"
// @Success 200 {array} map[string]interface{}
// @Router /api/playlists [get]
func (h *Handler) ListPlaylists(c echo.Context) error {
	user, _ := currentUser(c)
	limit, offset := parseLimitOffset(c, 50, 200)
	rows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT p.id, p.user_id, p.name, p.description, p.public, p.created_at, u.username,
		       (SELECT COUNT(*) FROM playlist_songs ps WHERE ps.playlist_id = p.id) as song_count,
		       (SELECT ps2.song_id FROM playlist_songs ps2 WHERE ps2.playlist_id = p.id ORDER BY ps2.position LIMIT 1) as first_song_id
		FROM playlists p
		JOIN users u ON u.id = p.user_id
		WHERE p.public = 1 OR p.user_id = ?
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`, user.ID, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "PLAYLIST_LIST_FAILED"})
	}
	defer rows.Close()
	var res = []map[string]any{}
	for rows.Next() {
		var id, uid int64
		var name, desc string
		var pub bool
		var created string
		var owner string
		var songCount int
		var firstSongID *int64
		if err := rows.Scan(&id, &uid, &name, &desc, &pub, &created, &owner, &songCount, &firstSongID); err == nil {
			item := map[string]any{
				"id":          id,
				"name":        name,
				"description": desc,
				"public":      pub,
				"created_at":  created,
				"owner":       map[string]any{"id": uid, "username": owner},
				"song_count":  songCount,
			}
			if firstSongID != nil {
				item["first_song_id"] = *firstSongID
			}
			res = append(res, item)
		}
	}
	return c.JSON(http.StatusOK, res)
}

// CreatePlaylist godoc
// @Summary Create playlist
// @Tags Playlists
// @Accept json
// @Produce json
// @Param body body playlistRequest true "playlist"
// @Success 200 {object} map[string]interface{}
// @Router /api/playlists [post]
func (h *Handler) CreatePlaylist(c echo.Context) error {
	user, _ := currentUser(c)
	var req playlistRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	res, err := h.db.ExecContext(c.Request().Context(), `
		INSERT INTO playlists(user_id, name, description, public) VALUES(?, ?, ?, ?)
	`, user.ID, req.Name, req.Description, req.Public)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "PLAYLIST_CREATE_FAILED"})
	}
	id, _ := res.LastInsertId()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"public":      req.Public,
	})
}

// GetPlaylist godoc
// @Summary Get playlist
// @Tags Playlists
// @Produce json
// @Param id path int true "Playlist ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /api/playlists/{id} [get]
func (h *Handler) GetPlaylist(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var name, desc string
	var ownerID int64
	var pub bool
	err := h.db.QueryRowContext(c.Request().Context(), `
		SELECT id, user_id, name, description, public FROM playlists WHERE id = ?
	`, id).Scan(&id, &ownerID, &name, &desc, &pub)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "playlist not found", "code": "NOT_FOUND"})
	}
	if !pub && ownerID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
	}
	songs, _ := db.GetSongsByPlaylist(c.Request().Context(), h.db, id)
	return c.JSON(http.StatusOK, map[string]any{
		"id":          id,
		"name":        name,
		"description": desc,
		"public":      pub,
		"songs":       songs,
		"owner":       map[string]interface{}{"id": ownerID},
	})
}

// UpdatePlaylist godoc
// @Summary Update playlist
// @Tags Playlists
// @Accept json
// @Produce json
// @Param id path int true "Playlist ID"
// @Param body body playlistRequest true "playlist"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /api/playlists/{id} [put]
func (h *Handler) UpdatePlaylist(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var owner int64
	if err := h.db.QueryRowContext(c.Request().Context(), `SELECT user_id FROM playlists WHERE id = ?`, id).Scan(&owner); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "playlist not found", "code": "NOT_FOUND"})
	}
	if owner != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
	}
	var req playlistRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	_, err := h.db.ExecContext(c.Request().Context(), `
		UPDATE playlists SET name = ?, description = ?, public = ? WHERE id = ?
	`, req.Name, req.Description, req.Public, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "PLAYLIST_UPDATE_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":          id,
		"name":        req.Name,
		"description": req.Description,
		"public":      req.Public,
	})
}

// DeletePlaylist godoc
// @Summary Delete playlist
// @Tags Playlists
// @Param id path int true "Playlist ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Router /api/playlists/{id} [delete]
func (h *Handler) DeletePlaylist(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var owner int64
	if err := h.db.QueryRowContext(c.Request().Context(), `SELECT user_id FROM playlists WHERE id = ?`, id).Scan(&owner); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "playlist not found", "code": "NOT_FOUND"})
	}
	if owner != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
	}
	if _, err := h.db.ExecContext(c.Request().Context(), `DELETE FROM playlists WHERE id = ?`, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "PLAYLIST_DELETE_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// AddPlaylistSong godoc
// @Summary Add song to playlist
// @Tags Playlists
// @Accept json
// @Produce json
// @Param id path int true "Playlist ID"
// @Param body body map[string]int true "song payload"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Router /api/playlists/{id}/songs [post]
func (h *Handler) AddPlaylistSong(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var owner int64
	if err := h.db.QueryRowContext(c.Request().Context(), `SELECT user_id FROM playlists WHERE id = ?`, id).Scan(&owner); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "playlist not found", "code": "NOT_FOUND"})
	}
	if owner != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
	}
	var payload struct {
		SongID   int64 `json:"song_id" validate:"required"`
		Position int   `json:"position"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	if !h.songExists(c.Request().Context(), payload.SongID) {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "song not found", "code": "NOT_FOUND"})
	}
	if payload.Position == 0 {
		payload.Position = int(time.Now().Unix())
	}
	if _, err := h.db.ExecContext(c.Request().Context(), `
		INSERT OR REPLACE INTO playlist_songs(playlist_id, song_id, position) VALUES(?, ?, ?)
	`, id, payload.SongID, payload.Position); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "PLAYLIST_ADD_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// DeletePlaylistSong godoc
// @Summary Remove song from playlist
// @Tags Playlists
// @Param id path int true "Playlist ID"
// @Param song_id path int true "Song ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Router /api/playlists/{id}/songs/{song_id} [delete]
func (h *Handler) DeletePlaylistSong(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	songID, _ := strconv.ParseInt(c.Param("song_id"), 10, 64)
	var owner int64
	if err := h.db.QueryRowContext(c.Request().Context(), `SELECT user_id FROM playlists WHERE id = ?`, id).Scan(&owner); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "playlist not found", "code": "NOT_FOUND"})
	}
	if owner != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
	}
	if _, err := h.db.ExecContext(c.Request().Context(), `DELETE FROM playlist_songs WHERE playlist_id = ? AND song_id = ?`, id, songID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "PLAYLIST_REMOVE_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// ReorderPlaylistSongs godoc
// @Summary Reorder playlist songs
// @Tags Playlists
// @Accept json
// @Produce json
// @Param id path int true "Playlist ID"
// @Param body body map[string][]int64 true "song ids"
// @Success 200 {object} map[string]bool
// @Router /api/playlists/{id}/reorder [put]
func (h *Handler) ReorderPlaylistSongs(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var owner int64
	if err := h.db.QueryRowContext(c.Request().Context(), `SELECT user_id FROM playlists WHERE id = ?`, id).Scan(&owner); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "playlist not found", "code": "NOT_FOUND"})
	}
	if owner != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
	}
	var payload struct {
		SongIDs []int64 `json:"song_ids" validate:"required"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	// ensure all songs exist and belong to playlist
	for _, sid := range payload.SongIDs {
		if !h.songExists(c.Request().Context(), sid) {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "song not found", "code": "NOT_FOUND"})
		}
	}
	tx, err := h.db.BeginTx(c.Request().Context(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "REORDER_FAILED"})
	}
	for idx, sid := range payload.SongIDs {
		if _, err := tx.ExecContext(c.Request().Context(), `UPDATE playlist_songs SET position = ? WHERE playlist_id = ? AND song_id = ?`, idx+1, id, sid); err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "REORDER_FAILED"})
		}
	}
	if err := tx.Commit(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "REORDER_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}
