package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"korus/internal/database"

	"github.com/jackc/pgx/v5"
)

type BatchMetadataService struct {
	db              *database.DB
	metadataService *MetadataService
}

type BatchResult struct {
	ProcessedCount int
	SuccessCount   int
	ErrorCount     int
	Errors         []error
	Duration       time.Duration
	SuccessFiles   []ProcessedSong
}

type ProcessedSong struct {
	SongID   int
	FilePath string
}

func NewBatchMetadataService(db *database.DB) *BatchMetadataService {
	return &BatchMetadataService{
		db:              db,
		metadataService: NewMetadataService(db),
	}
}

func (bms *BatchMetadataService) ProcessBatch(ctx context.Context, filePaths []string) (*BatchResult, error) {
	start := time.Now()
	result := &BatchResult{
		ProcessedCount: len(filePaths),
		SuccessFiles:   make([]ProcessedSong, 0, len(filePaths)),
	}

	log.Printf("🚀 Starting batch processing of %d files", len(filePaths))

	// Phase 1: Extract metadata from all files (parallel, no DB)
	log.Printf("📖 Phase 1: Extracting metadata from %d files", len(filePaths))
	metadataList := make([]*ExtractedMetadata, 0, len(filePaths))
	for _, filePath := range filePaths {
		metadata, err := bms.metadataService.extractMetadata(filePath)
		if err != nil {
			log.Printf("❌ Failed to extract metadata from %s: %v", filePath, err)
			result.Errors = append(result.Errors, fmt.Errorf("failed to extract metadata from %s: %w", filePath, err))
			result.ErrorCount++
			continue
		}
		metadataList = append(metadataList, metadata)
	}

	log.Printf("📖 Phase 1 complete: %d metadata extracted, %d errors", len(metadataList), result.ErrorCount)

	// Phase 2: Batch database operations
	if len(metadataList) > 0 {
		log.Printf("💾 Phase 2: Storing %d songs to database", len(metadataList))
		err := bms.storeBatchMetadata(ctx, metadataList, result)
		if err != nil {
			log.Printf("❌ Failed to store batch metadata: %v", err)
			return nil, fmt.Errorf("failed to store batch metadata: %w", err)
		}
		log.Printf("✅ Phase 2 complete: Database operations successful")
	} else {
		log.Printf("⚠️  No metadata to store (all files failed extraction)")
	}

	result.Duration = time.Since(start)
	result.SuccessCount = result.ProcessedCount - result.ErrorCount

	log.Printf("🎉 Batch processing completed: %d processed, %d success, %d errors in %v",
		result.ProcessedCount, result.SuccessCount, result.ErrorCount, result.Duration)

	return result, nil
}

