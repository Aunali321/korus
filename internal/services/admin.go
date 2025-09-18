package services

import (
	"context"
	"encoding/json"
	"fmt"

	"korus/internal/database"
	"korus/internal/models"
)

type AdminService struct {
	db *database.DB
}

func NewAdminService(db *database.DB) *AdminService {
	return &AdminService{db: db}
}

// TriggerLibraryScan triggers a full library scan by creating a job in the queue
func (as *AdminService) TriggerLibraryScan(ctx context.Context) (int, error) {
	// Create a scan job payload
	payload := map[string]interface{}{
		"type":      "full_scan",
		"recursive": true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal scan payload: %w", err)
	}

	// Insert job into queue
	query := `
		INSERT INTO job_queue (job_type, payload, status, created_at)
		VALUES ($1, $2, 'pending', NOW())
		RETURNING id
	`

	var jobID int
	err = as.db.QueryRowContext(ctx, query, "scan", payloadBytes).Scan(&jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to create scan job: %w", err)
	}

	return jobID, nil
}

// GetSystemStatus returns system status information
func (as *AdminService) GetSystemStatus(ctx context.Context) (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// Get library statistics
	libraryStats := make(map[string]interface{})

	// Get total counts
	countsQuery := `
		SELECT 
			(SELECT COUNT(*) FROM songs) as song_count,
			(SELECT COUNT(*) FROM albums) as album_count,
			(SELECT COUNT(*) FROM artists) as artist_count,
			(SELECT COUNT(*) FROM users) as user_count,
			(SELECT COUNT(*) FROM playlists) as playlist_count
	`
	var songCount, albumCount, artistCount, userCount, playlistCount int
	err := as.db.QueryRowContext(ctx, countsQuery).
		Scan(&songCount, &albumCount, &artistCount, &userCount, &playlistCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get library counts: %w", err)
	}

	libraryStats["songs"] = songCount
	libraryStats["albums"] = albumCount
	libraryStats["artists"] = artistCount
	libraryStats["users"] = userCount
	libraryStats["playlists"] = playlistCount

	// Get total duration
	var totalDuration int64
	durationQuery := `SELECT COALESCE(SUM(duration), 0) FROM songs`
	err = as.db.QueryRowContext(ctx, durationQuery).Scan(&totalDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get total duration: %w", err)
	}
	libraryStats["total_duration"] = totalDuration

	status["library"] = libraryStats

	// Get recent scan history
	scanHistory, err := as.GetRecentScanHistory(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get scan history: %w", err)
	}
	status["recent_scans"] = scanHistory

	// Get job queue status
	jobStats, err := as.getJobQueueStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get job queue stats: %w", err)
	}
	status["job_queue"] = jobStats

	// Get recent activity (play counts for last 24 hours, 7 days, 30 days)
	activityStats, err := as.getActivityStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity stats: %w", err)
	}
	status["activity"] = activityStats

	return status, nil
}

// GetRecentScanHistory returns recent scan history
func (as *AdminService) GetRecentScanHistory(ctx context.Context, limit int) ([]models.ScanHistory, error) {
	query := `
		SELECT id, started_at, completed_at, songs_added, songs_updated, songs_removed
		FROM scan_history
		ORDER BY started_at DESC
		LIMIT $1
	`

	rows, err := as.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query scan history: %w", err)
	}
	defer rows.Close()

	history := make([]models.ScanHistory, 0)
	for rows.Next() {
		var scan models.ScanHistory
		err := rows.Scan(&scan.ID, &scan.StartedAt, &scan.CompletedAt,
			&scan.SongsAdded, &scan.SongsUpdated, &scan.SongsRemoved)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scan history: %w", err)
		}
		history = append(history, scan)
	}

	return history, rows.Err()
}

