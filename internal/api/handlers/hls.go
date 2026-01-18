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

	"github.com/Aunali321/korus/internal/services/hls"
)

type HLSHandler struct {
	db      *sql.DB
	hls     *hls.Service
	formats map[string][]int
}

func NewHLSHandler(db *sql.DB, hlsService *hls.Service) *HLSHandler {
	return &HLSHandler{
		db:  db,
		hls: hlsService,
		formats: map[string][]int{
			"mp3":  {128, 192, 256, 320},
			"aac":  {128, 192, 256},
			"opus": {64, 96, 128, 192, 256},
			"flac": {0},
			"alac": {0},
		},
	}
}

type hlsTrackMeta struct {
	ID         int64
	Path       string
	Title      string
	Artist     string
	DurationMs int
	SampleRate int
	BitDepth   int
	Channels   int
}

func (h *HLSHandler) getTrackMeta(c echo.Context, id int64) (*hlsTrackMeta, error) {
	ctx := c.Request().Context()
	var meta hlsTrackMeta
	var durationMs, sampleRate, bitDepth, channels sql.NullInt64
	var artistName sql.NullString

	err := h.db.QueryRowContext(ctx, `
		SELECT s.id, s.file_path, s.title, s.duration_ms, s.sample_rate, s.bit_depth, s.channels,
		       (SELECT GROUP_CONCAT(a.name, ', ') FROM artists a 
		        JOIN song_artists sa ON sa.artist_id = a.id 
		        WHERE sa.song_id = s.id) as artist_name
		FROM songs s WHERE s.id = ?
	`, id).Scan(&meta.ID, &meta.Path, &meta.Title, &durationMs, &sampleRate, &bitDepth, &channels, &artistName)

	if err != nil {
		return nil, err
	}

	meta.DurationMs = int(durationMs.Int64)
	meta.SampleRate = int(sampleRate.Int64)
	meta.BitDepth = int(bitDepth.Int64)
	meta.Channels = int(channels.Int64)
	meta.Artist = artistName.String
	if meta.Artist == "" {
		meta.Artist = "Unknown Artist"
	}

	return &meta, nil
}

func (h *HLSHandler) validateFormat(format string, bitrate int) error {
	supported, ok := h.formats[format]
	if !ok {
		return fmt.Errorf("unsupported format: %s", format)
	}

	if bitrate == 0 {
		return nil
	}

	for _, b := range supported {
		if b == bitrate || b == 0 {
			return nil
		}
	}

	return fmt.Errorf("unsupported bitrate %d for format %s", bitrate, format)
}

// Manifest godoc
// @Summary Get HLS manifest for a track
// @Description Serves the m3u8 manifest for HLS streaming
// @Tags Streaming
// @Produce application/vnd.apple.mpegurl
// @Param id path int true "Track ID"
// @Param format query string false "Audio format" Enums(aac, mp3, opus, flac, alac) default(aac)
// @Param bitrate query int false "Bitrate in kbps"
// @Param token query string false "Auth token for player"
// @Success 200 {string} string "HLS manifest"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /stream/{id}/manifest.m3u8 [get]
// @Security BearerAuth
func (h *HLSHandler) Manifest(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid track id", "code": "INVALID_ID"})
	}

	format := c.QueryParam("format")
	if format == "" {
		format = "aac"
	}

	bitrate, _ := strconv.Atoi(c.QueryParam("bitrate"))
	token := c.QueryParam("token")

	if err := h.validateFormat(format, bitrate); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "INVALID_FORMAT"})
	}

	meta, err := h.getTrackMeta(c, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "track not found", "code": "NOT_FOUND"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "database error", "code": "DB_ERROR"})
	}

	if _, err := os.Stat(meta.Path); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "audio file not found", "code": "FILE_NOT_FOUND"})
	}

	ctx := c.Request().Context()
	req := hls.SegmentRequest{
		TrackID:    meta.ID,
		SourcePath: meta.Path,
		Format:     format,
		Bitrate:    bitrate,
		DurationMs: meta.DurationMs,
		SampleRate: meta.SampleRate,
		BitDepth:   meta.BitDepth,
		Channels:   meta.Channels,
	}

	manifest, err := h.hls.Generator.GenerateManifest(ctx, req, token)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "manifest generation failed", "code": "MANIFEST_FAILED"})
	}

	c.Response().Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	return c.Blob(http.StatusOK, "application/vnd.apple.mpegurl", manifest)
}

