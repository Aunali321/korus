package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"korus/internal/database"
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

type EmbeddingExtractionHandler struct {
	embeddingService   *services.EmbeddingService
	recommenderService *services.RecommenderService
}

func NewEmbeddingExtractionHandler(embeddingService *services.EmbeddingService, recommenderService *services.RecommenderService) *EmbeddingExtractionHandler {
	return &EmbeddingExtractionHandler{embeddingService: embeddingService, recommenderService: recommenderService}
}

func (h *EmbeddingExtractionHandler) Handle(ctx context.Context, job *Job) error {
	if !h.embeddingService.Enabled() {
		log.Printf("Embedding service disabled; skipping job %d", job.ID)
		return nil
	}

	payload, ok := job.PayloadData.(EmbeddingExtractJobPayload)
	if !ok {
		return fmt.Errorf("invalid payload type for embedding extraction job")
	}

	log.Printf("Extracting embedding for file: %s", payload.FilePath)

	task := services.EmbeddingTask{FilePath: payload.FilePath, SongID: payload.SongID}
	processed, err := h.embeddingService.ProcessBatch(ctx, []services.EmbeddingTask{task})
	if err != nil {
		return fmt.Errorf("failed to extract embedding: %w", err)
	}

	if h.recommenderService != nil && len(processed) > 0 {
		songIDs := make([]int, 0, len(processed))
		for _, item := range processed {
			songIDs = append(songIDs, item.SongID)
		}
		if err := h.recommenderService.RefreshSongs(ctx, songIDs); err != nil {
			log.Printf("Failed to refresh recommender index for songs %v: %v", songIDs, err)
		}
	}

	return nil
}

type BatchEmbeddingExtractionHandler struct {
	embeddingService   *services.EmbeddingService
	recommenderService *services.RecommenderService
}

func NewBatchEmbeddingExtractionHandler(embeddingService *services.EmbeddingService, recommenderService *services.RecommenderService) *BatchEmbeddingExtractionHandler {
	return &BatchEmbeddingExtractionHandler{embeddingService: embeddingService, recommenderService: recommenderService}
}

func (h *BatchEmbeddingExtractionHandler) Handle(ctx context.Context, job *Job) error {
	if !h.embeddingService.Enabled() {
		log.Printf("Embedding service disabled; skipping batch job %d", job.ID)
		return nil
	}

	payload, ok := job.PayloadData.(BatchEmbeddingExtractJobPayload)
	if !ok {
		return fmt.Errorf("invalid payload type for batch embedding extraction job")
	}

	if len(payload.Entries) == 0 {
		return nil
	}

	log.Printf("Processing batch embedding extraction for %d files", len(payload.Entries))

	tasks := make([]services.EmbeddingTask, 0, len(payload.Entries))
	for _, entry := range payload.Entries {
		tasks = append(tasks, services.EmbeddingTask{FilePath: entry.FilePath, SongID: entry.SongID})
	}

	processed, err := h.embeddingService.ProcessBatch(ctx, tasks)
	if err != nil {
		return fmt.Errorf("failed to process embedding batch: %w", err)
	}

	if h.recommenderService != nil && len(processed) > 0 {
		songIDs := make([]int, 0, len(processed))
		for _, item := range processed {
			songIDs = append(songIDs, item.SongID)
		}
		if err := h.recommenderService.RefreshSongs(ctx, songIDs); err != nil {
			log.Printf("Failed to refresh recommender index for songs %v: %v", songIDs, err)
		}
	}

	return nil
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

	// Embeddings will be processed by a separate cleanup job after all metadata is complete

	// Log batch errors but don't fail the job
	for _, batchErr := range result.Errors {
		log.Printf("Batch processing error: %v", batchErr)
	}

	return nil
}

type ScanResult struct {
	FilesFound   int
	FilesAdded   int
	FilesUpdated int
	FilesRemoved int
	Duration     time.Duration
	Errors       []error
}

type ScanHandler struct {
	scanner          Scanner
	queue            *Queue
	embeddingsEnabled bool
	batchSize        int
}

