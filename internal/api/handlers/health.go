package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Health godoc
// @Summary Health check
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
	})
}
