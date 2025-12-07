package indexer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"korus/internal/config"
	"korus/internal/database"
	"korus/internal/services"

	"github.com/google/uuid"
)

// JobStatus represents the current state of a scan job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// Job represents an async scan job with progress tracking
type Job struct {
	ID          string     `json:"id"`
	Status      JobStatus  `json:"status"`
	Phase       string     `json:"phase"`
	Progress    int        `json:"progress"`
	Total       int        `json:"total"`
	Force       bool       `json:"force"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Result      *Result    `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type Options struct {
	Force bool
}

type Result struct {
	StartedAt       time.Time
	CompletedAt     time.Time
	Duration        time.Duration
	FilesDiscovered int
	FilesQueued     int
	FilesNew        int
	FilesUpdated    int
	FilesRemoved    int
	Ingested        int
	Errors          []error
}

type Status struct {
	Running   bool
	LastRun   *Result
	LastError string
}

type Service struct {
	db            *database.DB
	libraryCfg    *config.LibraryConfig
	batchMetadata *services.BatchMetadataService

	mu        sync.Mutex
	running   bool
	lastRun   *Result
	lastError error
	jobs      map[string]*Job
}

func NewService(db *database.DB, libraryCfg *config.LibraryConfig, batchMetadata *services.BatchMetadataService) *Service {
	return &Service{
		db:            db,
		libraryCfg:    libraryCfg,
		batchMetadata: batchMetadata,
		jobs:          make(map[string]*Job),
	}
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := Status{
		Running: s.running,
	}

	if s.lastRun != nil {
		copy := *s.lastRun
		status.LastRun = &copy
	}

	if s.lastError != nil {
		status.LastError = s.lastError.Error()
	}

	return status
}

// StartScanAsync starts a scan in the background and returns a job ID for tracking
func (s *Service) StartScanAsync(ctx context.Context, opts Options) (string, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return "", errors.New("indexer already running")
	}
	s.running = true

	jobID := uuid.New().String()
	job := &Job{
		ID:        jobID,
		Status:    JobStatusPending,
		Phase:     "initializing",
		Force:     opts.Force,
		StartedAt: time.Now(),
	}
	s.jobs[jobID] = job
	s.mu.Unlock()

	// Run scan in background goroutine
	go func() {
		// Use background context since the HTTP request context may be cancelled
		bgCtx := context.Background()

		s.mu.Lock()
		job.Status = JobStatusRunning
		job.Phase = "discovering"
		s.mu.Unlock()

		result, err := s.scanWithProgress(bgCtx, opts, job)

		s.mu.Lock()
		now := time.Now()
		job.CompletedAt = &now
		job.Result = result
		if err != nil {
			job.Status = JobStatusFailed
			job.Error = err.Error()
		} else {
			job.Status = JobStatusCompleted
		}
		s.running = false
		s.lastRun = result
		s.lastError = err
		s.mu.Unlock()
	}()

	return jobID, nil
}

// GetJob retrieves the status of a scan job by ID
func (s *Service) GetJob(jobID string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, errors.New("job not found")
	}

	// Return a copy to avoid data races
	copy := *job
	if job.Result != nil {
		resultCopy := *job.Result
		copy.Result = &resultCopy
	}
	return &copy, nil
}

