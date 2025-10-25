package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Database    DatabaseConfig
	Server      ServerConfig
	Auth        AuthConfig
	Library     LibraryConfig
	Cache       CacheConfig
	Recommender RecommenderConfig
}

type DatabaseConfig struct {
	URL         string
	MaxConns    int
	MinConns    int
	MaxConnTime time.Duration
	MaxIdleTime time.Duration
	HealthCheck time.Duration
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Environment  string
}

type AuthConfig struct {
	JWTSecret          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	AdminUsername      string
	AdminPassword      string
}

type LibraryConfig struct {
	MusicDir    string
	CacheDir    string
	ScanWorkers int
	FileTypes   []string
}

type CacheConfig struct {
	ArtworkMaxSize   int64
	MetadataMaxItems int
	MetadataTTL      time.Duration
}

type RecommenderConfig struct {
	Enabled              bool
	PythonRunner         string
	ProjectDir           string
	EncoderConfigPath    string
	BatchSize            int
	CleanupBatchSize     int
	IndexRefreshInterval time.Duration
	Segments             int
	ProjectDim           int
	SimilarityLimit      int
	ColdStartPolicy      string
}

func Load() (*Config, error) {
	cfg := &Config{
		Database: DatabaseConfig{
			URL:         getEnv("DATABASE_URL", "postgres://korus:korus@localhost/korus?sslmode=disable"),
			MaxConns:    getEnvInt("DB_MAX_CONNS", 20),
			MinConns:    getEnvInt("DB_MIN_CONNS", 5),
			MaxConnTime: getEnvDuration("DB_MAX_CONN_TIME", 1*time.Hour),
			MaxIdleTime: getEnvDuration("DB_MAX_IDLE_TIME", 30*time.Minute),
			HealthCheck: getEnvDuration("DB_HEALTH_CHECK", 1*time.Minute),
		},
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvInt("SERVER_PORT", 3000),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
			Environment:  getEnv("ENVIRONMENT", "development"),
		},
		Auth: AuthConfig{
			JWTSecret:          getEnv("JWT_SECRET", ""),
			AccessTokenExpiry:  getEnvDuration("ACCESS_TOKEN_EXPIRY", 15*time.Minute),
			RefreshTokenExpiry: getEnvDuration("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour),
			AdminUsername:      getEnv("ADMIN_USERNAME", "admin"),
			AdminPassword:      getEnv("ADMIN_PASSWORD", ""),
		},
		Library: LibraryConfig{
			MusicDir:    getEnv("MUSIC_DIR", "./music"),
			CacheDir:    getEnv("CACHE_DIR", "./cache"),
			ScanWorkers: getEnvInt("SCAN_WORKERS", 4),
			FileTypes:   []string{"mp3", "flac", "m4a", "ogg", "wav", "aac", "opus"},
		},
		Cache: CacheConfig{
			ArtworkMaxSize:   getEnvInt64("ARTWORK_MAX_SIZE", 100*1024*1024), // 100MB
			MetadataMaxItems: getEnvInt("METADATA_MAX_ITEMS", 10000),
			MetadataTTL:      getEnvDuration("METADATA_TTL", 24*time.Hour),
		},
		Recommender: RecommenderConfig{},
	}

	recommenderProjectDir := getEnv("RECOMMENDER_PROJECT_DIR", "./context/LongCat-Audio-Codec")
	recommenderEncoderConfig := getEnv("RECOMMENDER_ENCODER_CONFIG", "configs/LongCatAudioCodec_encoder.yaml")

	cfg.Recommender = RecommenderConfig{
		Enabled:              getEnvBool("RECOMMENDER_ENABLED", true),
		PythonRunner:         getEnv("RECOMMENDER_PYTHON_RUNNER", "uv run"),
		ProjectDir:           recommenderProjectDir,
		EncoderConfigPath:    recommenderEncoderConfig,
		BatchSize:            getEnvInt("RECOMMENDER_BATCH_SIZE", 4),
		CleanupBatchSize:     getEnvInt("RECOMMENDER_CLEANUP_BATCH_SIZE", 20),
		IndexRefreshInterval: getEnvDuration("RECOMMENDER_INDEX_REFRESH_INTERVAL", 5*time.Minute),
		Segments:             getEnvInt("RECOMMENDER_SEGMENTS", 6),
		ProjectDim:           getEnvInt("RECOMMENDER_PROJECT_DIM", 256),
		SimilarityLimit:      getEnvInt("RECOMMENDER_SIMILARITY_LIMIT", 50),
		ColdStartPolicy:      getEnv("RECOMMENDER_COLD_START_POLICY", "random_diverse"),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.Library.MusicDir == "" {
		return fmt.Errorf("MUSIC_DIR is required")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("SERVER_PORT must be between 1 and 65535")
	}

	if c.Recommender.Enabled {
		if c.Recommender.PythonRunner == "" {
			return fmt.Errorf("RECOMMENDER_PYTHON_RUNNER is required when recommender is enabled")
		}

		if c.Recommender.ProjectDir == "" {
			return fmt.Errorf("RECOMMENDER_PROJECT_DIR is required when recommender is enabled")
		}

		if c.Recommender.EncoderConfigPath == "" {
			return fmt.Errorf("RECOMMENDER_ENCODER_CONFIG is required when recommender is enabled")
		}

		if c.Recommender.Segments < 1 {
			return fmt.Errorf("RECOMMENDER_SEGMENTS must be at least 1")
		}

		if c.Recommender.ProjectDim < 0 {
			return fmt.Errorf("RECOMMENDER_PROJECT_DIM cannot be negative")
		}

		if c.Recommender.BatchSize <= 0 {
			return fmt.Errorf("RECOMMENDER_BATCH_SIZE must be positive")
		}

		if c.Recommender.CleanupBatchSize <= 0 {
			return fmt.Errorf("RECOMMENDER_CLEANUP_BATCH_SIZE must be positive")
		}

		if c.Recommender.SimilarityLimit <= 0 {
			return fmt.Errorf("RECOMMENDER_SIMILARITY_LIMIT must be positive")
		}

		switch c.Recommender.ColdStartPolicy {
		case "random_diverse", "empty":
		default:
			return fmt.Errorf("RECOMMENDER_COLD_START_POLICY must be 'random_diverse' or 'empty'")
		}
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch strings.ToLower(value) {
		case "true", "1", "yes", "y", "on":
			return true
		case "false", "0", "no", "n", "off":
			return false
		}
	}
	return defaultValue
}