// InitSegment godoc
// @Summary Get fMP4 init segment
// @Description Serves the fMP4 initialization segment for HLS
// @Tags Streaming
// @Produce video/mp4
// @Param id path int true "Track ID"
// @Param format query string false "Audio format" Enums(aac, mp3, opus, flac, alac) default(aac)
// @Param bitrate query int false "Bitrate in kbps"
// @Success 200 {file} binary "Init segment"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /stream/{id}/init.mp4 [get]
// @Security BearerAuth
func (h *HLSHandler) InitSegment(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid track id", "code": "INVALID_ID"})
	}

	format := c.QueryParam("format")
	if format == "" {
		format = "aac"
	}

	bitrate, _ := strconv.Atoi(c.QueryParam("bitrate"))

	if err := h.validateFormat(format, bitrate); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "INVALID_FORMAT"})
	}

	meta, err := h.getTrackMeta(c, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "track not found", "code": "NOT_FOUND"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "database error", "code": "DB_ERROR"})
	}

	ctx := c.Request().Context()
	req := hls.SegmentRequest{
		TrackID:    meta.ID,
		SourcePath: meta.Path,
		Format:     format,
		Bitrate:    bitrate,
		DurationMs: meta.DurationMs,
		SampleRate: meta.SampleRate,
		BitDepth:   meta.BitDepth,
		Channels:   meta.Channels,
	}

	data, err := h.hls.Generator.GenerateInitSegment(ctx, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "init segment generation failed", "code": "SEGMENT_FAILED"})
	}

	c.Response().Header().Set("Content-Type", "video/mp4")
	c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	return c.Blob(http.StatusOK, "video/mp4", data)
}

// Segment godoc
// @Summary Get HLS audio segment
// @Description Serves an audio segment for HLS streaming
// @Tags Streaming
// @Produce video/mp4
// @Param id path int true "Track ID"
// @Param segment path string true "Segment number (e.g., 0.m4s)"
// @Param format query string false "Audio format" Enums(aac, mp3, opus, flac, alac) default(aac)
// @Param bitrate query int false "Bitrate in kbps"
// @Success 200 {file} binary "Audio segment"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /stream/{id}/{segment}.m4s [get]
// @Security BearerAuth
func (h *HLSHandler) Segment(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid track id", "code": "INVALID_ID"})
	}

	segmentParam := c.Param("segment")
	segmentParam = strings.TrimSuffix(segmentParam, ".m4s")
	segmentNum, err := strconv.Atoi(segmentParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid segment number", "code": "INVALID_SEGMENT"})
	}

	format := c.QueryParam("format")
	if format == "" {
		format = "aac"
	}

	bitrate, _ := strconv.Atoi(c.QueryParam("bitrate"))

	if err := h.validateFormat(format, bitrate); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "INVALID_FORMAT"})
	}

	meta, err := h.getTrackMeta(c, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "track not found", "code": "NOT_FOUND"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "database error", "code": "DB_ERROR"})
	}

	segmentCount := hls.CalculateSegmentCount(meta.DurationMs, h.hls.SegmentDuration())
	if segmentNum < 0 || segmentNum >= segmentCount {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "segment not found", "code": "SEGMENT_NOT_FOUND"})
	}

	ctx := c.Request().Context()
	req := hls.SegmentRequest{
		TrackID:    meta.ID,
		SourcePath: meta.Path,
		Format:     format,
		Bitrate:    bitrate,
		SegmentNum: segmentNum,
		DurationMs: meta.DurationMs,
		SampleRate: meta.SampleRate,
		BitDepth:   meta.BitDepth,
		Channels:   meta.Channels,
	}

	data, err := h.hls.Generator.GenerateSegment(ctx, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "segment generation failed", "code": "SEGMENT_FAILED"})
	}

	c.Response().Header().Set("Content-Type", "video/mp4")
	c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	return c.Blob(http.StatusOK, "video/mp4", data)
}

