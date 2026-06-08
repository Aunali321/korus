package ai

import (
	"context"
	"fmt"

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"
)

// Effect is the structured payload a client-effect / mutate / ui tool returns in
// ToolResult.Details. The chat handler turns it into an SSE event; song IDs are
// resolved to full songs by the handler so both clients can act on them.
const (
	EffectAction = "action"
	EffectUI     = "ui"
)

type Effect struct {
	Kind       string         `json:"kind"`
	Action     string         `json:"action,omitempty"`
	SongIDs    []int64        `json:"song_ids,omitempty"`
	Position   string         `json:"position,omitempty"`
	PlaylistID int64          `json:"playlist_id,omitempty"`
	Indices    []int          `json:"indices,omitempty"`
	Order      []int64        `json:"order,omitempty"`
	Control    string         `json:"control,omitempty"`
	Spec       map[string]any `json:"spec,omitempty"`
}

type idsArgs struct {
	SongIDs []int64 `json:"song_ids"`
}

type queueArgs struct {
	SongIDs  []int64 `json:"song_ids"`
	Position string  `json:"position"`
}

type createPlaylistArgs struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SongIDs     []int64 `json:"song_ids"`
}

type addToPlaylistArgs struct {
	PlaylistID int64   `json:"playlist_id"`
	SongIDs    []int64 `json:"song_ids"`
}

type indicesArgs struct {
	Indices []int `json:"indices"`
}

type orderArgs struct {
	Order []int64 `json:"order"`
}

type controlArgs struct {
	Control string `json:"control"`
}

func actionResult(text string, eff Effect) agent.ToolResult {
	return agent.ToolResult{Content: []llm.Content{&llm.Text{Text: text}}, Details: eff}
}

