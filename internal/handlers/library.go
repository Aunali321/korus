package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"korus/internal/search"
	"korus/internal/services"
)

type LibraryHandler struct {
	libraryService *services.LibraryService
	searchService  *search.SearchService
}

func NewLibraryHandler(libraryService *services.LibraryService, searchService *search.SearchService) *LibraryHandler {
	return &LibraryHandler{
		libraryService: libraryService,
		searchService:  searchService,
	}
}

func (h *LibraryHandler) GetStats(c *gin.Context) {
	stats, err := h.libraryService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get library statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *LibraryHandler) GetArtists(c *gin.Context) {
	// Parse query parameters
	limit := parseIntParam(c, "limit", 50)
	offset := parseIntParam(c, "offset", 0)
	sort := c.DefaultQuery("sort", "name")

	artists, err := h.libraryService.GetArtists(c.Request.Context(), limit, offset, sort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get artists",
		})
		return
	}

	c.JSON(http.StatusOK, artists)
}

func (h *LibraryHandler) GetArtist(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid artist ID",
		})
		return
	}

	artist, err := h.libraryService.GetArtist(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Artist not found",
		})
		return
	}

	c.JSON(http.StatusOK, artist)
}

func (h *LibraryHandler) GetAlbums(c *gin.Context) {
	// Parse query parameters
	limit := parseIntParam(c, "limit", 50)
	offset := parseIntParam(c, "offset", 0)
	sort := c.DefaultQuery("sort", "artist")
	
	var year *int
	if yearStr := c.Query("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = &y
		}
	}

	albums, err := h.libraryService.GetAlbums(c.Request.Context(), limit, offset, sort, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get albums",
		})
		return
	}

	c.JSON(http.StatusOK, albums)
}

func (h *LibraryHandler) GetAlbum(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid album ID",
		})
		return
	}

	album, err := h.libraryService.GetAlbum(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Album not found",
		})
		return
	}

	c.JSON(http.StatusOK, album)
}

func (h *LibraryHandler) GetAlbumSongs(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid album ID",
		})
		return
	}

	// First check if album exists
	_, err = h.libraryService.GetAlbum(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Album not found",
		})
		return
	}

	songs, err := h.libraryService.GetAlbumSongs(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get album songs",
		})
		return
	}

	c.JSON(http.StatusOK, songs)
}

func (h *LibraryHandler) GetSongs(c *gin.Context) {
	idsParam := c.Query("ids")
	if idsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_ids",
			"message": "Song IDs are required",
		})
		return
	}

	// Parse comma-separated IDs
	idStrings := strings.Split(idsParam, ",")
	ids := make([]int, 0, len(idStrings))
	
	for _, idStr := range idStrings {
		if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_ids",
			"message": "No valid song IDs provided",
		})
		return
	}

	songs, err := h.libraryService.GetSongs(c.Request.Context(), ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get songs",
		})
		return
	}

	// Return 404 if no songs were found for any of the requested IDs
	if len(songs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "No songs found for the requested IDs",
		})
		return
	}

	c.JSON(http.StatusOK, songs)
}

func (h *LibraryHandler) GetSong(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid song ID",
		})
		return
	}

	song, err := h.libraryService.GetSong(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Song not found",
		})
		return
	}

	c.JSON(http.StatusOK, song)
}

func (h *LibraryHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_query",
			"message": "Search query is required",
		})
		return
	}

	// Parse search parameters
	searchType := c.Query("type") // "song", "album", "artist", or empty for all
	limit := parseIntParam(c, "limit", 20)
	offset := parseIntParam(c, "offset", 0)

	options := search.SearchOptions{
		Query:  query,
		Type:   searchType,
		Limit:  limit,
		Offset: offset,
	}

	results, err := h.searchService.Search(c.Request.Context(), options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "search_failed",
			"message": "Search request failed",
		})
		return
	}

	c.JSON(http.StatusOK, results)
}

// Helper function to parse integer parameters with defaults
func parseIntParam(c *gin.Context, key string, defaultValue int) int {
	if str := c.Query(key); str != "" {
		if val, err := strconv.Atoi(str); err == nil && val >= 0 {
			return val
		}
	}
	return defaultValue
}