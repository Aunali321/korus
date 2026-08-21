package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/services"
)

// Stats godoc
// @Summary Stats overview
// @Tags Stats
// @Produce json
// @Param period query string false "hour|today|week|month|year|all_time"
// @Success 200 {object} models.StatsReport
// @Router /stats [get]
// @Security BearerAuth
func (h *Handler) Stats(c echo.Context) error {
	user, _ := currentUser(c)
	period := services.ResolvePeriod(c.QueryParam("period"))
	report, err := h.stats.Report(c.Request().Context(), user.ID, period)
	if err != nil {
		return serviceError(err, "not found", "STATS_FAILED")
	}
	return c.JSON(http.StatusOK, report)
}

// Wrapped godoc
// @Summary Wrapped stats
// @Tags Stats
// @Produce json
// @Param period query string false "year|all_time"
// @Success 200 {object} models.ListeningSummary
// @Router /stats/wrapped [get]
// @Security BearerAuth
func (h *Handler) Wrapped(c echo.Context) error {
	user, _ := currentUser(c)
	period := services.ResolvePeriod(c.QueryParam("period"))
	summary, err := h.stats.Summary(c.Request().Context(), user.ID, period, 5, 5)
	if err != nil {
		return serviceError(err, "not found", "WRAPPED_FAILED")
	}
	return c.JSON(http.StatusOK, summary)
}

// Insights godoc
// @Summary Insights
// @Tags Stats
// @Produce json
// @Success 200 {object} models.Insights
// @Router /stats/insights [get]
// @Security BearerAuth
func (h *Handler) Insights(c echo.Context) error {
	user, _ := currentUser(c)
	insights, err := h.stats.Streaks(c.Request().Context(), user.ID)
	if err != nil {
		return serviceError(err, "not found", "INSIGHTS_FAILED")
	}
	return c.JSON(http.StatusOK, insights)
}

// Home godoc
// @Summary Home summary
// @Tags Stats
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /home [get]
// @Security BearerAuth
func (h *Handler) Home(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()

	recent, err := h.library.UserSongs(ctx, user.ID, services.SliceRecent, 0, 27)
	if err != nil {
		return serviceError(err, "not found", "HOME_FAILED")
	}
	recommended, err := h.library.UserSongs(ctx, user.ID, services.SliceTop, 0, 5)
	if err != nil {
		return serviceError(err, "not found", "HOME_FAILED")
	}
	newAdditions, err := h.library.Albums(ctx, 10)
	if err != nil {
		return serviceError(err, "not found", "HOME_FAILED")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"recent_plays":    recent,
		"recommendations": recommended,
		"new_additions":   newAdditions,
	})
}
