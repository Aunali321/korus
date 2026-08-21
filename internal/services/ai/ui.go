package ai

import (
	"context"

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"
)

// uiCatalog documents the nested UI spec for render_ui / Wrapped. Both the web
// and Flutter renderers implement exactly these node types.
const uiCatalog = `UI spec: a nested tree. Each node is {"type": string, "props": object, "children": [node...]}.
Node types:
- "section": props {title?: string, direction?: "vertical"|"horizontal", gap?: number}. A container — put nodes in children.
- "heading": props {value: string}.
- "text": props {value: string, muted?: bool, size?: "sm"|"base"|"lg"|"xl", weight?: "normal"|"bold"}.
- "stat": props {label: string, value: string}. A large KPI with a label.
- "badge": props {value: string}.
- "song_list": props {song_ids: number[]}. Renders each as an artwork card — pass IDs only, not titles.
- "song_card": props {song_id: number}.
- "bar_chart": props {data: [{label: string, value: number}...]}.
- "button": props {label: string, action: {type: "play_now"|"queue"|"open_album"|"open_artist"|"open_playlist", song_ids?: number[], album_id?: number, artist_id?: number, playlist_id?: number, position?: "next"|"end"}}.
- "divider": props {}.
- "image": props {url: string, alt?: string}.
Use only these types and reference songs by ID only.`

type uiArgs struct {
	Spec map[string]any `json:"spec"`
}

func (s *Service) uiTool() agent.Tool {
	return agent.NewTool(agent.ToolDef[uiArgs]{
		Name:        "render_ui",
		Description: "Render a rich UI card inline in the chat (charts, stat blocks, song lists) when showing is clearer than telling. " + uiCatalog,
		Label:       "render",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"spec": map[string]any{"type": "object", "description": "Root UI node; see the format in this tool's description."},
			},
			"required": []string{"spec"},
		},
		Run: func(ctx context.Context, _ string, a uiArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
			spec, _ := s.resolveSpecSongs(ctx, a.Spec).(map[string]any)
			return agent.ToolResult{Content: []llm.Content{&llm.Text{Text: "Rendered a UI card."}}, Details: Effect{Kind: EffectUI, Spec: spec}}, nil
		},
	})
}

// resolveSpecSongs walks a spec and replaces song_ids / song_id in node props
// with resolved song briefs so clients render them without extra lookups.
func (s *Service) resolveSpecSongs(ctx context.Context, node any) any {
	switch n := node.(type) {
	case map[string]any:
		if props, ok := n["props"].(map[string]any); ok {
			if ids := toInt64Slice(props["song_ids"]); len(ids) > 0 {
				props["songs"] = s.briefsByIDs(ctx, ids)
			}
			if id := toInt64(props["song_id"]); id > 0 {
				if song, err := s.library.Song(ctx, id); err == nil {
					props["song"] = brief(song)
				}
			}
		}
		if ch, ok := n["children"].([]any); ok {
			for i := range ch {
				ch[i] = s.resolveSpecSongs(ctx, ch[i])
			}
		}
		return n
	case []any:
		for i := range n {
			n[i] = s.resolveSpecSongs(ctx, n[i])
		}
		return n
	}
	return node
}

func (s *Service) briefsByIDs(ctx context.Context, ids []int64) []songBrief {
	songs, err := s.library.Songs(ctx, ids)
	if err != nil {
		return []songBrief{}
	}
	return briefs(songs)
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	}
	return 0
}

func toInt64Slice(v any) []int64 {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(arr))
	for _, e := range arr {
		if id := toInt64(e); id > 0 {
			out = append(out, id)
		}
	}
	return out
}
