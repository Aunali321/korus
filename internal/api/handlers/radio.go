package handlers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/models"
)

// Radio godoc
// @Summary Get similar songs for radio playback
// @Tags Radio
// @Produce json
// @Param id path int true "Song ID to seed radio from"
// @Param limit query int false "Number of songs to return" default(20)
// @Success 200 {object} map[string][]models.Song
// @Failure 404 {object} map[string]string
// @Router /radio/{id} [get]
// @Security BearerAuth
func (h *Handler) Radio(c echo.Context) error {
	var songID int64
	if err := echo.PathParamsBinder(c).Int64("id", &songID).BindError(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid id", "code": "BAD_REQUEST"})
	}

	limit := h.radioDefaultLimit
	if l := c.QueryParam("limit"); l != "" {
		if err := echo.QueryParamsBinder(c).Int("limit", &limit).BindError(); err != nil || limit <= 0 || limit > 100 {
			limit = h.radioDefaultLimit
		}
	}

	ctx := c.Request().Context()

	if h.ai != nil {
		user, _ := currentUser(c)
		ids, err := h.ai.Radio(ctx, user.ID, songID, limit)
		if err == nil && len(ids) > 0 {
			return c.JSON(http.StatusOK, map[string]any{"songs": h.songsByIDs(ctx, ids)})
		}
	}

	songs, err := h.library.SimilarSongs(ctx, songID, limit)
	if err != nil {
		return serviceError(err, "song not found", "RADIO_FAILED")
	}
	return c.JSON(http.StatusOK, map[string]any{"songs": songs})
}

func (h *Handler) songsByIDs(ctx context.Context, ids []int64) []models.Song {
	songs, err := h.library.Songs(ctx, ids)
	if err != nil {
		return []models.Song{}
	}
	return songs
}
