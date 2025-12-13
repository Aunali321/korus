package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration.
type Config struct {
	Addr                string
	DBPath              string
	MediaRoot           string
	JWTSecret           string
	TokenTTL            time.Duration
	RefreshTTL          time.Duration
	FFmpegPath          string
	FFprobePath         string
	ListenBrainzToken   string
	ListenBrainzUser    string
	MusicBrainzAgent    string
	EnableListenBrainz  bool
	EnableMusicBrainz   bool
	RateLimitAuthCount  int
	RateLimitAuthWindow time.Duration
	ScanWatch           bool
	ScanExcludePattern  string
	ScanEmbeddedCover   bool
}

// FromEnv builds Config from environment with sane defaults.
func FromEnv() (Config, error) {
	cfg := Config{
		Addr:                getenv("ADDR", ":8080"),
		DBPath:              getenv("DB_PATH", "./korus.db"),
		MediaRoot:           getenv("MEDIA_ROOT", "./media"),
		JWTSecret:           getenv("JWT_SECRET", ""),
		TokenTTL:            durationEnv("TOKEN_TTL", 24*time.Hour),
		RefreshTTL:          durationEnv("REFRESH_TTL", 30*24*time.Hour),
		FFmpegPath:          getenv("FFMPEG_PATH", "ffmpeg"),
		FFprobePath:         getenv("FFPROBE_PATH", "ffprobe"),
		ListenBrainzToken:   getenv("LISTENBRAINZ_TOKEN", ""),
		ListenBrainzUser:    getenv("LISTENBRAINZ_USER", ""),
		MusicBrainzAgent:    getenv("MUSICBRAINZ_AGENT", "Korus/0.1 (https://github.com/Aunali321/korus)"),
		EnableListenBrainz:  boolEnv("ENABLE_LISTENBRAINZ", false),
		EnableMusicBrainz:   boolEnv("ENABLE_MUSICBRAINZ", false),
		RateLimitAuthCount:  intEnv("RATE_LIMIT_AUTH_COUNT", 10),
		RateLimitAuthWindow: durationEnv("RATE_LIMIT_AUTH_WINDOW", time.Minute),
		ScanWatch:           boolEnv("SCAN_WATCH", false),
		ScanExcludePattern:  getenv("SCAN_EXCLUDE_PATTERN", ""),
		ScanEmbeddedCover:   boolEnv("SCAN_EMBEDDED_COVER", true),
	}
	if cfg.JWTSecret == "" {
		return cfg, errors.New("JWT_SECRET is required")
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func durationEnv(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func boolEnv(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func intEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
