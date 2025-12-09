package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type historyRequest struct {
	SongID           int64   `json:"song_id" validate:"required"`
	DurationListened int     `json:"duration_listened"`
	Timestamp        *int64  `json:"timestamp"`
	Source           string  `json:"source"`
	CompletionRate   float64 `json:"completion_rate"`
}

// RecordHistory godoc
// @Summary Record play history
// @Tags History
// @Accept json
// @Produce json
// @Param body body historyRequest true "history"
// @Success 200 {object} map[string]bool
// @Router /api/history [post]
func (h *Handler) RecordHistory(c echo.Context) error {
	user, _ := currentUser(c)
	var req historyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	ts := time.Now()
	if req.Timestamp != nil {
		ts = time.Unix(*req.Timestamp, 0)
	}
	_, err := h.db.ExecContext(c.Request().Context(), `
		INSERT INTO play_history(user_id, song_id, played_at, duration_listened, completion_rate, source)
		VALUES (?, ?, ?, ?, ?, ?)
	`, user.ID, req.SongID, ts, req.DurationListened, req.CompletionRate, req.Source)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "HISTORY_SAVE_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// ListHistory godoc
// @Summary List play history
// @Tags History
// @Produce json
// @Param limit query int false "limit (default 50, max 200)"
// @Param offset query int false "offset"
// @Success 200 {array} map[string]interface{}
// @Router /api/history [get]
func (h *Handler) ListHistory(c echo.Context) error {
	user, _ := currentUser(c)
	limit, offset := parseLimitOffset(c, 50, 200)
	rows, err := h.db.QueryContext(c.Request().Context(), `
		SELECT ph.id, ph.song_id, ph.played_at, ph.duration_listened, ph.completion_rate, ph.source,
		       s.title, s.file_path
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		WHERE ph.user_id = ?
		ORDER BY ph.played_at DESC
		LIMIT ? OFFSET ?
	`, user.ID, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "HISTORY_QUERY_FAILED"})
	}
	defer rows.Close()
	var res []map[string]interface{}
	for rows.Next() {
		var id, songID int64
		var playedAt string
		var dur int
		var comp float64
		var source, title, path string
		if err := rows.Scan(&id, &songID, &playedAt, &dur, &comp, &source, &title, &path); err == nil {
			res = append(res, map[string]interface{}{
				"id":                id,
				"song_id":           songID,
				"played_at":         playedAt,
				"duration_listened": dur,
				"completion_rate":   comp,
				"source":            source,
				"song": map[string]interface{}{
					"id":        songID,
					"title":     title,
					"file_path": path,
				},
			})
		}
	}
	return c.JSON(http.StatusOK, res)
}
