package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"korus/internal/database"
	"korus/internal/models"

	"github.com/jackc/pgx/v5"
)

const (
	JobTypeScan                  = "scan"
	JobTypeMetadataExtract       = "metadata_extract"
	JobTypeMetadataExtractBatch  = "metadata_extract_batch"
	JobTypeTranscode             = "transcode"
	JobTypeCleanup               = "cleanup"
	JobTypeStatsUpdate           = "stats_update"
	JobTypeEmbeddingExtract      = "embedding_extract"
	JobTypeEmbeddingExtractBatch = "embedding_extract_batch"
	JobTypeEmbeddingBatchCreator = "embedding_batch_creator"
)

const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

type Queue struct {
	db *database.DB
}

type Job struct {
	*models.Job
	PayloadData interface{} `json:"-"`
}

type ScanJobPayload struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"`
	Force     bool   `json:"force"`
}

type MetadataExtractJobPayload struct {
	FilePath string `json:"file_path"`
	SongID   *int   `json:"song_id,omitempty"`
}

type BatchMetadataExtractJobPayload struct {
	FilePaths []string `json:"file_paths"`
}

type EmbeddingExtractJobPayload struct {
	FilePath string `json:"file_path"`
	SongID   *int   `json:"song_id,omitempty"`
}

type BatchEmbeddingExtractJobPayload struct {
	Entries []BatchEmbeddingEntry `json:"entries"`
}

type BatchEmbeddingEntry struct {
	FilePath string `json:"file_path"`
	SongID   *int   `json:"song_id,omitempty"`
}

type EmbeddingBatchJobPayload struct {
	MaxSongsPerBatch int   `json:"max_songs_per_batch"`
	ScanStartTime    int64 `json:"scan_start_time"` // Unix timestamp to identify newly processed songs
}

type TranscodeJobPayload struct {
	SongID       int    `json:"song_id"`
	OutputFormat string `json:"output_format"`
	Bitrate      int    `json:"bitrate"`
}

type CleanupJobPayload struct {
	Type      string    `json:"type"`
	OlderThan time.Time `json:"older_than"`
}

type StatsUpdateJobPayload struct {
	UserID *int `json:"user_id,omitempty"`
}

func NewQueue(db *database.DB) *Queue {
	return &Queue{db: db}
}

func (q *Queue) Enqueue(ctx context.Context, jobType string, payload interface{}) (*Job, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := `
		INSERT INTO job_queue (job_type, payload, status, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, job_type, payload, status, created_at, processed_at, attempts, last_error
	`

	var job Job
	job.Job = &models.Job{}

	err = q.db.QueryRowContext(ctx, query, jobType, payloadBytes, JobStatusPending).
		Scan(&job.ID, &job.JobType, &job.Payload, &job.Status,
			&job.CreatedAt, &job.ProcessedAt, &job.Attempts, &job.LastError)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	job.PayloadData = payload
	return &job, nil
}

