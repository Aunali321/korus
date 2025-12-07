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

	"korus/internal/database"
	"korus/internal/models"

	"github.com/dhowden/tag"
	"github.com/jackc/pgx/v5"
	"gopkg.in/vansante/go-ffprobe.v2"
)

type MetadataService struct {
	db             *database.DB
	coverExtractor *CoverExtractor
	lyricsEnabled  bool
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
	CoverURL     string            // URL for cover image
	Lyrics       []ExtractedLyrics // Extracted lyrics data
}

type ExtractedLyrics struct {
	Content  string
	Type     string // "synced" or "unsynced"
	Source   string // "embedded", "external_lrc", "external_txt"
	Language string // ISO 639-2 language codes
}

func NewMetadataService(db *database.DB, lyricsEnabled bool) *MetadataService {
	coverExtractor := NewCoverExtractor("./static/covers")
	return &MetadataService{
		db:             db,
		coverExtractor: coverExtractor,
		lyricsEnabled:  lyricsEnabled,
	}
}

func (ms *MetadataService) ExtractAndStoreMetadata(ctx context.Context, filePath string) (*models.Song, error) {
	metadata, err := ms.extractMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata from %s: %w", filePath, err)
	}

	song, err := ms.storeMetadata(ctx, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to store metadata for %s: %w", filePath, err)
	}

	return song, nil
}

func (ms *MetadataService) extractMetadata(filePath string) (*ExtractedMetadata, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		// If we can't read tags, we still need accurate duration data
		return ms.fallbackMetadata(filePath, info)
	}

	duration, bitrate, format, err := ms.extractAudioProperties(filePath, file)
	if err != nil {
		return nil, fmt.Errorf("failed to extract audio properties: %w", err)
	}

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

	result.CoverURL = ms.extractCoverArt(filePath)

	if ms.lyricsEnabled {
		result.Lyrics = ms.extractLyrics(filePath, metadata)
	}

	return result, nil
}

func (ms *MetadataService) storeMetadata(ctx context.Context, metadata *ExtractedMetadata) (*models.Song, error) {
	var result *models.Song
	err := ms.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		artistID, err := ms.findOrCreateArtist(ctx, tx, metadata.Artist)
		if err != nil {
			return fmt.Errorf("failed to find/create artist: %w", err)
		}

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

		albumID, err := ms.findOrCreateAlbum(ctx, tx, metadata.Album, artistID, albumArtistID, metadata.Year, metadata.FilePath, metadata.CoverURL)
		if err != nil {
			return fmt.Errorf("failed to find/create album: %w", err)
		}

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
	var yearPtr *int
	if year > 0 {
		yearPtr = &year
	}

	albumCoverURL := ""
	if coverURL, err := ms.coverExtractor.ScanForExternalCover(songFilePath); err == nil {
		albumCoverURL = coverURL
	} else if songCoverURL != "" {
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
	var existingID int
	checkQuery := "SELECT id FROM songs WHERE file_path = $1"
	err := tx.QueryRow(ctx, checkQuery, metadata.FilePath).Scan(&existingID)

	var song models.Song

	if err == pgx.ErrNoRows {
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

	if len(metadata.Lyrics) > 0 {
		log.Printf("💾 Storing %d lyrics entries for song ID %d", len(metadata.Lyrics), song.ID)
		err = ms.storeLyrics(ctx, tx, song.ID, metadata.Lyrics)
		if err != nil {
			return nil, fmt.Errorf("failed to store lyrics: %w", err)
		}
		log.Printf("✅ Successfully stored lyrics for song ID %d", song.ID)
	} else {
		log.Printf("❌ No lyrics to store for song ID %d (metadata.Lyrics is empty)", song.ID)
	}

	return &song, nil
}

func (ms *MetadataService) storeLyrics(ctx context.Context, tx pgx.Tx, songID int, lyrics []ExtractedLyrics) error {
	if len(lyrics) == 0 {
		return nil
	}

	deleteQuery := "DELETE FROM lyrics WHERE song_id = $1"
	_, err := tx.Exec(ctx, deleteQuery, songID)
	if err != nil {
		return fmt.Errorf("failed to delete existing lyrics: %w", err)
	}

	valueStrings := make([]string, 0, len(lyrics))
	valueArgs := make([]interface{}, 0, len(lyrics)*5)
	paramIdx := 1

	for _, lyric := range lyrics {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, NOW())",
			paramIdx, paramIdx+1, paramIdx+2, paramIdx+3, paramIdx+4))
		valueArgs = append(valueArgs, songID, lyric.Content, lyric.Type, lyric.Source, lyric.Language)
		paramIdx += 5
	}

	insertQuery := fmt.Sprintf(`
		INSERT INTO lyrics (song_id, content, type, source, language, created_at)
		VALUES %s
	`, strings.Join(valueStrings, ", "))

	_, err = tx.Exec(ctx, insertQuery, valueArgs...)
	if err != nil {
		return fmt.Errorf("failed to bulk insert lyrics: %w", err)
	}

	return nil
}

