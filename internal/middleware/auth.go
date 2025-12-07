package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"korus/internal/auth"
	"korus/internal/models"
)

func AuthRequired(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "missing authorization header",
			})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		user, err := authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid or expired token",
			})
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)

		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "authentication required",
			})
			c.Abort()
			return
		}

		u, ok := user.(*models.User)
		if !ok || u.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "admin privileges required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func OptionalAuth(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		user, err := authService.ValidateToken(c.Request.Context(), token)
		if err == nil {
			c.Set("user", user)
			c.Set("user_id", user.ID)
			c.Set("user_role", user.Role)
		}

		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	u, ok := user.(*models.User)
	return u, ok
}

func GetCurrentUserID(c *gin.Context) (int, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	id, ok := userID.(int)
	return id, ok
}

func IsCurrentUserAdmin(c *gin.Context) bool {
	role, exists := c.Get("user_role")
	if !exists {
		return false
	}

	r, ok := role.(string)
	return ok && r == "admin"
}
