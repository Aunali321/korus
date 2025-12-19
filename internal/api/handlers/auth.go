package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/models"
)

type registerRequest struct {
	Username string `json:"username" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Register godoc
// @Summary Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body registerRequest true "registration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /auth/register [post]
func (h *Handler) Register(c echo.Context) error {
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	user, tokens, err := h.auth.Register(c.Request().Context(), req.Username, req.Email, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "REGISTER_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"user": sanitizeUser(user), "access_token": tokens.Access, "refresh_token": tokens.Refresh})
}

// Login godoc
// @Summary Login user
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "login"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *Handler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	user, tokens, err := h.auth.Login(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": err.Error(), "code": "UNAUTHORIZED"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"user": sanitizeUser(user), "access_token": tokens.Access, "refresh_token": tokens.Refresh})
}

// Logout godoc
// @Summary Logout user
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]bool
// @Failure 401 {object} map[string]string
// @Router /auth/logout [post]
func (h *Handler) Logout(c echo.Context) error {
	token := bearerToken(c.Request().Header.Get("Authorization"))
	if token == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "missing token", "code": "UNAUTHORIZED"})
	}
	if err := h.auth.Logout(c.Request().Context(), token); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "LOGOUT_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// Refresh godoc
// @Summary Refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body map[string]string true "refresh token"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /auth/refresh [post]
func (h *Handler) Refresh(c echo.Context) error {
	var payload struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	user, tokens, err := h.auth.Refresh(c.Request().Context(), payload.RefreshToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": err.Error(), "code": "UNAUTHORIZED"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"user": sanitizeUser(user), "access_token": tokens.Access, "refresh_token": tokens.Refresh})
}

// Me godoc
// @Summary Current user
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /auth/me [get]
func (h *Handler) Me(c echo.Context) error {
	user, err := currentUser(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized", "code": "UNAUTHORIZED"})
	}
	return c.JSON(http.StatusOK, sanitizeUser(user))
}

func currentUser(c echo.Context) (models.User, error) {
	v := c.Get("user")
	if v == nil {
		return models.User{}, echo.ErrUnauthorized
	}
	user, ok := v.(models.User)
	if !ok {
		return models.User{}, echo.ErrUnauthorized
	}
	return user, nil
}

func sanitizeUser(u models.User) map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"username":   u.Username,
		"email":      u.Email,
		"role":       u.Role,
		"created_at": u.CreatedAt,
	}
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}
