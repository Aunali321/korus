package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/services"
)

// setFavorite is the shared body of the six favourite and follow endpoints.
func (h *Handler) setFavorite(c echo.Context, kind services.EntityKind, on bool) error {
	user, _ := currentUser(c)
	err := h.library.SetFavorite(c.Request().Context(), user.ID, kind, pathID(c, "id"), on)
	if err != nil {
		return serviceError(err, string(kind)+" not found", "FAVORITE_FAILED")
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

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
	return h.setFavorite(c, services.EntitySong, true)
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
	return h.setFavorite(c, services.EntitySong, false)
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
	return h.setFavorite(c, services.EntityAlbum, true)
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
	return h.setFavorite(c, services.EntityAlbum, false)
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
	return h.setFavorite(c, services.EntityArtist, true)
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
	return h.setFavorite(c, services.EntityArtist, false)
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

	songs, err := h.library.UserSongs(ctx, user.ID, services.SliceFavorites, 0, -1)
	if err != nil {
		return serviceError(err, "not found", "FAVORITES_FAILED")
	}
	albums, err := h.library.FavoriteAlbums(ctx, user.ID)
	if err != nil {
		return serviceError(err, "not found", "FAVORITES_FAILED")
	}
	artists, err := h.library.FollowedArtists(ctx, user.ID)
	if err != nil {
		return serviceError(err, "not found", "FAVORITES_FAILED")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"songs":   songs,
		"albums":  albums,
		"artists": artists,
	})
}
