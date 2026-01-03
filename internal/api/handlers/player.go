package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/services"
)

// StreamingOptions godoc
// @Summary Get available streaming formats and bitrates
// @Tags Player
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /streaming/options [get]
func (h *Handler) StreamingOptions(c echo.Context) error {
	ffmpegAvailable := true
	if _, err := exec.LookPath(h.transcoder.FFmpegPath); err != nil {
		ffmpegAvailable = false
	}
	return c.JSON(http.StatusOK, map[string]any{
		"formats":          h.transcoder.Options(),
		"ffmpeg_available": ffmpegAvailable,
		"original_enabled": true,
	})
}

type songMetadata struct {
	Path       string
	DurationMs int
	SampleRate int
	BitDepth   int
	Channels   int
}

func (h *Handler) getSongMetadata(c echo.Context, id int64) (*songMetadata, error) {
	ctx := c.Request().Context()
	var meta songMetadata
	var durationMs, sampleRate, bitDepth, channels sql.NullInt64

	err := h.db.QueryRowContext(ctx, `
		SELECT file_path, duration_ms, sample_rate, bit_depth, channels
		FROM songs WHERE id = ?
	`, id).Scan(&meta.Path, &durationMs, &sampleRate, &bitDepth, &channels)

	if err != nil {
		return nil, err
	}

	meta.DurationMs = int(durationMs.Int64)
	meta.SampleRate = int(sampleRate.Int64)
	meta.BitDepth = int(bitDepth.Int64)
	meta.Channels = int(channels.Int64)

	return &meta, nil
}

// parseRangeHeader parses Range header and returns start byte offset
// Returns -1 if no range or invalid range
func parseRangeHeader(rangeHeader string, totalSize int64) (start int64, end int64) {
	if rangeHeader == "" {
		return 0, totalSize - 1
	}
	// Format: bytes=start-end or bytes=start-
	rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeHeader, "-")
	if len(parts) != 2 {
		return 0, totalSize - 1
	}

	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, totalSize - 1
	}

	if parts[1] == "" {
		end = totalSize - 1
	} else {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			end = totalSize - 1
		}
	}

	if start > end || start >= totalSize {
		return 0, totalSize - 1
	}

	return start, end
}

// Stream godoc
// @Summary Stream or transcode song
// @Tags Player
// @Produce octet-stream
// @Param id path int true "Song ID"
// @Param format query string false "mp3|aac|opus|wav"
// @Param bitrate query int false "bitrate kbps"
// @Param seek query number false "seek position in seconds"
// @Success 200 {file} binary
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /stream/{id} [get]
func (h *Handler) Stream(c echo.Context) error {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	meta, err := h.getSongMetadata(c, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "song not found", "code": "NOT_FOUND"})
	}

	format := c.QueryParam("format")
	bitrate, _ := strconv.Atoi(c.QueryParam("bitrate"))
	seekSec, _ := strconv.ParseFloat(c.QueryParam("seek"), 64)

	// No format = serve original file
	if format == "" {
		return c.File(meta.Path)
	}

	contentType, err := h.transcoder.Validate(format, bitrate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "INVALID_TRANSCODE"})
	}

	if _, err := exec.LookPath(h.transcoder.FFmpegPath); err != nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "ffmpeg not available", "code": "FFMPEG_MISSING"})
	}

	// WAV format with Range request support
	if format == "wav" {
		return h.streamWAV(c, meta, contentType)
	}

	// Other formats: transcode with optional seek
	return h.streamTranscode(c, meta.Path, format, bitrate, contentType, seekSec)
}

func (h *Handler) streamWAV(c echo.Context, meta *songMetadata, contentType string) error {
	ctx := c.Request().Context()

	// Calculate total WAV size
	totalSize := services.WAVSize(meta.DurationMs, meta.SampleRate, meta.BitDepth, meta.Channels)
	if totalSize == 0 {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": "missing audio metadata for WAV streaming",
			"code":  "METADATA_MISSING",
		})
	}

	rangeHeader := c.Request().Header.Get("Range")
	start, end := parseRangeHeader(rangeHeader, totalSize)

	req := services.TranscodeRequest{
		Format:     "wav",
		Path:       meta.Path,
		DurationMs: meta.DurationMs,
		SampleRate: meta.SampleRate,
		BitDepth:   meta.BitDepth,
		Channels:   meta.Channels,
	}

	resp := c.Response()
	resp.Header().Set(echo.HeaderContentType, contentType)
	resp.Header().Set("Accept-Ranges", "bytes")

	var args []string
	var err error

	if start > 0 {
		// Seeking requested
		args, err = h.transcoder.WAVSeekArgs(req, start)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "TRANSCODE_FAILED"})
		}

		contentLength := end - start + 1
		resp.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
		resp.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, totalSize))
		resp.WriteHeader(http.StatusPartialContent)
	} else {
		// Full file from start
		args, err = h.transcoder.Args(req)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "TRANSCODE_FAILED"})
		}

		resp.Header().Set("Content-Length", strconv.FormatInt(totalSize, 10))
		resp.WriteHeader(http.StatusOK)
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

	_, _ = io.Copy(resp, stdout)
	return cmd.Wait()
}

func (h *Handler) streamTranscode(c echo.Context, path, format string, bitrate int, contentType string, seekSec float64) error {
	ctx := c.Request().Context()

	args, err := h.transcoder.Args(services.TranscodeRequest{Format: format, Bitrate: bitrate, Path: path, SeekSec: seekSec})
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

	_, _ = io.Copy(resp, stdout)
	return cmd.Wait()
}

// Artwork godoc
// @Summary Get artwork for song or album
// @Tags Player
// @Produce png
// @Produce jpeg
// @Param id path int true "Song or Album ID"
// @Param type query string false "album to get album artwork directly, otherwise song lookup"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /artwork/{id} [get]
func (h *Handler) Artwork(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	artworkType := c.QueryParam("type") // "album" or default to song

	var cover string
	var err error

	if artworkType == "album" {
		// Direct album lookup
		err = h.db.QueryRowContext(ctx, `SELECT cover_path FROM albums WHERE id = ?`, id).Scan(&cover)
	} else {
		// Song -> album lookup (default)
		err = h.db.QueryRowContext(ctx, `
			SELECT cover_path FROM albums WHERE id = (SELECT album_id FROM songs WHERE id = ?)
		`, id).Scan(&cover)
	}

	if err != nil || cover == "" {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artwork not found", "code": "NOT_FOUND"})
	}
	info, err := os.Stat(cover)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artwork not found", "code": "NOT_FOUND"})
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Response().Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))

	return c.File(cover)
}

// ArtistImage godoc
// @Summary Get artist image
// @Tags Player
// @Param id path int true "Artist ID"
// @Produce image/jpeg
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /artist-image/{id} [get]
func (h *Handler) ArtistImage(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var imagePath string
	err := h.db.QueryRowContext(ctx, `SELECT image_path FROM artists WHERE id = ?`, id).Scan(&imagePath)
	if err != nil || imagePath == "" {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artist image not found", "code": "NOT_FOUND"})
	}

	info, err := os.Stat(imagePath)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artist image not found", "code": "NOT_FOUND"})
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Response().Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))

	return c.File(imagePath)
}

// Lyrics godoc
// @Summary Get lyrics for song
// @Tags Player
// @Param id path int true "Song ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /lyrics/{id} [get]
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