// GetPendingJobs returns jobs that are currently pending or processing
func (as *AdminService) GetPendingJobs(ctx context.Context) ([]models.Job, error) {
	query := `
		SELECT id, job_type, payload, status, created_at, processed_at, attempts, last_error
		FROM job_queue
		WHERE status IN ('pending', 'processing')
		ORDER BY created_at DESC
	`

	rows, err := as.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]models.Job, 0)
	for rows.Next() {
		var job models.Job
		err := rows.Scan(&job.ID, &job.JobType, &job.Payload, &job.Status,
			&job.CreatedAt, &job.ProcessedAt, &job.Attempts, &job.LastError)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// getJobQueueStats returns job queue statistics
func (as *AdminService) getJobQueueStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get job counts by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM job_queue
		GROUP BY status
	`

	rows, err := as.db.QueryContext(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query job status counts: %w", err)
	}
	defer rows.Close()

	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job status: %w", err)
		}
		statusCounts[status] = count
	}
	stats["by_status"] = statusCounts

	// Get job counts by type
	typeQuery := `
		SELECT job_type, COUNT(*) as count
		FROM job_queue
		WHERE created_at > NOW() - INTERVAL '24 hours'
		GROUP BY job_type
	`

	rows2, err := as.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query job type counts: %w", err)
	}
	defer rows2.Close()

	typeCounts := make(map[string]int)
	for rows2.Next() {
		var jobType string
		var count int
		err := rows2.Scan(&jobType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job type: %w", err)
		}
		typeCounts[jobType] = count
	}
	stats["by_type_24h"] = typeCounts

	return stats, nil
}

// getActivityStats returns activity statistics
func (as *AdminService) getActivityStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get play counts for different time periods
	periodsQuery := `
		SELECT 
			COUNT(CASE WHEN played_at > NOW() - INTERVAL '24 hours' THEN 1 END) as plays_24h,
			COUNT(CASE WHEN played_at > NOW() - INTERVAL '7 days' THEN 1 END) as plays_7d,
			COUNT(CASE WHEN played_at > NOW() - INTERVAL '30 days' THEN 1 END) as plays_30d,
			COUNT(*) as total_plays
		FROM play_history
	`

	var plays24h, plays7d, plays30d, totalPlays int
	err := as.db.QueryRowContext(ctx, periodsQuery).
		Scan(&plays24h, &plays7d, &plays30d, &totalPlays)
	if err != nil {
		return nil, fmt.Errorf("failed to get play counts: %w", err)
	}

	stats["plays_24h"] = plays24h
	stats["plays_7d"] = plays7d
	stats["plays_30d"] = plays30d
	stats["total_plays"] = totalPlays

	// Get active users (users who played something in the last 30 days)
	activeUsersQuery := `
		SELECT COUNT(DISTINCT user_id)
		FROM play_history
		WHERE played_at > NOW() - INTERVAL '30 days'
	`

	var activeUsers int
	err = as.db.QueryRowContext(ctx, activeUsersQuery).Scan(&activeUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users count: %w", err)
	}
	stats["active_users_30d"] = activeUsers

	return stats, nil
}

// CleanupOldJobs removes completed and failed jobs older than specified days
func (as *AdminService) CleanupOldJobs(ctx context.Context, olderThanDays int) (int, error) {
	query := `
		DELETE FROM job_queue
		WHERE status IN ('completed', 'failed') 
		AND created_at < NOW() - INTERVAL '%d days'
	`

	result, err := as.db.ExecContext(ctx, fmt.Sprintf(query, olderThanDays))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old jobs: %w", err)
	}

	rowsAffected := int(result.RowsAffected())
	return rowsAffected, nil
}

// CleanupOldSessions removes expired user sessions
func (as *AdminService) CleanupOldSessions(ctx context.Context) (int, error) {
	query := `DELETE FROM user_sessions WHERE expires_at < NOW()`

	result, err := as.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old sessions: %w", err)
	}

	rowsAffected := int(result.RowsAffected())
	return rowsAffected, nil
}
