package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"
)

const wrappedSystem = `You are an art director making a SHAREABLE "Wrapped" poster for a music lover — one refined, screenshot-worthy piece about their year (or month). Mood: MOODY, SERENE, CINEMATIC.

TASTE IS EVERYTHING. Aim for the restraint of a fashion-magazine spread, an A24 film poster, a museum print, an Apple keynote slide — made by a real art director. It must NOT look like a website template.

BANNED — these are dated, tasteless "web-template" tells. Use NONE of them:
- numbers that count up / odometer effects
- content that slides or fades in from the sides; scroll-triggered reveals; parallax sections
- frosted glass / glassmorphism panels
- rainbow or purple→blue gradients on text
- a big centered hero that fades up; rows of generic rounded "stat boxes"

INSTEAD, DESIGN LIKE PRINT: confident editorial typography (strong type contrast, real hierarchy), intentional asymmetric composition, generous negative space, one cohesive moody palette pulled from their album art. Build atmosphere FROM their covers — a large cover blurred and darkened into the backdrop, vignette, fine film grain, deep cinematic tones — so the page is bathed in the colors of their music.

MOTION — AMBIENT, NOT THEATRICAL. The piece is mostly still and self-assured, like a cinemagraph: at most the grain breathes, the backdrop drifts a hair, a faint light slowly shifts — continuous, slow, barely noticeable. NO entrance animations, NO scroll effects, NO counting. When unsure, leave it still. CSS only.

SHAREABLE — NAMES VISIBLE, REAL NUMBERS. The headline numbers (minutes, plays, days, new artists, their #1 song & artist) are bold and clear, printed at their REAL values as plain text (no JS, so no counters — they'd freeze at 0). Every top song shows its TITLE + ARTIST, every top artist its NAME, with counts. Almost no prose — a line or two at most. Real data only; never invent titles, artists, numbers, or events.

TECH:
- Album covers ONLY via the given song ids: <img src="/api/artwork/SONG_ID">.
- ONE self-contained HTML body — markup plus a single <style> block, absolutely NO <script>. Responsive; great on a phone.
- Fonts: @import real Google Fonts at the top of your <style>, exact CSS2 format, e.g. @import url('https://fonts.googleapis.com/css2?family=Family+Name:wght@400;700&display=swap'); (spaces become '+', weights after 'wght@'). Never guess a name or the syntax.
- Leave the top-right corner clear (~64px) for app controls.

First call get_listening_stats for the period, then design and call submit_wrapped exactly once with the finished HTML.`

type topItem struct {
	ID     int64  `json:"id,omitempty"`
	Name   string `json:"name"`
	Artist string `json:"artist,omitempty"`
	Plays  int    `json:"plays"`
}

type wrappedStats struct {
	Period        string     `json:"period"`
	TotalPlays    int        `json:"total_plays"`
	TotalMinutes  int        `json:"total_minutes"`
	DaysListened  int        `json:"days_listened"`
	AvgPerDay     int        `json:"avg_plays_per_day"`
	UniqueSongs   int        `json:"unique_songs"`
	UniqueArtists int        `json:"unique_artists"`
	NewArtists    int        `json:"new_artists_discovered"`
	TopSongs      []topItem  `json:"top_songs"`
	TopArtists    []topItem  `json:"top_artists"`
}

// Wrapped returns the Wrapped recap for a period as a self-contained HTML body,
// serving the cache unless refresh is set. Past periods are cached forever;
// periods with no listening yield an honest empty page and are not cached (so
// they fill in once data arrives). Returns whether the result came from cache.
func (s *Service) Wrapped(ctx context.Context, userID int64, periodType, periodKey string, refresh bool) (string, bool, error) {
	if !refresh {
		if html, ok := s.CachedWrapped(ctx, userID, periodType, periodKey); ok {
			return html, true, nil
		}
	}

	start, end, label, err := periodBounds(periodType, periodKey)
	if err != nil {
		return "", false, err
	}
	if s.periodPlayCount(ctx, userID, start, end) == 0 {
		return emptyWrappedHTML(label), false, nil
	}

	var html string
	tools := []agent.Tool{s.listeningStatsTool(userID), submitWrappedTool(&html)}
	prompt := fmt.Sprintf("Create the Wrapped recap for %s. First call get_listening_stats with period_type=%q and period_key=%q to pull the real numbers (album covers live at /api/artwork/<id>), then design and submit the HTML.", label, periodType, periodKey)
	if _, err := s.run(ctx, wrappedSystem, prompt, tools, nil); err != nil {
		return "", false, err
	}
	if html == "" {
		return "", false, fmt.Errorf("ai: no wrapped html produced")
	}
	_ = s.SaveWrapped(ctx, userID, periodType, periodKey, html)
	return html, false, nil
}

