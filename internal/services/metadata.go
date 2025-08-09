package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/jackc/pgx/v5"
	"gopkg.in/vansante/go-ffprobe.v2"
	"korus/internal/database"
	"korus/internal/models"
)

type MetadataService struct {
	db             *database.DB
	coverExtractor *CoverExtractor
}

type ExtractedMetadata struct {
	Title        string
	Artist       string
	Album        string
	AlbumArtist  string
	Year         int
	TrackNumber  int
	DiscNumber   int
	Duration     int // in seconds
	Bitrate      int
	Format       string
	FilePath     string
	FileSize     int64
	FileModified time.Time
	CoverURL     string // URL for cover image
}

func NewMetadataService(db *database.DB) *MetadataService {
	coverExtractor := NewCoverExtractor("./static/covers")
	return &MetadataService{
		db:             db,
		coverExtractor: coverExtractor,
	}
}

func (ms *MetadataService) ExtractAndStoreMetadata(ctx context.Context, filePath string) (*models.Song, error) {
	// Extract metadata from file
	metadata, err := ms.extractMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata from %s: %w", filePath, err)
	}

	// Store in database
	song, err := ms.storeMetadata(ctx, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to store metadata for %s: %w", filePath, err)
	}

	return song, nil
}

func (ms *MetadataService) extractMetadata(filePath string) (*ExtractedMetadata, error) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Open file for metadata reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read metadata tags
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		// If we can't read tags, we still need accurate duration data
		return ms.fallbackMetadata(filePath, info)
	}

	// Extract duration and bitrate (format-specific)
	duration, bitrate, format, err := ms.extractAudioProperties(filePath, file)
	if err != nil {
		return nil, fmt.Errorf("failed to extract audio properties: %w", err)
	}

	// Build metadata struct
	result := &ExtractedMetadata{
		Title:        metadata.Title(),
		Artist:       metadata.Artist(),
		Album:        metadata.Album(),
		AlbumArtist:  metadata.AlbumArtist(),
		Year:         metadata.Year(),
		TrackNumber:  extractTrackNumber(metadata.Track()),
		DiscNumber:   extractDiscNumber(metadata.Disc()),
		Duration:     duration,
		Bitrate:      bitrate,
		Format:       format,
		FilePath:     filePath,
		FileSize:     info.Size(),
		FileModified: info.ModTime(),
	}

	// Use fallback values if tags are empty
	if result.Title == "" {
		result.Title = ms.filenameWithoutExt(filePath)
	}
	if result.Artist == "" {
		result.Artist = "Unknown Artist"
	}
	if result.Album == "" {
		result.Album = "Unknown Album"
	}
	if result.AlbumArtist == "" {
		result.AlbumArtist = result.Artist
	}
	if result.DiscNumber == 0 {
		result.DiscNumber = 1
	}

	// Extract cover art (try multiple methods in order of preference)
	result.CoverURL = ms.extractCoverArt(filePath)

	return result, nil
}

func (ms *MetadataService) storeMetadata(ctx context.Context, metadata *ExtractedMetadata) (*models.Song, error) {
	var result *models.Song
	err := ms.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Find or create artist
		artistID, err := ms.findOrCreateArtist(ctx, tx, metadata.Artist)
		if err != nil {
			return fmt.Errorf("failed to find/create artist: %w", err)
		}

		// Find or create album artist (if different)
		var albumArtistID *int
		if metadata.AlbumArtist != "" && metadata.AlbumArtist != metadata.Artist {
			id, err := ms.findOrCreateArtist(ctx, tx, metadata.AlbumArtist)
			if err != nil {
				return fmt.Errorf("failed to find/create album artist: %w", err)
			}
			albumArtistID = &id
		} else {
			albumArtistID = &artistID
		}

		// Find or create album
		albumID, err := ms.findOrCreateAlbum(ctx, tx, metadata.Album, artistID, albumArtistID, metadata.Year, metadata.FilePath, metadata.CoverURL)
		if err != nil {
			return fmt.Errorf("failed to find/create album: %w", err)
		}

		// Insert or update song
		song, err := ms.insertOrUpdateSong(ctx, tx, metadata, artistID, albumID)
		if err != nil {
			return fmt.Errorf("failed to insert/update song: %w", err)
		}

		result = song
		return nil
	})

	return result, err
}

func (ms *MetadataService) findOrCreateArtist(ctx context.Context, tx pgx.Tx, name string) (int, error) {
	// Atomic upsert - the correct way to handle concurrent inserts
	sortName := generateSortName(name)
	query := `
		INSERT INTO artists (name, sort_name) 
		VALUES ($1, $2) 
		ON CONFLICT (LOWER(name))
		DO UPDATE SET sort_name = EXCLUDED.sort_name
		RETURNING id
	`

	var artistID int
	err := tx.QueryRow(ctx, query, name, sortName).Scan(&artistID)
	if err != nil {
		return 0, fmt.Errorf("failed to find/create artist: %w", err)
	}

	return artistID, nil
}