func (ms *MetadataService) extractAudioProperties(filePath string, file *os.File) (duration, bitrate int, format string, err error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if len(ext) > 1 {
		format = ext[1:]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	probeData, err := ffprobe.ProbeURL(ctx, filePath)
	if err != nil {
		return 0, 0, format, fmt.Errorf("failed to probe audio file: %w", err)
	}

	audioStream := probeData.FirstAudioStream()
	if audioStream == nil {
		return 0, 0, format, fmt.Errorf("no audio stream found in file")
	}

	if probeData.Format.DurationSeconds != 0 {
		duration = int(math.Round(probeData.Format.DurationSeconds))
	} else {
		return 0, 0, format, fmt.Errorf("no duration information available")
	}

	if audioStream.BitRate != "" {
		if parsedBitrate := parseInt(audioStream.BitRate); parsedBitrate > 0 {
			bitrate = parsedBitrate / 1000
		}
	}

	if audioStream.CodecName != "" {
		format = audioStream.CodecName
	}

	return duration, bitrate, format, nil
}

func (ms *MetadataService) fallbackMetadata(filePath string, info os.FileInfo) (*ExtractedMetadata, error) {
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

	if audioStream.BitRate != "" {
		if parsedBitrate := parseInt(audioStream.BitRate); parsedBitrate > 0 {
			bitrate = parsedBitrate / 1000
		}
	}

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
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

func (ms *MetadataService) extractCoverArt(filePath string) string {
	log.Printf("🖼️  Extracting cover art for: %s", filePath)

	if coverURL, err := ms.coverExtractor.ScanForSongSpecificCover(filePath); err == nil {
		log.Printf("✅ Found song-specific cover: %s", coverURL)
		return coverURL
	} else {
		log.Printf("❌ No song-specific cover found: %v", err)
	}

	if coverURL, err := ms.coverExtractor.ExtractEmbeddedCover(filePath); err == nil {
		log.Printf("✅ Found embedded cover: %s", coverURL)
		return coverURL
	} else {
		log.Printf("❌ No embedded cover found: %v", err)
	}

	if coverURL, err := ms.coverExtractor.ScanForExternalCover(filePath); err == nil {
		log.Printf("✅ Found external cover: %s", coverURL)
		return coverURL
	} else {
		log.Printf("❌ No external cover found: %v", err)
	}

	log.Printf("❌ No cover art found for: %s", filePath)
	return ""
}

func (ms *MetadataService) extractLyrics(filePath string, metadata tag.Metadata) []ExtractedLyrics {
	log.Printf("🎵 Extracting lyrics for: %s", filePath)
	var lyrics []ExtractedLyrics

	if embeddedLyrics := ms.extractEmbeddedLyrics(metadata); embeddedLyrics != nil {
		lyrics = append(lyrics, *embeddedLyrics)
		log.Printf("✅ Found embedded lyrics")
	}

	if lrcLyrics := ms.extractLrcFileWithMetadata(filePath, metadata); lrcLyrics != nil {
		lyrics = append(lyrics, *lrcLyrics)
		log.Printf("✅ Found .lrc file")
	}

	if txtLyrics := ms.extractTxtFile(filePath); txtLyrics != nil {
		lyrics = append(lyrics, *txtLyrics)
		log.Printf("✅ Found .txt file")
	}

	if len(lyrics) == 0 {
		log.Printf("❌ No lyrics found for: %s", filePath)
	}

	return lyrics
}

func (ms *MetadataService) extractEmbeddedLyrics(metadata tag.Metadata) *ExtractedLyrics {
	if metadata == nil {
		return nil
	}

	lyricsText := metadata.Lyrics()
	if lyricsText == "" {
		return nil
	}

	return &ExtractedLyrics{
		Content:  lyricsText,
		Type:     "unsynced", // dhowden/tag extracts unsynchronized lyrics
		Source:   "embedded",
		Language: "eng", // Default to English, could be enhanced to detect language
	}
}

func (ms *MetadataService) extractLrcFileWithMetadata(audioFilePath string, songMetadata tag.Metadata) *ExtractedLyrics {
	baseName := strings.TrimSuffix(audioFilePath, filepath.Ext(audioFilePath))
	lrcPath := baseName + ".lrc"

	if _, err := os.Stat(lrcPath); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(lrcPath)
	if err != nil {
		log.Printf("Failed to open LRC file %s: %v", lrcPath, err)
		return nil
	}
	defer file.Close()

	parser := NewLRCParser()
	lrcDoc, err := parser.Parse(file)
	if err != nil {
		log.Printf("Failed to parse LRC file %s: %v", lrcPath, err)
		return nil
	}

	if len(lrcDoc.Lines) == 0 {
		log.Printf("No lyrics lines found in LRC file %s", lrcPath)
		return nil
	}

	if lrcDoc.Metadata.Title == "" && songMetadata != nil {
		lrcDoc.Metadata.Title = songMetadata.Title()
	}
	if lrcDoc.Metadata.Artist == "" && songMetadata != nil {
		lrcDoc.Metadata.Artist = songMetadata.Artist()
	}
	if lrcDoc.Metadata.Album == "" && songMetadata != nil {
		lrcDoc.Metadata.Album = songMetadata.Album()
	}
	if lrcDoc.Metadata.Language == "" {
		detectedLang := lrcDoc.detectLanguageFromContent()
		lrcDoc.Metadata.Language = detectedLang
	}

	jsonContent, err := lrcDoc.ToJSON()
	if err != nil {
		log.Printf("Failed to convert LRC to JSON %s: %v", lrcPath, err)
		return nil
	}

	return &ExtractedLyrics{
		Content:  jsonContent,
		Type:     "synced",
		Source:   "external_lrc",
		Language: lrcDoc.DetectLanguage(),
	}
}

func (ms *MetadataService) extractTxtFile(audioFilePath string) *ExtractedLyrics {
	baseName := strings.TrimSuffix(audioFilePath, filepath.Ext(audioFilePath))
	txtPath := baseName + ".txt"

	if _, err := os.Stat(txtPath); os.IsNotExist(err) {
		return nil
	}

	content, err := os.ReadFile(txtPath)
	if err != nil {
		log.Printf("Failed to read TXT file %s: %v", txtPath, err)
		return nil
	}

	if len(strings.TrimSpace(string(content))) == 0 {
		return nil
	}

	return &ExtractedLyrics{
		Content:  string(content),
		Type:     "unsynced",
		Source:   "external_txt",
		Language: "eng",
	}
}
