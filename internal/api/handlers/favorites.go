package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/models"
)

// FavSong godoc
// @Summary Favorite song
// @Tags Favorites
// @Produce json
// @Param id path int true "Song ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Router /favorites/songs/{id} [post]
// @Security BearerAuth
func (h *Handler) FavSong(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !h.songExists(c.Request().Context(), id) {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "song not found", "code": "NOT_FOUND"})
	}
	if _, err := h.db.ExecContext(c.Request().Context(), `INSERT OR IGNORE INTO favorites_songs(user_id, song_id) VALUES(?, ?)`, user.ID, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "FAV_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// UnfavSong godoc
// @Summary Unfavorite song
// @Tags Favorites
// @Produce json
// @Param id path int true "Song ID"
// @Success 200 {object} map[string]bool
// @Router /favorites/songs/{id} [delete]
// @Security BearerAuth
func (h *Handler) UnfavSong(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if _, err := h.db.ExecContext(c.Request().Context(), `DELETE FROM favorites_songs WHERE user_id = ? AND song_id = ?`, user.ID, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "UNFAV_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// FavAlbum godoc
// @Summary Favorite album
// @Tags Favorites
// @Produce json
// @Param id path int true "Album ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Router /favorites/albums/{id} [post]
// @Security BearerAuth
func (h *Handler) FavAlbum(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !h.albumExists(c.Request().Context(), id) {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "album not found", "code": "NOT_FOUND"})
	}
	if _, err := h.db.ExecContext(c.Request().Context(), `INSERT OR IGNORE INTO favorites_albums(user_id, album_id) VALUES(?, ?)`, user.ID, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "FAV_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// UnfavAlbum godoc
// @Summary Unfavorite album
// @Tags Favorites
// @Produce json
// @Param id path int true "Album ID"
// @Success 200 {object} map[string]bool
// @Router /favorites/albums/{id} [delete]
// @Security BearerAuth
func (h *Handler) UnfavAlbum(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if _, err := h.db.ExecContext(c.Request().Context(), `DELETE FROM favorites_albums WHERE user_id = ? AND album_id = ?`, user.ID, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "UNFAV_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// FollowArtist godoc
// @Summary Follow artist
// @Tags Favorites
// @Produce json
// @Param id path int true "Artist ID"
// @Success 200 {object} map[string]bool
// @Failure 404 {object} map[string]string
// @Router /follows/artists/{id} [post]
// @Security BearerAuth
func (h *Handler) FollowArtist(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if !h.artistExists(c.Request().Context(), id) {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "artist not found", "code": "NOT_FOUND"})
	}
	if _, err := h.db.ExecContext(c.Request().Context(), `INSERT OR IGNORE INTO follows_artists(user_id, artist_id) VALUES(?, ?)`, user.ID, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "FOLLOW_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// UnfollowArtist godoc
// @Summary Unfollow artist
// @Tags Favorites
// @Produce json
// @Param id path int true "Artist ID"
// @Success 200 {object} map[string]bool
// @Router /follows/artists/{id} [delete]
// @Security BearerAuth
func (h *Handler) UnfollowArtist(c echo.Context) error {
	user, _ := currentUser(c)
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if _, err := h.db.ExecContext(c.Request().Context(), `DELETE FROM follows_artists WHERE user_id = ? AND artist_id = ?`, user.ID, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "UNFOLLOW_FAILED"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// ListFavorites godoc
// @Summary List favorites
// @Tags Favorites
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /favorites [get]
// @Security BearerAuth
func (h *Handler) ListFavorites(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()
	songs, _ := db.GetSongsByFavorites(ctx, h.db, user.ID)
	_ = db.PopulateSongArtists(ctx, h.db, songs)
	albums, _ := h.fetchAlbumsByFav(ctx, user.ID)
	artists, _ := h.fetchArtistsByFollow(ctx, user.ID)
	return c.JSON(http.StatusOK, map[string]any{
		"songs":   songs,
		"albums":  albums,
		"artists": artists,
	})
}

func (h *Handler) fetchAlbumsByFav(ctx context.Context, userID int64) ([]models.Album, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT a.id, a.title, a.cover_path
		FROM favorites_albums f
		JOIN albums a ON a.id = f.album_id
		WHERE f.user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Album
	for rows.Next() {
		var a models.Album
		if err := rows.Scan(&a.ID, &a.Title, &a.CoverPath); err == nil {
			res = append(res, a)
		}
	}
	return res, nil
}

func (h *Handler) fetchArtistsByFollow(ctx context.Context, userID int64) ([]models.Artist, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT ar.id, ar.name
		FROM follows_artists f
		JOIN artists ar ON ar.id = f.artist_id
		WHERE f.user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Artist
	for rows.Next() {
		var a models.Artist
		if err := rows.Scan(&a.ID, &a.Name); err == nil {
			res = append(res, a)
		}
	}
	return res, nil
}

func (h *Handler) songExists(ctx context.Context, id int64) bool {
	var exists int
	_ = h.db.QueryRowContext(ctx, `SELECT 1 FROM songs WHERE id = ?`, id).Scan(&exists)
	return exists == 1
}

func (h *Handler) albumExists(ctx context.Context, id int64) bool {
	var exists int
	_ = h.db.QueryRowContext(ctx, `SELECT 1 FROM albums WHERE id = ?`, id).Scan(&exists)
	return exists == 1
}

func (h *Handler) artistExists(ctx context.Context, id int64) bool {
	var exists int
	_ = h.db.QueryRowContext(ctx, `SELECT 1 FROM artists WHERE id = ?`, id).Scan(&exists)
	return exists == 1
}