func (ms *MetadataService) findOrCreateAlbum(ctx context.Context, tx pgx.Tx, name string, artistID int, albumArtistID *int, year int, songFilePath string, songCoverURL string) (int, error) {
	// Atomic upsert for albums - handles concurrent inserts properly
	var yearPtr *int
	if year > 0 {
		yearPtr = &year
	}

	// Extract album cover (prefer external covers, fallback to song's embedded cover)
	albumCoverURL := ""
	if coverURL, err := ms.coverExtractor.ScanForExternalCover(songFilePath); err == nil {
		albumCoverURL = coverURL
	} else if songCoverURL != "" {
		// Fallback: use the song's cover for the album if no external cover found
		albumCoverURL = songCoverURL
	}

	query := `
		INSERT INTO albums (name, artist_id, album_artist_id, year, cover_path, date_added) 
		VALUES ($1, $2, $3, $4, $5, NOW()) 
		ON CONFLICT (LOWER(name), artist_id)
		DO UPDATE SET 
			album_artist_id = EXCLUDED.album_artist_id, 
			year = EXCLUDED.year,
			cover_path = COALESCE(EXCLUDED.cover_path, albums.cover_path)
		RETURNING id
	`

	var albumID int
	err := tx.QueryRow(ctx, query, name, artistID, albumArtistID, yearPtr, nullString(albumCoverURL)).Scan(&albumID)
	if err != nil {
		return 0, fmt.Errorf("failed to find/create album: %w", err)
	}

	return albumID, nil
}

func (ms *MetadataService) insertOrUpdateSong(ctx context.Context, tx pgx.Tx, metadata *ExtractedMetadata, artistID, albumID int) (*models.Song, error) {
	// Check if song already exists
	var existingID int
	checkQuery := "SELECT id FROM songs WHERE file_path = $1"
	err := tx.QueryRow(ctx, checkQuery, metadata.FilePath).Scan(&existingID)

	var song models.Song

	if err == pgx.ErrNoRows {
		// Insert new song
		insertQuery := `
			INSERT INTO songs (title, album_id, artist_id, track_number, disc_number, duration, 
							 file_path, file_size, file_modified, bitrate, format, cover_path, date_added) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW()) 
			RETURNING id, title, album_id, artist_id, track_number, disc_number, duration, 
					  file_path, file_size, file_modified, bitrate, format, cover_path, date_added
		`

		err = tx.QueryRow(ctx, insertQuery,
			metadata.Title, albumID, artistID,
			nullInt(metadata.TrackNumber), metadata.DiscNumber, metadata.Duration,
			metadata.FilePath, metadata.FileSize, metadata.FileModified,
			nullInt(metadata.Bitrate), nullString(metadata.Format), nullString(metadata.CoverURL)).
			Scan(&song.ID, &song.Title, &song.AlbumID, &song.ArtistID,
				&song.TrackNumber, &song.DiscNumber, &song.Duration,
				&song.FilePath, &song.FileSize, &song.FileModified,
				&song.Bitrate, &song.Format, &song.CoverPath, &song.DateAdded)
	} else if err == nil {
		// Update existing song
		updateQuery := `
			UPDATE songs 
			SET title = $2, album_id = $3, artist_id = $4, track_number = $5, 
				disc_number = $6, duration = $7, file_size = $8, file_modified = $9, 
				bitrate = $10, format = $11, cover_path = $12
			WHERE id = $1 
			RETURNING id, title, album_id, artist_id, track_number, disc_number, duration, 
					  file_path, file_size, file_modified, bitrate, format, cover_path, date_added
		`

		err = tx.QueryRow(ctx, updateQuery, existingID,
			metadata.Title, albumID, artistID,
			nullInt(metadata.TrackNumber), metadata.DiscNumber, metadata.Duration,
			metadata.FileSize, metadata.FileModified,
			nullInt(metadata.Bitrate), nullString(metadata.Format), nullString(metadata.CoverURL)).
			Scan(&song.ID, &song.Title, &song.AlbumID, &song.ArtistID,
				&song.TrackNumber, &song.DiscNumber, &song.Duration,
				&song.FilePath, &song.FileSize, &song.FileModified,
				&song.Bitrate, &song.Format, &song.CoverPath, &song.DateAdded)
	} else {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return &song, nil
}

func (ms *MetadataService) extractAudioProperties(filePath string, file *os.File) (duration, bitrate int, format string, err error) {
	// Get format from file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if len(ext) > 1 {
		format = ext[1:] // Remove the dot
	}

	// Use FFprobe to get accurate audio properties
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	probeData, err := ffprobe.ProbeURL(ctx, filePath)
	if err != nil {
		return 0, 0, format, fmt.Errorf("failed to probe audio file: %w", err)
	}

	// Find the first audio stream
	audioStream := probeData.FirstAudioStream()
	if audioStream == nil {
		return 0, 0, format, fmt.Errorf("no audio stream found in file")
	}

	// Extract duration (convert from float seconds to int)
	if probeData.Format.DurationSeconds != 0 {
		duration = int(math.Round(probeData.Format.DurationSeconds))
	} else {
		return 0, 0, format, fmt.Errorf("no duration information available")
	}

	// Extract bitrate
	if audioStream.BitRate != "" {
		if parsedBitrate := parseInt(audioStream.BitRate); parsedBitrate > 0 {
			bitrate = parsedBitrate / 1000 // Convert from bps to kbps
		}
	}

	// Use codec name if available, otherwise fall back to file extension
	if audioStream.CodecName != "" {
		format = audioStream.CodecName
	}

	return duration, bitrate, format, nil
}

