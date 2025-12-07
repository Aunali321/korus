package scanner

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"korus/internal/config"
	"korus/internal/database"
	"korus/internal/indexer"

	"github.com/fsnotify/fsnotify"
)

type Scanner struct {
	db      *database.DB
	indexer *indexer.Service
	config  *config.LibraryConfig
	watcher *fsnotify.Watcher

	mu          sync.RWMutex
	watchedDirs map[string]bool
	stopCh      chan struct{}
	wg          sync.WaitGroup

	batchMu       sync.Mutex
	batchPending  map[string]struct{}
	batchTimer    *time.Timer
	batchInterval time.Duration
	batchNeedFull bool
}

func New(db *database.DB, indexerSvc *indexer.Service, cfg *config.LibraryConfig) (*Scanner, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Scanner{
		db:            db,
		indexer:       indexerSvc,
		config:        cfg,
		watcher:       watcher,
		watchedDirs:   make(map[string]bool),
		stopCh:        make(chan struct{}),
		batchPending:  make(map[string]struct{}),
		batchInterval: 10 * time.Second,
	}, nil
}

func (s *Scanner) Start(ctx context.Context) error {
	resolvedPath, err := filepath.EvalSymlinks(s.config.MusicDir)
	if err != nil {
		return fmt.Errorf("failed to resolve music directory: %w", err)
	}

	if err := s.addWatchRecursive(resolvedPath); err != nil {
		return fmt.Errorf("failed to watch music directory: %w", err)
	}

	s.wg.Add(1)
	go s.watchLoop(ctx)

	log.Printf("👁️  File watcher started, watching: %s", s.config.MusicDir)
	return nil
}

func (s *Scanner) Stop() {
	close(s.stopCh)
	s.wg.Wait()

	s.batchMu.Lock()
	if s.batchTimer != nil {
		s.batchTimer.Stop()
	}
	s.batchMu.Unlock()

	if s.watcher != nil {
		s.watcher.Close()
	}

	log.Println("👁️  File watcher stopped")
}

func (s *Scanner) watchLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			s.handleEvent(event)
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

func (s *Scanner) handleEvent(event fsnotify.Event) {
	basename := filepath.Base(event.Name)
	if strings.HasPrefix(basename, ".") || strings.HasSuffix(basename, "~") {
		return
	}

	if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove) == 0 {
		return
	}

	isRemove := event.Op&fsnotify.Remove == fsnotify.Remove

	if isRemove {
		if !s.isSupportedAudioFile(event.Name) {
			return
		}
	} else {
		info, err := os.Stat(event.Name)
		if err != nil {
			return
		}
		if info.IsDir() {
			if event.Op&fsnotify.Create == fsnotify.Create {
				s.addWatchRecursive(event.Name)
			}
			return
		}
		if !s.isSupportedAudioFile(event.Name) {
			return
		}
	}

	log.Printf("📁 File event: %s %s", event.Op, event.Name)

	affectedDir := filepath.Dir(event.Name)

	s.batchMu.Lock()
	defer s.batchMu.Unlock()

	s.batchPending[affectedDir] = struct{}{}

	if isRemove {
		s.batchNeedFull = true
	}

	if s.batchTimer != nil {
		s.batchTimer.Stop()
	}
	s.batchTimer = time.AfterFunc(s.batchInterval, s.processBatch)
}

func (s *Scanner) processBatch() {
	s.batchMu.Lock()
	needFull := s.batchNeedFull
	pendingDirs := make([]string, 0, len(s.batchPending))
	for dir := range s.batchPending {
		pendingDirs = append(pendingDirs, dir)
	}
	s.batchPending = make(map[string]struct{})
	s.batchNeedFull = false
	s.batchMu.Unlock()

	if len(pendingDirs) == 0 && !needFull {
		return
	}

	log.Printf("🔄 Processing batched changes: %d directories, full=%v", len(pendingDirs), needFull)

	if needFull {
		log.Println("🔄 Starting incremental scan (delete detected)")
	} else {
		log.Println("🔄 Starting incremental scan (files changed)")
	}

	jobID, err := s.indexer.StartScanAsync(context.Background(), indexer.Options{Force: false})
	if err != nil {
		log.Printf("⏭️  Skipping scan: %v", err)
	} else {
		log.Printf("✅ Scan started with job ID: %s", jobID)
	}
}

func (s *Scanner) addWatchRecursive(root string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(filepath.Base(path), ".") {
				return filepath.SkipDir
			}
			if !s.watchedDirs[path] {
				if err := s.watcher.Add(path); err != nil {
					log.Printf("Failed to watch directory %s: %v", path, err)
					return nil
				}
				s.watchedDirs[path] = true
			}
		}

		return nil
	})
}

func (s *Scanner) isSupportedAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return false
	}
	ext = ext[1:]

	for _, supported := range s.config.FileTypes {
		if ext == supported {
			return true
		}
	}
	return false
}