func (s *Service) Scan(ctx context.Context, opts Options) (*Result, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, errors.New("indexer already running")
	}
	s.running = true
	s.mu.Unlock()

	start := time.Now()
	result := &Result{StartedAt: start}
	var runErr error

	defer func() {
		s.mu.Lock()
		s.running = false
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(start)
		s.lastRun = result
		s.lastError = runErr
		s.mu.Unlock()
	}()

	root := s.libraryCfg.MusicDir
	if root == "" {
		runErr = errors.New("music directory not configured")
		return result, runErr
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		runErr = fmt.Errorf("failed to resolve music dir: %w", err)
		return result, runErr
	}

	snapshots, err := walkLibrary(absRoot, s.libraryCfg.FileTypes)
	if err != nil {
		runErr = err
		return result, runErr
	}

	result.FilesDiscovered = len(snapshots)

	existing, err := s.loadExistingSongs(ctx)
	if err != nil {
		runErr = err
		return result, runErr
	}

	historyID, err := s.insertScanHistory(ctx, start)
	if err != nil {
		log.Printf("indexer: failed to record scan start: %v", err)
	}

	defer func() {
		if historyID > 0 {
			if err := s.completeScanHistory(ctx, historyID, result); err != nil {
				log.Printf("indexer: failed to finalize scan history: %v", err)
			}
		}
	}()

	toProcess := make([]string, 0, len(snapshots))
	processedMap := make(map[string]struct{}, len(snapshots))
	for _, snap := range snapshots {
		key := normalizePath(snap.Path)
		processedMap[key] = struct{}{}

		record, ok := existing[key]
		if !ok {
			result.FilesNew++
			toProcess = append(toProcess, snap.Path)
			continue
		}

		if opts.Force || fileChanged(record, snap) {
			result.FilesUpdated++
			toProcess = append(toProcess, snap.Path)
		}
	}

	for path, record := range existing {
		if _, ok := processedMap[path]; !ok {
			if err := s.deleteSong(ctx, record.ID); err != nil {
				log.Printf("indexer: failed to remove missing song %s: %v", path, err)
				result.Errors = append(result.Errors, err)
				continue
			}
			result.FilesRemoved++
		}
	}

	if result.FilesNew == 0 && result.FilesUpdated == 0 && result.FilesRemoved == 0 && !opts.Force {
		return result, nil
	}

	if opts.Force && result.FilesNew == 0 && result.FilesUpdated == 0 {
		toProcess = make([]string, 0, len(snapshots))
		for _, snap := range snapshots {
			toProcess = append(toProcess, snap.Path)
		}
		result.FilesNew = len(toProcess)
	}

	result.FilesQueued = len(toProcess)

	if len(toProcess) > 0 {
		if err := s.ingest(ctx, toProcess, result); err != nil {
			runErr = err
			return result, runErr
		}
	}

	return result, nil
}

// scanWithProgress performs a scan while updating the job's progress for async tracking
func (s *Service) scanWithProgress(ctx context.Context, opts Options, job *Job) (*Result, error) {
	start := time.Now()
	result := &Result{StartedAt: start}

	defer func() {
		result.CompletedAt = time.Now()
		result.Duration = result.CompletedAt.Sub(start)
	}()

	root := s.libraryCfg.MusicDir
	if root == "" {
		return result, errors.New("music directory not configured")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return result, fmt.Errorf("failed to resolve music dir: %w", err)
	}

	snapshots, err := walkLibrary(absRoot, s.libraryCfg.FileTypes)
	if err != nil {
		return result, err
	}

	result.FilesDiscovered = len(snapshots)

	// Update job with discovered files count
	s.mu.Lock()
	job.Total = len(snapshots)
	job.Phase = "analyzing"
	s.mu.Unlock()

	existing, err := s.loadExistingSongs(ctx)
	if err != nil {
		return result, err
	}

	historyID, err := s.insertScanHistory(ctx, start)
	if err != nil {
		log.Printf("indexer: failed to record scan start: %v", err)
	}

	defer func() {
		if historyID > 0 {
			if err := s.completeScanHistory(ctx, historyID, result); err != nil {
				log.Printf("indexer: failed to finalize scan history: %v", err)
			}
		}
	}()

	toProcess := make([]string, 0, len(snapshots))
	processedMap := make(map[string]struct{}, len(snapshots))
	for _, snap := range snapshots {
		key := normalizePath(snap.Path)
		processedMap[key] = struct{}{}

		record, ok := existing[key]
		if !ok {
			result.FilesNew++
			toProcess = append(toProcess, snap.Path)
			continue
		}

		if opts.Force || fileChanged(record, snap) {
			result.FilesUpdated++
			toProcess = append(toProcess, snap.Path)
		}
	}

	for path, record := range existing {
		if _, ok := processedMap[path]; !ok {
			if err := s.deleteSong(ctx, record.ID); err != nil {
				log.Printf("indexer: failed to remove missing song %s: %v", path, err)
				result.Errors = append(result.Errors, err)
				continue
			}
			result.FilesRemoved++
		}
	}

	if result.FilesNew == 0 && result.FilesUpdated == 0 && result.FilesRemoved == 0 && !opts.Force {
		return result, nil
	}

	if opts.Force && result.FilesNew == 0 && result.FilesUpdated == 0 {
		toProcess = make([]string, 0, len(snapshots))
		for _, snap := range snapshots {
			toProcess = append(toProcess, snap.Path)
		}
		result.FilesNew = len(toProcess)
	}

	result.FilesQueued = len(toProcess)

	// Update job for ingestion phase
	s.mu.Lock()
	job.Total = len(toProcess)
	job.Phase = "ingesting"
	job.Progress = 0
	s.mu.Unlock()

	if len(toProcess) > 0 {
		if err := s.ingestWithProgress(ctx, toProcess, result, job); err != nil {
			return result, err
		}
	}

	return result, nil
}

