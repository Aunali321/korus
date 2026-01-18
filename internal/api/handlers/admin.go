package handlers

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/Aunali321/korus/internal/db"
	"github.com/labstack/echo/v4"
)

// StartScan godoc
// @Summary Start library scan
// @Tags Library
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 409 {object} map[string]string
// @Router /scan [post]
// @Security BearerAuth
func (h *Handler) StartScan(c echo.Context) error {
	scanID, err := h.scanner.StartScan(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, map[string]string{"error": err.Error(), "code": "SCAN_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"scan_id": scanID, "status": "running"})
}

// ScanStatus godoc
// @Summary Get scan status
// @Tags Library
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /scan/status [get]
// @Security BearerAuth
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
// @Router /admin/system [get]
// @Security BearerAuth
func (h *Handler) SystemInfo(c echo.Context) error {
	var users, songs, artists, albums int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM songs`).Scan(&songs)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM artists`).Scan(&artists)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM albums`).Scan(&albums)

	// Get database size
	var dbSize int64
	dbFile := "korus.db"
	if info, err := os.Stat(dbFile); err == nil {
		dbSize = info.Size()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"version":       "1.0.0",
		"uptime":        int(time.Since(startTime).Seconds()),
		"database_size": dbSize,
		"total_songs":   songs,
		"total_albums":  albums,
		"total_artists": artists,
		"total_users":   users,
		"media_root":    h.mediaRoot,
		"go_version":    runtime.Version(),
		"hostname":      hostname(),
	})
}

// CleanupSessions godoc
// @Summary Cleanup sessions
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body map[string]int true "older_than_days"
// @Success 200 {object} map[string]int64
// @Router /admin/sessions/cleanup [delete]
// @Security BearerAuth
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
// @Failure 503 {object} map[string]string
// @Router /admin/musicbrainz/enrich [post]
// @Security BearerAuth
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

// GetAppSettings godoc
// @Summary Get app settings
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /admin/settings [get]
// @Security BearerAuth
func (h *Handler) GetAppSettings(c echo.Context) error {
	ctx := c.Request().Context()
	radioEnabled := false
	if val, err := db.GetAppSetting(ctx, h.db, "radio_enabled"); err == nil && val == "true" {
		radioEnabled = true
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"radio_enabled": radioEnabled,
	})
}

// UpdateAppSettings godoc
// @Summary Update app settings
// @Tags Admin
// @Accept json
// @Produce json
// @Param settings body map[string]interface{} true "Settings"
// @Success 200 {object} map[string]interface{}
// @Router /admin/settings [put]
// @Security BearerAuth
func (h *Handler) UpdateAppSettings(c echo.Context) error {
	var payload struct {
		RadioEnabled *bool `json:"radio_enabled"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}

	ctx := c.Request().Context()
	if payload.RadioEnabled != nil {
		val := "false"
		if *payload.RadioEnabled {
			val = "true"
		}
		if err := db.SetAppSetting(ctx, h.db, "radio_enabled", val); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to save settings", "code": "INTERNAL_ERROR"})
		}
	}

	radioEnabled := false
	if val, err := db.GetAppSetting(ctx, h.db, "radio_enabled"); err == nil && val == "true" {
		radioEnabled = true
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"radio_enabled": radioEnabled,
	})
}
