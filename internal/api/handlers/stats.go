package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Stats godoc
// @Summary Stats overview
// @Tags Stats
// @Produce json
// @Param period query string false "hour|today|week|month|year|all_time"
// @Success 200 {object} map[string]interface{}
// @Router /api/stats [get]
func (h *Handler) Stats(c echo.Context) error {
	user, _ := currentUser(c)
	period := c.QueryParam("period")
	start, end := resolvePeriod(period)
	ctx := c.Request().Context()

	overview := h.overview(ctx, user.ID, start, end)

	topSongs := h.rankSongs(ctx, user.ID, start, end, 10)
	topArtists := h.rankArtists(ctx, user.ID, start, end, 10)
	topAlbums := h.rankAlbums(ctx, user.ID, start, end, 10)
	topGenres := h.rankGenres(ctx, user.ID, start, end, 5)
	patterns := h.listeningPatterns(ctx, user.ID, start, end)
	discovery := h.discoveryStats(ctx, user.ID, start, end, overview)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"period":             map[string]string{"start": start.Format(time.RFC3339), "end": end.Format(time.RFC3339)},
		"overview":           overview,
		"top_songs":          topSongs,
		"top_artists":        topArtists,
		"top_albums":         topAlbums,
		"top_genres":         topGenres,
		"listening_patterns": patterns,
		"discovery":          discovery,
	})
}

// Wrapped godoc
// @Summary Wrapped stats
// @Tags Stats
// @Produce json
// @Param period query string false "year|all_time"
// @Success 200 {object} map[string]interface{}
// @Router /api/stats/wrapped [get]
func (h *Handler) Wrapped(c echo.Context) error {
	user, _ := currentUser(c)
	start, end := resolvePeriod(c.QueryParam("period"))
	ctx := c.Request().Context()
	topSongs := h.rankSongs(ctx, user.ID, start, end, 10)
	topArtists := h.rankArtists(ctx, user.ID, start, end, 10)
	topAlbums := h.rankAlbums(ctx, user.ID, start, end, 10)
	overview := h.overview(ctx, user.ID, start, end)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"period": map[string]string{"start": start.Format("2006-01-02"), "end": end.Format("2006-01-02")},
		"summary": map[string]interface{}{
			"total_plays":       overview["total_plays"],
			"total_time":        overview["total_time"],
			"days_listened":     int(end.Sub(start).Hours()) / 24,
			"avg_plays_per_day": safeDivInt(overview["total_plays"], int(end.Sub(start).Hours()/24+1)),
		},
		"top_songs":   topSongs,
		"top_artists": topArtists,
		"top_albums":  topAlbums,
		"milestones":  []interface{}{},
	})
}

// Insights godoc
// @Summary Insights
// @Tags Stats
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/stats/insights [get]
func (h *Handler) Insights(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()
	currentStreak, longest := h.streaks(ctx, user.ID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"current_streak": map[string]interface{}{"days": currentStreak, "type": "play"},
		"longest_streak": map[string]interface{}{"days": longest},
		"trends":         []interface{}{},
		"fun_facts":      []interface{}{},
	})
}

// Social godoc
// @Summary Social stats
// @Tags Stats
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/stats/social [get]
func (h *Handler) Social(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()
	leaderboard := h.leaderboard(ctx, 10)
	rank := 0
	for i, item := range leaderboard {
		if uid, ok := item["user"].(map[string]interface{})["id"].(int64); ok && uid == user.ID {
			rank = i + 1
			break
		}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"your_rank":   rank,
		"total_users": len(leaderboard),
		"leaderboard": leaderboard,
		"taste_match": []interface{}{},
	})
}

// Home godoc
// @Summary Home summary
// @Tags Stats
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/home [get]
func (h *Handler) Home(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()
	recent, _ := h.recentPlays(ctx, user.ID, 10)
	recommended := h.rankSongs(ctx, user.ID, time.Now().AddDate(0, -1, 0), time.Now(), 5)
	newAdditions, _ := h.fetchSongs(ctx, 10)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"recent_plays":  recent,
		"recommended":   recommended,
		"new_additions": newAdditions,
	})
}

func resolvePeriod(period string) (time.Time, time.Time) {
	now := time.Now()
	switch period {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, start.Add(24 * time.Hour)
	case "week":
		start := now.AddDate(0, 0, -7)
		return start, now
	case "month":
		start := now.AddDate(0, -1, 0)
		return start, now
	case "year":
		start := now.AddDate(-1, 0, 0)
		return start, now
	default:
		start := now.AddDate(0, 0, -30)
		return start, now
	}
}

