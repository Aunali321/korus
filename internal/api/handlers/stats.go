package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/Aunali321/korus/internal/db"
)

// Stats godoc
// @Summary Stats overview
// @Tags Stats
// @Produce json
// @Param period query string false "hour|today|week|month|year|all_time"
// @Success 200 {object} map[string]interface{}
// @Router /stats [get]
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
		"total_plays":        overview["total_plays"],
		"total_duration":     overview["total_time"],
		"unique_songs":       overview["unique_songs"],
		"unique_artists":     overview["unique_artists"],
		"unique_albums":      overview["unique_albums"],
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
// @Router /stats/wrapped [get]
func (h *Handler) Wrapped(c echo.Context) error {
	user, _ := currentUser(c)
	start, end := resolvePeriod(c.QueryParam("period"))
	ctx := c.Request().Context()
	overview := h.overview(ctx, user.ID, start, end)

	startStr := start.Format(time.RFC3339)
	endStr := end.Format(time.RFC3339)

	// Get top 5 songs with artist info
	topSongs := []map[string]any{}
	songRows, err := h.db.QueryContext(ctx, `
		SELECT s.id, s.title, ar.id, ar.name, COUNT(*) as plays
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums al ON al.id = s.album_id
		JOIN artists ar ON ar.id = al.artist_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
		GROUP BY s.id
		ORDER BY plays DESC
		LIMIT 5
	`, user.ID, startStr, endStr)
	if err == nil {
		defer songRows.Close()
		for songRows.Next() {
			var songID, artistID, plays int64
			var songTitle, artistName string
			if songRows.Scan(&songID, &songTitle, &artistID, &artistName, &plays) == nil {
				topSongs = append(topSongs, map[string]any{
					"id":    songID,
					"title": songTitle,
					"artist": map[string]any{
						"id":   artistID,
						"name": artistName,
					},
					"plays": plays,
				})
			}
		}
	}

	// Get top 5 artists
	topArtists := []map[string]any{}
	artistRows, err := h.db.QueryContext(ctx, `
		SELECT ar.id, ar.name, COUNT(*) as plays
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums al ON al.id = s.album_id
		JOIN artists ar ON ar.id = al.artist_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
		GROUP BY ar.id
		ORDER BY plays DESC
		LIMIT 5
	`, user.ID, startStr, endStr)
	if err == nil {
		defer artistRows.Close()
		for artistRows.Next() {
			var artistID, plays int64
			var artistName string
			if artistRows.Scan(&artistID, &artistName, &plays) == nil {
				topArtists = append(topArtists, map[string]any{
					"id":    artistID,
					"name":  artistName,
					"plays": plays,
				})
			}
		}
	}

	// Get top 5 albums with artist info
	topAlbums := []map[string]any{}
	albumRows, err := h.db.QueryContext(ctx, `
		SELECT al.id, al.title, ar.id, ar.name, COUNT(*) as plays
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums al ON al.id = s.album_id
		JOIN artists ar ON ar.id = al.artist_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
		GROUP BY al.id
		ORDER BY plays DESC
		LIMIT 5
	`, user.ID, startStr, endStr)
	if err == nil {
		defer albumRows.Close()
		for albumRows.Next() {
			var albumID, artistID, plays int64
			var albumTitle, artistName string
			if albumRows.Scan(&albumID, &albumTitle, &artistID, &artistName, &plays) == nil {
				topAlbums = append(topAlbums, map[string]any{
					"id":    albumID,
					"title": albumTitle,
					"artist": map[string]any{
						"id":   artistID,
						"name": artistName,
					},
					"plays": plays,
				})
			}
		}
	}

	totalTime, _ := overview["total_time"].(int64)
	totalMinutes := totalTime / 60
	totalPlays, _ := overview["total_plays"].(int64)
	uniqueSongs, _ := overview["unique_songs"].(int64)
	uniqueArtists, _ := overview["unique_artists"].(int64)

	var daysListened int
	_ = h.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT DATE(played_at))
		FROM play_history
		WHERE user_id = ? AND played_at BETWEEN ? AND ?
	`, user.ID, startStr, endStr).Scan(&daysListened)

	var avgPlaysPerDay float64
	if daysListened > 0 {
		avgPlaysPerDay = float64(totalPlays) / float64(daysListened)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"period":            c.QueryParam("period"),
		"top_songs":         topSongs,
		"top_artists":       topArtists,
		"top_albums":        topAlbums,
		"total_minutes":     totalMinutes,
		"total_plays":       totalPlays,
		"days_listened":     daysListened,
		"avg_plays_per_day": avgPlaysPerDay,
		"unique_songs":      uniqueSongs,
		"unique_artists":    uniqueArtists,
		"milestones":        []string{},
	})
}

// Insights godoc
// @Summary Insights
// @Tags Stats
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /stats/insights [get]
func (h *Handler) Insights(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()
	currentStreak, longest := h.streaks(ctx, user.ID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"current_streak": currentStreak,
		"longest_streak": longest,
		"trends":         []interface{}{},
		"fun_facts":      []interface{}{},
	})
}

// Home godoc
// @Summary Home summary
// @Tags Stats
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /home [get]
func (h *Handler) Home(c echo.Context) error {
	user, _ := currentUser(c)
	ctx := c.Request().Context()
	recent, _ := db.GetSongsByRecentPlays(ctx, h.db, user.ID, 10)
	recommended, _ := db.GetSongsByTopPlayed(ctx, h.db, user.ID, 5)
	newAdditions, _ := h.fetchAlbums(ctx, 10)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"recent_plays":    recent,
		"recommendations": recommended,
		"new_additions":   newAdditions,
	})
}

func resolvePeriod(period string) (time.Time, time.Time) {
	now := time.Now()
	switch period {
	case "hour":
		return now.Add(-time.Hour), now
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
	case "all_time":
		// Return a very old date to capture all history
		start := time.Date(2000, 1, 1, 0, 0, 0, 0, now.Location())
		return start, now
	default:
		start := now.AddDate(0, 0, -30)
		return start, now
	}
}

func (h *Handler) rankSongs(ctx context.Context, userID int64, start, end time.Time, limit int) []map[string]interface{} {
	rows, err := h.db.QueryContext(ctx, `
		SELECT s.id, s.title, s.album_id, s.duration, ar.id, ar.name,
		       COUNT(*) as plays, COALESCE(SUM(ph.duration_listened),0) as total_time, COALESCE(AVG(ph.completion_rate),0) as avg_comp
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums al ON al.id = s.album_id
		JOIN artists ar ON ar.id = al.artist_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
		GROUP BY s.id, s.title, s.album_id, s.duration, ar.id, ar.name
		ORDER BY plays DESC
		LIMIT ?
	`, userID, start.Format(time.RFC3339), end.Format(time.RFC3339), limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []map[string]interface{}
	for rows.Next() {
		var id, albumID, duration, artistID int64
		var title, artistName string
		var plays, totalTime int64
		var avg float64
		if err := rows.Scan(&id, &title, &albumID, &duration, &artistID, &artistName, &plays, &totalTime, &avg); err == nil {
			res = append(res, map[string]interface{}{
				"song": map[string]interface{}{
					"id":       id,
					"title":    title,
					"album_id": albumID,
					"duration": duration,
					"artist":   map[string]interface{}{"id": artistID, "name": artistName},
				},
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
		SELECT ph.song_id, ph.played_at, s.title, s.album_id,
		       ar.id, ar.name
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums al ON al.id = s.album_id
		JOIN artists ar ON ar.id = al.artist_id
		WHERE ph.user_id = ?
		ORDER BY ph.played_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []map[string]interface{}{}
	for rows.Next() {
		var songID, albumID, artistID int64
		var playedAt, title, artistName string
		if err := rows.Scan(&songID, &playedAt, &title, &albumID, &artistID, &artistName); err == nil {
			res = append(res, map[string]interface{}{
				"id":       songID,
				"title":    title,
				"album_id": albumID,
				"artist":   map[string]interface{}{"id": artistID, "name": artistName},
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
	`, userID, start.Format(time.RFC3339), end.Format(time.RFC3339), limit)
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
		SELECT al.id, al.title, al.artist_id, ar.id, ar.name,
		       COUNT(*) as plays, COALESCE(SUM(ph.duration_listened),0) as total_time, COALESCE(AVG(ph.completion_rate),0) as comp
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		JOIN albums al ON al.id = s.album_id
		JOIN artists ar ON ar.id = al.artist_id
		WHERE ph.user_id = ? AND ph.played_at BETWEEN ? AND ?
		GROUP BY al.id, al.title, al.artist_id, ar.id, ar.name
		ORDER BY plays DESC
		LIMIT ?
	`, userID, start.Format(time.RFC3339), end.Format(time.RFC3339), limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var res []map[string]interface{}
	for rows.Next() {
		var id, artistID, artistIDFromJoin int64
		var title, artistName string
		var plays, totalTime int64
		var comp float64
		if err := rows.Scan(&id, &title, &artistID, &artistIDFromJoin, &artistName, &plays, &totalTime, &comp); err == nil {
			res = append(res, map[string]interface{}{
				"album": map[string]interface{}{
					"id":        id,
					"title":     title,
					"artist_id": artistID,
					"artist":    map[string]interface{}{"id": artistIDFromJoin, "name": artistName},
				},
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
	`, format, userID, start.Format(time.RFC3339), end.Format(time.RFC3339))
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
	startStr := start.Format(time.RFC3339)
	endStr := end.Format(time.RFC3339)

	var newSongs int
	_ = h.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT song_id) FROM play_history
		WHERE user_id = ? AND played_at BETWEEN ? AND ?
		AND song_id NOT IN (SELECT DISTINCT song_id FROM play_history WHERE user_id = ? AND played_at < ?)
	`, userID, startStr, endStr, userID, startStr).Scan(&newSongs)

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
	`, userID, startStr, endStr, userID, startStr).Scan(&newArtists)

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
	`, userID, start.Format(time.RFC3339), end.Format(time.RFC3339)).Scan(&totalPlays, &totalTime, &uniqueSongs, &uniqueArtists, &uniqueAlbums, &avgCompletion)
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
	if len(dates) == 0 {
		return 0, 0
	}

	// Check if the first date (most recent) is today or yesterday to count current streak
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	current, longest := 0, 0
	for i, d := range dates {
		if i == 0 {
			// Current streak only starts if played today or yesterday
			if d.Equal(today) || d.Equal(yesterday) {
				current = 1
			} else {
				current = 0 // Streak is broken
			}
		} else {
			prev := dates[i-1]
			diff := prev.Sub(d).Hours() / 24
			if diff == 1 {
				// Consecutive day
				if current > 0 {
					current++
				}
			} else if diff > 1 {
				// Streak broken, only track longest
				if current > longest {
					longest = current
				}
				current = 0
			}
		}
		if current > longest {
			longest = current
		}
	}
	return current, longest
}

func safeDivInt(v interface{}, divisor int) float64 {
	n, ok := v.(int64)
	if !ok || divisor == 0 {
		return 0
	}
	return float64(n) / float64(divisor)
}
