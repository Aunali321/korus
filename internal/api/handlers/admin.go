package handlers

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/labstack/echo/v4"
)

// StartScan godoc
// @Summary Start library scan
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/scan [post]
func (h *Handler) StartScan(c echo.Context) error {
	id, err := h.scanner.StartScan(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "SCAN_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"scan_id": id, "status": "running"})
}

// ScanStatus godoc
// @Summary Get scan status
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/scan/status [get]
func (h *Handler) ScanStatus(c echo.Context) error {
	status, err := h.scanner.Status(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "SCAN_STATUS_FAILED"})
	}
	return c.JSON(http.StatusOK, status)
}

// SystemInfo godoc
// @Summary System info
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/system [get]
func (h *Handler) SystemInfo(c echo.Context) error {
	var users, songs, artists, albums int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM songs`).Scan(&songs)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM artists`).Scan(&artists)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM albums`).Scan(&albums)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"library": map[string]int{
			"users":   users,
			"songs":   songs,
			"artists": artists,
			"albums":  albums,
		},
		"storage": map[string]interface{}{
			"media_root": h.mediaRoot,
		},
		"server": map[string]interface{}{
			"go_version": runtime.Version(),
			"uptime":     time.Since(startTime).String(),
			"hostname":   hostname(),
		},
	})
}

// CleanupSessions godoc
// @Summary Cleanup sessions
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body map[string]int true "older_than_days"
// @Success 200 {object} map[string]int64
// @Router /api/admin/sessions/cleanup [delete]
func (h *Handler) CleanupSessions(c echo.Context) error {
	var payload struct {
		OlderThanDays int `json:"older_than_days" validate:"required"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	removed, err := h.auth.CleanupSessions(c.Request().Context(), time.Duration(payload.OlderThanDays)*24*time.Hour)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "CLEANUP_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]int64{"deleted": removed})
}

// Enrich godoc
// @Summary MusicBrainz enrich
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body map[string]string true "type/id"
// @Success 200 {object} map[string]string
// @Router /api/admin/musicbrainz/enrich [post]
func (h *Handler) Enrich(c echo.Context) error {
	if h.musicBrainz == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "musicbrainz disabled", "code": "MB_DISABLED"})
	}
	var payload struct {
		Type string `json:"type" validate:"required"`
		ID   string `json:"id" validate:"required"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	mbid, err := h.musicBrainz.Enrich(c.Request().Context(), payload.Type, payload.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "MB_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]string{"mbid": mbid})
}

var startTime = time.Now()

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}
