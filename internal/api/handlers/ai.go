package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	aisvc "github.com/Aunali321/korus/internal/services/ai"
)

func aiDisabled() error {
	return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "AI is not enabled", "code": "AI_DISABLED"})
}

type chatRequest struct {
	ConversationID int64   `json:"conversation_id"`
	Message        string  `json:"message"`
	NowPlayingID   int64   `json:"now_playing_id"`
	QueueIDs       []int64 `json:"queue_ids"`
	Shuffle        bool    `json:"shuffle"`
	Repeat         string  `json:"repeat"`
}

// AIChat streams an assistant turn as Server-Sent Events. Events are JSON
// objects with a "type": text, tool, action, done, or error.
// @Summary Chat with the assistant
// @Tags AI
// @Accept json
// @Produce text/event-stream
// @Param body body chatRequest true "Message and player context"
// @Success 200 {string} string "SSE stream of JSON events: text, tool, action, ui, done, error"
// @Router /ai/chat [post]
// @Security BearerAuth
func (h *Handler) AIChat(c echo.Context) error {
	if h.ai == nil {
		return aiDisabled()
	}
	user, _ := currentUser(c)

	var req chatRequest
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "message required", "code": "BAD_REQUEST"})
	}
	ctx := c.Request().Context()

	convID := req.ConversationID
	var history []aisvc.ChatMessage
	if convID == 0 {
		id, err := h.ai.CreateConversation(ctx, user.ID, conversationTitle(req.Message))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "INTERNAL_ERROR"})
		}
		convID = id
	} else {
		owner, err := h.ai.ConversationOwner(ctx, convID)
		if err != nil || owner != user.ID {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "conversation not found", "code": "NOT_FOUND"})
		}
		history, _ = h.ai.LoadHistory(ctx, convID)
	}

	res := c.Response()
	res.Header().Set("Content-Type", "text/event-stream")
	res.Header().Set("Cache-Control", "no-cache")
	res.Header().Set("Connection", "keep-alive")
	res.Header().Set("X-Accel-Buffering", "no")
	res.WriteHeader(http.StatusOK)

	var wmu sync.Mutex
	send := func(v any) {
		b, _ := json.Marshal(v)
		wmu.Lock()
		fmt.Fprintf(res, "data: %s\n\n", b)
		res.Flush()
		wmu.Unlock()
	}

	// Reasoning phases can be silent for tens of seconds; a comment frame keeps
	// idle-timeout proxies from closing the stream.
	stopPing := make(chan struct{})
	defer close(stopPing)
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stopPing:
				return
			case <-t.C:
				wmu.Lock()
				fmt.Fprint(res, ": ping\n\n")
				res.Flush()
				wmu.Unlock()
			}
		}
	}()

	var assistant strings.Builder
	sink := aisvc.ChatSink{
		OnText: func(delta string) {
			assistant.WriteString(delta)
			send(map[string]any{"type": "text", "delta": delta})
		},
		OnTool: func(name, phase string) {
			send(map[string]any{"type": "tool", "name": name, "phase": phase})
		},
		OnEffect: func(eff aisvc.Effect) {
			if eff.Kind == aisvc.EffectUI {
				send(map[string]any{"type": "ui", "spec": eff.Spec})
				return
			}
			payload := map[string]any{"type": "action", "action": eff.Action}
			// set_queue sends an empty list to mean "clear", so the songs key
			// is always present for the queue-shaped actions.
			if len(eff.SongIDs) > 0 || eff.Action == "set_queue" {
				payload["songs"] = h.songsByIDs(ctx, eff.SongIDs)
			}
			if eff.Mode != "" {
				payload["mode"] = eff.Mode
			}
			if eff.PlaylistID != 0 {
				payload["playlist_id"] = eff.PlaylistID
			}
			if eff.Control != "" {
				payload["control"] = eff.Control
			}
			if eff.Entity != "" {
				payload["entity"] = eff.Entity
				payload["entity_id"] = eff.EntityID
				payload["on"] = eff.On
			}
			send(payload)
		},
	}

	h.ai.AppendMessage(ctx, convID, "user", req.Message)

	repeat := req.Repeat
	if repeat == "" {
		repeat = "off"
	}
	pc := aisvc.PlayerContext{
		NowPlayingID: req.NowPlayingID,
		QueueIDs:     req.QueueIDs,
		Shuffle:      req.Shuffle,
		Repeat:       repeat,
	}
	if err := h.ai.Chat(ctx, user.ID, history, req.Message, pc, sink); err != nil {
		send(map[string]any{"type": "error", "message": err.Error()})
		return nil
	}
	if t := strings.TrimSpace(assistant.String()); t != "" {
		h.ai.AppendMessage(ctx, convID, "assistant", t)
	}
	send(map[string]any{"type": "done", "conversation_id": convID})
	return nil
}

