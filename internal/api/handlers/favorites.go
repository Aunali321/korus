package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/models"
)

// FavSong godoc
// @Summary Favorite song
// @Tags Favorites
// @Param id path int true "Song ID"
// @Success 200 {object} map[string]bool
// @Router /api/favorites/songs/{id} [post]
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
// @Param id path int true "Song ID"
// @Success 200 {object} map[string]bool
// @Router /api/favorites/songs/{id} [delete]
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
// @Param id path int true "Album ID"
// @Success 200 {object} map[string]bool
// @Router /api/favorites/albums/{id} [post]
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
// @Param id path int true "Album ID"
// @Success 200 {object} map[string]bool
// @Router /api/favorites/albums/{id} [delete]
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
// @Param id path int true "Artist ID"
// @Success 200 {object} map[string]bool
// @Router /api/follows/artists/{id} [post]
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
// @Param id path int true "Artist ID"
// @Success 200 {object} map[string]bool
// @Router /api/follows/artists/{id} [delete]
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
// @Router /api/favorites [get]
func (h *Handler) ListFavorites(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()
	songs, _ := h.fetchSongsByFav(ctx, user.ID)
	albums, _ := h.fetchAlbumsByFav(ctx, user.ID)
	artists, _ := h.fetchArtistsByFollow(ctx, user.ID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"songs":   songs,
		"albums":  albums,
		"artists": artists,
	})
}

func (h *Handler) fetchSongsByFav(ctx context.Context, userID int64) ([]models.Song, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT s.id, s.title, s.file_path
		FROM favorites_songs f
		JOIN songs s ON s.id = f.song_id
		WHERE f.user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.Song
	for rows.Next() {
		var s models.Song
		if err := rows.Scan(&s.ID, &s.Title, &s.FilePath); err == nil {
			res = append(res, s)
		}
	}
	return res, nil
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
