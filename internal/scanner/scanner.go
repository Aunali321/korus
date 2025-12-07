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

	"github.com/fsnotify/fsnotify"
	"korus/internal/config"
	"korus/internal/database"
)

type JobQueue interface {
	Enqueue(ctx context.Context, jobType string, payload interface{}) (interface{}, error)
}

type Scanner struct {
	db          *database.DB
	jobQueue    JobQueue
	config      *config.LibraryConfig
	watcher     *fsnotify.Watcher
	mu          sync.RWMutex
	watchedDirs map[string]bool
	debouncer   *Debouncer
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

type Debouncer struct {
	mu       sync.Mutex
	pending  map[string]*time.Timer
	delay    time.Duration
	callback func(string)
}

type ScanResult struct {
	FilesFound   int
	FilesAdded   int
	FilesUpdated int
	FilesRemoved int
	Duration     time.Duration
	Errors       []error
}

func NewScanner(db *database.DB, jobQueue JobQueue, config *config.LibraryConfig) (*Scanner, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	debouncer := &Debouncer{
		pending: make(map[string]*time.Timer),
		delay:   5 * time.Second, // 5 second debounce
	}

	scanner := &Scanner{
		db:          db,
		jobQueue:    jobQueue,
		config:      config,
		watcher:     watcher,
		watchedDirs: make(map[string]bool),
		debouncer:   debouncer,
		stopCh:      make(chan struct{}),
	}

	debouncer.callback = scanner.handleDebouncedChange

	return scanner, nil
}

func (s *Scanner) Start(ctx context.Context) error {
	// Resolve symbolic links for watching
	resolvedPath, err := filepath.EvalSymlinks(s.config.MusicDir)
	if err != nil {
		return fmt.Errorf("failed to resolve music directory path for watching: %w", err)
	}

	// Add initial watch on resolved music directory
	if err := s.addWatchRecursive(resolvedPath); err != nil {
		return fmt.Errorf("failed to watch music directory: %w", err)
	}

	// Start watcher goroutine
	s.wg.Add(1)
	go s.watchLoop(ctx)

	log.Printf("File scanner started, watching: %s", s.config.MusicDir)
	return nil
}

func (s *Scanner) Stop() {
	close(s.stopCh)
	s.wg.Wait()

	if s.watcher != nil {
		s.watcher.Close()
	}

	log.Println("File scanner stopped")
}

func (s *Scanner) ScanLibrary(ctx context.Context, force bool) (*ScanResult, error) {
	start := time.Now()
	result := &ScanResult{}

	log.Printf("Starting %s library scan of %s",
		func() string {
			if force {
				return "full"
			}
			return "incremental"
		}(), s.config.MusicDir)

	// Resolve symbolic links first
	resolvedPath, err := filepath.EvalSymlinks(s.config.MusicDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve music directory path: %w", err)
	}
	log.Printf("Resolved music path: %s -> %s", s.config.MusicDir, resolvedPath)

	// Collect files for batch processing
	filesToProcess := make([]string, 0)

	// Walk through resolved music directory
	err = filepath.Walk(resolvedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("error accessing %s: %w", path, err))
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is a supported audio format
		if !s.isSupportedAudioFile(path) {
			return nil
		}

		result.FilesFound++

		// Log progress every 100 files
		if result.FilesFound%100 == 0 {
			log.Printf("📊 Found %d audio files so far...", result.FilesFound)
		}

		// Check if we should process this file
		shouldProcess, err := s.shouldProcessFile(ctx, path, info, force)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("error checking file %s: %w", path, err))
			return nil
		}

		if shouldProcess {
			filesToProcess = append(filesToProcess, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Create batch jobs (process files in batches of 25)
	batchSize := 25
	for i := 0; i < len(filesToProcess); i += batchSize {
		end := i + batchSize
		if end > len(filesToProcess) {
			end = len(filesToProcess)
		}

		batch := filesToProcess[i:end]
		payload := map[string]interface{}{
			"file_paths": batch,
		}

		_, err := s.jobQueue.Enqueue(ctx, "metadata_extract_batch", payload)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to enqueue batch job: %w", err))
			continue
		}

		result.FilesAdded += len(batch)
	}

	// Clean up removed files if not a force scan
	if !force {
		removed, err := s.cleanupRemovedFiles(ctx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to cleanup removed files: %w", err))
		}
		result.FilesRemoved = removed
	}

	result.Duration = time.Since(start)
	log.Printf("Library scan completed in %v: %d files found, %d to process, %d removed",
		result.Duration, result.FilesFound, result.FilesAdded, result.FilesRemoved)

	return result, nil
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
			s.handleFileEvent(event)
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

func (s *Scanner) handleFileEvent(event fsnotify.Event) {
	// Filter out events we don't care about
	if !s.shouldHandleEvent(event) {
		return
	}

	log.Printf("File event: %s %s", event.Op, event.Name)

	// Handle directory creation - add to watch list
	if event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			s.addWatchRecursive(event.Name)
		}
	}

	// Debounce file changes
	s.debouncer.debounce(event.Name)
}

