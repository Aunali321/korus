package hls

import (
	"fmt"
	"math"
	"strings"
)

type TrackInfo struct {
	ID         int64
	Title      string
	Artist     string
	DurationMs int
	Format     string
	Bitrate    int
}

type ManifestConfig struct {
	SegmentDuration int
	Token           string
}

func GenerateManifest(track TrackInfo, cfg ManifestConfig) string {
	durationSec := float64(track.DurationMs) / 1000.0
	segmentCount := int(math.Ceil(durationSec / float64(cfg.SegmentDuration)))

	var sb strings.Builder

	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:7\n")
	sb.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", cfg.SegmentDuration))
	sb.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	sb.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")

	params := buildQueryParams(track.Format, track.Bitrate, cfg.Token)
	sb.WriteString(fmt.Sprintf("#EXT-X-MAP:URI=\"init.mp4%s\"\n", params))

	for i := 0; i < segmentCount; i++ {
		var segDuration float64
		if i == segmentCount-1 {
			remaining := durationSec - float64(i*cfg.SegmentDuration)
			segDuration = remaining
		} else {
			segDuration = float64(cfg.SegmentDuration)
		}

		if i == 0 {
			sb.WriteString(fmt.Sprintf("#EXTINF:%.3f,%s - %s\n", segDuration, track.Artist, track.Title))
		} else {
			sb.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", segDuration))
		}

		sb.WriteString(fmt.Sprintf("%d.m4s%s\n", i, params))
	}

	sb.WriteString("#EXT-X-ENDLIST\n")

	return sb.String()
}

func buildQueryParams(format string, bitrate int, token string) string {
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

func CalculateSegmentCount(durationMs int, segmentDuration int) int {
	durationSec := float64(durationMs) / 1000.0
	return int(math.Ceil(durationSec / float64(segmentDuration)))
}

func CalculateSegmentDuration(durationMs int, segmentIndex int, segmentDuration int) float64 {
	durationSec := float64(durationMs) / 1000.0
	segmentCount := CalculateSegmentCount(durationMs, segmentDuration)

	if segmentIndex == segmentCount-1 {
		remaining := durationSec - float64(segmentIndex*segmentDuration)
		return remaining
	}

	return float64(segmentDuration)
}

func CalculateSegmentStart(segmentIndex int, segmentDuration int) float64 {
	return float64(segmentIndex * segmentDuration)
}