// emptyWrappedHTML is the honest stand-in for a period with no listening — far
// better than letting the model fabricate a recap out of nothing.
func emptyWrappedHTML(label string) string {
	return fmt.Sprintf(`<div style="min-height:100vh;display:flex;align-items:center;justify-content:center;text-align:center;padding:2rem;font-family:system-ui,sans-serif;color:#e7e5e4">`+
		`<div style="max-width:30rem">`+
		`<div style="font-style:italic;font-size:1.1rem;color:#a8a29e">%s</div>`+
		`<h1 style="font-family:Georgia,serif;font-size:2.4rem;font-weight:600;margin:.4rem 0 1rem;color:#fafaf9">Nothing to wrap up yet</h1>`+
		`<p style="line-height:1.7;color:#a8a29e">No listening recorded for this stretch. Once you start playing, your recap fills in — top songs, the artists you leaned on, and the hours you spent here.</p>`+
		`</div></div>`, label)
}

var monthNames = [...]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

func periodBounds(periodType, periodKey string) (start, end, label string, err error) {
	switch periodType {
	case "year":
		var y int
		if _, e := fmt.Sscanf(periodKey, "%d", &y); e != nil || y < 1970 {
			return "", "", "", fmt.Errorf("bad year %q", periodKey)
		}
		return fmt.Sprintf("%04d-01-01", y), fmt.Sprintf("%04d-01-01", y+1), fmt.Sprintf("%d", y), nil
	case "month":
		var y, m int
		if _, e := fmt.Sscanf(periodKey, "%d-%d", &y, &m); e != nil || m < 1 || m > 12 {
			return "", "", "", fmt.Errorf("bad month %q", periodKey)
		}
		ny, nm := y, m+1
		if nm > 12 {
			ny, nm = y+1, 1
		}
		return fmt.Sprintf("%04d-%02d-01", y, m), fmt.Sprintf("%04d-%02d-01", ny, nm), fmt.Sprintf("%s %d", monthNames[m-1], y), nil
	}
	return "", "", "", fmt.Errorf("unknown period type %q", periodType)
}

// wrappedStats computes period totals via lexicographic date-prefix bounds,
// which work because timestamps and bounds both start with YYYY-MM-DD.
func (s *Service) wrappedStats(ctx context.Context, userID int64, label, start, end string) (wrappedStats, error) {
	w := wrappedStats{Period: label, TopSongs: []topItem{}, TopArtists: []topItem{}}

	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COALESCE(SUM(duration_listened), 0) / 60,
		       COUNT(DISTINCT DATE(played_at)),
		       COUNT(DISTINCT song_id)
		FROM play_history
		WHERE user_id = ? AND played_at >= ? AND played_at < ?
	`, userID, start, end).Scan(&w.TotalPlays, &w.TotalMinutes, &w.DaysListened, &w.UniqueSongs)
	if err != nil {
		return w, err
	}
	if w.DaysListened > 0 {
		w.AvgPerDay = w.TotalPlays / w.DaysListened
	}

	s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT sa.artist_id)
		FROM play_history ph
		JOIN song_artists sa ON sa.song_id = ph.song_id AND sa.role = 'primary'
		WHERE ph.user_id = ? AND ph.played_at >= ? AND ph.played_at < ?
	`, userID, start, end).Scan(&w.UniqueArtists)

	s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT sa.artist_id)
		FROM play_history ph
		JOIN song_artists sa ON sa.song_id = ph.song_id AND sa.role = 'primary'
		WHERE ph.user_id = ? AND ph.played_at >= ? AND ph.played_at < ?
		  AND sa.artist_id NOT IN (
		      SELECT sa2.artist_id FROM play_history ph2
		      JOIN song_artists sa2 ON sa2.song_id = ph2.song_id AND sa2.role = 'primary'
		      WHERE ph2.user_id = ? AND ph2.played_at < ?
		  )
	`, userID, start, end, userID, start).Scan(&w.NewArtists)

	songRows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.title,
		       COALESCE(
		         (SELECT a.name FROM song_artists sa JOIN artists a ON a.id = sa.artist_id
		          WHERE sa.song_id = s.id AND sa.role = 'primary' ORDER BY sa.position LIMIT 1),
		         ar_alb.name, 'Unknown'),
		       COUNT(*) AS plays
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		LEFT JOIN albums al ON al.id = s.album_id
		LEFT JOIN artists ar_alb ON ar_alb.id = al.artist_id
		WHERE ph.user_id = ? AND ph.played_at >= ? AND ph.played_at < ?
		GROUP BY s.id ORDER BY plays DESC LIMIT 8
	`, userID, start, end)
	if err == nil {
		for songRows.Next() {
			var t topItem
			if songRows.Scan(&t.ID, &t.Name, &t.Artist, &t.Plays) == nil {
				w.TopSongs = append(w.TopSongs, t)
			}
		}
		songRows.Close()
	}

	artRows, err := s.db.QueryContext(ctx, `
		SELECT ar.name, COUNT(*) AS plays
		FROM play_history ph
		JOIN song_artists sa ON sa.song_id = ph.song_id AND sa.role = 'primary'
		JOIN artists ar ON ar.id = sa.artist_id
		WHERE ph.user_id = ? AND ph.played_at >= ? AND ph.played_at < ?
		GROUP BY ar.id ORDER BY plays DESC LIMIT 6
	`, userID, start, end)
	if err == nil {
		for artRows.Next() {
			var t topItem
			if artRows.Scan(&t.Name, &t.Plays) == nil {
				w.TopArtists = append(w.TopArtists, t)
			}
		}
		artRows.Close()
	}

	return w, nil
}

