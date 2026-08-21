package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/services"
)

// serviceError maps a service-layer error onto the HTTP response shape the
// clients expect. notFound names the missing entity, e.g. "playlist not found".
func serviceError(err error, notFound, failureCode string) error {
	switch {
	case errors.Is(err, services.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": notFound, "code": "NOT_FOUND"})
	case errors.Is(err, services.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
	case errors.Is(err, services.ErrInvalid):
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "BAD_REQUEST"})
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": failureCode})
	}
}

func bindAndValidate(c echo.Context, payload any) error {
	if err := c.Bind(payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	return nil
}

func pathID(c echo.Context, name string) int64 {
	id, _ := strconv.ParseInt(c.Param(name), 10, 64)
	return id
}

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
