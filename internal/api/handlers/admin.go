package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Aunali321/korus/internal/db"
	"github.com/labstack/echo/v4"
	_ "modernc.org/sqlite"
)

// StartScan godoc
// @Summary Start library scan
// @Tags Library
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 409 {object} map[string]string
// @Router /scan [post]
// @Security BearerAuth
func (h *Handler) StartScan(c echo.Context) error {
	scanID, err := h.scanner.StartScan(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, map[string]string{"error": err.Error(), "code": "SCAN_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"scan_id": scanID, "status": "running"})
}

// ScanStatus godoc
// @Summary Get scan status
// @Tags Library
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /scan/status [get]
// @Security BearerAuth
func (h *Handler) ScanStatus(c echo.Context) error {
	status, err := h.scanner.Status(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "SCAN_STATUS_FAILED"})
	}
	return c.JSON(http.StatusOK, status)
}

// SystemInfo godoc
// @Summary System info
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /admin/system [get]
// @Security BearerAuth
func (h *Handler) SystemInfo(c echo.Context) error {
	var users, songs, artists, albums int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM songs`).Scan(&songs)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM artists`).Scan(&artists)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM albums`).Scan(&albums)

	// Get database size
	var dbSize int64
	if info, err := os.Stat(h.dbPath); err == nil {
		dbSize = info.Size()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"version":       "1.0.0",
		"uptime":        int(time.Since(startTime).Seconds()),
		"database_size": dbSize,
		"total_songs":   songs,
		"total_albums":  albums,
		"total_artists": artists,
		"total_users":   users,
		"media_root":    h.mediaRoot,
		"go_version":    runtime.Version(),
		"hostname":      hostname(),
	})
}

// CleanupSessions godoc
// @Summary Cleanup sessions
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body map[string]int true "older_than_days"
// @Success 200 {object} map[string]int64
// @Router /admin/sessions/cleanup [delete]
// @Security BearerAuth
func (h *Handler) CleanupSessions(c echo.Context) error {
	var payload struct {
		OlderThanDays int `json:"older_than_days" validate:"required"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	removed, err := h.auth.CleanupSessions(c.Request().Context(), time.Duration(payload.OlderThanDays)*24*time.Hour)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "CLEANUP_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]int64{"deleted": removed})
}

