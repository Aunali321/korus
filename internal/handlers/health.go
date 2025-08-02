package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"korus/internal/database"
)

type HealthHandler struct {
	db *database.DB
}

func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"message": "Korus server is running",
	})
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Check database health
	dbHealthy := true
	var dbError string
	if err := h.db.Health(ctx); err != nil {
		dbHealthy = false
		dbError = err.Error()
	}

	status := "healthy"
	httpStatus := http.StatusOK
	
	if !dbHealthy {
		status = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	response := gin.H{
		"status": status,
		"checks": gin.H{
			"database": gin.H{
				"status": func() string {
					if dbHealthy {
						return "healthy"
					}
					return "unhealthy"
				}(),
			},
		},
	}

	if !dbHealthy {
		response["checks"].(gin.H)["database"].(gin.H)["error"] = dbError
	}

	// Add database stats if healthy
	if dbHealthy {
		stats := h.db.Stats()
		response["checks"].(gin.H)["database"].(gin.H)["stats"] = gin.H{
			"total_connections":     stats.TotalConns(),
			"idle_connections":      stats.IdleConns(),
			"acquired_connections":  stats.AcquiredConns(),
			"constructing_connections": stats.ConstructingConns(),
		}
	}

	c.JSON(httpStatus, response)
}