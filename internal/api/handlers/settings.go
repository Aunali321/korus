package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/Aunali321/korus/internal/models"
	"github.com/labstack/echo/v4"
)

type UserSettings struct {
	StreamingPreset  string `json:"streaming_preset"`
	StreamingFormat  string `json:"streaming_format,omitempty"`
	StreamingBitrate int    `json:"streaming_bitrate,omitempty"`
}

// GetSettings godoc
// @Summary Get user settings
// @Tags Settings
// @Produce json
// @Success 200 {object} UserSettings
// @Router /api/settings [get]
func (h *Handler) GetSettings(c echo.Context) error {
	user := c.Get("user").(models.User)
	userID := user.ID

	var settings UserSettings
	var format sql.NullString
	var bitrate sql.NullInt64

	err := h.db.QueryRowContext(c.Request().Context(), `
		SELECT streaming_preset, streaming_format, streaming_bitrate
		FROM user_settings WHERE user_id = ?
	`, userID).Scan(&settings.StreamingPreset, &format, &bitrate)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusOK, UserSettings{StreamingPreset: "original"})
	}
	if err != nil {
		slog.Error("failed to get settings", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to get settings"})
	}

	if format.Valid {
		settings.StreamingFormat = format.String
	}
	if bitrate.Valid {
		settings.StreamingBitrate = int(bitrate.Int64)
	}

	return c.JSON(http.StatusOK, settings)
}

// UpdateSettings godoc
// @Summary Update user settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body UserSettings true "Settings"
// @Success 200 {object} UserSettings
// @Router /api/settings [put]
func (h *Handler) UpdateSettings(c echo.Context) error {
	user := c.Get("user").(models.User)
	userID := user.ID

	var req UserSettings
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.StreamingPreset == "" {
		req.StreamingPreset = "original"
	}

	validPresets := map[string]bool{"original": true, "lossless": true, "very_high": true, "high": true, "medium": true, "low": true, "custom": true}
	if !validPresets[req.StreamingPreset] {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid preset"})
	}

	var format any = nil
	var bitrate any = nil
	if req.StreamingPreset == "custom" || req.StreamingFormat != "" {
		if _, err := h.transcoder.Validate(req.StreamingFormat, req.StreamingBitrate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		format = req.StreamingFormat
		bitrate = req.StreamingBitrate
	}

	_, err := h.db.ExecContext(c.Request().Context(), `
		INSERT INTO user_settings (user_id, streaming_preset, streaming_format, streaming_bitrate, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			streaming_preset = excluded.streaming_preset,
			streaming_format = excluded.streaming_format,
			streaming_bitrate = excluded.streaming_bitrate,
			updated_at = CURRENT_TIMESTAMP
	`, userID, req.StreamingPreset, format, bitrate)

	if err != nil {
		slog.Error("failed to save settings", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
	}

	return c.JSON(http.StatusOK, req)
}
