package services

import (
	"context"
	"fmt"
	"time"

	"korus/internal/database"
	"korus/internal/models"
)

type AdminService struct {
	db      *database.DB
	indexer LibraryIndexer
}

func NewAdminService(db *database.DB, indexer LibraryIndexer) *AdminService {
	return &AdminService{db: db, indexer: indexer}
}

type LibraryIndexer interface {
	StartScanAsync(ctx context.Context, force bool) (string, error)
	GetJob(jobID string) (*LibraryScanJob, error)
	Status() LibraryIndexerStatus
}

type LibraryScanJob struct {
	ID          string             `json:"id"`
	Status      string             `json:"status"`
	Phase       string             `json:"phase"`
	Progress    int                `json:"progress"`
	Total       int                `json:"total"`
	Force       bool               `json:"force"`
	StartedAt   time.Time          `json:"startedAt"`
	CompletedAt *time.Time         `json:"completedAt,omitempty"`
	Result      *LibraryScanResult `json:"result,omitempty"`
	Error       string             `json:"error,omitempty"`
}

type LibraryScanResult struct {
	StartedAt       time.Time
	CompletedAt     time.Time
	Duration        time.Duration
	FilesDiscovered int
	FilesQueued     int
	FilesNew        int
	FilesUpdated    int
	FilesRemoved    int
	Ingested        int
	Errors          []error `json:"-"`
}

type LibraryIndexerStatus struct {
	Running   bool               `json:"running"`
	LastRun   *LibraryScanResult `json:"lastRun,omitempty"`
	LastError string             `json:"lastError,omitempty"`
}

func (as *AdminService) TriggerLibraryScan(ctx context.Context, force bool) (string, error) {
	if as.indexer == nil {
		return "", fmt.Errorf("indexer not configured")
	}

	return as.indexer.StartScanAsync(ctx, force)
}

func (as *AdminService) GetScanJob(jobID string) (*LibraryScanJob, error) {
	if as.indexer == nil {
		return nil, fmt.Errorf("indexer not configured")
	}

	return as.indexer.GetJob(jobID)
}

func (as *AdminService) GetSystemStatus(ctx context.Context) (map[string]interface{}, error) {
	status := make(map[string]interface{})

	libraryStats := make(map[string]interface{})

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

	var totalDuration int64
	durationQuery := `SELECT COALESCE(SUM(duration), 0) FROM songs`
	err = as.db.QueryRowContext(ctx, durationQuery).Scan(&totalDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get total duration: %w", err)
	}
	libraryStats["totalDuration"] = totalDuration

	status["library"] = libraryStats

	scanHistory, err := as.GetRecentScanHistory(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get scan history: %w", err)
	}
	status["recentScans"] = scanHistory

	if as.indexer != nil {
		status["indexer"] = as.indexer.Status()
	}

	// Get recent activity (play counts for last 24 hours, 7 days, 30 days)
	activityStats, err := as.getActivityStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity stats: %w", err)
	}
	status["activity"] = activityStats

	return status, nil
}

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

func (as *AdminService) getActivityStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

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

	stats["plays24h"] = plays24h
	stats["plays7d"] = plays7d
	stats["plays30d"] = plays30d
	stats["totalPlays"] = totalPlays

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
	stats["activeUsers30d"] = activeUsers

	return stats, nil
}

func (as *AdminService) CleanupOldSessions(ctx context.Context) (int, error) {
	query := `DELETE FROM user_sessions WHERE expires_at < NOW()`

	result, err := as.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old sessions: %w", err)
	}

	rowsAffected := int(result.RowsAffected())
	return rowsAffected, nil
}
