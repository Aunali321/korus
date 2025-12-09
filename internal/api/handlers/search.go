package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Search godoc
// @Summary Search library
// @Tags Search
// @Produce json
// @Param q query string true "query"
// @Param limit query int false "max items (default 25, max 200)"
// @Param offset query int false "offset"
// @Success 200 {object} map[string]interface{}
// @Router /api/search [get]
func (h *Handler) Search(c echo.Context) error {
	q := c.QueryParam("q")
	limit, offset := parseLimitOffset(c, 25, 200)
	res, err := h.search.Search(c.Request().Context(), q, limit, offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "SEARCH_FAILED"})
	}
	return c.JSON(http.StatusOK, res)
}
