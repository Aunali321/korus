package services

import (
	"errors"
	"fmt"
)

type TranscodeRequest struct {
	Format  string
	Bitrate int
	Path    string
	SeekSec float64 // Time offset in seconds for seeking
	// For WAV: needed to calculate Content-Length and seeking
	DurationMs int
	SampleRate int
	BitDepth   int
	Channels   int
}

type FormatOption struct {
	Format   string `json:"format"`
	Bitrates []int  `json:"bitrates"`
	MimeType string `json:"mime_type"`
}

type Transcoder struct {
	FFmpegPath string
}

const wavHeaderSize = 44

// pcmCodec returns the FFmpeg PCM codec for the given bit depth
func pcmCodec(bitDepth int) string {
	switch bitDepth {
	case 24:
		return "pcm_s24le"
	case 32:
		return "pcm_s32le"
	default:
		return "pcm_s16le"
	}
}

// rawPCMFormat returns the FFmpeg raw format for the given bit depth (no header)
func rawPCMFormat(bitDepth int) string {
	switch bitDepth {
	case 24:
		return "s24le"
	case 32:
		return "s32le"
	default:
		return "s16le"
	}
}

var (
	allowedFormats = map[string][]int{
		"mp3":  {128, 192, 256, 320},
		"aac":  {128, 192, 256},
		"opus": {64, 96, 128, 192, 256},
		"wav":  {0}, // 0 means lossless, no bitrate selection
	}
	contentTypes = map[string]string{
		"mp3":  "audio/mpeg",
		"aac":  "audio/mp4",
		"opus": "audio/ogg",
		"wav":  "audio/wav",
	}
)

func NewTranscoder(ffmpegPath string) *Transcoder {
	return &Transcoder{FFmpegPath: ffmpegPath}
}

func (t *Transcoder) Options() []FormatOption {
	return []FormatOption{
		{Format: "wav", Bitrates: allowedFormats["wav"], MimeType: contentTypes["wav"]},
		{Format: "mp3", Bitrates: allowedFormats["mp3"], MimeType: contentTypes["mp3"]},
		{Format: "aac", Bitrates: allowedFormats["aac"], MimeType: contentTypes["aac"]},
		{Format: "opus", Bitrates: allowedFormats["opus"], MimeType: contentTypes["opus"]},
	}
}

func (t *Transcoder) Validate(format string, bitrate int) (string, error) {
	supported, ok := allowedFormats[format]
	if !ok {
		return "", errors.New("invalid format")
	}
	if bitrate == 0 {
		return contentTypes[format], nil
	}
	for _, b := range supported {
		if b == bitrate {
			return contentTypes[format], nil
		}
	}
	return "", errors.New("invalid bitrate")
}

// WAVSize calculates the total WAV file size from audio metadata
// Formula: header (44 bytes) + (sample_rate * channels * bytes_per_sample * duration_seconds)
func WAVSize(durationMs, sampleRate, bitDepth, channels int) int64 {
	if sampleRate == 0 || bitDepth == 0 || channels == 0 {
		return 0
	}
	bytesPerSample := bitDepth / 8
	durationSec := float64(durationMs) / 1000.0
	audioDataSize := int64(float64(sampleRate*channels*bytesPerSample) * durationSec)
	return wavHeaderSize + audioDataSize
}

// WAVBytesPerSecond returns bytes per second for seeking calculations
func WAVBytesPerSecond(sampleRate, bitDepth, channels int) int {
	if sampleRate == 0 || bitDepth == 0 || channels == 0 {
		return 0
	}
	return sampleRate * channels * (bitDepth / 8)
}

// WAVSeekArgs returns FFmpeg args for seeking to a specific byte offset in WAV output
// Returns the args and the byte offset to skip in output (for partial WAV header handling)
func (t *Transcoder) WAVSeekArgs(req TranscodeRequest, byteOffset int64) ([]string, error) {
	if req.Format != "wav" {
		return nil, errors.New("WAVSeekArgs only supports wav format")
	}

	bytesPerSec := WAVBytesPerSecond(req.SampleRate, req.BitDepth, req.Channels)
	if bytesPerSec == 0 {
		return nil, errors.New("invalid audio metadata")
	}

	// Calculate time offset from byte offset
	// Subtract header size for audio data offset
	audioByteOffset := byteOffset - wavHeaderSize
	if audioByteOffset < 0 {
		audioByteOffset = 0
	}
	timeOffsetSec := float64(audioByteOffset) / float64(bytesPerSec)

	return []string{
		"-ss", fmt.Sprintf("%.3f", timeOffsetSec),
		"-i", req.Path,
		"-vn",
		"-c:a", pcmCodec(req.BitDepth),
		"-ar", fmt.Sprintf("%d", req.SampleRate),
		"-ac", fmt.Sprintf("%d", req.Channels),
		"-f", rawPCMFormat(req.BitDepth),
		"-",
	}, nil
}

func (t *Transcoder) Args(req TranscodeRequest) ([]string, error) {
	if _, err := t.Validate(req.Format, req.Bitrate); err != nil {
		return nil, err
	}

	// Build seek args if seeking is requested
	var seekArgs []string
	if req.SeekSec > 0 {
		seekArgs = []string{"-ss", fmt.Sprintf("%.3f", req.SeekSec)}
	}

	switch req.Format {
	case "wav":
		codec := pcmCodec(req.BitDepth)
		args := append(seekArgs, "-i", req.Path, "-vn", "-c:a", codec)
		if req.SampleRate > 0 {
			args = append(args, "-ar", fmt.Sprintf("%d", req.SampleRate),
				"-ac", fmt.Sprintf("%d", req.Channels))
		}
		args = append(args, "-f", "wav", "-")
		return args, nil
	case "mp3":
		br := req.Bitrate
		if br == 0 {
			br = allowedFormats["mp3"][len(allowedFormats["mp3"])-1]
		}
		args := append(seekArgs, "-i", req.Path, "-vn", "-acodec", "libmp3lame", "-b:a", fmt.Sprintf("%dk", br), "-f", "mp3", "-")
		return args, nil
	case "aac":
		br := req.Bitrate
		if br == 0 {
			br = allowedFormats["aac"][len(allowedFormats["aac"])-1]
		}
		args := append(seekArgs, "-i", req.Path, "-vn", "-c:a", "aac", "-b:a", fmt.Sprintf("%dk", br), "-f", "adts", "-")
		return args, nil
	case "opus":
		br := req.Bitrate
		if br == 0 {
			br = allowedFormats["opus"][len(allowedFormats["opus"])-1]
		}
		args := append(seekArgs, "-i", req.Path, "-vn", "-c:a", "libopus", "-b:a", fmt.Sprintf("%dk", br), "-f", "opus", "-")
		return args, nil
	default:
		return nil, errors.New("unsupported format")
	}
}
