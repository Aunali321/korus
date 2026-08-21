package ai

import (
	"context"
	"errors"
	"fmt"

	"github.com/aunali321/pi-go/agent"

	"github.com/Aunali321/korus/internal/services"
)

// Effect is the structured payload a write tool returns in ToolResult.Details.
// The chat handler turns it into an SSE event; song ids are resolved to full
// songs there so both clients can act on it without another round trip.
const (
	EffectAction = "action"
	EffectUI     = "ui"
)

type Effect struct {
	Kind       string         `json:"kind"`
	Action     string         `json:"action,omitempty"`
	SongIDs    []int64        `json:"song_ids,omitempty"`
	Mode       string         `json:"mode,omitempty"`
	PlaylistID int64          `json:"playlist_id,omitempty"`
	Control    string         `json:"control,omitempty"`
	Entity     string         `json:"entity,omitempty"`
	EntityID   int64          `json:"entity_id,omitempty"`
	On         bool           `json:"on,omitempty"`
	Spec       map[string]any `json:"spec,omitempty"`
}

type playArgs struct {
	SongIDs []int64 `json:"song_ids"`
	Mode    string  `json:"mode"`
}

type queueArgs struct {
	SongIDs []int64 `json:"song_ids"`
}

type controlArgs struct {
	Control string `json:"control"`
}

type createPlaylistArgs struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SongIDs     []int64 `json:"song_ids"`
}

type updatePlaylistArgs struct {
	PlaylistID int64   `json:"playlist_id"`
	SongIDs    []int64 `json:"song_ids"`
	Mode       string  `json:"mode"`
}

type favoriteArgs struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
	On   *bool  `json:"on"`
}

func actionResult(text string, eff Effect) agent.ToolResult {
	result := textResult(text)
	result.Details = eff
	return result
}

