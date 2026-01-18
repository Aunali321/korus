package hls

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Generator struct {
	ffmpegPath      string
	segmentDuration int
	cache           *Cache
	generating      map[string]*sync.Mutex
	genMu           sync.Mutex
}

type GeneratorConfig struct {
	FFmpegPath      string
	SegmentDuration int
	Cache           *Cache
}

func NewGenerator(cfg GeneratorConfig) *Generator {
	return &Generator{
		ffmpegPath:      cfg.FFmpegPath,
		segmentDuration: cfg.SegmentDuration,
		cache:           cfg.Cache,
		generating:      make(map[string]*sync.Mutex),
	}
}

type SegmentRequest struct {
	TrackID    int64
	SourcePath string
	Format     string
	Bitrate    int
	SegmentNum int
	DurationMs int
	SampleRate int
	BitDepth   int
	Channels   int
}

func (g *Generator) getGenerationLock(key string) *sync.Mutex {
	g.genMu.Lock()
	defer g.genMu.Unlock()

	if mu, ok := g.generating[key]; ok {
		return mu
	}

	mu := &sync.Mutex{}
	g.generating[key] = mu
	return mu
}

func (g *Generator) trackKey(req SegmentRequest) string {
	return fmt.Sprintf("%d:%s:%d", req.TrackID, req.Format, req.Bitrate)
}

// GenerateAllSegments generates all segments for a track at once using ffmpeg's HLS muxer
func (g *Generator) GenerateAllSegments(ctx context.Context, req SegmentRequest) error {
	trackKey := g.trackKey(req)
	cacheKey := g.cache.InitKey(req.TrackID, req.Format, req.Bitrate)

	// Check if already generated
	if g.cache.Has(cacheKey) {
		return nil
	}

	// Lock to prevent concurrent generation of same track
	mu := g.getGenerationLock(trackKey)
	mu.Lock()
	defer mu.Unlock()

	// Double-check after acquiring lock
	if g.cache.Has(cacheKey) {
		return nil
	}

	start := time.Now()

	// Create temp directory for HLS output
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("hls-%d-*", req.TrackID))
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	playlistPath := filepath.Join(tmpDir, "playlist.m3u8")
	segmentPattern := filepath.Join(tmpDir, "segment%d.m4s")

	args := []string{
		"-i", req.SourcePath,
		"-vn",
	}

	args = append(args, g.codecArgs(req)...)

	args = append(args,
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", g.segmentDuration),
		"-hls_playlist_type", "vod",
		"-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", "init.mp4",
		"-hls_segment_filename", segmentPattern,
		playlistPath,
	)

	cmd := exec.CommandContext(ctx, g.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		slog.Error("HLS generation failed",
			"track_id", req.TrackID,
			"format", req.Format,
			"error", err,
			"stderr", stderr.String(),
		)
		return fmt.Errorf("HLS generation failed: %w", err)
	}

	// Read the generated playlist to get exact segment info
	playlistData, err := os.ReadFile(playlistPath)
	if err != nil {
		return fmt.Errorf("read playlist: %w", err)
	}

	// Cache the ffmpeg-generated manifest (we'll transform URLs later)
	manifestKey := g.cache.ManifestKey(req.TrackID, req.Format, req.Bitrate)
	if err := g.cache.Put(manifestKey, playlistData, ".m3u8"); err != nil {
		slog.Warn("failed to cache manifest", "error", err)
	}

	// Read and cache init segment
	initPath := filepath.Join(tmpDir, "init.mp4")
	initData, err := os.ReadFile(initPath)
	if err != nil {
		return fmt.Errorf("read init segment: %w", err)
	}
	if err := g.cache.Put(cacheKey, initData, ".mp4"); err != nil {
		slog.Warn("failed to cache init segment", "error", err)
	}

	// Read and cache all segments
	files, err := filepath.Glob(filepath.Join(tmpDir, "segment*.m4s"))
	if err != nil {
		return fmt.Errorf("glob segments: %w", err)
	}

	for _, segPath := range files {
		// Extract segment number from filename
		base := filepath.Base(segPath)
		numStr := strings.TrimPrefix(base, "segment")
		numStr = strings.TrimSuffix(numStr, ".m4s")
		segNum, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}

		segData, err := os.ReadFile(segPath)
		if err != nil {
			slog.Warn("failed to read segment", "segment", segNum, "error", err)
			continue
		}
		segKey := g.cache.SegmentKey(req.TrackID, req.Format, req.Bitrate, segNum)
		if err := g.cache.Put(segKey, segData, ".m4s"); err != nil {
			slog.Warn("failed to cache segment", "segment", segNum, "error", err)
		}
	}

	slog.Info("HLS segments generated",
		"track_id", req.TrackID,
		"format", req.Format,
		"segment_count", len(files),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return nil
}

func (g *Generator) GenerateInitSegment(ctx context.Context, req SegmentRequest) ([]byte, error) {
	cacheKey := g.cache.InitKey(req.TrackID, req.Format, req.Bitrate)

	if data, ok := g.cache.Get(cacheKey); ok {
		slog.Debug("init segment cache hit", "track_id", req.TrackID, "format", req.Format)
		return data, nil
	}

	// Generate all segments (will cache them)
	if err := g.GenerateAllSegments(ctx, req); err != nil {
		return nil, err
	}

	// Now get from cache
	if data, ok := g.cache.Get(cacheKey); ok {
		return data, nil
	}

	return nil, errors.New("init segment not found after generation")
}

