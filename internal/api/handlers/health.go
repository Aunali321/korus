package handlers

import (
	"net/http"
	"time"

	"github.com/Aunali321/korus/internal/db"
	"github.com/labstack/echo/v4"
)

// Health godoc
// @Summary Health check
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *Handler) Health(c echo.Context) error {
	radioEnabled := false
	if val, err := db.GetAppSetting(c.Request().Context(), h.db, "radio_enabled"); err == nil && val == "true" {
		radioEnabled = true
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":        "ok",
		"timestamp":     time.Now().UTC(),
		"radio_enabled": radioEnabled,
	})
}
