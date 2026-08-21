package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Library godoc
// @Summary Get library overview
// @Tags Library
// @Produce json
// @Param limit query int false "max items (default: all)"
// @Success 200 {object} map[string]interface{}
// @Router /library [get]
// @Security BearerAuth
func (h *Handler) Library(c echo.Context) error {
	ctx := c.Request().Context()
	limit := parseOptionalLimit(c)

	artists, err := h.library.Artists(ctx, limit)
	if err != nil {
		return serviceError(err, "not found", "LIBRARY_FAILED")
	}
	albums, err := h.library.Albums(ctx, limit)
	if err != nil {
		return serviceError(err, "not found", "LIBRARY_FAILED")
	}
	songs, err := h.library.RecentlyAdded(ctx, limit)
	if err != nil {
		return serviceError(err, "not found", "LIBRARY_FAILED")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"artists": artists,
		"albums":  albums,
		"songs":   songs,
	})
}

// Artist godoc
// @Summary Get artist by id
// @Tags Library
// @Produce json
// @Param id path int true "Artist ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /artists/{id} [get]
// @Security BearerAuth
func (h *Handler) Artist(c echo.Context) error {
	ctx := c.Request().Context()
	id := pathID(c, "id")

	artist, err := h.library.Artist(ctx, id)
	if err != nil {
		return serviceError(err, "artist not found", "ARTIST_FAILED")
	}
	albums, err := h.library.AlbumsByArtist(ctx, id)
	if err != nil {
		return serviceError(err, "artist not found", "ARTIST_FAILED")
	}
	songs, err := h.library.SongsByArtist(ctx, id)
	if err != nil {
		return serviceError(err, "artist not found", "ARTIST_FAILED")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"id":         artist.ID,
		"name":       artist.Name,
		"bio":        artist.Bio,
		"image_path": artist.ImagePath,
		"mbid":       artist.MBID,
		"albums":     albums,
		"songs":      songs,
	})
}

// Album godoc
// @Summary Get album by id
// @Tags Library
// @Produce json
// @Param id path int true "Album ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /albums/{id} [get]
// @Security BearerAuth
func (h *Handler) Album(c echo.Context) error {
	ctx := c.Request().Context()
	id := pathID(c, "id")

	album, err := h.library.Album(ctx, id)
	if err != nil {
		return serviceError(err, "album not found", "ALBUM_FAILED")
	}
	songs, err := h.library.SongsByAlbum(ctx, id)
	if err != nil {
		return serviceError(err, "album not found", "ALBUM_FAILED")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"id":         album.ID,
		"title":      album.Title,
		"year":       album.Year,
		"cover_path": album.CoverPath,
		"mbid":       album.MBID,
		"artist":     album.Artist,
		"songs":      songs,
	})
}

// Song godoc
// @Summary Get song by id
// @Tags Library
// @Produce json
// @Param id path int true "Song ID"
// @Success 200 {object} models.Song
// @Failure 404 {object} map[string]string
// @Router /songs/{id} [get]
// @Security BearerAuth
func (h *Handler) Song(c echo.Context) error {
	song, err := h.library.Song(c.Request().Context(), pathID(c, "id"))
	if err != nil {
		return serviceError(err, "song not found", "SONG_FAILED")
	}
	return c.JSON(http.StatusOK, song)
}