type Scanner interface {
	ScanLibrary(ctx context.Context, force bool) (*ScanResult, error)
}

func NewScanHandler(scanner Scanner, queue *Queue, embeddingsEnabled bool, batchSize int) *ScanHandler {
	return &ScanHandler{
		scanner:          scanner,
		queue:            queue,
		embeddingsEnabled: embeddingsEnabled,
		batchSize:        batchSize,
	}
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

	// Enqueue embedding batch creator job if files were added/updated and embeddings are enabled
	if h.embeddingsEnabled && h.queue != nil && (result.FilesAdded > 0 || result.FilesUpdated > 0) {
		batchPayload := EmbeddingBatchJobPayload{
			MaxSongsPerBatch: h.batchSize,
			ScanStartTime:    time.Now().Unix(),
		}

		if _, err := h.queue.Enqueue(ctx, JobTypeEmbeddingBatchCreator, batchPayload); err != nil {
			log.Printf("Failed to enqueue embedding batch creator job: %v", err)
		} else {
			log.Printf("Enqueued embedding batch creator job for %d new/updated files", result.FilesAdded+result.FilesUpdated)
		}
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

type EmbeddingBatchHandler struct {
	db                    *database.DB
	queue                 *Queue
	batchSize             int
	embeddingService      *services.EmbeddingService
	recommenderService    *services.RecommenderService
}

func NewEmbeddingBatchHandler(db *database.DB, queue *Queue, batchSize int, embeddingService *services.EmbeddingService, recommenderService *services.RecommenderService) *EmbeddingBatchHandler {
	return &EmbeddingBatchHandler{
		db:                 db,
		queue:              queue,
		batchSize:          batchSize,
		embeddingService:   embeddingService,
		recommenderService: recommenderService,
	}
}

func (h *EmbeddingBatchHandler) Handle(ctx context.Context, job *Job) error {
	payload, ok := job.PayloadData.(EmbeddingBatchJobPayload)
	if !ok {
		return fmt.Errorf("invalid payload type for embedding batch creator job")
	}

	log.Printf("Starting embedding batch creation: max batch size=%d", payload.MaxSongsPerBatch)

	// Find all songs that don't have embeddings yet
	query := `
		SELECT s.id, s.file_path, s.updated_at
		FROM songs s
		LEFT JOIN song_embeddings se ON s.id = se.song_id
		WHERE se.song_id IS NULL
		ORDER BY s.updated_at DESC
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query songs without embeddings: %w", err)
	}
	defer rows.Close()

	type SongInfo struct {
		ID       int    `json:"id"`
		FilePath string `json:"file_path"`
	}

	var songs []SongInfo
	for rows.Next() {
		var song SongInfo
		if err := rows.Scan(&song.ID, &song.FilePath); err != nil {
			log.Printf("Failed to scan song row: %v", err)
			continue
		}
		songs = append(songs, song)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating song rows: %w", err)
	}

	if len(songs) == 0 {
		log.Printf("No songs need embedding processing")
		return nil
	}

	log.Printf("Found %d songs that need embedding processing", len(songs))

	// Create batch jobs
	batchCount := 0
	for i := 0; i < len(songs); i += payload.MaxSongsPerBatch {
		end := i + payload.MaxSongsPerBatch
		if end > len(songs) {
			end = len(songs)
		}

		batchSongs := songs[i:end]
		entries := make([]BatchEmbeddingEntry, 0, len(batchSongs))

		for _, song := range batchSongs {
			entries = append(entries, BatchEmbeddingEntry{
				FilePath: song.FilePath,
				SongID:   &song.ID,
			})
		}

		batchPayload := BatchEmbeddingExtractJobPayload{Entries: entries}
		if _, err := h.queue.Enqueue(ctx, JobTypeEmbeddingExtractBatch, batchPayload); err != nil {
			return fmt.Errorf("failed to enqueue embedding batch job: %w", err)
		}

		batchCount++
		log.Printf("Enqueued embedding batch %d with %d songs", batchCount, len(batchSongs))
	}

	log.Printf("Successfully created %d embedding batch jobs for %d songs", batchCount, len(songs))
	return nil
}