// effectTools mutate state or drive the client's player. They run sequentially
// (set on the chat config) so playback actions apply in call order.
func (s *Service) effectTools(userID int64) []agent.Tool {
	songIDsProp := map[string]any{
		"type":        "array",
		"items":       map[string]any{"type": "integer"},
		"description": "Song IDs drawn from tool results.",
	}

	return []agent.Tool{
		agent.NewTool(agent.ToolDef[idsArgs]{
			Name:        "play_now",
			Description: "Replace the play queue with these songs and start playing immediately. Use when the user wants to play something right now.",
			Label:       "play",
			Schema:      map[string]any{"type": "object", "properties": map[string]any{"song_ids": songIDsProp}, "required": []string{"song_ids"}},
			Run: func(_ context.Context, _ string, a idsArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				return actionResult(fmt.Sprintf("Playing %d songs now.", len(a.SongIDs)), Effect{Kind: EffectAction, Action: "play_now", SongIDs: a.SongIDs}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[queueArgs]{
			Name:        "queue_songs",
			Description: "Add songs to the queue WITHOUT changing the currently playing song. position 'next' (default) inserts right after the current song so they play next; 'end' appends.",
			Label:       "queue",
			Schema: map[string]any{"type": "object", "properties": map[string]any{
				"song_ids": songIDsProp,
				"position": map[string]any{"type": "string", "enum": []string{"next", "end"}, "description": "Where to insert. Default 'next'."},
			}, "required": []string{"song_ids"}},
			Run: func(_ context.Context, _ string, a queueArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				pos := a.Position
				if pos != "end" {
					pos = "next"
				}
				return actionResult(fmt.Sprintf("Queued %d songs (%s).", len(a.SongIDs), pos), Effect{Kind: EffectAction, Action: "queue", SongIDs: a.SongIDs, Position: pos}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[indicesArgs]{
			Name:        "remove_from_queue",
			Description: "Remove songs from the queue by their zero-based positions. Does not affect the currently playing song.",
			Label:       "remove from queue",
			Schema: map[string]any{"type": "object", "properties": map[string]any{
				"indices": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Zero-based queue positions to remove."},
			}, "required": []string{"indices"}},
			Run: func(_ context.Context, _ string, a indicesArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				return actionResult(fmt.Sprintf("Removed %d from the queue.", len(a.Indices)), Effect{Kind: EffectAction, Action: "remove_from_queue", Indices: a.Indices}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[orderArgs]{
			Name:        "reorder_queue",
			Description: "Reorder the upcoming queue. Provide the song IDs in the new order.",
			Label:       "reorder queue",
			Schema: map[string]any{"type": "object", "properties": map[string]any{
				"order": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Song IDs in the desired new order."},
			}, "required": []string{"order"}},
			Run: func(_ context.Context, _ string, a orderArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				return actionResult("Reordered the queue.", Effect{Kind: EffectAction, Action: "reorder_queue", Order: a.Order}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[struct{}]{
			Name:        "clear_queue",
			Description: "Clear the upcoming queue (the currently playing song keeps playing).",
			Label:       "clear queue",
			Schema:      map[string]any{"type": "object", "properties": map[string]any{}},
			Run: func(_ context.Context, _ string, _ struct{}, _ agent.UpdateFunc) (agent.ToolResult, error) {
				return actionResult("Cleared the queue.", Effect{Kind: EffectAction, Action: "clear_queue"}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[controlArgs]{
			Name:        "playback_control",
			Description: "Control playback: pause, resume, next, or previous.",
			Label:       "playback",
			Schema: map[string]any{"type": "object", "properties": map[string]any{
				"control": map[string]any{"type": "string", "enum": []string{"pause", "resume", "next", "previous"}},
			}, "required": []string{"control"}},
			Run: func(_ context.Context, _ string, a controlArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				return actionResult(fmt.Sprintf("Playback: %s.", a.Control), Effect{Kind: EffectAction, Action: "playback_control", Control: a.Control}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[createPlaylistArgs]{
			Name:        "create_playlist",
			Description: "Create a new playlist for the user containing the given songs, in order.",
			Label:       "create playlist",
			Schema: map[string]any{"type": "object", "properties": map[string]any{
				"name":        map[string]any{"type": "string"},
				"description": map[string]any{"type": "string", "description": "Optional short description."},
				"song_ids":    songIDsProp,
			}, "required": []string{"name", "song_ids"}},
			Run: func(ctx context.Context, _ string, a createPlaylistArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				res, err := s.db.ExecContext(ctx, `INSERT INTO playlists(user_id, name, description, public) VALUES(?, ?, ?, 0)`, userID, a.Name, a.Description)
				if err != nil {
					return agent.ToolResult{}, err
				}
				pid, _ := res.LastInsertId()
				if err := s.addPlaylistSongs(ctx, pid, a.SongIDs, 0); err != nil {
					return agent.ToolResult{}, err
				}
				return actionResult(fmt.Sprintf("Created playlist %q with %d songs.", a.Name, len(a.SongIDs)), Effect{Kind: EffectAction, Action: "playlist_created", PlaylistID: pid}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[addToPlaylistArgs]{
			Name:        "add_to_playlist",
			Description: "Append songs to an existing playlist owned by the user.",
			Label:       "add to playlist",
			Schema: map[string]any{"type": "object", "properties": map[string]any{
				"playlist_id": map[string]any{"type": "integer"},
				"song_ids":    songIDsProp,
			}, "required": []string{"playlist_id", "song_ids"}},
			Run: func(ctx context.Context, _ string, a addToPlaylistArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				var owner int64
				if err := s.db.QueryRowContext(ctx, `SELECT user_id FROM playlists WHERE id = ?`, a.PlaylistID).Scan(&owner); err != nil || owner != userID {
					return agent.ToolResult{Content: []llm.Content{&llm.Text{Text: "Playlist not found or not owned by you."}}}, nil
				}
				var maxPos int
				s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), -1) FROM playlist_songs WHERE playlist_id = ?`, a.PlaylistID).Scan(&maxPos)
				if err := s.addPlaylistSongs(ctx, a.PlaylistID, a.SongIDs, maxPos+1); err != nil {
					return agent.ToolResult{}, err
				}
				return actionResult(fmt.Sprintf("Added %d songs to the playlist.", len(a.SongIDs)), Effect{Kind: EffectAction, Action: "playlist_updated", PlaylistID: a.PlaylistID}), nil
			},
		}),
	}
}

func (s *Service) addPlaylistSongs(ctx context.Context, playlistID int64, songIDs []int64, startPos int) error {
	for i, sid := range songIDs {
		if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO playlist_songs(playlist_id, song_id, position) VALUES(?, ?, ?)`, playlistID, sid, startPos+i); err != nil {
			return err
		}
	}
	return nil
}