type songRecord struct {
	ID           int
	FilePath     string
	FileSize     int64
	FileModified time.Time
}

type fileSnapshot struct {
	Path     string
	Size     int64
	Modified time.Time
}

func (s *Service) loadExistingSongs(ctx context.Context) (map[string]songRecord, error) {
	query := `SELECT id, file_path, file_size, file_modified FROM songs`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query songs: %w", err)
	}
	defer rows.Close()

	items := make(map[string]songRecord)
	for rows.Next() {
		var record songRecord
		if err := rows.Scan(&record.ID, &record.FilePath, &record.FileSize, &record.FileModified); err != nil {
			return nil, fmt.Errorf("failed to scan song: %w", err)
		}
		items[normalizePath(record.FilePath)] = record
	}

	return items, rows.Err()
}

func (s *Service) deleteSong(ctx context.Context, songID int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM songs WHERE id = $1`, songID)
	return err
}

func (s *Service) ingest(ctx context.Context, files []string, result *Result) error {
	batchSize := s.libraryCfg.IngestBatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	workers := s.libraryCfg.IngestWorkers
	if workers <= 0 {
		workers = 1
	}

	for i := 0; i < len(files); i += batchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		batchResult, err := s.batchMetadata.ProcessBatchWithOptions(ctx, batch, services.BatchOptions{Workers: workers})
		if err != nil {
			return fmt.Errorf("batch ingest failed: %w", err)
		}

		result.Ingested += batchResult.SuccessCount
		if len(batchResult.Errors) > 0 {
			for _, batchErr := range batchResult.Errors {
				result.Errors = append(result.Errors, batchErr)
			}
		}
	}

	return nil
}

// ingestWithProgress processes files while updating job progress
func (s *Service) ingestWithProgress(ctx context.Context, files []string, result *Result, job *Job) error {
	batchSize := s.libraryCfg.IngestBatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	workers := s.libraryCfg.IngestWorkers
	if workers <= 0 {
		workers = 1
	}

	processed := 0
	for i := 0; i < len(files); i += batchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		batchResult, err := s.batchMetadata.ProcessBatchWithOptions(ctx, batch, services.BatchOptions{Workers: workers})
		if err != nil {
			return fmt.Errorf("batch ingest failed: %w", err)
		}

		result.Ingested += batchResult.SuccessCount
		if len(batchResult.Errors) > 0 {
			for _, batchErr := range batchResult.Errors {
				result.Errors = append(result.Errors, batchErr)
			}
		}

		// Update job progress
		processed += len(batch)
		s.mu.Lock()
		job.Progress = processed
		s.mu.Unlock()
	}

	return nil
}

func walkLibrary(root string, extensions []string) ([]fileSnapshot, error) {
	extSet := make(map[string]struct{}, len(extensions))
	for _, ext := range extensions {
		clean := strings.ToLower(strings.TrimPrefix(ext, "."))
		if clean != "" {
			extSet[clean] = struct{}{}
		}
	}

	snapshots := make([]fileSnapshot, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(entry.Name()), "."))
		if _, ok := extSet[ext]; !ok {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return nil
		}

		snapshots = append(snapshots, fileSnapshot{
			Path:     path,
			Size:     info.Size(),
			Modified: info.ModTime().UTC(),
		})
		return nil
	})

	return snapshots, err
}

func fileChanged(record songRecord, snap fileSnapshot) bool {
	if record.FileSize != snap.Size {
		return true
	}

	// Allow slight differences due to filesystem precision
	delta := record.FileModified.Sub(snap.Modified)
	if delta < 0 {
		delta = -delta
	}
	return delta > time.Second
}

func normalizePath(path string) string {
	return filepath.Clean(path)
}

func (s *Service) insertScanHistory(ctx context.Context, started time.Time) (int, error) {
	var id int
	err := s.db.QueryRowContext(ctx, `INSERT INTO scan_history (started_at) VALUES ($1) RETURNING id`, started).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) completeScanHistory(ctx context.Context, id int, result *Result) error {
	if id <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `UPDATE scan_history SET completed_at = $2, songs_added = $3, songs_updated = $4, songs_removed = $5 WHERE id = $1`,
		id, result.CompletedAt, result.FilesNew, result.FilesUpdated, result.FilesRemoved)
	return err
}
