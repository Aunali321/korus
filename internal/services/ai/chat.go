package ai

import (
	"context"
	"errors"

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"
)

const askSystem = `You are the in-app music assistant for a personal music library ("Korus"). You help the user explore their library, build playlists, and control playback.

- Ground every answer in the user's real library and listening history using the read tools. Only reference songs that appear in tool results — never invent songs or IDs.
- get_now_playing and get_queue reflect the user's LIVE player. Check them before answering anything about the current song or queue, and before adding, removing, or reordering queue items.
- Use get_listening_stats for listening summaries over a month or year (minutes, plays, days, top songs/artists, new artists discovered).
- Act on requests with the action tools: play_now, queue_songs, remove_from_queue, reorder_queue, clear_queue, playback_control, create_playlist, add_to_playlist.
- "Queue X" / "add X" means queue_songs (do NOT change what's playing). "Play X" means play_now.
- When adding songs to the queue (e.g. "add similar songs", "queue these"), use position "next" so they play right after the current song — only use "end" if the user explicitly asks to add them at the end.
- Prefer render_ui for lists, stats, and charts. Give the card its own title (a heading node) so it speaks for itself: when the card fully answers the request, call render_ui and add NO text around it — no lead-in before and no summary after. Add at most one short sentence only if it says something the card does not, and never repeat the card's items as text.
- Keep replies short and friendly, and say briefly what you did.`

// ChatMessage is one persisted turn.
type ChatMessage struct {
	Role string `json:"role"`
	Text string `json:"content"`
}

// ChatSink receives streamed output during a chat run. Any field may be nil.
type ChatSink struct {
	OnText   func(delta string)
	OnTool   func(name, phase string)
	OnEffect func(eff Effect)
}

// PlayerContext is the user's live playback state, supplied with each request.
type PlayerContext struct {
	NowPlayingID int64
	QueueIDs     []int64
}

// playerTools expose the user's live now-playing and queue (passed per request;
// now-playing falls back to persisted state when not supplied).
func (s *Service) playerTools(userID int64, pc PlayerContext) []agent.Tool {
	return []agent.Tool{
		agent.NewTool(agent.ToolDef[struct{}]{
			Name:        "get_now_playing",
			Description: "Get the song the user is currently playing, if any.",
			Label:       "now playing",
			Schema:      map[string]any{"type": "object", "properties": map[string]any{}},
			Run: func(ctx context.Context, _ string, _ struct{}, _ agent.UpdateFunc) (agent.ToolResult, error) {
				id := pc.NowPlayingID
				if id == 0 {
					_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(current_song_id, 0) FROM player_state WHERE user_id = ?`, userID).Scan(&id)
				}
				if id == 0 {
					return jsonResult(map[string]any{"now_playing": nil}), nil
				}
				b, err := s.songBriefByID(ctx, id)
				if err != nil {
					return jsonResult(map[string]any{"now_playing": nil}), nil
				}
				return jsonResult(map[string]any{"now_playing": b}), nil
			},
		}),
		agent.NewTool(agent.ToolDef[struct{}]{
			Name:        "get_queue",
			Description: "Get the songs in the play queue, in order. Check this before adding, removing, or reordering queue items.",
			Label:       "queue",
			Schema:      map[string]any{"type": "object", "properties": map[string]any{}},
			Run: func(ctx context.Context, _ string, _ struct{}, _ agent.UpdateFunc) (agent.ToolResult, error) {
				ids := pc.QueueIDs
				if len(ids) > 100 {
					ids = ids[:100]
				}
				briefs := s.briefsByIDs(ctx, ids)
				return jsonResult(map[string]any{"queue": briefs, "count": len(briefs)}), nil
			},
		}),
	}
}

// Chat runs the assistant for one user turn, replaying prior history, and
// streams output through sink.
func (s *Service) Chat(ctx context.Context, userID int64, history []ChatMessage, userText string, pc PlayerContext, sink ChatSink) error {
	tools := append(s.readTools(userID), s.playerTools(userID, pc)...)
	tools = append(tools, s.effectTools(userID)...)
	tools = append(tools, s.listeningStatsTool(userID))
	tools = append(tools, s.uiTool())
	agentCtx := &agent.Context{SystemPrompt: askSystem, Tools: tools}
	for _, m := range history {
		if m.Role == "assistant" {
			agentCtx.Messages = append(agentCtx.Messages, &llm.AssistantMessage{Content: []llm.Content{&llm.Text{Text: m.Text}}})
		} else {
			agentCtx.Messages = append(agentCtx.Messages, &llm.UserMessage{Content: []llm.Content{&llm.Text{Text: m.Text}}})
		}
	}

	cfg := &agent.Config{
		Model:         s.model,
		Reasoning:     s.reasoning,
		ToolExecution: agent.ModeSequential,
		Options: llm.StreamOptions{
			APIKey:         s.apiKey,
			CacheRetention: llm.CacheShort,
			MaxTokens:      s.model.MaxTokens,
		},
	}

	emit := func(e agent.Event) {
		switch ev := e.(type) {
		case agent.MessageUpdate:
			if d, ok := ev.Event.(llm.TextDeltaEvent); ok && sink.OnText != nil {
				sink.OnText(d.Delta)
			}
		case agent.ToolExecutionStart:
			if sink.OnTool != nil {
				sink.OnTool(ev.ToolName, "start")
			}
		case agent.ToolExecutionEnd:
			if eff, ok := ev.Result.Details.(Effect); ok && sink.OnEffect != nil {
				sink.OnEffect(eff)
			}
			if sink.OnTool != nil {
				sink.OnTool(ev.ToolName, "end")
			}
		}
	}

	msgs := agent.Run(ctx, []agent.AgentMessage{llm.TextUser(userText)}, agentCtx, cfg, emit)
	for _, m := range msgs {
		if am, ok := m.(*llm.AssistantMessage); ok && (am.StopReason == llm.StopError || am.StopReason == llm.StopAborted) {
			if am.ErrorMessage != "" {
				return errors.New(am.ErrorMessage)
			}
			return errors.New("ai: chat run failed")
		}
	}
	return nil
}
