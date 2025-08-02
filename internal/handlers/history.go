package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"korus/internal/middleware"
	"korus/internal/services"
)

type HistoryHandler struct {
	historyService *services.HistoryService
}

func NewHistoryHandler(historyService *services.HistoryService) *HistoryHandler {
	return &HistoryHandler{
		historyService: historyService,
	}
}

type ScrobbleRequest struct {
	SongID       int       `json:"songId" binding:"required"`
	PlayedAt     time.Time `json:"playedAt"`
	PlayDuration *int      `json:"playDuration,omitempty"`
}

// Scrobble records a song play event
func (h *HistoryHandler) Scrobble(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	var req ScrobbleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// If no playedAt time provided, use current time
	if req.PlayedAt.IsZero() {
		req.PlayedAt = time.Now()
	}

	// Get client IP address
	clientIP := c.ClientIP()

	err := h.historyService.RecordPlay(c.Request.Context(), user.ID, req.SongID, req.PlayedAt, req.PlayDuration, &clientIP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to record play",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Play recorded successfully",
	})
}

// GetRecentHistory returns user's recent listening history
func (h *HistoryHandler) GetRecentHistory(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	// Parse limit parameter
	limit := parseIntParam(c, "limit", 50)
	if limit > 200 {
		limit = 200 // Cap at 200 to prevent excessive queries
	}

	history, err := h.historyService.GetRecentHistory(c.Request.Context(), user.ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get listening history",
		})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetUserStats returns user's listening statistics
func (h *HistoryHandler) GetUserStats(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	stats, err := h.historyService.GetUserStats(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get user statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetHomeData returns personalized home page data
func (h *HistoryHandler) GetHomeData(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	// Parse limit parameter for each section
	limit := parseIntParam(c, "limit", 10)
	if limit > 50 {
		limit = 50 // Cap at 50 to prevent excessive queries
	}

	homeData, err := h.historyService.GetHomeData(c.Request.Context(), user.ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get home data",
		})
		return
	}

	c.JSON(http.StatusOK, homeData)
}