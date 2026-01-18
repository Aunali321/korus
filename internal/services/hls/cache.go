package hls

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type CacheEntry struct {
	Path       string
	Size       int64
	AccessTime time.Time
	CreateTime time.Time
}

type Cache struct {
	mu          sync.RWMutex
	dir         string
	maxSizeMB   int64
	minTTL      time.Duration
	entries     map[string]*CacheEntry
	currentSize int64
}

type CacheConfig struct {
	Dir       string
	MaxSizeMB int64
	MinTTL    time.Duration
}

func NewCache(cfg CacheConfig) (*Cache, error) {
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	c := &Cache{
		dir:       cfg.Dir,
		maxSizeMB: cfg.MaxSizeMB,
		minTTL:    cfg.MinTTL,
		entries:   make(map[string]*CacheEntry),
	}

	if err := c.loadExisting(); err != nil {
		slog.Warn("failed to load existing cache entries", "error", err)
	}

	return c, nil
}

func (c *Cache) loadExisting() error {
	return filepath.WalkDir(c.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		key := filepath.Base(path)
		key = key[:len(key)-len(filepath.Ext(key))]

		c.entries[key] = &CacheEntry{
			Path:       path,
			Size:       info.Size(),
			AccessTime: info.ModTime(),
			CreateTime: info.ModTime(),
		}
		c.currentSize += info.Size()

		return nil
	})
}

func (c *Cache) CacheKey(trackID int64, format string, bitrate int, segmentType string, segmentNum int) string {
	data := fmt.Sprintf("%d:%s:%d:%s:%d", trackID, format, bitrate, segmentType, segmentNum)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

func (c *Cache) ManifestKey(trackID int64, format string, bitrate int) string {
	data := fmt.Sprintf("%d:%s:%d:manifest", trackID, format, bitrate)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

func (c *Cache) InitKey(trackID int64, format string, bitrate int) string {
	data := fmt.Sprintf("%d:%s:%d:init", trackID, format, bitrate)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

func (c *Cache) SegmentKey(trackID int64, format string, bitrate int, segmentNum int) string {
	data := fmt.Sprintf("%d:%s:%d:segment:%d", trackID, format, bitrate, segmentNum)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	data, err := os.ReadFile(entry.Path)
	if err != nil {
		c.mu.Lock()
		delete(c.entries, key)
		c.currentSize -= entry.Size
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	entry.AccessTime = time.Now()
	c.mu.Unlock()

	return data, true
}

func (c *Cache) GetPath(key string) (string, bool) {
	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return "", false
	}

	if _, err := os.Stat(entry.Path); err != nil {
		c.mu.Lock()
		delete(c.entries, key)
		c.currentSize -= entry.Size
		c.mu.Unlock()
		return "", false
	}

	c.mu.Lock()
	entry.AccessTime = time.Now()
	c.mu.Unlock()

	return entry.Path, true
}

func (c *Cache) Put(key string, data []byte, ext string) error {
	path := filepath.Join(c.dir, key+ext)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if old, exists := c.entries[key]; exists {
		c.currentSize -= old.Size
	}

	c.entries[key] = &CacheEntry{
		Path:       path,
		Size:       int64(len(data)),
		AccessTime: time.Now(),
		CreateTime: time.Now(),
	}
	c.currentSize += int64(len(data))

	return nil
}

func (c *Cache) PutFile(key string, srcPath string, ext string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}
	return c.Put(key, data, ext)
}

func (c *Cache) Has(key string) bool {
	c.mu.RLock()
	_, exists := c.entries[key]
	c.mu.RUnlock()
	return exists
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		return
	}

	os.Remove(entry.Path)
	c.currentSize -= entry.Size
	delete(c.entries, key)
}

func (c *Cache) InvalidateTrack(trackID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prefix := fmt.Sprintf("%d:", trackID)
	for key, entry := range c.entries {
		keyData := fmt.Sprintf("%d:", trackID)
		hash := sha256.Sum256([]byte(keyData))
		if key[:8] == hex.EncodeToString(hash[:4]) {
			os.Remove(entry.Path)
			c.currentSize -= entry.Size
			delete(c.entries, key)
		}
	}

	_ = prefix
}

func (c *Cache) CurrentSizeMB() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return float64(c.currentSize) / (1024 * 1024)
}

func (c *Cache) EntryCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func (c *Cache) Evict() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	maxBytes := c.maxSizeMB * 1024 * 1024
	if c.currentSize <= maxBytes {
		return 0
	}

	type entryWithKey struct {
		key   string
		entry *CacheEntry
	}

	var eligible []entryWithKey
	minTime := time.Now().Add(-c.minTTL)

	for key, entry := range c.entries {
		if entry.CreateTime.Before(minTime) {
			eligible = append(eligible, entryWithKey{key, entry})
		}
	}

	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].entry.AccessTime.Before(eligible[j].entry.AccessTime)
	})

	evicted := 0
	for _, e := range eligible {
		if c.currentSize <= maxBytes {
			break
		}

		if err := os.Remove(e.entry.Path); err != nil {
			slog.Warn("failed to remove cache file", "path", e.entry.Path, "error", err)
			continue
		}

		c.currentSize -= e.entry.Size
		delete(c.entries, e.key)
		evicted++
	}

	slog.Info("cache eviction complete",
		"evicted", evicted,
		"current_size_mb", float64(c.currentSize)/(1024*1024),
		"max_size_mb", c.maxSizeMB,
	)

	return evicted
}

func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, entry := range c.entries {
		os.Remove(entry.Path)
		delete(c.entries, key)
	}
	c.currentSize = 0

	return nil
}
