package ai

import (
	"context"
	"encoding/json"

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"
)

type songBrief struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Album  string `json:"album,omitempty"`
	Year   int    `json:"year,omitempty"`
}

// briefSelect yields (id, title, artist, album, year); artist prefers the
// song's primary performer, falling back to the album artist.
const briefSelect = `
SELECT s.id, s.title,
  COALESCE(
    (SELECT a.name FROM song_artists sa JOIN artists a ON a.id = sa.artist_id
     WHERE sa.song_id = s.id AND sa.role = 'primary' ORDER BY sa.position LIMIT 1),
    ar_alb.name, 'Unknown') AS artist,
  COALESCE(al.title, '') AS album,
  COALESCE(al.year, 0) AS year
FROM songs s
LEFT JOIN albums al ON al.id = s.album_id
LEFT JOIN artists ar_alb ON ar_alb.id = al.artist_id
`

func clampLimit(n, def, max int) int {
	if n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

func (s *Service) queryBriefs(ctx context.Context, query string, args ...any) ([]songBrief, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]songBrief, 0)
	for rows.Next() {
		var b songBrief
		if err := rows.Scan(&b.ID, &b.Title, &b.Artist, &b.Album, &b.Year); err != nil {
			continue
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func jsonResult(v any) agent.ToolResult {
	b, err := json.Marshal(v)
	if err != nil {
		return agent.ToolResult{Content: []llm.Content{&llm.Text{Text: "error: " + err.Error()}}, Details: map[string]any{}}
	}
	return agent.ToolResult{Content: []llm.Content{&llm.Text{Text: string(b)}}, Details: v}
}

type searchArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type limitArgs struct {
	Limit int `json:"limit"`
}

type artistArgs struct {
	Artist string `json:"artist"`
	Limit  int    `json:"limit"`
}

// readTools returns the read-only library tools bound to a user.
func (s *Service) readTools(userID int64) []agent.Tool {
	limitProp := map[string]any{
		"limit": map[string]any{"type": "integer", "description": "Max results (default 20)."},
	}

	return []agent.Tool{
		agent.NewTool(agent.ToolDef[searchArgs]{
			Name:        "search_songs",
			Description: "Search the music library by song title, artist name, or album. Returns matching songs with their IDs.",
			Label:       "search library",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Free-text query: a title, artist, or album."},
					"limit": map[string]any{"type": "integer", "description": "Max results (default 20)."},
				},
				"required": []string{"query"},
			},
			Run: func(ctx context.Context, _ string, a searchArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				limit := clampLimit(a.Limit, 20, 100)
				pat := "%" + a.Query + "%"
				rows, err := s.queryBriefs(ctx, briefSelect+`
WHERE s.title LIKE ?
   OR al.title LIKE ?
   OR EXISTS (SELECT 1 FROM song_artists sa JOIN artists a ON a.id = sa.artist_id
              WHERE sa.song_id = s.id AND a.name LIKE ?)
ORDER BY s.id
LIMIT ?`, pat, pat, pat, limit)
				if err != nil {
					return agent.ToolResult{}, err
				}
				return jsonResult(rows), nil
			},
		}),

		agent.NewTool(agent.ToolDef[limitArgs]{
			Name:        "get_top_tracks",
			Description: "Get the user's most-played songs of all time, most-played first.",
			Label:       "top tracks",
			Schema:      map[string]any{"type": "object", "properties": limitProp},
			Run: func(ctx context.Context, _ string, a limitArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				limit := clampLimit(a.Limit, 20, 100)
				rows, err := s.queryBriefs(ctx, briefSelect+`
JOIN play_history ph ON ph.song_id = s.id
WHERE ph.user_id = ?
GROUP BY s.id
ORDER BY COUNT(*) DESC
LIMIT ?`, userID, limit)
				if err != nil {
					return agent.ToolResult{}, err
				}
				return jsonResult(rows), nil
			},
		}),

		agent.NewTool(agent.ToolDef[limitArgs]{
			Name:        "get_recent_plays",
			Description: "Get the songs the user played most recently, newest first.",
			Label:       "recent plays",
			Schema:      map[string]any{"type": "object", "properties": limitProp},
			Run: func(ctx context.Context, _ string, a limitArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				limit := clampLimit(a.Limit, 20, 100)
				rows, err := s.queryBriefs(ctx, briefSelect+`
JOIN play_history ph ON ph.song_id = s.id
WHERE ph.user_id = ?
GROUP BY s.id
ORDER BY MAX(ph.played_at) DESC
LIMIT ?`, userID, limit)
				if err != nil {
					return agent.ToolResult{}, err
				}
				return jsonResult(rows), nil
			},
		}),

		agent.NewTool(agent.ToolDef[limitArgs]{
			Name:        "get_favorites",
			Description: "Get the user's favorited (liked) songs.",
			Label:       "favorites",
			Schema:      map[string]any{"type": "object", "properties": limitProp},
			Run: func(ctx context.Context, _ string, a limitArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				limit := clampLimit(a.Limit, 50, 200)
				rows, err := s.queryBriefs(ctx, briefSelect+`
JOIN favorites_songs f ON f.song_id = s.id
WHERE f.user_id = ?
ORDER BY f.created_at DESC
LIMIT ?`, userID, limit)
				if err != nil {
					return agent.ToolResult{}, err
				}
				return jsonResult(rows), nil
			},
		}),

		agent.NewTool(agent.ToolDef[artistArgs]{
			Name:        "get_artist_songs",
			Description: "Get songs by a given artist name (matches the song's performer or the album artist).",
			Label:       "artist songs",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"artist": map[string]any{"type": "string", "description": "Artist name (partial match allowed)."},
					"limit":  map[string]any{"type": "integer", "description": "Max results (default 50)."},
				},
				"required": []string{"artist"},
			},
			Run: func(ctx context.Context, _ string, a artistArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
				limit := clampLimit(a.Limit, 50, 200)
				pat := "%" + a.Artist + "%"
				rows, err := s.queryBriefs(ctx, briefSelect+`
WHERE ar_alb.name LIKE ?
   OR EXISTS (SELECT 1 FROM song_artists sa JOIN artists a ON a.id = sa.artist_id
              WHERE sa.song_id = s.id AND a.name LIKE ?)
ORDER BY s.id
LIMIT ?`, pat, pat, limit)
				if err != nil {
					return agent.ToolResult{}, err
				}
				return jsonResult(rows), nil
			},
		}),
	}
}

// submitSongIDsTool is a terminal tool: the agent calls it once with its final
// ordered song IDs, which are written to *out, ending the run.
func submitSongIDsTool(out *[]int64) agent.Tool {
	type submitArgs struct {
		SongIDs []int64 `json:"song_ids"`
	}
	return agent.NewTool(agent.ToolDef[submitArgs]{
		Name:        "submit_songs",
		Description: "Deliver your final ordered list of chosen song IDs. Call this exactly once when finished.",
		Label:       "submit",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"song_ids": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "integer"},
					"description": "Ordered list of chosen song IDs drawn from the library.",
				},
			},
			"required": []string{"song_ids"},
		},
		Run: func(_ context.Context, _ string, a submitArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
			*out = append(*out, a.SongIDs...)
			return agent.ToolResult{Content: []llm.Content{&llm.Text{Text: "Recorded."}}, Terminate: true}, nil
		},
	})
}
