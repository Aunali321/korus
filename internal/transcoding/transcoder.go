package transcoding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// Supported formats and their valid bitrates
var supportedFormats = map[string][]int{
	"mp3":  {128, 192, 256, 320},
	"aac":  {128, 192, 256},
	"opus": {64, 96, 128, 192},
}

// Format to FFmpeg codec mapping
var formatCodecs = map[string]string{
	"mp3":  "libmp3lame",
	"aac":  "aac",
	"opus": "libopus",
}

// Format to content type mapping
var formatContentTypes = map[string]string{
	"mp3":  "audio/mpeg",
	"aac":  "audio/mp4",
	"opus": "audio/ogg",
}

// Transcoder handles audio transcoding via FFmpeg
type Transcoder struct {
	ffmpegPath string
	available  bool
}

// New creates a new Transcoder instance
func New() *Transcoder {
	t := &Transcoder{}
	t.checkFFmpeg()
	return t
}

// checkFFmpeg verifies FFmpeg is available
func (t *Transcoder) checkFFmpeg() {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.available = false
		return
	}
	t.ffmpegPath = path
	t.available = true
}

// IsAvailable returns whether FFmpeg is available
func (t *Transcoder) IsAvailable() bool {
	return t.available
}

// ValidateParams validates format and bitrate parameters
func ValidateParams(format string, bitrateStr string) (int, error) {
	if format == "" {
		return 0, errors.New("format is required")
	}

	validBitrates, ok := supportedFormats[format]
	if !ok {
		return 0, fmt.Errorf("unsupported format: %s (supported: mp3, aac, opus)", format)
	}

	if bitrateStr == "" {
		// Return default bitrate for format
		return validBitrates[len(validBitrates)-1], nil
	}

	bitrate, err := strconv.Atoi(bitrateStr)
	if err != nil {
		return 0, fmt.Errorf("invalid bitrate: %s", bitrateStr)
	}

	// Check if bitrate is valid for this format
	valid := false
	for _, b := range validBitrates {
		if b == bitrate {
			valid = true
			break
		}
	}

	if !valid {
		return 0, fmt.Errorf("invalid bitrate %d for format %s (valid: %v)", bitrate, format, validBitrates)
	}

	return bitrate, nil
}

// GetContentType returns the content type for a format
func GetContentType(format string) string {
	if ct, ok := formatContentTypes[format]; ok {
		return ct
	}
	return "application/octet-stream"
}

// Stream transcodes the input file and writes to the output writer
func (t *Transcoder) Stream(ctx context.Context, inputPath string, format string, bitrate int, w io.Writer) error {
	if !t.available {
		return errors.New("ffmpeg not available")
	}

	codec, ok := formatCodecs[format]
	if !ok {
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Build FFmpeg arguments
	args := []string{
		"-i", inputPath,
		"-vn",         // No video
		"-c:a", codec, // Audio codec
		"-b:a", fmt.Sprintf("%dk", bitrate), // Bitrate
	}

	// Format-specific options
	switch format {
	case "mp3":
		args = append(args, "-f", "mp3")
	case "aac":
		args = append(args, "-f", "adts") // ADTS container for streaming
	case "opus":
		args = append(args, "-f", "opus")
	}

	// Output to stdout
	args = append(args, "pipe:1")

	cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)
	cmd.Stdout = w

	// Capture stderr for debugging
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if context was cancelled (client disconnected)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg error: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}