func (bms *BatchMetadataService) storeBatchMetadata(ctx context.Context, metadataList []*ExtractedMetadata, result *BatchResult) error {
	log.Printf("💾 Starting database transaction for %d songs", len(metadataList))
	return bms.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Collect unique artists and albums for batch processing
		log.Printf("🔍 Collecting unique artists and albums...")
		artistMap := make(map[string]bool)
		albumMap := make(map[string]albumKey)

		for _, metadata := range metadataList {
			artistMap[strings.ToLower(metadata.Artist)] = true
			if metadata.AlbumArtist != "" && metadata.AlbumArtist != metadata.Artist {
				artistMap[strings.ToLower(metadata.AlbumArtist)] = true
			}

			albumKeyName := strings.ToLower(metadata.Album)
			if existingAlbum, exists := albumMap[albumKeyName]; exists {
				// Keep existing cover if it exists, otherwise use this song's cover
				coverURL := existingAlbum.coverURL
				if coverURL == "" && metadata.CoverURL != "" {
					coverURL = metadata.CoverURL
				}
				albumMap[albumKeyName] = albumKey{
					name:     metadata.Album,
					artist:   metadata.Artist,
					coverURL: coverURL,
				}
			} else {
				albumMap[albumKeyName] = albumKey{
					name:     metadata.Album,
					artist:   metadata.Artist,
					coverURL: metadata.CoverURL,
				}
			}
		}

		log.Printf("🎤 Found %d unique artists, %d unique albums", len(artistMap), len(albumMap))

		// Batch insert/update artists
		log.Printf("🎤 Batch upserting %d artists...", len(artistMap))
		artistIDs, err := bms.batchUpsertArtists(ctx, tx, artistMap, metadataList)
		if err != nil {
			log.Printf("❌ Failed to batch upsert artists: %v", err)
			return fmt.Errorf("failed to batch upsert artists: %w", err)
		}
		log.Printf("✅ Created/found %d artists", len(artistIDs))

		// Batch insert/update albums
		log.Printf("💿 Batch upserting %d albums...", len(albumMap))
		albumIDs, err := bms.batchUpsertAlbums(ctx, tx, albumMap, artistIDs)
		if err != nil {
			log.Printf("❌ Failed to batch upsert albums: %v", err)
			return fmt.Errorf("failed to batch upsert albums: %w", err)
		}
		log.Printf("✅ Created/found %d albums", len(albumIDs))

		// Insert/update songs
		log.Printf("🎵 Inserting %d songs...", len(metadataList))
		successCount := 0
		for i, metadata := range metadataList {
			songID, err := bms.insertSong(ctx, tx, metadata, artistIDs, albumIDs)
			if err != nil {
				log.Printf("❌ Failed to insert song [%d/%d] %s: %v", i+1, len(metadataList), metadata.Title, err)
				result.Errors = append(result.Errors, fmt.Errorf("failed to insert song %s: %w", metadata.FilePath, err))
				result.ErrorCount++
				continue
			}
			successCount++
			result.SuccessFiles = append(result.SuccessFiles, ProcessedSong{SongID: songID, FilePath: metadata.FilePath})
		}

		log.Printf("✅ Successfully inserted %d/%d songs", successCount, len(metadataList))
		return nil
	})
}

type albumKey struct {
	name     string
	artist   string
	coverURL string
}

func (bms *BatchMetadataService) batchUpsertArtists(ctx context.Context, tx pgx.Tx, artistMap map[string]bool, metadataList []*ExtractedMetadata) (map[string]int, error) {
	result := make(map[string]int)

	if len(artistMap) == 0 {
		return result, nil
	}

	// First, get existing artists
	artistNames := make([]string, 0, len(artistMap))
	for name := range artistMap {
		artistNames = append(artistNames, name)
	}

	query := `SELECT id, LOWER(name) as name_lower FROM artists WHERE LOWER(name) = ANY($1)`
	rows, err := tx.Query(ctx, query, artistNames)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing artists: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var nameLower string
		if err := rows.Scan(&id, &nameLower); err != nil {
			continue
		}
		result[nameLower] = id
		delete(artistMap, nameLower) // Remove from map so we don't insert it
	}

	// Insert new artists in batch
	if len(artistMap) > 0 {
		valueStrings := make([]string, 0, len(artistMap))
		valueArgs := make([]interface{}, 0, len(artistMap)*2)
		i := 1

		// Create a map of original names for proper casing
		originalNames := make(map[string]string)
		for _, metadata := range metadataList {
			originalNames[strings.ToLower(metadata.Artist)] = metadata.Artist
			if metadata.AlbumArtist != "" && metadata.AlbumArtist != metadata.Artist {
				originalNames[strings.ToLower(metadata.AlbumArtist)] = metadata.AlbumArtist
			}
		}

		for nameLower := range artistMap {
			originalName := originalNames[nameLower]
			if originalName == "" {
				originalName = nameLower // fallback
			}

			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", i, i+1))
			valueArgs = append(valueArgs, originalName, generateSortName(originalName))
			i += 2
		}

		insertQuery := fmt.Sprintf(`
			INSERT INTO artists (name, sort_name)
			VALUES %s
			ON CONFLICT ((LOWER(name)))
			DO UPDATE SET sort_name = EXCLUDED.sort_name
			RETURNING id, LOWER(name) as name_lower
		`, strings.Join(valueStrings, ","))

		rows, err := tx.Query(ctx, insertQuery, valueArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to batch insert artists: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id int
			var nameLower string
			if err := rows.Scan(&id, &nameLower); err != nil {
				continue
			}
			result[nameLower] = id
		}
	}

	return result, nil
}

