package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/models"
	"github.com/Aunali321/korus/internal/services"
)

func Auth(auth *services.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var token string
			header := c.Request().Header.Get("Authorization")
			if header != "" {
				parts := strings.SplitN(header, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					token = parts[1]
				}
			}
			// Fallback to query parameter for media endpoints
			if token == "" {
				token = c.QueryParam("token")
			}
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "missing authorization", "code": "UNAUTHORIZED"})
			}
			user, err := auth.ValidateToken(c.Request().Context(), token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "invalid token", "code": "UNAUTHORIZED"})
			}
			c.Set("user", user)
			return next(c)
		}
	}
}

func AdminOnly(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		v := c.Get("user")
		if v == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized", "code": "UNAUTHORIZED"})
		}
		user := v.(models.User)
		if user.Role != "admin" {
			return echo.NewHTTPError(http.StatusForbidden, map[string]string{"error": "forbidden", "code": "FORBIDDEN"})
		}
		return next(c)
	}
}