// periodPlayCount is a cheap guard for the empty state — far lighter than the
// full stats query, and it lets us skip the LLM entirely when there's nothing.
func (s *Service) periodPlayCount(ctx context.Context, userID int64, start, end string) int {
	var n int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM play_history WHERE user_id = ? AND played_at >= ? AND played_at < ?`, userID, start, end).Scan(&n)
	return n
}

type statsArgs struct {
	PeriodType string `json:"period_type"`
	PeriodKey  string `json:"period_key"`
}

// listeningStatsTool exposes a period's listening stats (plays, minutes, days,
// unique songs/artists, new artists, ranked top songs and artists) as a tool, so
// the agent pulls real numbers itself. Backs Wrapped and is also handed to the
// Ask chat for listening-summary questions.
func (s *Service) listeningStatsTool(userID int64) agent.Tool {
	return agent.NewTool(agent.ToolDef[statsArgs]{
		Name:        "get_listening_stats",
		Description: `Get the user's listening stats for a period: total plays and minutes, days listened, unique songs/artists, new artists discovered, and ranked top songs and artists. period_type is "month" or "year"; period_key is "YYYY-MM" (month) or "YYYY" (year), defaulting to the current period.`,
		Label:       "listening stats",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"period_type": map[string]any{"type": "string", "enum": []string{"month", "year"}, "description": `"month" or "year".`},
				"period_key":  map[string]any{"type": "string", "description": `"YYYY-MM" for a month or "YYYY" for a year. Defaults to the current period.`},
			},
		},
		Run: func(ctx context.Context, _ string, a statsArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
			periodType := a.PeriodType
			if periodType != "year" {
				periodType = "month"
			}
			periodKey := a.PeriodKey
			if periodKey == "" {
				now := time.Now()
				if periodType == "year" {
					periodKey = now.Format("2006")
				} else {
					periodKey = now.Format("2006-01")
				}
			}
			start, end, label, err := periodBounds(periodType, periodKey)
			if err != nil {
				return jsonResult(map[string]any{"error": err.Error()}), nil
			}
			stats, err := s.wrappedStats(ctx, userID, label, start, end)
			if err != nil {
				return agent.ToolResult{}, err
			}
			return jsonResult(stats), nil
		},
	})
}

type submitHTMLArgs struct {
	HTML string `json:"html"`
}

func submitWrappedTool(out *string) agent.Tool {
	return agent.NewTool(agent.ToolDef[submitHTMLArgs]{
		Name:        "submit_wrapped",
		Description: "Deliver the finished, self-contained Wrapped recap as an HTML body (markup plus an optional single <style> block, no scripts). Call exactly once.",
		Label:       "submit",
		Schema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"html": map[string]any{"type": "string", "description": "The complete HTML body."}},
			"required":   []string{"html"},
		},
		Run: func(_ context.Context, _ string, a submitHTMLArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
			*out = a.HTML
			return agent.ToolResult{Content: []llm.Content{&llm.Text{Text: "Saved."}}, Terminate: true}, nil
		},
	})
}
