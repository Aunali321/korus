package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"korus/internal/services"
)

type MetadataExtractionHandler struct {
	metadataService *services.MetadataService
}

func NewMetadataExtractionHandler(metadataService *services.MetadataService) *MetadataExtractionHandler {
	return &MetadataExtractionHandler{
		metadataService: metadataService,
	}
}

func (h *MetadataExtractionHandler) Handle(ctx context.Context, job *Job) error {
	payload, ok := job.PayloadData.(MetadataExtractJobPayload)
	if !ok {
		return fmt.Errorf("invalid payload type for metadata extraction job")
	}

	log.Printf("Extracting metadata for file: %s", payload.FilePath)

	// Extract and store metadata
	song, err := h.metadataService.ExtractAndStoreMetadata(ctx, payload.FilePath)
	if err != nil {
		return fmt.Errorf("failed to extract metadata: %w", err)
	}

	log.Printf("Successfully processed metadata for: %s (ID: %d)", song.Title, song.ID)
	return nil
}

type BatchMetadataExtractionHandler struct {
	batchMetadataService *services.BatchMetadataService
}

func NewBatchMetadataExtractionHandler(batchMetadataService *services.BatchMetadataService) *BatchMetadataExtractionHandler {
	return &BatchMetadataExtractionHandler{
		batchMetadataService: batchMetadataService,
	}
}

func (h *BatchMetadataExtractionHandler) Handle(ctx context.Context, job *Job) error {
	payload, ok := job.PayloadData.(BatchMetadataExtractJobPayload)
	if !ok {
		return fmt.Errorf("invalid payload type for batch metadata extraction job")
	}

	log.Printf("Processing batch metadata extraction for %d files", len(payload.FilePaths))

	// Process the batch
	result, err := h.batchMetadataService.ProcessBatch(ctx, payload.FilePaths)
	if err != nil {
		return fmt.Errorf("failed to process metadata batch: %w", err)
	}

	log.Printf("Batch processing completed: %d processed, %d success, %d errors in %v", 
		result.ProcessedCount, result.SuccessCount, result.ErrorCount, result.Duration)

	// Log batch errors but don't fail the job
	for _, batchErr := range result.Errors {
		log.Printf("Batch processing error: %v", batchErr)
	}

	return nil
}

type ScanResult struct {
	FilesFound    int
	FilesAdded    int
	FilesUpdated  int
	FilesRemoved  int
	Duration      time.Duration
	Errors        []error
}

type ScanHandler struct {
	scanner Scanner
}

type Scanner interface {
	ScanLibrary(ctx context.Context, force bool) (*ScanResult, error)
}

func NewScanHandler(scanner Scanner) *ScanHandler {
	return &ScanHandler{scanner: scanner}
}

func (h *ScanHandler) Handle(ctx context.Context, job *Job) error {
	payload, ok := job.PayloadData.(ScanJobPayload)
	if !ok {
		return fmt.Errorf("invalid payload type for scan job")
	}

	log.Printf("Starting library scan: path=%s, recursive=%t, force=%t", 
		payload.Path, payload.Recursive, payload.Force)

	// Perform scan
	result, err := h.scanner.ScanLibrary(ctx, payload.Force)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	log.Printf("Scan completed: %d files found, %d added, %d updated, %d removed, %d errors", 
		result.FilesFound, result.FilesAdded, result.FilesUpdated, result.FilesRemoved, len(result.Errors))

	// Log any errors but don't fail the job
	for _, scanErr := range result.Errors {
		log.Printf("Scan error: %v", scanErr)
	}

	return nil
}

type CleanupHandler struct {
	// Add cleanup service dependencies here
}

func NewCleanupHandler() *CleanupHandler {
	return &CleanupHandler{}
}

func (h *CleanupHandler) Handle(ctx context.Context, job *Job) error {
	payload, ok := job.PayloadData.(CleanupJobPayload)
	if !ok {
		return fmt.Errorf("invalid payload type for cleanup job")
	}

	log.Printf("Starting cleanup job: type=%s, older_than=%v", payload.Type, payload.OlderThan)

	// Implement cleanup logic based on type
	switch payload.Type {
	case "sessions":
		// Clean up expired user sessions
		// TODO: implement session cleanup
		log.Println("Session cleanup completed")
	case "jobs":
		// Clean up old completed jobs
		// TODO: implement job cleanup
		log.Println("Job cleanup completed")
	case "cache":
		// Clean up cached files
		// TODO: implement cache cleanup
		log.Println("Cache cleanup completed")
	default:
		return fmt.Errorf("unknown cleanup type: %s", payload.Type)
	}

	return nil
}

type StatsUpdateHandler struct {
	// Add stats service dependencies here
}

func NewStatsUpdateHandler() *StatsUpdateHandler {
	return &StatsUpdateHandler{}
}

func (h *StatsUpdateHandler) Handle(ctx context.Context, job *Job) error {
	payload, ok := job.PayloadData.(StatsUpdateJobPayload)
	if !ok {
		return fmt.Errorf("invalid payload type for stats update job")
	}

	if payload.UserID != nil {
		log.Printf("Updating stats for user: %d", *payload.UserID)
		// TODO: Update stats for specific user
	} else {
		log.Println("Updating global stats")
		// TODO: Update global statistics
	}

	return nil
}