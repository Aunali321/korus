package handlers

import (
	"net/http"
	"strings"

	"korus/internal/services"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService *services.AdminService
}

func NewAdminHandler(adminService *services.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

// TriggerLibraryScan triggers a full library scan (async)
func (h *AdminHandler) TriggerLibraryScan(c *gin.Context) {
	force := parseBoolQuery(c, "force")

	jobID, err := h.adminService.TriggerLibraryScan(c.Request.Context(), force)
	if err != nil {
		if err.Error() == "indexer already running" {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "scan_in_progress",
				"message": "A library scan is already running",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to start library scan",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Library scan started",
		"jobId":   jobID,
		"force":   force,
	})
}

// GetScanJob returns the status of a scan job
func (h *AdminHandler) GetScanJob(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_job_id",
			"message": "Job ID is required",
		})
		return
	}

	job, err := h.adminService.GetScanJob(jobID)
	if err != nil {
		if err.Error() == "job not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "job_not_found",
				"message": "Scan job not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get scan job status",
		})
		return
	}

	c.JSON(http.StatusOK, mapScanJob(job))
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
		"message":      "Sessions cleanup completed",
		"deletedCount": deletedCount,
	})
}

func parseBoolQuery(c *gin.Context, key string) bool {
	value := strings.ToLower(c.DefaultQuery(key, "false"))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func mapIndexerResult(res *services.LibraryScanResult) gin.H {
	if res == nil {
		return gin.H{}
	}
	errors := make([]string, 0, len(res.Errors))
	for _, err := range res.Errors {
		errors = append(errors, err.Error())
	}
	return gin.H{
		"startedAt":       res.StartedAt,
		"completedAt":     res.CompletedAt,
		"duration":        res.Duration.String(),
		"filesDiscovered": res.FilesDiscovered,
		"filesQueued":     res.FilesQueued,
		"filesNew":        res.FilesNew,
		"filesUpdated":    res.FilesUpdated,
		"filesRemoved":    res.FilesRemoved,
		"ingested":        res.Ingested,
		"errors":          errors,
	}
}

func mapScanJob(job *services.LibraryScanJob) gin.H {
	if job == nil {
		return gin.H{}
	}
	response := gin.H{
		"id":        job.ID,
		"status":    job.Status,
		"phase":     job.Phase,
		"progress":  job.Progress,
		"total":     job.Total,
		"force":     job.Force,
		"startedAt": job.StartedAt,
	}
	if job.CompletedAt != nil {
		response["completedAt"] = job.CompletedAt
	}
	if job.Result != nil {
		response["result"] = mapIndexerResult(job.Result)
	}
	if job.Error != "" {
		response["error"] = job.Error
	}
	return response
}