// @Summary List conversations
// @Tags AI
// @Produce json
// @Success 200 {object} map[string][]ai.Conversation
// @Router /ai/conversations [get]
// @Security BearerAuth
func (h *Handler) ListConversations(c echo.Context) error {
	if h.ai == nil {
		return aiDisabled()
	}
	user, _ := currentUser(c)
	convs, err := h.ai.ListConversations(c.Request().Context(), user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "INTERNAL_ERROR"})
	}
	return c.JSON(http.StatusOK, map[string]any{"conversations": convs})
}

// @Summary Get conversation messages
// @Tags AI
// @Produce json
// @Param id path int true "Conversation ID"
// @Success 200 {object} map[string][]ai.ChatMessage
// @Router /ai/conversations/{id} [get]
// @Security BearerAuth
func (h *Handler) GetConversation(c echo.Context) error {
	if h.ai == nil {
		return aiDisabled()
	}
	user, _ := currentUser(c)
	var id int64
	if err := echo.PathParamsBinder(c).Int64("id", &id).BindError(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid id", "code": "BAD_REQUEST"})
	}
	ctx := c.Request().Context()
	owner, err := h.ai.ConversationOwner(ctx, id)
	if err != nil || owner != user.ID {
		return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "conversation not found", "code": "NOT_FOUND"})
	}
	msgs, err := h.ai.LoadHistory(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "INTERNAL_ERROR"})
	}
	return c.JSON(http.StatusOK, map[string]any{"messages": msgs})
}

// @Summary Delete conversation
// @Tags AI
// @Produce json
// @Param id path int true "Conversation ID"
// @Success 200 {object} map[string]bool
// @Router /ai/conversations/{id} [delete]
// @Security BearerAuth
func (h *Handler) DeleteConversation(c echo.Context) error {
	if h.ai == nil {
		return aiDisabled()
	}
	user, _ := currentUser(c)
	var id int64
	if err := echo.PathParamsBinder(c).Int64("id", &id).BindError(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{"error": "invalid id", "code": "BAD_REQUEST"})
	}
	if err := h.ai.DeleteConversation(c.Request().Context(), id, user.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "INTERNAL_ERROR"})
	}
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

// AIWrapped returns the cached Wrapped HTML diary page for a period, generating
// and caching it on first request. period=month|year, optional key (YYYY-MM / YYYY).
// @Summary Wrapped diary page
// @Tags AI
// @Produce json
// @Param period query string false "month|year"
// @Param key query string false "YYYY-MM or YYYY"
// @Param refresh query bool false "Regenerate instead of serving the cached page"
// @Success 200 {object} map[string]any
// @Router /ai/wrapped [get]
// @Security BearerAuth
func (h *Handler) AIWrapped(c echo.Context) error {
	if h.ai == nil {
		return aiDisabled()
	}
	user, _ := currentUser(c)

	periodType := c.QueryParam("period")
	if periodType != "year" {
		periodType = "month"
	}
	periodKey := c.QueryParam("key")
	if periodKey == "" {
		now := time.Now()
		if periodType == "year" {
			periodKey = now.Format("2006")
		} else {
			periodKey = now.Format("2006-01")
		}
	}

	ctx := c.Request().Context()
	refresh := c.QueryParam("refresh") == "1" || c.QueryParam("refresh") == "true"
	html, cached, err := h.ai.Wrapped(ctx, user.ID, periodType, periodKey, refresh)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "INTERNAL_ERROR"})
	}
	return c.JSON(http.StatusOK, map[string]any{"html": html, "period_type": periodType, "period_key": periodKey, "cached": cached})
}

func conversationTitle(msg string) string {
	msg = strings.TrimSpace(msg)
	r := []rune(msg)
	if len(r) > 60 {
		return string(r[:60])
	}
	return msg
}
