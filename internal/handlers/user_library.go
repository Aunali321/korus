package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"korus/internal/middleware"
	"korus/internal/services"
)

type UserLibraryHandler struct {
	userLibraryService *services.UserLibraryService
}

func NewUserLibraryHandler(userLibraryService *services.UserLibraryService) *UserLibraryHandler {
	return &UserLibraryHandler{
		userLibraryService: userLibraryService,
	}
}

type LikeSongsRequest struct {
	SongIDs []int `json:"songIds" binding:"required"`
}

// GetLikedSongs returns user's liked songs
func (h *UserLibraryHandler) GetLikedSongs(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	// Parse query parameters
	limit := parseIntParam(c, "limit", 50)
	offset := parseIntParam(c, "offset", 0)
	sort := c.DefaultQuery("sort", "liked_at")

	songs, err := h.userLibraryService.GetLikedSongs(c.Request.Context(), user.ID, limit, offset, sort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get liked songs",
		})
		return
	}

	c.JSON(http.StatusOK, songs)
}

// GetLikedAlbums returns user's liked albums
func (h *UserLibraryHandler) GetLikedAlbums(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	// Parse query parameters
	limit := parseIntParam(c, "limit", 50)
	offset := parseIntParam(c, "offset", 0)

	albums, err := h.userLibraryService.GetLikedAlbums(c.Request.Context(), user.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get liked albums",
		})
		return
	}

	c.JSON(http.StatusOK, albums)
}

// GetFollowedArtists returns user's followed artists
func (h *UserLibraryHandler) GetFollowedArtists(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	// Parse query parameters
	limit := parseIntParam(c, "limit", 50)
	offset := parseIntParam(c, "offset", 0)

	artists, err := h.userLibraryService.GetFollowedArtists(c.Request.Context(), user.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get followed artists",
		})
		return
	}

	c.JSON(http.StatusOK, artists)
}

// LikeSongs adds songs to user's liked songs
func (h *UserLibraryHandler) LikeSongs(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	var req LikeSongsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if len(req.SongIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "empty_request",
			"message": "No songs provided",
		})
		return
	}

	err := h.userLibraryService.LikeSongs(c.Request.Context(), user.ID, req.SongIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to like songs",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// UnlikeSongs removes songs from user's liked songs
func (h *UserLibraryHandler) UnlikeSongs(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	var req LikeSongsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	if len(req.SongIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "empty_request",
			"message": "No songs provided",
		})
		return
	}

	err := h.userLibraryService.UnlikeSongs(c.Request.Context(), user.ID, req.SongIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to unlike songs",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// LikeAlbum adds an album to user's liked albums
func (h *UserLibraryHandler) LikeAlbum(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid album ID",
		})
		return
	}

	err = h.userLibraryService.LikeAlbum(c.Request.Context(), user.ID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to like album",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// UnlikeAlbum removes an album from user's liked albums
func (h *UserLibraryHandler) UnlikeAlbum(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid album ID",
		})
		return
	}

	err = h.userLibraryService.UnlikeAlbum(c.Request.Context(), user.ID, id)
	if err != nil {
		if err.Error() == "album not found in user's liked albums" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Album not found in liked albums",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to unlike album",
			})
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// FollowArtist adds an artist to user's followed artists
func (h *UserLibraryHandler) FollowArtist(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid artist ID",
		})
		return
	}

	err = h.userLibraryService.FollowArtist(c.Request.Context(), user.ID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to follow artist",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// UnfollowArtist removes an artist from user's followed artists
func (h *UserLibraryHandler) UnfollowArtist(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid artist ID",
		})
		return
	}

	err = h.userLibraryService.UnfollowArtist(c.Request.Context(), user.ID, id)
	if err != nil {
		if err.Error() == "artist not found in user's followed artists" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Artist not found in followed artists",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to unfollow artist",
			})
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}