// writeTools change playback or the user's library. They run sequentially (set
// on the chat config) so playback actions apply in call order.
func (s *Service) writeTools(userID int64) []agent.Tool {
	songIDsProp := map[string]any{
		"type":        "array",
		"items":       map[string]any{"type": "integer"},
		"description": "Song ids drawn from tool results.",
	}

	return []agent.Tool{
		agent.NewTool(agent.ToolDef[playArgs]{
			Name: "play",
			Description: "Put songs on the player. mode 'now' replaces the queue and starts playing immediately. " +
				"mode 'next' inserts them right after the current song without interrupting it. mode 'end' appends them. " +
				"\"Play X\" means now; \"queue X\" or \"add X\" means next unless the user asks for the end.",
			Label: "play",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"song_ids": songIDsProp,
					"mode":     map[string]any{"type": "string", "enum": []string{"now", "next", "end"}, "description": "Default 'now'."},
				},
				"required": []string{"song_ids", "mode"},
			},
			Run: func(_ context.Context, _ string, a playArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				mode := a.Mode
				if mode != "next" && mode != "end" {
					mode = "now"
				}
				verb := map[string]string{"now": "Playing", "next": "Queued next", "end": "Added to the end"}[mode]
				return actionResult(fmt.Sprintf("%s: %d songs.", verb, len(a.SongIDs)),
					Effect{Kind: EffectAction, Action: "play", SongIDs: a.SongIDs, Mode: mode}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[queueArgs]{
			Name: "set_queue",
			Description: "Replace the whole upcoming queue with exactly these songs, in this order. " +
				"Read the queue with get_player first, then send back what it should become: this is how you remove songs, reorder them, or clear the queue (send an empty list). " +
				"The currently playing song keeps playing.",
			Label: "set queue",
			Schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"song_ids": songIDsProp},
				"required":   []string{"song_ids"},
			},
			Run: func(_ context.Context, _ string, a queueArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				return actionResult(fmt.Sprintf("Queue set to %d songs.", len(a.SongIDs)),
					Effect{Kind: EffectAction, Action: "set_queue", SongIDs: a.SongIDs}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[controlArgs]{
			Name: "playback_control",
			Description: "Control playback: pause, resume, skip forward or back, or change shuffle and repeat. " +
				"'stop' ends the session outright, clearing the queue and leaving nothing playing; " +
				"use it for \"stop the music\" and 'pause' when they mean to resume later.",
			Label: "playback",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"control": map[string]any{
						"type": "string",
						"enum": []string{"pause", "resume", "stop", "next", "previous",
							"shuffle_on", "shuffle_off", "repeat_off", "repeat_one", "repeat_all"},
					},
				},
				"required": []string{"control"},
			},
			Run: func(_ context.Context, _ string, a controlArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				return actionResult(fmt.Sprintf("Playback: %s.", a.Control),
					Effect{Kind: EffectAction, Action: "playback_control", Control: a.Control}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[createPlaylistArgs]{
			Name:        "create_playlist",
			Description: "Create a new playlist containing the given songs, in order. The reply gives you the new playlist's id.",
			Label:       "create playlist",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":        map[string]any{"type": "string"},
					"description": map[string]any{"type": "string", "description": "Optional short description."},
					"song_ids":    songIDsProp,
				},
				"required": []string{"name", "song_ids"},
			},
			Run: func(ctx context.Context, _ string, a createPlaylistArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				playlist, err := s.playlists.Create(ctx, userID, a.Name, a.Description, false, a.SongIDs)
				if err != nil {
					return agent.ToolResult{}, err
				}
				return actionResult(
					fmt.Sprintf("Created playlist %q (id %d) with %d songs.", playlist.Name, playlist.ID, playlist.SongCount),
					Effect{Kind: EffectAction, Action: "playlist_created", PlaylistID: playlist.ID}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[updatePlaylistArgs]{
			Name: "update_playlist",
			Description: "Change an existing playlist the user owns. mode 'append' adds these songs to the end. " +
				"mode 'replace' makes these songs the entire playlist, which is how you remove songs or reorder it: " +
				"read it with get_details first, then send back what it should become.",
			Label: "update playlist",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"playlist_id": map[string]any{"type": "integer"},
					"song_ids":    songIDsProp,
					"mode":        map[string]any{"type": "string", "enum": []string{"append", "replace"}},
				},
				"required": []string{"playlist_id", "song_ids", "mode"},
			},
			Run: func(ctx context.Context, _ string, a updatePlaylistArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				mode := services.ModeAppend
				if a.Mode == "replace" {
					mode = services.ModeReplace
				}
				if err := s.playlists.SetSongs(ctx, userID, a.PlaylistID, a.SongIDs, mode); err != nil {
					return toolFailure(err, "That playlist is not yours to change."), nil
				}
				return actionResult(fmt.Sprintf("Playlist %d updated (%s, %d songs).", a.PlaylistID, mode, len(a.SongIDs)),
					Effect{Kind: EffectAction, Action: "playlist_updated", PlaylistID: a.PlaylistID}), nil
			},
		}),

		agent.NewTool(agent.ToolDef[favoriteArgs]{
			Name:        "set_favorite",
			Description: "Like or unlike a song or album, or follow or unfollow an artist. Set on to false to undo.",
			Label:       "favorite",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type": map[string]any{"type": "string", "enum": []string{"song", "album", "artist"}},
					"id":   map[string]any{"type": "integer", "description": "Id of that kind, from an earlier tool result."},
					"on":   map[string]any{"type": "boolean", "description": "True to like or follow (default), false to undo."},
				},
				"required": []string{"type", "id"},
			},
			Run: func(ctx context.Context, _ string, a favoriteArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				on := a.On == nil || *a.On
				if err := s.library.SetFavorite(ctx, userID, services.EntityKind(a.Type), a.ID, on); err != nil {
					return toolFailure(err, fmt.Sprintf("No %s with id %d.", a.Type, a.ID)), nil
				}
				verb := map[bool]map[string]string{
					true:  {"song": "Liked", "album": "Liked", "artist": "Now following"},
					false: {"song": "Unliked", "album": "Unliked", "artist": "Unfollowed"},
				}[on][a.Type]
				return actionResult(fmt.Sprintf("%s %s %d.", verb, a.Type, a.ID),
					Effect{Kind: EffectAction, Action: "favorite_changed", Entity: a.Type, EntityID: a.ID, On: on}), nil
			},
		}),
	}
}

// toolFailure reports a rejected write back to the model as a normal result so
// it can explain or retry, rather than as an error that aborts the run.
func toolFailure(err error, message string) agent.ToolResult {
	switch {
	case errors.Is(err, services.ErrNotFound), errors.Is(err, services.ErrForbidden), errors.Is(err, services.ErrInvalid):
		return textResult(message)
	default:
		return textResult("That did not work: " + err.Error())
	}
}