func (s *Scanner) handleDebouncedChange(path string) {
	ctx := context.Background()

	// Check if file still exists and is supported
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File was deleted
			s.handleFileDeleted(ctx, path)
		}
		return
	}

	if info.IsDir() || !s.isSupportedAudioFile(path) {
		return
	}

	// File was created or modified - enqueue metadata extraction
	payload := map[string]interface{}{
		"file_path": path,
	}

	_, err = s.jobQueue.Enqueue(ctx, "metadata_extract", payload)
	if err != nil {
		log.Printf("Failed to enqueue metadata extraction for %s: %v", path, err)
	}
}

func (s *Scanner) handleFileDeleted(ctx context.Context, path string) {
	// Remove from database
	query := "DELETE FROM songs WHERE file_path = $1"
	result, err := s.db.ExecContext(ctx, query, path)
	if err != nil {
		log.Printf("Failed to delete file from database %s: %v", path, err)
		return
	}

	if result.RowsAffected() > 0 {
		log.Printf("Removed deleted file from database: %s", path)
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
			// Skip hidden directories
			if strings.HasPrefix(filepath.Base(path), ".") {
				return filepath.SkipDir
			}

			// Add to watcher if not already watched
			if !s.watchedDirs[path] {
				if err := s.watcher.Add(path); err != nil {
					log.Printf("Failed to watch directory %s: %v", path, err)
					return nil // Continue walking
				}
				s.watchedDirs[path] = true
				log.Printf("Added watch on directory: %s", path)
			}
		}

		return nil
	})
}

func (s *Scanner) shouldHandleEvent(event fsnotify.Event) bool {
	// Ignore temporary files and hidden files
	basename := filepath.Base(event.Name)
	if strings.HasPrefix(basename, ".") || strings.HasSuffix(basename, "~") {
		return false
	}

	// Only handle create, write, and remove events
	return event.Op&fsnotify.Create == fsnotify.Create ||
		event.Op&fsnotify.Write == fsnotify.Write ||
		event.Op&fsnotify.Remove == fsnotify.Remove
}

func (s *Scanner) isSupportedAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return false
	}

	// Remove the dot prefix
	ext = ext[1:]

	for _, supportedExt := range s.config.FileTypes {
		if ext == supportedExt {
			return true
		}
	}

	return false
}

func (s *Scanner) shouldProcessFile(ctx context.Context, path string, info os.FileInfo, force bool) (bool, error) {
	if force {
		return true, nil
	}

	// Check if file exists in database and if it's been modified
	query := "SELECT file_modified FROM songs WHERE file_path = $1"
	var dbModified time.Time

	err := s.db.QueryRowContext(ctx, query, path).Scan(&dbModified)
	if err != nil {
		// File not in database, should process
		return true, nil
	}

	// Compare modification times
	return info.ModTime().After(dbModified), nil
}

func (s *Scanner) cleanupRemovedFiles(ctx context.Context) (int, error) {
	// Get all file paths from database
	query := "SELECT file_path FROM songs"
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to query songs: %w", err)
	}
	defer rows.Close()

	var removedCount int
	for rows.Next() {
		var filePath string
		if err := rows.Scan(&filePath); err != nil {
			continue
		}

		// Check if file still exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// File no longer exists, remove from database
			deleteQuery := "DELETE FROM songs WHERE file_path = $1"
			if _, err := s.db.ExecContext(ctx, deleteQuery, filePath); err != nil {
				log.Printf("Failed to remove deleted file %s from database: %v", filePath, err)
				continue
			}
			removedCount++
			log.Printf("Removed deleted file from database: %s", filePath)
		}
	}

	return removedCount, rows.Err()
}

func (d *Debouncer) debounce(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel existing timer if any
	if timer, exists := d.pending[key]; exists {
		timer.Stop()
	}

	// Create new timer
	d.pending[key] = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.pending, key)
		d.mu.Unlock()

		d.callback(key)
	})
}