func (ms *MetadataService) fallbackMetadata(filePath string, info os.FileInfo) (*ExtractedMetadata, error) {
	// Create basic metadata from filename and path
	filename := ms.filenameWithoutExt(filePath)

	// Even for fallback metadata, we require accurate duration data
	format := strings.ToLower(filepath.Ext(filePath)[1:])
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	probeData, err := ffprobe.ProbeURL(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to probe audio file for fallback metadata: %w", err)
	}

	audioStream := probeData.FirstAudioStream()
	if audioStream == nil {
		return nil, fmt.Errorf("no audio stream found in file for fallback metadata")
	}

	var duration, bitrate int
	if probeData.Format.DurationSeconds != 0 {
		duration = int(math.Round(probeData.Format.DurationSeconds))
	} else {
		return nil, fmt.Errorf("no duration information available for fallback metadata")
	}

	// Extract bitrate
	if audioStream.BitRate != "" {
		if parsedBitrate := parseInt(audioStream.BitRate); parsedBitrate > 0 {
			bitrate = parsedBitrate / 1000 // Convert from bps to kbps
		}
	}

	// Use codec name if available, otherwise fall back to file extension
	if audioStream.CodecName != "" {
		format = audioStream.CodecName
	}

	return &ExtractedMetadata{
		Title:        filename,
		Artist:       "Unknown Artist",
		Album:        "Unknown Album",
		AlbumArtist:  "Unknown Artist",
		Year:         0,
		TrackNumber:  0,
		DiscNumber:   1,
		Duration:     duration,
		Bitrate:      bitrate,
		Format:       format,
		FilePath:     filePath,
		FileSize:     info.Size(),
		FileModified: info.ModTime(),
	}, nil
}

func (ms *MetadataService) filenameWithoutExt(filePath string) string {
	filename := filepath.Base(filePath)
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// Helper functions

func extractTrackNumber(track int, _ int) int {
	return track
}

func extractDiscNumber(disc int, _ int) int {
	if disc == 0 {
		return 1
	}
	return disc
}

func generateSortName(name string) string {
	// Remove articles from the beginning for sorting
	lower := strings.ToLower(strings.TrimSpace(name))
	articles := []string{"the ", "a ", "an "}

	for _, article := range articles {
		if strings.HasPrefix(lower, article) {
			return strings.TrimSpace(name[len(article):]) + ", " + strings.Title(article[:len(article)-1])
		}
	}

	return name
}

func nullInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func nullString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	// Simple integer parsing - handles basic numeric strings
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// extractCoverArt tries multiple methods to find cover art for a song
func (ms *MetadataService) extractCoverArt(filePath string) string {
	log.Printf("🖼️  Extracting cover art for: %s", filePath)

	// Try 1: Song-specific cover (highest priority)
	if coverURL, err := ms.coverExtractor.ScanForSongSpecificCover(filePath); err == nil {
		log.Printf("✅ Found song-specific cover: %s", coverURL)
		return coverURL
	} else {
		log.Printf("❌ No song-specific cover found: %v", err)
	}

	// Try 2: Embedded cover art
	if coverURL, err := ms.coverExtractor.ExtractEmbeddedCover(filePath); err == nil {
		log.Printf("✅ Found embedded cover: %s", coverURL)
		return coverURL
	} else {
		log.Printf("❌ No embedded cover found: %v", err)
	}

	// Try 3: External cover in same directory (folder.jpg, cover.png, etc.)
	if coverURL, err := ms.coverExtractor.ScanForExternalCover(filePath); err == nil {
		log.Printf("✅ Found external cover: %s", coverURL)
		return coverURL
	} else {
		log.Printf("❌ No external cover found: %v", err)
	}

	log.Printf("❌ No cover art found for: %s", filePath)
	// No cover found
	return ""
}
