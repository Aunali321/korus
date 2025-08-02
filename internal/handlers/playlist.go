package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"korus/internal/middleware"
	"korus/internal/services"
)

type PlaylistHandler struct {
	playlistService *services.PlaylistService
}

func NewPlaylistHandler(playlistService *services.PlaylistService) *PlaylistHandler {
	return &PlaylistHandler{
		playlistService: playlistService,
	}
}

type CreatePlaylistRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type UpdatePlaylistRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type AddSongsRequest struct {
	SongIDs  []int `json:"songIds" binding:"required"`
	Position *int  `json:"position,omitempty"`
}

type RemoveSongsRequest struct {
	SongIDs []int `json:"songIds" binding:"required"`
}

type ReorderSongsRequest struct {
	SongIDs []int `json:"songIds" binding:"required"`
}

// GetUserPlaylists returns all playlists owned by the current user
func (h *PlaylistHandler) GetUserPlaylists(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	playlists, err := h.playlistService.GetUserPlaylists(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get user playlists",
		})
		return
	}

	c.JSON(http.StatusOK, playlists)
}

// CreatePlaylist creates a new playlist
func (h *PlaylistHandler) CreatePlaylist(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "authentication required",
		})
		return
	}

	var req CreatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Set default visibility if not provided
	if req.Visibility == "" {
		req.Visibility = "private"
	}

	// Validate visibility
	if req.Visibility != "private" && req.Visibility != "public" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_visibility",
			"message": "Visibility must be 'private' or 'public'",
		})
		return
	}

	playlist, err := h.playlistService.CreatePlaylist(c.Request.Context(), user.ID, req.Name, req.Description, req.Visibility)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create playlist",
		})
		return
	}

	c.JSON(http.StatusCreated, playlist)
}

// GetPlaylist returns a specific playlist with its songs
func (h *PlaylistHandler) GetPlaylist(c *gin.Context) {
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
			"message": "Invalid playlist ID",
		})
		return
	}

	playlist, songs, err := h.playlistService.GetPlaylist(c.Request.Context(), id, user.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Playlist not found",
		})
		return
	}

	response := gin.H{
		"id":          playlist.ID,
		"user_id":     playlist.UserID,
		"name":        playlist.Name,
		"description": playlist.Description,
		"visibility":  playlist.Visibility,
		"created_at":  playlist.CreatedAt,
		"updated_at":  playlist.UpdatedAt,
		"song_count":  playlist.SongCount,
		"duration":    playlist.Duration,
		"songs":       songs,
	}

	c.JSON(http.StatusOK, response)
}

// UpdatePlaylist updates playlist metadata
func (h *PlaylistHandler) UpdatePlaylist(c *gin.Context) {
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
			"message": "Invalid playlist ID",
		})
		return
	}

	var req UpdatePlaylistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Set default visibility if not provided
	if req.Visibility == "" {
		req.Visibility = "private"
	}

	// Validate visibility
	if req.Visibility != "private" && req.Visibility != "public" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_visibility",
			"message": "Visibility must be 'private' or 'public'",
		})
		return
	}

	playlist, err := h.playlistService.UpdatePlaylist(c.Request.Context(), id, user.ID, req.Name, req.Description, req.Visibility)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Playlist not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to update playlist",
			})
		}
		return
	}

	c.JSON(http.StatusOK, playlist)
}

// DeletePlaylist deletes a playlist
func (h *PlaylistHandler) DeletePlaylist(c *gin.Context) {
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
			"message": "Invalid playlist ID",
		})
		return
	}

	err = h.playlistService.DeletePlaylist(c.Request.Context(), id, user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Playlist not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to delete playlist",
			})
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// AddSongsToPlaylist adds songs to a playlist
func (h *PlaylistHandler) AddSongsToPlaylist(c *gin.Context) {
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
			"message": "Invalid playlist ID",
		})
		return
	}

	var req AddSongsRequest
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

	err = h.playlistService.AddSongsToPlaylist(c.Request.Context(), id, user.ID, req.SongIDs, req.Position)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Playlist not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to add songs to playlist",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Songs added to playlist"})
}

// RemoveSongsFromPlaylist removes songs from a playlist
func (h *PlaylistHandler) RemoveSongsFromPlaylist(c *gin.Context) {
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
			"message": "Invalid playlist ID",
		})
		return
	}

	var req RemoveSongsRequest
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

	err = h.playlistService.RemoveSongsFromPlaylist(c.Request.Context(), id, user.ID, req.SongIDs)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Playlist not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to remove songs from playlist",
			})
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ReorderPlaylistSongs reorders all songs in a playlist
func (h *PlaylistHandler) ReorderPlaylistSongs(c *gin.Context) {
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
			"message": "Invalid playlist ID",
		})
		return
	}

	var req ReorderSongsRequest
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

	err = h.playlistService.ReorderPlaylistSongs(c.Request.Context(), id, user.ID, req.SongIDs)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Playlist not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to reorder playlist songs",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist songs reordered"})
}