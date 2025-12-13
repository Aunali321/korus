package services

import (
	"errors"
	"fmt"
)

type TranscodeRequest struct {
	Format  string
	Bitrate int
	Path    string
}

type Transcoder struct {
	FFmpegPath string
}

var (
	allowedFormats = map[string][]int{
		"mp3":  {128, 192, 256, 320},
		"aac":  {128, 192, 256},
		"opus": {64, 96, 128, 192},
	}
	contentTypes = map[string]string{
		"mp3":  "audio/mpeg",
		"aac":  "audio/mp4",
		"opus": "audio/ogg",
	}
)

func NewTranscoder(ffmpegPath string) *Transcoder {
	return &Transcoder{FFmpegPath: ffmpegPath}
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

func (t *Transcoder) Args(req TranscodeRequest) ([]string, error) {
	if _, err := t.Validate(req.Format, req.Bitrate); err != nil {
		return nil, err
	}
	br := req.Bitrate
	if br == 0 {
		br = allowedFormats[req.Format][len(allowedFormats[req.Format])-1]
	}
	switch req.Format {
	case "mp3":
		return []string{"-i", req.Path, "-vn", "-acodec", "libmp3lame", "-b:a", fmt.Sprintf("%dk", br), "-f", "mp3", "-"}, nil
	case "aac":
		return []string{"-i", req.Path, "-vn", "-c:a", "aac", "-b:a", fmt.Sprintf("%dk", br), "-f", "adts", "-"}, nil
	case "opus":
		return []string{"-i", req.Path, "-vn", "-c:a", "libopus", "-b:a", fmt.Sprintf("%dk", br), "-f", "opus", "-"}, nil
	default:
		return nil, errors.New("unsupported format")
	}
}
