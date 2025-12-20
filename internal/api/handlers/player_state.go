package handlers

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Aunali321/korus/internal/models"
	"github.com/labstack/echo/v4"
)

type PlayerState struct {
	CurrentSongID *int    `json:"current_song_id"`
	Queue         []int   `json:"queue"`
	QueueIndex    int     `json:"queue_index"`
	Progress      float64 `json:"progress"`
}

// GetPlayerState godoc
// @Summary Get player state
// @Tags Player
// @Produce json
// @Success 200 {object} PlayerState
// @Router /player/state [get]
func (h *Handler) GetPlayerState(c echo.Context) error {
	user := c.Get("user").(models.User)

	var currentSongID sql.NullInt64
	var queueJSON sql.NullString
	var queueIndex int
	var progress float64

	err := h.db.QueryRowContext(c.Request().Context(), `
		SELECT current_song_id, queue_song_ids, queue_index, progress
		FROM player_state WHERE user_id = ?
	`, user.ID).Scan(&currentSongID, &queueJSON, &queueIndex, &progress)

	if err == sql.ErrNoRows {
		return c.JSON(http.StatusOK, PlayerState{Queue: []int{}})
	}
	if err != nil {
		slog.Error("failed to get player state", "error", err, "user_id", user.ID)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to get player state"})
	}

	state := PlayerState{
		QueueIndex: queueIndex,
		Progress:   progress,
		Queue:      []int{},
	}

	if currentSongID.Valid {
		id := int(currentSongID.Int64)
		state.CurrentSongID = &id
	}

	if queueJSON.Valid && queueJSON.String != "" {
		if err := json.Unmarshal([]byte(queueJSON.String), &state.Queue); err != nil {
			slog.Warn("failed to parse queue JSON", "error", err)
			state.Queue = []int{}
		}
	}

	return c.JSON(http.StatusOK, state)
}

// SavePlayerState godoc
// @Summary Save player state
// @Tags Player
// @Accept json
// @Produce json
// @Param state body PlayerState true "Player state"
// @Success 200 {object} map[string]bool
// @Router /player/state [put]
// @Router /player/state [post]
func (h *Handler) SavePlayerState(c echo.Context) error {
	user := c.Get("user").(models.User)

	var req PlayerState
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	var currentSongID any = nil
	if req.CurrentSongID != nil {
		currentSongID = *req.CurrentSongID
	}

	queueJSON := "[]"
	if len(req.Queue) > 0 {
		bytes, err := json.Marshal(req.Queue)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid queue"})
		}
		queueJSON = string(bytes)
	}

	_, err := h.db.ExecContext(c.Request().Context(), `
		INSERT INTO player_state (user_id, current_song_id, queue_song_ids, queue_index, progress, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			current_song_id = excluded.current_song_id,
			queue_song_ids = excluded.queue_song_ids,
			queue_index = excluded.queue_index,
			progress = excluded.progress,
			updated_at = CURRENT_TIMESTAMP
	`, user.ID, currentSongID, queueJSON, req.QueueIndex, req.Progress)

	if err != nil {
		slog.Error("failed to save player state", "error", err, "user_id", user.ID)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "failed to save player state"})
	}

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}