// Download godoc
// @Summary Download track
// @Description Download a track in original or transcoded format
// @Tags Streaming
// @Produce octet-stream
// @Param id path int true "Track ID"
// @Param format query string false "Target format" Enums(mp3, aac, opus, flac, alac)
// @Param bitrate query int false "Bitrate in kbps"
// @Success 200 {file} binary "Audio file"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /download/{id} [get]
// @Security BearerAuth
func (h *HLSHandler) Download(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid track id", "code": "INVALID_ID"})
	}

	format := c.QueryParam("format")
	bitrate, _ := strconv.Atoi(c.QueryParam("bitrate"))

	meta, err := h.getTrackMeta(c, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "track not found", "code": "NOT_FOUND"})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "database error", "code": "DB_ERROR"})
	}

	if _, err := os.Stat(meta.Path); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "audio file not found", "code": "FILE_NOT_FOUND"})
	}

	if format == "" {
		filename := sanitizeFilename(meta.Title) + getExtension(meta.Path)
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		return c.File(meta.Path)
	}

	if err := h.validateFormat(format, bitrate); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": err.Error(), "code": "INVALID_FORMAT"})
	}

	ctx := c.Request().Context()
	data, err := h.hls.Generator.Transcode(ctx, meta.Path, format, bitrate)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "transcode failed", "code": "TRANSCODE_FAILED"})
	}

	ext := "." + format
	if format == "alac" {
		ext = ".m4a"
	}

	filename := sanitizeFilename(meta.Title) + ext
	contentType := getContentType(format)

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Content-Length", strconv.Itoa(len(data)))
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	return c.Blob(http.StatusOK, contentType, data)
}

// StreamingOptions godoc
// @Summary Get available streaming formats
// @Description Returns available formats, bitrates, and streaming capabilities
// @Tags Streaming
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /streaming/options [get]
func (h *HLSHandler) StreamingOptions(c echo.Context) error {
	ffmpegAvailable := true
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		ffmpegAvailable = false
	}

	type formatOption struct {
		Format   string `json:"format"`
		Bitrates []int  `json:"bitrates"`
		MimeType string `json:"mime_type"`
	}

	options := []formatOption{
		{Format: "aac", Bitrates: h.formats["aac"], MimeType: "audio/mp4"},
		{Format: "mp3", Bitrates: h.formats["mp3"], MimeType: "audio/mpeg"},
		{Format: "opus", Bitrates: h.formats["opus"], MimeType: "audio/ogg"},
		{Format: "flac", Bitrates: h.formats["flac"], MimeType: "audio/flac"},
		{Format: "alac", Bitrates: h.formats["alac"], MimeType: "audio/mp4"},
	}

	return c.JSON(http.StatusOK, map[string]any{
		"formats":          options,
		"ffmpeg_available": ffmpegAvailable,
		"original_enabled": true,
		"hls_enabled":      true,
	})
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

func getExtension(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 1 {
		return "." + parts[len(parts)-1]
	}
	return ""
}

func getContentType(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "aac":
		return "audio/mp4"
	case "opus":
		return "audio/ogg"
	case "flac":
		return "audio/flac"
	case "alac":
		return "audio/mp4"
	default:
		return "application/octet-stream"
	}
}

