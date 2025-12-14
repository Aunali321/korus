package handlers

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

func parseLimitOffset(c echo.Context, def, max int) (limit int, offset int) {
	limit = def
	offset = 0
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > max {
		limit = max
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

// parseOptionalLimit returns -1 (unlimited) by default, or the specified limit if provided
func parseOptionalLimit(c echo.Context) int {
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return -1 // SQLite treats -1 as LIMIT ALL
}
