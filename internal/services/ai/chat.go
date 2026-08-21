package ai

import (
	"context"

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"
)

const askSystem = `You are the in-app music assistant for a personal music library ("Korus"). You help the user explore their library, build playlists, and control playback.

- Ground every answer in the user's real library using the read tools. Only reference songs, albums, artists and playlists that appeared in a tool result, and never invent an id, a title, or a number.
- search_library finds things by name and hands you ids. get_details expands an id into what is inside it: an album's tracks, an artist's albums and biography, a playlist's songs, a song's lyrics.
- my_library is the user themselves: what they play most, what they played recently, what they liked, what they keep skipping, what they own but have never played, their playlists, and the artists they follow. Reach for "unplayed" when they ask to rediscover their own library, and "skipped" when you need to know what to avoid.
- get_player is the live player. Check it before answering anything about the current song or the queue, and always before changing the queue.
- get_listening_stats gives real numbers for a month or a year: minutes, plays, days listened, top songs and artists.
- Act with the write tools. "Play X" means play with mode "now". "Queue X" or "add X" means mode "next", which leaves the current song playing. Only use "end" if they ask for it.
- To remove, reorder, or clear queue items, read the queue with get_player, then send the queue you want with set_queue. The same applies to playlists: read with get_details, write with update_playlist in "replace" mode.
- Prefer render_ui for lists, stats, and charts. Give the card its own title (a heading node) so it speaks for itself: when the card fully answers the request, call render_ui and add NO text around it, no lead-in before and no summary after. Add at most one short sentence only if it says something the card does not, and never repeat the card's items as text.
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
	Shuffle      bool
	Repeat       string
}

// maxQueueBriefs bounds how much of a long queue is described to the model.
const maxQueueBriefs = 100

// playerTool exposes the live player in one call: what is playing, what is
// queued, and the shuffle and repeat modes. Now-playing falls back to the
// persisted state when the request did not carry it.
func (s *Service) playerTool(userID int64, pc PlayerContext) agent.Tool {
	return agent.NewTool(agent.ToolDef[struct{}]{
		Name:        "get_player",
		Description: "The live player: the song currently playing, the upcoming queue in order, and the shuffle and repeat settings. Check this before answering anything about the queue and before changing it.",
		Label:       "player",
		Schema:      map[string]any{"type": "object", "properties": map[string]any{}},
		Run: func(ctx context.Context, _ string, _ struct{}, _ agent.UpdateFunc) (agent.ToolResult, error) {
			out := map[string]any{
				"now_playing": nil,
				"shuffle":     pc.Shuffle,
				"repeat":      pc.Repeat,
			}

			id := pc.NowPlayingID
			if id == 0 {
				_ = s.db.QueryRowContext(ctx,
					`SELECT COALESCE(current_song_id, 0) FROM player_state WHERE user_id = ?`, userID).Scan(&id)
			}
			if id != 0 {
				if song, err := s.library.Song(ctx, id); err == nil {
					out["now_playing"] = brief(song)
				}
			}

			ids := pc.QueueIDs
			if len(ids) > maxQueueBriefs {
				ids = ids[:maxQueueBriefs]
			}
			songs, err := s.library.Songs(ctx, ids)
			if err != nil {
				return agent.ToolResult{}, err
			}
			out["queue"] = briefs(songs)
			out["queue_length"] = len(pc.QueueIDs)
			return jsonResult(out), nil
		},
	})
}

// Chat runs the assistant for one user turn, replaying prior history, and
// streams output through sink.
func (s *Service) Chat(ctx context.Context, userID int64, history []ChatMessage, userText string, pc PlayerContext, sink ChatSink) error {
	tools := append(s.readTools(userID), s.playerTool(userID, pc), s.listeningStatsTool(userID), s.uiTool())
	tools = append(tools, s.writeTools(userID)...)

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
		Options:       s.streamOptions(),
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
	return runError(msgs, "ai: chat run failed")
}