func (g *Generator) GenerateSegment(ctx context.Context, req SegmentRequest) ([]byte, error) {
	cacheKey := g.cache.SegmentKey(req.TrackID, req.Format, req.Bitrate, req.SegmentNum)

	if data, ok := g.cache.Get(cacheKey); ok {
		slog.Debug("segment cache hit", "track_id", req.TrackID, "segment", req.SegmentNum)
		return data, nil
	}

	// Generate all segments (will cache them)
	if err := g.GenerateAllSegments(ctx, req); err != nil {
		return nil, err
	}

	// Now get from cache
	if data, ok := g.cache.Get(cacheKey); ok {
		return data, nil
	}

	return nil, fmt.Errorf("segment %d not found after generation", req.SegmentNum)
}

func (g *Generator) codecArgs(req SegmentRequest) []string {
	switch req.Format {
	case "mp3":
		bitrate := req.Bitrate
		if bitrate == 0 {
			bitrate = 320
		}
		// MP3 in fMP4 isn't well supported, use AAC instead
		return []string{"-c:a", "aac", "-b:a", fmt.Sprintf("%dk", bitrate)}

	case "aac":
		bitrate := req.Bitrate
		if bitrate == 0 {
			bitrate = 256
		}
		return []string{"-c:a", "aac", "-b:a", fmt.Sprintf("%dk", bitrate)}

	case "opus":
		bitrate := req.Bitrate
		if bitrate == 0 {
			bitrate = 256
		}
		return []string{"-c:a", "libopus", "-b:a", fmt.Sprintf("%dk", bitrate)}

	case "flac":
		return []string{"-strict", "-2", "-c:a", "flac"}

	case "alac":
		return []string{"-c:a", "alac"}

	default:
		return []string{"-c:a", "aac", "-b:a", "256k"}
	}
}

func (g *Generator) GenerateManifest(ctx context.Context, req SegmentRequest, token string) ([]byte, error) {
	manifestKey := g.cache.ManifestKey(req.TrackID, req.Format, req.Bitrate)

	// Check if we have cached manifest
	if data, ok := g.cache.Get(manifestKey); ok {
		// Transform the cached manifest to use our URL structure
		return g.transformManifest(data, req, token), nil
	}

	// Generate segments first (this will also cache the manifest)
	if err := g.GenerateAllSegments(ctx, req); err != nil {
		return nil, err
	}

	// Now get from cache
	if data, ok := g.cache.Get(manifestKey); ok {
		return g.transformManifest(data, req, token), nil
	}

	return nil, errors.New("manifest not found after generation")
}

// transformManifest rewrites the ffmpeg-generated manifest to use our URL structure
func (g *Generator) transformManifest(data []byte, req SegmentRequest, token string) []byte {
	var sb strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(data))

	params := buildTransformParams(req.Format, req.Bitrate, token)
	segmentRegex := regexp.MustCompile(`^segment(\d+)\.m4s$`)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#EXT-X-MAP:URI=") {
			// Rewrite init segment URL
			sb.WriteString(fmt.Sprintf("#EXT-X-MAP:URI=\"init.mp4%s\"\n", params))
		} else if segmentRegex.MatchString(line) {
			// Rewrite segment URL
			matches := segmentRegex.FindStringSubmatch(line)
			if len(matches) == 2 {
				sb.WriteString(fmt.Sprintf("%s.m4s%s\n", matches[1], params))
			}
		} else {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return []byte(sb.String())
}

func buildTransformParams(format string, bitrate int, token string) string {
	var params []string

	if format != "" {
		params = append(params, fmt.Sprintf("format=%s", format))
	}
	if bitrate > 0 {
		params = append(params, fmt.Sprintf("bitrate=%d", bitrate))
	}
	if token != "" {
		params = append(params, fmt.Sprintf("token=%s", token))
	}

	if len(params) == 0 {
		return ""
	}

	return "?" + strings.Join(params, "&")
}

func (g *Generator) Transcode(ctx context.Context, sourcePath string, format string, bitrate int) ([]byte, error) {
	args := []string{
		"-i", sourcePath,
		"-vn",
	}

	args = append(args, g.codecArgs(SegmentRequest{Format: format, Bitrate: bitrate})...)

	switch format {
	case "mp3":
		args = append(args, "-f", "mp3")
	case "aac":
		args = append(args, "-f", "adts")
	case "opus":
		args = append(args, "-f", "opus")
	case "flac":
		args = append(args, "-f", "flac")
	case "alac":
		args = append(args, "-f", "ipod")
	default:
		args = append(args, "-f", "mp3")
	}

	args = append(args, "-")

	cmd := exec.CommandContext(ctx, g.ffmpegPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("transcode failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func (g *Generator) TranscodeToFile(ctx context.Context, sourcePath string, format string, bitrate int, outputPath string) error {
	args := []string{
		"-i", sourcePath,
		"-vn",
	}

	args = append(args, g.codecArgs(SegmentRequest{Format: format, Bitrate: bitrate})...)
	args = append(args, "-y", outputPath)

	cmd := exec.CommandContext(ctx, g.ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("transcode failed: %w, stderr: %s", err, stderr.String())
	}

	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("output file not created")
	}

	return nil
}