// Stream godoc
// @Summary Stream a track
// @Description Stream audio - serves original file or redirects to HLS manifest
// @Tags Streaming
// @Produce audio/*
// @Param id path int true "Track ID"
// @Param format query string false "Target format" Enums(aac, mp3, opus, flac, alac)
// @Param bitrate query string false "Bitrate in kbps"
// @Param token query string false "Auth token for player"
// @Success 200 {file} binary "Audio stream"
// @Success 307 {string} string "Redirect to HLS manifest"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /stream/{id} [get]
// @Security BearerAuth
func (h *HLSHandler) Stream(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid track id", "code": "INVALID_ID"})
	}

	format := c.QueryParam("format")
	bitrate := c.QueryParam("bitrate")
	token := c.QueryParam("token")

	// If no format specified, serve original file directly for playback
	if format == "" {
		meta, err := h.getTrackMeta(c, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "track not found", "code": "NOT_FOUND"})
			}
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "database error", "code": "DB_ERROR"})
		}

		if _, err := os.Stat(meta.Path); err != nil {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "audio file not found", "code": "FILE_NOT_FOUND"})
		}

		c.Response().Header().Set("Access-Control-Allow-Origin", "*")
		c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return c.File(meta.Path)
	}

	// With format, redirect to HLS manifest
	params := []string{}
	params = append(params, "format="+format)
	if bitrate != "" {
		params = append(params, "bitrate="+bitrate)
	}
	if token != "" {
		params = append(params, "token="+token)
	}

	redirectURL := fmt.Sprintf("/api/stream/%d/manifest.m3u8?%s", id, strings.Join(params, "&"))
	return c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// Artwork godoc
// @Summary Get artwork for a track or album
// @Description Returns cover artwork image
// @Tags Streaming
// @Produce image/*
// @Param id path int true "Track or Album ID"
// @Param type query string false "Type of artwork" Enums(track, album) default(track)
// @Success 200 {file} binary "Artwork image"
// @Failure 404 {object} map[string]string
// @Router /artwork/{id} [get]
func (h *HLSHandler) Artwork(c echo.Context) error {
	ctx := c.Request().Context()
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	artworkType := c.QueryParam("type")

	var cover string
	var filePath string
	var err error

	if artworkType == "album" {
		err = h.db.QueryRowContext(ctx, `SELECT cover_path FROM albums WHERE id = ?`, id).Scan(&cover)
	} else {
		err = h.db.QueryRowContext(ctx, `
			SELECT a.cover_path, s.file_path FROM albums a 
			JOIN songs s ON s.album_id = a.id 
			WHERE s.id = ?
		`, id).Scan(&cover, &filePath)
	}

	if err == nil && cover != "" {
		if info, statErr := os.Stat(cover); statErr == nil {
			c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			c.Response().Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))
			return c.File(cover)
		}
	}

	if filePath != "" {
		tempFile, extractErr := hlsExtractArtwork(filePath)
		if extractErr == nil && tempFile != "" {
			defer os.Remove(tempFile)
			c.Response().Header().Set("Cache-Control", "public, max-age=3600")
			return c.File(tempFile)
		}
	}

	return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artwork not found", "code": "NOT_FOUND"})
}

func hlsExtractArtwork(audioPath string) (string, error) {
	tempFile, err := os.CreateTemp("", "artwork-*.jpg")
	if err != nil {
		return "", err
	}
	tempFile.Close()

	cmd := exec.Command("ffmpeg", "-y", "-i", audioPath, "-an", "-vcodec", "mjpeg", "-frames:v", "1", tempFile.Name())
	if err := cmd.Run(); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	info, err := os.Stat(tempFile.Name())
	if err != nil || info.Size() == 0 {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("no artwork extracted")
	}

	return tempFile.Name(), nil
}

// ArtistImage godoc
// @Summary Get artist image
// @Description Returns artist image
// @Tags Library
// @Produce image/*
// @Param id path int true "Artist ID"
// @Success 200 {file} binary "Artist image"
// @Failure 404 {object} map[string]string
// @Router /artists/{id}/image [get]
func (h *HLSHandler) ArtistImage(c echo.Context) error {
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
// @Summary Get lyrics for a track
// @Description Returns lyrics and synced lyrics if available
// @Tags Library
// @Produce json
// @Param id path int true "Track ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /lyrics/{id} [get]
// @Security BearerAuth
func (h *HLSHandler) Lyrics(c echo.Context) error {
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

var _ io.Reader = nil