func (bms *BatchMetadataService) batchUpsertAlbums(ctx context.Context, tx pgx.Tx, albumMap map[string]albumKey, artistIDs map[string]int) (map[string]int, error) {
	result := make(map[string]int)

	if len(albumMap) == 0 {
		return result, nil
	}

	// For simplicity, process albums one by one (still much faster than individual transactions)
	for albumNameLower, albumInfo := range albumMap {
		artistID, exists := artistIDs[strings.ToLower(albumInfo.artist)]
		if !exists {
			continue // Skip if artist not found
		}

		query := `
			INSERT INTO albums (name, artist_id, album_artist_id, cover_path, date_added)
			VALUES ($1, $2, $2, $3, NOW())
			ON CONFLICT ((LOWER(name)), artist_id)
			DO UPDATE SET 
				artist_id = EXCLUDED.artist_id,
				cover_path = COALESCE(EXCLUDED.cover_path, albums.cover_path)
			RETURNING id
		`

		var albumID int
		err := tx.QueryRow(ctx, query, albumInfo.name, artistID, nullString(albumInfo.coverURL)).Scan(&albumID)
		if err != nil {
			continue // Skip failed albums
		}

		result[albumNameLower] = albumID
	}

	return result, nil
}

func (bms *BatchMetadataService) insertSong(ctx context.Context, tx pgx.Tx, metadata *ExtractedMetadata, artistIDs map[string]int, albumIDs map[string]int) (int, error) {
	artistID, exists := artistIDs[strings.ToLower(metadata.Artist)]
	if !exists {
		return 0, fmt.Errorf("artist not found: %s", metadata.Artist)
	}

	albumID, exists := albumIDs[strings.ToLower(metadata.Album)]
	if !exists {
		return 0, fmt.Errorf("album not found: %s", metadata.Album)
	}

	query := `
		INSERT INTO songs (
			title, album_id, artist_id, track_number, disc_number,
			duration, file_path, file_size, file_modified, bitrate,
			format, cover_path, date_added
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (file_path)
		DO UPDATE SET
			title = EXCLUDED.title,
			album_id = EXCLUDED.album_id,
			artist_id = EXCLUDED.artist_id,
			track_number = EXCLUDED.track_number,
			disc_number = EXCLUDED.disc_number,
			duration = EXCLUDED.duration,
			file_size = EXCLUDED.file_size,
			file_modified = EXCLUDED.file_modified,
			bitrate = EXCLUDED.bitrate,
			format = EXCLUDED.format,
			cover_path = EXCLUDED.cover_path
		RETURNING id
	`

	var songID int
	err := tx.QueryRow(ctx, query,
		metadata.Title, albumID, artistID, metadata.TrackNumber, metadata.DiscNumber,
		metadata.Duration, metadata.FilePath, metadata.FileSize, metadata.FileModified,
		metadata.Bitrate, metadata.Format, nullString(metadata.CoverURL),
	).Scan(&songID)

	if err != nil {
		return 0, err
	}

	// Store lyrics if any were extracted
	if len(metadata.Lyrics) > 0 {
		log.Printf("💾 Batch storing %d lyrics entries for song ID %d", len(metadata.Lyrics), songID)
		err = bms.storeLyrics(ctx, tx, songID, metadata.Lyrics)
		if err != nil {
			return 0, fmt.Errorf("failed to store lyrics in batch: %w", err)
		}
		log.Printf("✅ Successfully stored lyrics in batch for song ID %d", songID)
	}

	return songID, nil
}

// storeLyrics stores lyrics data for a song (reused from MetadataService)
func (bms *BatchMetadataService) storeLyrics(ctx context.Context, tx pgx.Tx, songID int, lyrics []ExtractedLyrics) error {
	// First, delete any existing lyrics for this song
	deleteQuery := "DELETE FROM lyrics WHERE song_id = $1"
	_, err := tx.Exec(ctx, deleteQuery, songID)
	if err != nil {
		return fmt.Errorf("failed to delete existing lyrics: %w", err)
	}

	// Insert new lyrics
	for _, lyric := range lyrics {
		insertQuery := `
			INSERT INTO lyrics (song_id, content, type, source, language, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`
		_, err = tx.Exec(ctx, insertQuery,
			songID, lyric.Content, lyric.Type, lyric.Source, lyric.Language)
		if err != nil {
			return fmt.Errorf("failed to insert lyrics: %w", err)
		}
	}

	return nil
}
