package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/Aunali321/korus/internal/models"
	"github.com/labstack/echo/v4"
)

type UserSettings struct {
	Shuffle bool   `json:"shuffle"`
	Repeat  string `json:"repeat"`
}

// GetSettings godoc
// @Summary Get user settings
// @Tags Settings
// @Produce json
// @Success 200 {object} UserSettings
// @Router /settings [get]
func (h *Handler) GetSettings(c echo.Context) error {
	user := c.Get("user").(models.User)
	userID := user.ID

	var shuffle int
	var repeat string

	err := h.db.QueryRowContext(c.Request().Context(), `
		SELECT shuffle, repeat FROM user_settings WHERE user_id = ?
	`, userID).Scan(&shuffle, &repeat)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusOK, UserSettings{Shuffle: false, Repeat: "off"})
	}
	if err != nil {
		slog.Error("failed to get settings", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to get settings"})
	}

	return c.JSON(http.StatusOK, UserSettings{Shuffle: shuffle == 1, Repeat: repeat})
}

// UpdateSettings godoc
// @Summary Update user settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body UserSettings true "Settings"
// @Success 200 {object} UserSettings
// @Router /settings [put]
func (h *Handler) UpdateSettings(c echo.Context) error {
	user := c.Get("user").(models.User)
	userID := user.ID

	var req UserSettings
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	validRepeats := map[string]bool{"off": true, "one": true, "all": true}
	if !validRepeats[req.Repeat] {
		req.Repeat = "off"
	}

	shuffleInt := 0
	if req.Shuffle {
		shuffleInt = 1
	}

	_, err := h.db.ExecContext(c.Request().Context(), `
		INSERT INTO user_settings (user_id, shuffle, repeat, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			shuffle = excluded.shuffle,
			repeat = excluded.repeat,
			updated_at = CURRENT_TIMESTAMP
	`, userID, shuffleInt, req.Repeat)

	if err != nil {
		slog.Error("failed to save settings", "error", err, "user_id", userID)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to save settings"})
	}

	return c.JSON(http.StatusOK, req)
}
