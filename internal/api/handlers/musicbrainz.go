package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// SubmitListen godoc
// @Summary Submit listen to ListenBrainz
// @Tags MusicBrainz
// @Accept json
// @Produce json
// @Param body body map[string]int64 true "song_id"
// @Success 200 {object} map[string]bool
// @Router /api/musicbrainz/submit-listen [post]
func (h *Handler) SubmitListen(c echo.Context) error {
	if h.listenBrainz == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "listenbrainz disabled", "code": "LB_DISABLED"})
	}
	var payload struct {
		SongID int64 `json:"song_id" validate:"required"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	var title string
	if err := h.db.QueryRowContext(c.Request().Context(), `SELECT title FROM songs WHERE id = ?`, payload.SongID).Scan(&title); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "song not found", "code": "NOT_FOUND"})
	}
	go h.listenBrainz.SubmitListen(c.Request().Context(), title, time.Now().Unix())
	return c.JSON(http.StatusOK, map[string]bool{"submitted": true})
}

// Recommendations godoc
// @Summary Get recommendations
// @Tags MusicBrainz
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/musicbrainz/recommendations [get]
func (h *Handler) Recommendations(c echo.Context) error {
	// Placeholder stub.
	return c.JSON(http.StatusOK, map[string]interface{}{"songs": []interface{}{}})
}
