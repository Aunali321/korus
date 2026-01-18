package services

import (
	"errors"
	"fmt"
)

type TranscodeRequest struct {
	Format  string
	Bitrate int
	Path    string
	SeekSec float64
}

type FormatOption struct {
	Format   string `json:"format"`
	Bitrates []int  `json:"bitrates"`
	MimeType string `json:"mime_type"`
}

type Transcoder struct {
	FFmpegPath string
}

var (
	allowedFormats = map[string][]int{
		"mp3":  {128, 192, 256, 320},
		"aac":  {128, 192, 256},
		"opus": {64, 96, 128, 192, 256},
		"flac": {0},
		"alac": {0},
	}
	contentTypes = map[string]string{
		"mp3":  "audio/mpeg",
		"aac":  "audio/mp4",
		"opus": "audio/ogg",
		"flac": "audio/flac",
		"alac": "audio/mp4",
	}
)

func NewTranscoder(ffmpegPath string) *Transcoder {
	return &Transcoder{FFmpegPath: ffmpegPath}
}

func (t *Transcoder) Options() []FormatOption {
	return []FormatOption{
		{Format: "aac", Bitrates: allowedFormats["aac"], MimeType: contentTypes["aac"]},
		{Format: "mp3", Bitrates: allowedFormats["mp3"], MimeType: contentTypes["mp3"]},
		{Format: "opus", Bitrates: allowedFormats["opus"], MimeType: contentTypes["opus"]},
		{Format: "flac", Bitrates: allowedFormats["flac"], MimeType: contentTypes["flac"]},
		{Format: "alac", Bitrates: allowedFormats["alac"], MimeType: contentTypes["alac"]},
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
		if b == bitrate || b == 0 {
			return contentTypes[format], nil
		}
	}
	return "", errors.New("invalid bitrate")
}

func (t *Transcoder) Args(req TranscodeRequest) ([]string, error) {
	if _, err := t.Validate(req.Format, req.Bitrate); err != nil {
		return nil, err
	}

	var seekArgs []string
	if req.SeekSec > 0 {
		seekArgs = []string{"-ss", fmt.Sprintf("%.3f", req.SeekSec)}
	}

	switch req.Format {
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
	case "flac":
		args := append(seekArgs, "-i", req.Path, "-vn", "-c:a", "flac", "-f", "flac", "-")
		return args, nil
	case "alac":
		args := append(seekArgs, "-i", req.Path, "-vn", "-c:a", "alac", "-f", "ipod", "-")
		return args, nil
	default:
		return nil, errors.New("unsupported format")
	}
}
