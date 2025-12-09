package handlers

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/services"
)

// Stream godoc
// @Summary Stream or transcode song
// @Tags Player
// @Param id path int true "Song ID"
// @Param format query string false "mp3|aac|opus"
// @Param bitrate query int false "bitrate kbps"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/stream/{id} [get]
func (h *Handler) Stream(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var path string
	err := h.db.QueryRowContext(ctx, `SELECT file_path FROM songs WHERE id = ?`, id).Scan(&path)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "song not found", "code": "NOT_FOUND"})
	}
	format := c.QueryParam("format")
	bitrate, _ := strconv.Atoi(c.QueryParam("bitrate"))
	if format == "" {
		return c.File(path)
	}
	contentType, err := h.transcoder.Validate(format, bitrate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "INVALID_TRANSCODE"})
	}
	if _, err := exec.LookPath(h.transcoder.FFmpegPath); err != nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "ffmpeg not available", "code": "FFMPEG_MISSING"})
	}
	args, err := h.transcoder.Args(services.TranscodeRequest{Format: format, Bitrate: bitrate, Path: path})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "INVALID_TRANSCODE"})
	}
	cmd := exec.CommandContext(ctx, h.transcoder.FFmpegPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "transcode failed", "code": "TRANSCODE_FAILED"})
	}
	if err := cmd.Start(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "transcode failed", "code": "TRANSCODE_FAILED"})
	}
	defer cmd.Process.Kill()
	resp := c.Response()
	resp.Header().Set(echo.HeaderContentType, contentType)
	resp.WriteHeader(http.StatusOK)
	if _, err := io.Copy(resp, stdout); err != nil {
		return err
	}
	return cmd.Wait()
}

// Artwork godoc
// @Summary Get artwork for song
// @Tags Player
// @Param id path int true "Song ID"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/artwork/{id} [get]
func (h *Handler) Artwork(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var cover string
	err := h.db.QueryRowContext(ctx, `
		SELECT cover_path FROM albums WHERE id = (SELECT album_id FROM songs WHERE id = ?)
	`, id).Scan(&cover)
	if err != nil || cover == "" {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artwork not found", "code": "NOT_FOUND"})
	}
	if _, err := os.Stat(cover); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artwork not found", "code": "NOT_FOUND"})
	}
	return c.File(cover)
}

// Lyrics godoc
// @Summary Get lyrics for song
// @Tags Player
// @Param id path int true "Song ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /api/lyrics/{id} [get]
func (h *Handler) Lyrics(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var lyrics, synced string
	err := h.db.QueryRowContext(ctx, `SELECT lyrics, lyrics_synced FROM songs WHERE id = ?`, id).Scan(&lyrics, &synced)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "lyrics not found", "code": "NOT_FOUND"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"lyrics": lyrics,
		"synced": synced,
		"source": "embedded",
	})
}
