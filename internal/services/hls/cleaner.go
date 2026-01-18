package hls

import (
	"context"
	"log/slog"
	"time"
)

type Cleaner struct {
	cache    *Cache
	interval time.Duration
	stopCh   chan struct{}
}

func NewCleaner(cache *Cache, interval time.Duration) *Cleaner {
	return &Cleaner{
		cache:    cache,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (c *Cleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	slog.Info("HLS cache cleaner started",
		"interval", c.interval,
		"max_size_mb", c.cache.maxSizeMB,
	)

	for {
		select {
		case <-ctx.Done():
			slog.Info("HLS cache cleaner stopping due to context cancellation")
			return
		case <-c.stopCh:
			slog.Info("HLS cache cleaner stopped")
			return
		case <-ticker.C:
			c.runCleanup()
		}
	}
}

func (c *Cleaner) Stop() {
	close(c.stopCh)
}

func (c *Cleaner) runCleanup() {
	currentSizeMB := c.cache.CurrentSizeMB()
	maxSizeMB := float64(c.cache.maxSizeMB)

	slog.Debug("cache cleanup check",
		"current_size_mb", currentSizeMB,
		"max_size_mb", maxSizeMB,
		"entry_count", c.cache.EntryCount(),
	)

	if currentSizeMB > maxSizeMB*0.9 {
		evicted := c.cache.Evict()
		slog.Info("cache cleanup completed",
			"evicted_count", evicted,
			"new_size_mb", c.cache.CurrentSizeMB(),
		)
	}
}

func (c *Cleaner) ForceCleanup() int {
	return c.cache.Evict()
}
