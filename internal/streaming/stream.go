package streaming

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"korus/internal/services"
)

type StreamingService struct {
	libraryService *services.LibraryService
}

type rangeSpec struct {
	start int64
	end   int64
}

func NewStreamingService(libraryService *services.LibraryService) *StreamingService {
	return &StreamingService{
		libraryService: libraryService,
	}
}

func (s *StreamingService) StreamSong(c *gin.Context) {
	songID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid song ID",
		})
		return
	}

	// Get song from database
	song, err := s.libraryService.GetSong(c.Request.Context(), songID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Song not found",
		})
		return
	}

	// Check if file exists
	if _, err := os.Stat(song.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "file_not_found",
			"message": "Audio file not found on disk",
		})
		return
	}

	// Open file
	file, err := os.Open(song.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "file_error",
			"message": "Failed to open audio file",
		})
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "file_error",
			"message": "Failed to get file information",
		})
		return
	}

	fileSize := fileInfo.Size()
	lastModified := fileInfo.ModTime()

	// Set content type based on file extension
	contentType := getContentType(song.FilePath)

	// Set basic headers
	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", "bytes")
	c.Header("Last-Modified", lastModified.Format(http.TimeFormat))
	c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year

	// Handle conditional requests
	if checkNotModified(c, lastModified) {
		c.Status(http.StatusNotModified)
		return
	}

	// Parse Range header if present
	rangeHeader := c.GetHeader("Range")
	if rangeHeader == "" {
		// No range request, serve entire file
		c.Header("Content-Length", strconv.FormatInt(fileSize, 10))
		c.Status(http.StatusOK)
		io.Copy(c.Writer, file)
		return
	}

	// Parse range specification
	ranges, err := parseRangeHeader(rangeHeader, fileSize)
	if err != nil {
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if len(ranges) != 1 {
		// Multiple ranges not supported for now
		c.Header("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Handle single range request
	r := ranges[0]
	contentLength := r.end - r.start + 1

	// Set partial content headers
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", r.start, r.end, fileSize))
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Status(http.StatusPartialContent)

	// Seek to start position
	if _, err := file.Seek(r.start, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "seek_error",
			"message": "Failed to seek in audio file",
		})
		return
	}

	// Copy the requested range
	io.CopyN(c.Writer, file, contentLength)
}

func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".m4a", ".aac":
		return "audio/mp4"
	case ".ogg":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	case ".webm":
		return "audio/webm"
	default:
		return "application/octet-stream"
	}
}

func checkNotModified(c *gin.Context, lastModified time.Time) bool {
	// Check If-Modified-Since header
	if modSince := c.GetHeader("If-Modified-Since"); modSince != "" {
		if t, err := time.Parse(http.TimeFormat, modSince); err == nil {
			// Truncate to seconds for comparison
			if lastModified.Truncate(time.Second).Equal(t.Truncate(time.Second)) || lastModified.Before(t) {
				return true
			}
		}
	}

	// Check If-Unmodified-Since header
	if unmodSince := c.GetHeader("If-Unmodified-Since"); unmodSince != "" {
		if t, err := time.Parse(http.TimeFormat, unmodSince); err == nil {
			if lastModified.After(t) {
				c.Status(http.StatusPreconditionFailed)
				return true
			}
		}
	}

	return false
}

func parseRangeHeader(rangeHeader string, fileSize int64) ([]rangeSpec, error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil, fmt.Errorf("unsupported range unit")
	}

	rangeSpecStr := strings.TrimPrefix(rangeHeader, "bytes=")
	ranges := strings.Split(rangeSpecStr, ",")

	var parsedRanges []rangeSpec

	for _, r := range ranges {
		r = strings.TrimSpace(r)

		if strings.HasPrefix(r, "-") {
			// Suffix range: -500 means last 500 bytes
			suffixLength, err := strconv.ParseInt(r[1:], 10, 64)
			if err != nil || suffixLength <= 0 || suffixLength > fileSize {
				return nil, fmt.Errorf("invalid suffix range")
			}

			start := fileSize - suffixLength
			if start < 0 {
				start = 0
			}

			parsedRanges = append(parsedRanges, rangeSpec{
				start: start,
				end:   fileSize - 1,
			})
		} else if strings.HasSuffix(r, "-") {
			// Prefix range: 500- means from byte 500 to end
			start, err := strconv.ParseInt(r[:len(r)-1], 10, 64)
			if err != nil || start < 0 || start >= fileSize {
				return nil, fmt.Errorf("invalid prefix range")
			}

			parsedRanges = append(parsedRanges, rangeSpec{
				start: start,
				end:   fileSize - 1,
			})
		} else {
			// Full range: 200-1023
			parts := strings.Split(r, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid range format")
			}

			start, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil || start < 0 {
				return nil, fmt.Errorf("invalid range start")
			}

			end, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil || end < start || end >= fileSize {
				return nil, fmt.Errorf("invalid range end")
			}

			parsedRanges = append(parsedRanges, rangeSpec{
				start: start,
				end:   end,
			})
		}
	}

	if len(parsedRanges) == 0 {
		return nil, fmt.Errorf("no valid ranges")
	}

	return parsedRanges, nil
}
