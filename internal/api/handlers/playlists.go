package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/services"
)

type playlistRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
}

// ListPlaylists godoc
// @Summary List playlists
// @Tags Playlists
// @Produce json
// @Param limit query int false "max items (default 50, max 200)"
// @Param offset query int false "offset"
// @Success 200 {array} models.Playlist
// @Router /playlists [get]
// @Security BearerAuth
func (h *Handler) ListPlaylists(c echo.Context) error {
	user, _ := currentUser(c)
	limit, offset := parseLimitOffset(c, 50, 200)
	playlists, err := h.playlists.List(c.Request().Context(), user.ID, limit, offset)
	if err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_LIST_FAILED")
	}
	return c.JSON(http.StatusOK, playlists)
}

// CreatePlaylist godoc
// @Summary Create playlist
// @Tags Playlists
// @Accept json
// @Produce json
// @Param body body playlistRequest true "playlist"
// @Success 200 {object} models.Playlist
// @Router /playlists [post]
// @Security BearerAuth
func (h *Handler) CreatePlaylist(c echo.Context) error {
	user, _ := currentUser(c)
	var req playlistRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	playlist, err := h.playlists.Create(c.Request().Context(), user.ID, req.Name, req.Description, req.Public, nil)
	if err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_CREATE_FAILED")
	}
	return c.JSON(http.StatusOK, playlist)
}

// GetPlaylist godoc
// @Summary Get playlist
// @Tags Playlists
// @Produce json
// @Param id path int true "Playlist ID"
// @Success 200 {object} models.Playlist
// @Failure 404 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /playlists/{id} [get]
// @Security BearerAuth
func (h *Handler) GetPlaylist(c echo.Context) error {
	user, _ := currentUser(c)
	playlist, err := h.playlists.GetWithSongs(c.Request().Context(), user.ID, pathID(c, "id"))
	if err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_GET_FAILED")
	}
	return c.JSON(http.StatusOK, playlist)
}

// UpdatePlaylist godoc
// @Summary Update playlist
// @Tags Playlists
// @Accept json
// @Produce json
// @Param id path int true "Playlist ID"
// @Param body body playlistRequest true "playlist"
// @Success 200 {object} models.Playlist
// @Failure 404 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /playlists/{id} [put]
// @Security BearerAuth
func (h *Handler) UpdatePlaylist(c echo.Context) error {
	user, _ := currentUser(c)
	var req playlistRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	playlist, err := h.playlists.UpdateMeta(c.Request().Context(), user.ID, pathID(c, "id"), req.Name, req.Description, req.Public)
	if err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_UPDATE_FAILED")
	}
	return c.JSON(http.StatusOK, playlist)
}

// UploadPlaylistCover godoc
// @Summary Upload playlist cover
// @Tags Playlists
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Playlist ID"
// @Param cover formData file true "Cover image"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /playlists/{id}/cover [post]
// @Security BearerAuth
func (h *Handler) UploadPlaylistCover(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()
	id := pathID(c, "id")

	if _, err := h.playlists.Get(ctx, user.ID, id); err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_GET_FAILED")
	}

	file, err := c.FormFile("cover")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "no file provided", "code": "BAD_REQUEST"})
	}
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "failed to read file", "code": "BAD_REQUEST"})
	}
	defer src.Close()

	coversDir := filepath.Join(h.mediaRoot, ".korus", "covers")
	if err := os.MkdirAll(coversDir, 0755); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to create covers dir", "code": "INTERNAL_ERROR"})
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	destPath := filepath.Join(coversDir, fmt.Sprintf("playlist_%d%s", id, ext))

	dst, err := os.Create(destPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to save file", "code": "INTERNAL_ERROR"})
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to save file", "code": "INTERNAL_ERROR"})
	}

	if err := h.playlists.SetCover(ctx, user.ID, id, destPath); err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_UPDATE_FAILED")
	}
	return c.JSON(http.StatusOK, map[string]string{"cover_path": destPath})
}

// GetPlaylistCover godoc
// @Summary Get playlist cover
// @Tags Playlists
// @Produce image/*
// @Param id path int true "Playlist ID"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /playlists/{id}/cover [get]
func (h *Handler) GetPlaylistCover(c echo.Context) error {
	coverPath, err := h.playlists.Cover(c.Request().Context(), pathID(c, "id"))
	if err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_GET_FAILED")
	}
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "cover file not found", "code": "NOT_FOUND"})
	}
	return c.File(coverPath)
}

// DeletePlaylist godoc
// @Summary Delete playlist
// @Tags Playlists
// @Produce json
// @Param id path int true "Playlist ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /playlists/{id} [delete]
// @Security BearerAuth
func (h *Handler) DeletePlaylist(c echo.Context) error {
	user, _ := currentUser(c)
	if err := h.playlists.Delete(c.Request().Context(), user.ID, pathID(c, "id")); err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_DELETE_FAILED")
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// AddPlaylistSong godoc
// @Summary Add song to playlist
// @Tags Playlists
// @Accept json
// @Produce json
// @Param id path int true "Playlist ID"
// @Param body body map[string]int true "song payload"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /playlists/{id}/songs [post]
// @Security BearerAuth
func (h *Handler) AddPlaylistSong(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()
	var payload struct {
		SongID int64 `json:"song_id" validate:"required"`
	}
	if err := bindAndValidate(c, &payload); err != nil {
		return err
	}
	if !h.library.SongExists(ctx, payload.SongID) {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "song not found", "code": "NOT_FOUND"})
	}
	if err := h.playlists.SetSongs(ctx, user.ID, pathID(c, "id"), []int64{payload.SongID}, services.ModeAppend); err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_ADD_FAILED")
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// DeletePlaylistSong godoc
// @Summary Remove song from playlist
// @Tags Playlists
// @Produce json
// @Param id path int true "Playlist ID"
// @Param song_id path int true "Song ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /playlists/{id}/songs/{song_id} [delete]
// @Security BearerAuth
func (h *Handler) DeletePlaylistSong(c echo.Context) error {
	user, _ := currentUser(c)
	err := h.playlists.RemoveSongs(c.Request().Context(), user.ID, pathID(c, "id"), []int64{pathID(c, "song_id")})
	if err != nil {
		return serviceError(err, "playlist not found", "PLAYLIST_REMOVE_FAILED")
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// ReorderPlaylistSongs godoc
// @Summary Reorder playlist songs
// @Tags Playlists
// @Accept json
// @Produce json
// @Param id path int true "Playlist ID"
// @Param body body map[string][]int64 true "song ids"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /playlists/{id}/reorder [put]
// @Security BearerAuth
func (h *Handler) ReorderPlaylistSongs(c echo.Context) error {
	user, _ := currentUser(c)
	var payload struct {
		SongIDs []int64 `json:"song_ids" validate:"required"`
	}
	if err := bindAndValidate(c, &payload); err != nil {
		return err
	}
	if err := h.playlists.Reorder(c.Request().Context(), user.ID, pathID(c, "id"), payload.SongIDs); err != nil {
		return serviceError(err, "playlist not found", "REORDER_FAILED")
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}
