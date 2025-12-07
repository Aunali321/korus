package services

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

type CoverExtractor struct {
	coversDir string
}

func NewCoverExtractor(coversDir string) *CoverExtractor {
	return &CoverExtractor{
		coversDir: coversDir,
	}
}

// ExtractEmbeddedCover extracts cover art embedded in audio file
func (ce *CoverExtractor) ExtractEmbeddedCover(filePath string) (string, error) {
	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read metadata
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return "", fmt.Errorf("failed to read metadata: %w", err)
	}

	// Get embedded picture
	picture := metadata.Picture()
	if picture == nil {
		return "", fmt.Errorf("no embedded cover art found")
	}

	// Ensure covers directory exists
	if err := os.MkdirAll(ce.coversDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create covers directory: %w", err)
	}

	// Generate unique filename based on content hash
	hash := md5.Sum(picture.Data)
	ext := ce.getImageExtension(picture.MIMEType)
	filename := fmt.Sprintf("%x%s", hash, ext)
	coverPath := filepath.Join(ce.coversDir, filename)

	// Check if file already exists
	if _, err := os.Stat(coverPath); err == nil {
		// File already exists, return relative path for URL
		return ce.getRelativeURL(filename), nil
	}

	// Write cover image to disk
	if err := os.WriteFile(coverPath, picture.Data, 0644); err != nil {
		return "", fmt.Errorf("failed to write cover file: %w", err)
	}

	return ce.getRelativeURL(filename), nil
}

// ScanForExternalCover looks for common cover art files in the same directory as the audio file
func (ce *CoverExtractor) ScanForExternalCover(audioFilePath string) (string, error) {
	dir := filepath.Dir(audioFilePath)

	// Common cover art filenames (in order of preference)
	coverNames := []string{
		"cover", "folder", "front", "albumart", "album",
	}

	// Supported image extensions including WebP
	extensions := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

	// Search for cover files
	for _, name := range coverNames {
		for _, ext := range extensions {
			coverFile := filepath.Join(dir, name+ext)
			if _, err := os.Stat(coverFile); err == nil {
				return ce.copyCoverToStatic(coverFile)
			}
		}
	}

	return "", fmt.Errorf("no external cover art found")
}

// ScanForSongSpecificCover looks for cover art specific to a song
func (ce *CoverExtractor) ScanForSongSpecificCover(audioFilePath string) (string, error) {
	dir := filepath.Dir(audioFilePath)
	baseName := ce.getFileBaseName(audioFilePath)

	// Supported image extensions including WebP
	extensions := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

	// Look for song-specific cover (same name as audio file)
	for _, ext := range extensions {
		coverFile := filepath.Join(dir, baseName+ext)
		if _, err := os.Stat(coverFile); err == nil {
			return ce.copyCoverToStatic(coverFile)
		}
	}

	return "", fmt.Errorf("no song-specific cover art found")
}

// copyCoverToStatic copies an external cover file to the static covers directory
func (ce *CoverExtractor) copyCoverToStatic(coverFilePath string) (string, error) {
	// Read the cover file
	coverData, err := os.ReadFile(coverFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read cover file: %w", err)
	}

	// Validate that it's a valid image
	if !ce.isValidImageData(coverData) {
		return "", fmt.Errorf("invalid image data")
	}

	// Ensure covers directory exists
	if err := os.MkdirAll(ce.coversDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create covers directory: %w", err)
	}

	// Generate unique filename based on content hash
	hash := md5.Sum(coverData)
	ext := strings.ToLower(filepath.Ext(coverFilePath))
	filename := fmt.Sprintf("%x%s", hash, ext)
	staticCoverPath := filepath.Join(ce.coversDir, filename)

	// Check if file already exists
	if _, err := os.Stat(staticCoverPath); err == nil {
		// File already exists, return relative path
		return ce.getRelativeURL(filename), nil
	}

	// Copy to static directory
	if err := os.WriteFile(staticCoverPath, coverData, 0644); err != nil {
		return "", fmt.Errorf("failed to write cover to static directory: %w", err)
	}

	return ce.getRelativeURL(filename), nil
}

// getImageExtension returns appropriate file extension based on MIME type
func (ce *CoverExtractor) getImageExtension(mimeType string) string {
	switch strings.ToLower(mimeType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg" // Default fallback
	}
}

// isValidImageData performs basic validation to check if data is a valid image
func (ce *CoverExtractor) isValidImageData(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// Check for common image file signatures
	// JPEG: FF D8
	if data[0] == 0xFF && data[1] == 0xD8 {
		return true
	}

	// PNG: 89 50 4E 47
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true
	}

	// WebP: 52 49 46 46 (RIFF) ... 57 45 42 50 (WEBP)
	if len(data) >= 12 &&
		data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return true
	}

	// GIF87a or GIF89a
	if len(data) >= 6 &&
		((data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 && data[4] == 0x37 && data[5] == 0x61) ||
			(data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 && data[4] == 0x39 && data[5] == 0x61)) {
		return true
	}

	return false
}

// getFileBaseName returns filename without extension
func (ce *CoverExtractor) getFileBaseName(filePath string) string {
	base := filepath.Base(filePath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// getRelativeURL converts a filename to a relative URL for serving
func (ce *CoverExtractor) getRelativeURL(filename string) string {
	return "/covers/" + filename
}
