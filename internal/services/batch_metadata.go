package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
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

type BatchOptions struct {
	Workers int
}

func NewBatchMetadataService(db *database.DB, lyricsEnabled bool) *BatchMetadataService {
	return &BatchMetadataService{
		db:              db,
		metadataService: NewMetadataService(db, lyricsEnabled),
	}
}

func (bms *BatchMetadataService) ProcessBatch(ctx context.Context, filePaths []string) (*BatchResult, error) {
	return bms.ProcessBatchWithOptions(ctx, filePaths, BatchOptions{})
}

func (bms *BatchMetadataService) ProcessBatchWithOptions(ctx context.Context, filePaths []string, opts BatchOptions) (*BatchResult, error) {
	start := time.Now()
	result := &BatchResult{
		ProcessedCount: len(filePaths),
		SuccessFiles:   make([]ProcessedSong, 0, len(filePaths)),
	}

	if len(filePaths) == 0 {
		return result, nil
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = 1
	}

	log.Printf("🚀 Starting batch processing of %d files", len(filePaths))

	type metaOutput struct {
		metadata *ExtractedMetadata
		err      error
		path     string
	}

	jobs := make(chan string)
	outputs := make(chan metaOutput)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if ctx.Err() != nil {
					return
				}
				metadata, err := bms.metadataService.extractMetadata(path)
				outputs <- metaOutput{metadata: metadata, err: err, path: path}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, path := range filePaths {
			if ctx.Err() != nil {
				return
			}
			jobs <- path
		}
	}()

	go func() {
		wg.Wait()
		close(outputs)
	}()

	log.Printf("📖 Phase 1: Extracting metadata from %d files (workers=%d)", len(filePaths), workers)

	metadataList := make([]*ExtractedMetadata, 0, len(filePaths))
	for output := range outputs {
		if output.err != nil {
			if errors.Is(output.err, context.Canceled) || errors.Is(output.err, context.DeadlineExceeded) {
				return nil, output.err
			}
			log.Printf("❌ Failed to extract metadata from %s: %v", output.path, output.err)
			result.Errors = append(result.Errors, fmt.Errorf("failed to extract metadata from %s: %w", output.path, output.err))
			result.ErrorCount++
			continue
		}
		metadataList = append(metadataList, output.metadata)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
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

		// Bulk insert/update songs
		log.Printf("🎵 Bulk upserting %d songs...", len(metadataList))
		songIDs, err := bms.bulkUpsertSongs(ctx, tx, metadataList, artistIDs, albumIDs)
		if err != nil {
			log.Printf("❌ Failed to bulk upsert songs: %v", err)
			return fmt.Errorf("failed to bulk upsert songs: %w", err)
		}
		log.Printf("✅ Bulk upserted %d songs", len(songIDs))

		// Build success files and lyrics map
		songLyricsMap := make(map[int][]ExtractedLyrics)
		for _, metadata := range metadataList {
			songID, exists := songIDs[metadata.FilePath]
			if !exists {
				result.Errors = append(result.Errors, fmt.Errorf("song ID not found for %s", metadata.FilePath))
				result.ErrorCount++
				continue
			}
			result.SuccessFiles = append(result.SuccessFiles, ProcessedSong{
				SongID:   songID,
				FilePath: metadata.FilePath,
			})
			if len(metadata.Lyrics) > 0 {
				songLyricsMap[songID] = metadata.Lyrics
			}
		}

		// Bulk insert all lyrics at once
		if len(songLyricsMap) > 0 {
			if err := bms.bulkStoreLyrics(ctx, tx, songLyricsMap); err != nil {
				return fmt.Errorf("failed to bulk store lyrics: %w", err)
			}
		}

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

	// Build bulk insert values
	valueStrings := make([]string, 0, len(albumMap))
	valueArgs := make([]interface{}, 0, len(albumMap)*3)
	paramIdx := 1

	for _, albumInfo := range albumMap {
		artistID, exists := artistIDs[strings.ToLower(albumInfo.artist)]
		if !exists {
			continue
		}
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, NOW())",
			paramIdx, paramIdx+1, paramIdx+1, paramIdx+2))
		valueArgs = append(valueArgs, albumInfo.name, artistID, nullString(albumInfo.coverURL))
		paramIdx += 3
	}

	if len(valueStrings) == 0 {
		return result, nil
	}

	// Bulk upsert all albums
	insertQuery := fmt.Sprintf(`
		INSERT INTO albums (name, artist_id, album_artist_id, cover_path, date_added)
		VALUES %s
		ON CONFLICT ((LOWER(name)), artist_id)
		DO UPDATE SET 
			cover_path = COALESCE(EXCLUDED.cover_path, albums.cover_path)
	`, strings.Join(valueStrings, ", "))

	_, err := tx.Exec(ctx, insertQuery, valueArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk insert albums: %w", err)
	}

	// Now query back all album IDs
	albumNames := make([]string, 0, len(albumMap))
	for _, albumInfo := range albumMap {
		albumNames = append(albumNames, strings.ToLower(albumInfo.name))
	}

	selectQuery := `SELECT id, LOWER(name) FROM albums WHERE LOWER(name) = ANY($1)`
	rows, err := tx.Query(ctx, selectQuery, albumNames)
	if err != nil {
		return nil, fmt.Errorf("failed to query album IDs: %w", err)
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

	return result, nil
}

