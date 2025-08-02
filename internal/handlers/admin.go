package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"korus/internal/services"
)

type AdminHandler struct {
	adminService *services.AdminService
}

func NewAdminHandler(adminService *services.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

// TriggerLibraryScan triggers a full library scan
func (h *AdminHandler) TriggerLibraryScan(c *gin.Context) {
	jobID, err := h.adminService.TriggerLibraryScan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to trigger library scan",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Library scan triggered successfully",
		"job_id":  jobID,
	})
}

// GetSystemStatus returns comprehensive system status
func (h *AdminHandler) GetSystemStatus(c *gin.Context) {
	status, err := h.adminService.GetSystemStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get system status",
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetScanHistory returns recent scan history
func (h *AdminHandler) GetScanHistory(c *gin.Context) {
	limit := parseIntParam(c, "limit", 10)
	if limit > 50 {
		limit = 50 // Cap at 50
	}

	history, err := h.adminService.GetRecentScanHistory(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get scan history",
		})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetPendingJobs returns currently pending/processing jobs
func (h *AdminHandler) GetPendingJobs(c *gin.Context) {
	jobs, err := h.adminService.GetPendingJobs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get pending jobs",
		})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

// CleanupJobs removes old completed/failed jobs
func (h *AdminHandler) CleanupJobs(c *gin.Context) {
	// Default to cleaning up jobs older than 7 days
	days := parseIntParam(c, "days", 7)
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365 // Cap at 1 year
	}

	deletedCount, err := h.adminService.CleanupOldJobs(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to cleanup jobs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Jobs cleanup completed",
		"deleted_count":  deletedCount,
		"older_than_days": days,
	})
}

// CleanupSessions removes expired user sessions
func (h *AdminHandler) CleanupSessions(c *gin.Context) {
	deletedCount, err := h.adminService.CleanupOldSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to cleanup sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Sessions cleanup completed",
		"deleted_count": deletedCount,
	})
}