func (q *Queue) Dequeue(ctx context.Context, jobTypes []string) (*Job, error) {
	tx, err := q.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Build query with job type filter
	query := `
		SELECT id, job_type, payload, status, created_at, processed_at, attempts, last_error
		FROM job_queue
		WHERE status = $1 AND job_type = ANY($2)
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var job Job
	job.Job = &models.Job{}

	err = tx.QueryRow(ctx, query, JobStatusPending, jobTypes).
		Scan(&job.ID, &job.JobType, &job.Payload, &job.Status,
			&job.CreatedAt, &job.ProcessedAt, &job.Attempts, &job.LastError)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	// Mark job as processing
	_, err = tx.Exec(ctx,
		"UPDATE job_queue SET status = $1, attempts = attempts + 1 WHERE id = $2",
		JobStatusProcessing, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark job as processing: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Unmarshal payload based on job type
	if err := q.unmarshalPayload(&job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &job, nil
}

func (q *Queue) Complete(ctx context.Context, jobID int) error {
	query := `
		UPDATE job_queue
		SET status = $1, processed_at = NOW()
		WHERE id = $2
	`

	_, err := q.db.ExecContext(ctx, query, JobStatusCompleted, jobID)
	if err != nil {
		return fmt.Errorf("failed to mark job as completed: %w", err)
	}

	return nil
}

func (q *Queue) Fail(ctx context.Context, jobID int, errorMsg string) error {
	query := `
		UPDATE job_queue
		SET status = $1, last_error = $2, processed_at = NOW()
		WHERE id = $3
	`

	_, err := q.db.ExecContext(ctx, query, JobStatusFailed, errorMsg, jobID)
	if err != nil {
		return fmt.Errorf("failed to mark job as failed: %w", err)
	}

	return nil
}

func (q *Queue) Retry(ctx context.Context, jobID int, maxAttempts int) error {
	query := `
		UPDATE job_queue
		SET status = CASE
			WHEN attempts < $2 THEN $3
			ELSE $4
		END,
		last_error = CASE
			WHEN attempts >= $2 THEN 'Max retry attempts exceeded'
			ELSE last_error
		END
		WHERE id = $1
	`

	_, err := q.db.ExecContext(ctx, query, jobID, maxAttempts, JobStatusPending, JobStatusFailed)
	if err != nil {
		return fmt.Errorf("failed to retry job: %w", err)
	}

	return nil
}

func (q *Queue) GetJob(ctx context.Context, jobID int) (*Job, error) {
	query := `
		SELECT id, job_type, payload, status, created_at, processed_at, attempts, last_error
		FROM job_queue
		WHERE id = $1
	`

	var job Job
	job.Job = &models.Job{}

	err := q.db.QueryRowContext(ctx, query, jobID).
		Scan(&job.ID, &job.JobType, &job.Payload, &job.Status,
			&job.CreatedAt, &job.ProcessedAt, &job.Attempts, &job.LastError)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if err := q.unmarshalPayload(&job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &job, nil
}

func (q *Queue) ListJobs(ctx context.Context, status string, limit, offset int) ([]Job, error) {
	query := `
		SELECT id, job_type, payload, status, created_at, processed_at, attempts, last_error
		FROM job_queue
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := q.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var job Job
		job.Job = &models.Job{}

		err := rows.Scan(&job.ID, &job.JobType, &job.Payload, &job.Status,
			&job.CreatedAt, &job.ProcessedAt, &job.Attempts, &job.LastError)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		if err := q.unmarshalPayload(&job); err != nil {
			// Log error but continue with other jobs
			fmt.Printf("Warning: failed to unmarshal payload for job %d: %v\n", job.ID, err)
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func (q *Queue) CleanupCompletedJobs(ctx context.Context, olderThan time.Time) (int, error) {
	query := `
		DELETE FROM job_queue
		WHERE status = $1 AND processed_at < $2
	`

	result, err := q.db.ExecContext(ctx, query, JobStatusCompleted, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup completed jobs: %w", err)
	}

	return int(result.RowsAffected()), nil
}

func (q *Queue) GetQueueStats(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*)
		FROM job_queue
		GROUP BY status
	`

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int

		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan queue stats: %w", err)
		}

		stats[status] = count
	}

	return stats, rows.Err()
}

func (q *Queue) unmarshalPayload(job *Job) error {
	if job.Payload == nil {
		return nil
	}

	switch job.JobType {
	case JobTypeScan:
		var payload ScanJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		job.PayloadData = payload

	case JobTypeMetadataExtract:
		var payload MetadataExtractJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		job.PayloadData = payload

	case JobTypeMetadataExtractBatch:
		var payload BatchMetadataExtractJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		job.PayloadData = payload

	case JobTypeTranscode:
		var payload TranscodeJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		job.PayloadData = payload

	case JobTypeCleanup:
		var payload CleanupJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		job.PayloadData = payload

	case JobTypeStatsUpdate:
		var payload StatsUpdateJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		job.PayloadData = payload

	case JobTypeEmbeddingExtract:
		var payload EmbeddingExtractJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		job.PayloadData = payload

	case JobTypeEmbeddingExtractBatch:
		var payload BatchEmbeddingExtractJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		job.PayloadData = payload

	case JobTypeEmbeddingBatchCreator:
		var payload EmbeddingBatchJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		job.PayloadData = payload
	}

	return nil
}