func (h *Handler) rankSongs(ctx context.Context, userID int64, start, end time.Time, limit int) []map[string]interface{} {
	rows, err := h.db.QueryContext(ctx, `
		SELECT s.id, s.title, COUNT(*) as plays, COALESCE(SUM(ph.duration_listened),0) as total_time, COALESCE(AVG(ph.completion_rate),0) as avg_comp
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
		GROUP BY s.id, s.title
		ORDER BY plays DESC
		LIMIT ?
	`, userID, start, end, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []map[string]interface{}
	for rows.Next() {
		var id int64
		var title string
		var plays, totalTime int64
		var avg float64
		if err := rows.Scan(&id, &title, &plays, &totalTime, &avg); err == nil {
			res = append(res, map[string]interface{}{
				"song":           map[string]interface{}{"id": id, "title": title},
				"play_count":     plays,
				"total_time":     totalTime,
				"avg_completion": avg,
			})
		}
	}
	return res
}

func (h *Handler) recentPlays(ctx context.Context, userID int64, limit int) ([]map[string]interface{}, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT ph.song_id, ph.played_at, s.title
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		WHERE ph.user_id = ?
		ORDER BY ph.played_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []map[string]interface{}
	for rows.Next() {
		var songID int64
		var playedAt, title string
		if err := rows.Scan(&songID, &playedAt, &title); err == nil {
			res = append(res, map[string]interface{}{
				"song":      map[string]interface{}{"id": songID, "title": title},
				"played_at": playedAt,
			})
		}
	}
	return res, nil
}

func (h *Handler) rankArtists(ctx context.Context, userID int64, start, end time.Time, limit int) []map[string]interface{} {
	rows, err := h.db.QueryContext(ctx, `
		SELECT a.id, a.name, COUNT(*) as plays, COALESCE(SUM(ph.duration_listened),0) as total_time, COUNT(DISTINCT s.id) as songs
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums al ON al.id = s.album_id
		JOIN artists a ON a.id = al.artist_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
		GROUP BY a.id, a.name
		ORDER BY plays DESC
		LIMIT ?
	`, userID, start, end, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []map[string]interface{}
	for rows.Next() {
		var id int64
		var name string
		var plays, totalTime, songs int64
		if err := rows.Scan(&id, &name, &plays, &totalTime, &songs); err == nil {
			res = append(res, map[string]interface{}{
				"artist":       map[string]interface{}{"id": id, "name": name},
				"play_count":   plays,
				"total_time":   totalTime,
				"unique_songs": songs,
			})
		}
	}
	return res
}

func (h *Handler) rankAlbums(ctx context.Context, userID int64, start, end time.Time, limit int) []map[string]interface{} {
	rows, err := h.db.QueryContext(ctx, `
		SELECT al.id, al.title, COUNT(*) as plays, COALESCE(SUM(ph.duration_listened),0) as total_time, COALESCE(AVG(ph.completion_rate),0) as comp
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums al ON al.id = s.album_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
		GROUP BY al.id, al.title
		ORDER BY plays DESC
		LIMIT ?
	`, userID, start, end, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []map[string]interface{}
	for rows.Next() {
		var id int64
		var title string
		var plays, totalTime int64
		var comp float64
		if err := rows.Scan(&id, &title, &plays, &totalTime, &comp); err == nil {
			res = append(res, map[string]interface{}{
				"album":           map[string]interface{}{"id": id, "title": title},
				"play_count":      plays,
				"total_time":      totalTime,
				"completion_rate": comp,
			})
		}
	}
	return res
}

func (h *Handler) rankGenres(ctx context.Context, userID int64, start, end time.Time, limit int) []map[string]interface{} {
	// Genres not modeled; return empty placeholder
	return []map[string]interface{}{}
}

func (h *Handler) listeningPatterns(ctx context.Context, userID int64, start, end time.Time) map[string][]map[string]interface{} {
	byHour := h.aggregate(ctx, userID, start, end, "%H")
	byDay := h.aggregate(ctx, userID, start, end, "%w")
	byMonth := h.aggregate(ctx, userID, start, end, "%m")
	return map[string][]map[string]interface{}{
		"by_hour":  byHour,
		"by_day":   byDay,
		"by_month": byMonth,
	}
}

func (h *Handler) aggregate(ctx context.Context, userID int64, start, end time.Time, format string) []map[string]interface{} {
	rows, err := h.db.QueryContext(ctx, `
		SELECT strftime(?, played_at) as bucket, COUNT(*) FROM play_history
		WHERE user_id = ? AND played_at BETWEEN ? AND ?
		GROUP BY bucket ORDER BY bucket
	`, format, userID, start, end)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []map[string]interface{}
	for rows.Next() {
		var label string
		var count int
		if err := rows.Scan(&label, &count); err == nil {
			res = append(res, map[string]interface{}{"label": label, "value": count})
		}
	}
	return res
}

func (h *Handler) discoveryStats(ctx context.Context, userID int64, start, end time.Time, overview map[string]interface{}) map[string]interface{} {
	var newSongs int
	_ = h.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT song_id) FROM play_history
		WHERE user_id = ? AND played_at BETWEEN ? AND ?
		AND song_id NOT IN (SELECT DISTINCT song_id FROM play_history WHERE user_id = ? AND played_at < ?)
	`, userID, start, end, userID, start).Scan(&newSongs)

	var newArtists int
	_ = h.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT a.id) FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums al ON al.id = s.album_id
		JOIN artists a ON a.id = al.artist_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
		AND a.id NOT IN (
			SELECT a2.id FROM play_history ph2
			JOIN songs s2 ON s2.id = ph2.song_id
			JOIN albums al2 ON al2.id = s2.album_id
			JOIN artists a2 ON a2.id = al2.artist_id
			WHERE ph2.user_id = ? AND ph2.played_at < ?
		)
	`, userID, start, end, userID, start).Scan(&newArtists)

	totalPlays, _ := overview["total_plays"].(int64)
	exploration := 0.0
	if totalPlays > 0 {
		exploration = float64(newSongs) / float64(totalPlays)
	}
	return map[string]interface{}{
		"new_artists":      newArtists,
		"new_songs":        newSongs,
		"exploration_rate": exploration,
	}
}

func (h *Handler) overview(ctx context.Context, userID int64, start, end time.Time) map[string]interface{} {
	var totalPlays int64
	var totalTime int64
	var uniqueSongs int64
	var uniqueArtists int64
	var uniqueAlbums int64
	var avgCompletion float64
	_ = h.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(duration_listened),0), COUNT(DISTINCT song_id), COUNT(DISTINCT a.artist_id), COUNT(DISTINCT s.album_id), COALESCE(AVG(completion_rate),0)
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums a ON a.id = s.album_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
	`, userID, start, end).Scan(&totalPlays, &totalTime, &uniqueSongs, &uniqueArtists, &uniqueAlbums, &avgCompletion)
	return map[string]interface{}{
		"total_plays":         totalPlays,
		"total_time":          totalTime,
		"unique_songs":        uniqueSongs,
		"unique_artists":      uniqueArtists,
		"unique_albums":       uniqueAlbums,
		"avg_completion_rate": avgCompletion,
	}
}

func (h *Handler) streaks(ctx context.Context, userID int64) (int, int) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT DATE(played_at) as d FROM play_history WHERE user_id = ? GROUP BY d ORDER BY d DESC
	`, userID)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()
	var dates []time.Time
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err == nil {
			if t, err := time.Parse("2006-01-02", d); err == nil {
				dates = append(dates, t)
			}
		}
	}
	current, longest := 0, 0
	prev := time.Time{}
	for _, d := range dates {
		if prev.IsZero() || prev.Sub(d) == 24*time.Hour {
			current++
		} else if prev.Sub(d) > 24*time.Hour {
			current = 1
		}
		if current > longest {
			longest = current
		}
		prev = d
	}
	return current, longest
}

func (h *Handler) leaderboard(ctx context.Context, limit int) []map[string]interface{} {
	rows, err := h.db.QueryContext(ctx, `
		SELECT u.id, u.username, COUNT(ph.id) as plays, COALESCE(SUM(ph.duration_listened),0) as total_time
		FROM users u
		LEFT JOIN play_history ph ON ph.user_id = u.id
		GROUP BY u.id, u.username
		ORDER BY plays DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []map[string]interface{}
	for rows.Next() {
		var id int64
		var username string
		var plays, totalTime int64
		if err := rows.Scan(&id, &username, &plays, &totalTime); err == nil {
			res = append(res, map[string]interface{}{
				"user":       map[string]interface{}{"id": id, "username": username},
				"play_count": plays,
				"total_time": totalTime,
			})
		}
	}
	return res
}

func safeDivInt(v interface{}, divisor int) float64 {
	n, ok := v.(int64)
	if !ok || divisor == 0 {
		return 0
	}
	return float64(n) / float64(divisor)
}
