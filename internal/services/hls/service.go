package hls

import (
	"context"
	"time"
)

type Service struct {
	Cache     *Cache
	Generator *Generator
	Cleaner   *Cleaner
	Config    ServiceConfig
}

type ServiceConfig struct {
	CacheDir        string
	CacheSizeMB     int64
	CacheTTLHours   int
	CacheMinTTL     time.Duration
	SegmentDuration int
	FFmpegPath      string
	CleanupInterval time.Duration
}

func DefaultConfig() ServiceConfig {
	return ServiceConfig{
		CacheDir:        "./cache/hls",
		CacheSizeMB:     5000,
		CacheTTLHours:   24,
		CacheMinTTL:     time.Hour,
		SegmentDuration: 4,
		FFmpegPath:      "ffmpeg",
		CleanupInterval: 5 * time.Minute,
	}
}

func NewService(cfg ServiceConfig) (*Service, error) {
	cache, err := NewCache(CacheConfig{
		Dir:       cfg.CacheDir,
		MaxSizeMB: cfg.CacheSizeMB,
		MinTTL:    cfg.CacheMinTTL,
	})
	if err != nil {
		return nil, err
	}

	generator := NewGenerator(GeneratorConfig{
		FFmpegPath:      cfg.FFmpegPath,
		SegmentDuration: cfg.SegmentDuration,
		Cache:           cache,
	})

	cleaner := NewCleaner(cache, cfg.CleanupInterval)

	return &Service{
		Cache:     cache,
		Generator: generator,
		Cleaner:   cleaner,
		Config:    cfg,
	}, nil
}

func (s *Service) Start(ctx context.Context) {
	go s.Cleaner.Start(ctx)
}

func (s *Service) Stop() {
	s.Cleaner.Stop()
}

func (s *Service) SegmentDuration() int {
	return s.Config.SegmentDuration
}