// bulkUpsertSongs bulk inserts all songs and returns a map of file_path -> song_id
func (bms *BatchMetadataService) bulkUpsertSongs(ctx context.Context, tx pgx.Tx, metadataList []*ExtractedMetadata, artistIDs map[string]int, albumIDs map[string]int) (map[string]int, error) {
	result := make(map[string]int)

	if len(metadataList) == 0 {
		return result, nil
	}

	// Build bulk insert values
	valueStrings := make([]string, 0, len(metadataList))
	valueArgs := make([]interface{}, 0, len(metadataList)*12)
	filePaths := make([]string, 0, len(metadataList))
	paramIdx := 1

	for _, metadata := range metadataList {
		artistID, exists := artistIDs[strings.ToLower(metadata.Artist)]
		if !exists {
			continue
		}
		albumID, exists := albumIDs[strings.ToLower(metadata.Album)]
		if !exists {
			continue
		}

		valueStrings = append(valueStrings, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, NOW())",
			paramIdx, paramIdx+1, paramIdx+2, paramIdx+3, paramIdx+4, paramIdx+5,
			paramIdx+6, paramIdx+7, paramIdx+8, paramIdx+9, paramIdx+10, paramIdx+11))
		valueArgs = append(valueArgs,
			metadata.Title, albumID, artistID, metadata.TrackNumber, metadata.DiscNumber,
			metadata.Duration, metadata.FilePath, metadata.FileSize, metadata.FileModified,
			metadata.Bitrate, metadata.Format, nullString(metadata.CoverURL))
		filePaths = append(filePaths, metadata.FilePath)
		paramIdx += 12
	}

	if len(valueStrings) == 0 {
		return result, nil
	}

	// Bulk upsert all songs
	insertQuery := fmt.Sprintf(`
		INSERT INTO songs (
			title, album_id, artist_id, track_number, disc_number,
			duration, file_path, file_size, file_modified, bitrate,
			format, cover_path, date_added
		) VALUES %s
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
	`, strings.Join(valueStrings, ", "))

	_, err := tx.Exec(ctx, insertQuery, valueArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk insert songs: %w", err)
	}

	// Query back all song IDs by file_path
	selectQuery := `SELECT id, file_path FROM songs WHERE file_path = ANY($1)`
	rows, err := tx.Query(ctx, selectQuery, filePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to query song IDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var filePath string
		if err := rows.Scan(&id, &filePath); err != nil {
			continue
		}
		result[filePath] = id
	}

	return result, nil
}

// bulkStoreLyrics stores all lyrics for multiple songs in one bulk operation
func (bms *BatchMetadataService) bulkStoreLyrics(ctx context.Context, tx pgx.Tx, songLyricsMap map[int][]ExtractedLyrics) error {
	if len(songLyricsMap) == 0 {
		return nil
	}

	// Collect all song IDs for bulk delete
	songIDs := make([]int, 0, len(songLyricsMap))
	for songID := range songLyricsMap {
		songIDs = append(songIDs, songID)
	}

	// Bulk delete existing lyrics for all songs
	deleteQuery := "DELETE FROM lyrics WHERE song_id = ANY($1)"
	_, err := tx.Exec(ctx, deleteQuery, songIDs)
	if err != nil {
		return fmt.Errorf("failed to bulk delete existing lyrics: %w", err)
	}

	// Count total lyrics for capacity
	totalLyrics := 0
	for _, lyrics := range songLyricsMap {
		totalLyrics += len(lyrics)
	}

	log.Printf("💾 Bulk inserting %d lyrics entries for %d songs", totalLyrics, len(songLyricsMap))

	// Build bulk insert query for ALL lyrics
	valueStrings := make([]string, 0, totalLyrics)
	valueArgs := make([]interface{}, 0, totalLyrics*5)
	paramIdx := 1

	for songID, lyrics := range songLyricsMap {
		for _, lyric := range lyrics {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, NOW())",
				paramIdx, paramIdx+1, paramIdx+2, paramIdx+3, paramIdx+4))
			valueArgs = append(valueArgs, songID, lyric.Content, lyric.Type, lyric.Source, lyric.Language)
			paramIdx += 5
		}
	}

	insertQuery := fmt.Sprintf(`
		INSERT INTO lyrics (song_id, content, type, source, language, created_at)
		VALUES %s
	`, strings.Join(valueStrings, ", "))

	_, err = tx.Exec(ctx, insertQuery, valueArgs...)
	if err != nil {
		return fmt.Errorf("failed to bulk insert lyrics: %w", err)
	}

	log.Printf("✅ Successfully bulk inserted %d lyrics entries", totalLyrics)
	return nil
}