// Enrich godoc
// @Summary MusicBrainz enrich
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body map[string]string true "type/id"
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /admin/musicbrainz/enrich [post]
// @Security BearerAuth
func (h *Handler) Enrich(c echo.Context) error {
	if h.musicBrainz == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "musicbrainz disabled", "code": "MB_DISABLED"})
	}
	var payload struct {
		Type string `json:"type" validate:"required"`
		ID   string `json:"id" validate:"required"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}
	if err := c.Validate(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "VALIDATION_ERROR"})
	}
	mbid, err := h.musicBrainz.Enrich(c.Request().Context(), payload.Type, payload.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "MB_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]string{"mbid": mbid})
}

var startTime = time.Now()

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

// GetAppSettings godoc
// @Summary Get app settings
// @Tags Admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /admin/settings [get]
// @Security BearerAuth
func (h *Handler) GetAppSettings(c echo.Context) error {
	ctx := c.Request().Context()
	radioEnabled := false
	if val, err := db.GetAppSetting(ctx, h.db, "radio_enabled"); err == nil && val == "true" {
		radioEnabled = true
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"radio_enabled": radioEnabled,
	})
}

// UpdateAppSettings godoc
// @Summary Update app settings
// @Tags Admin
// @Accept json
// @Produce json
// @Param settings body map[string]interface{} true "Settings"
// @Success 200 {object} map[string]interface{}
// @Router /admin/settings [put]
// @Security BearerAuth
func (h *Handler) UpdateAppSettings(c echo.Context) error {
	var payload struct {
		RadioEnabled *bool `json:"radio_enabled"`
	}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid payload", "code": "BAD_REQUEST"})
	}

	ctx := c.Request().Context()
	if payload.RadioEnabled != nil {
		val := "false"
		if *payload.RadioEnabled {
			val = "true"
		}
		if err := db.SetAppSetting(ctx, h.db, "radio_enabled", val); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to save settings", "code": "INTERNAL_ERROR"})
		}
	}

	radioEnabled := false
	if val, err := db.GetAppSetting(ctx, h.db, "radio_enabled"); err == nil && val == "true" {
		radioEnabled = true
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"radio_enabled": radioEnabled,
	})
}

// BackupDatabase godoc
// @Summary Backup database
// @Description Creates a backup of the SQLite database and streams it to the client
// @Tags Admin
// @Produce application/octet-stream
// @Success 200 {file} binary "Database backup file"
// @Failure 500 {object} map[string]string
// @Router /admin/database/backup [get]
// @Security BearerAuth
func (h *Handler) BackupDatabase(c echo.Context) error {
	tempFile, err := os.CreateTemp("", "korus-backup-*.db")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to create temp file", "code": "BACKUP_FAILED"})
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	backupDB, err := sql.Open("sqlite", tempPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to open backup db", "code": "BACKUP_FAILED"})
	}
	defer backupDB.Close()

	_, err = h.db.Exec(fmt.Sprintf(`VACUUM INTO '%s'`, tempPath))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("backup failed: %v", err), "code": "BACKUP_FAILED"})
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("korus-backup-%s.db", timestamp)

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Response().Header().Set("Content-Type", "application/octet-stream")

	return c.File(tempPath)
}

// RestoreDatabase godoc
// @Summary Restore database
// @Description Restores the database from an uploaded backup file. Server will restart after restore.
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Param backup formData file true "Database backup file"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/database/restore [post]
// @Security BearerAuth
func (h *Handler) RestoreDatabase(c echo.Context) error {
	file, err := c.FormFile("backup")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "backup file required", "code": "MISSING_FILE"})
	}

	if file.Size > 1<<30 {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "file too large (max 1GB)", "code": "FILE_TOO_LARGE"})
	}

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to open uploaded file", "code": "RESTORE_FAILED"})
	}
	defer src.Close()

	tempFile, err := os.CreateTemp("", "korus-restore-*.db")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to create temp file", "code": "RESTORE_FAILED"})
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	_, err = io.Copy(tempFile, src)
	tempFile.Close()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to save uploaded file", "code": "RESTORE_FAILED"})
	}

	if err := validateSQLiteDatabase(tempPath); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid database file: %v", err), "code": "INVALID_DATABASE"})
	}

	dbDir := filepath.Dir(h.dbPath)
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	safetyBackupPath := filepath.Join(dbDir, fmt.Sprintf("korus.db.backup.%s", timestamp))

	_, err = h.db.Exec(fmt.Sprintf(`VACUUM INTO '%s'`, safetyBackupPath))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to create safety backup: %v", err), "code": "BACKUP_FAILED"})
	}

	log.Printf("Created safety backup at: %s", safetyBackupPath)

	h.db.Close()

	if err := copyFile(tempPath, h.dbPath); err != nil {
		copyFile(safetyBackupPath, h.dbPath)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to restore database: %v", err), "code": "RESTORE_FAILED"})
	}

	os.Remove(h.dbPath + "-wal")
	os.Remove(h.dbPath + "-shm")

	log.Printf("Database restored from uploaded file. Server will restart...")

	c.JSON(http.StatusOK, map[string]string{
		"message":       "Database restored successfully. Server will exit and should be restarted by your process manager (Docker, systemd, etc.)",
		"safety_backup": safetyBackupPath,
	})

	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	return nil
}

func validateSQLiteDatabase(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("cannot open as sqlite: %w", err)
	}
	defer db.Close()

	var result string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("integrity check returned: %s", result)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check tables: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("not a valid Korus database (missing users table)")
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return dstFile.Sync()
}
