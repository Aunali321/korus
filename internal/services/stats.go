package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Aunali321/korus/internal/models"
)

// StatsService owns every aggregation over play_history. It is the one place
// listening numbers are computed, so /stats, /stats/wrapped and the
// assistant's get_listening_stats tool cannot disagree.
type StatsService struct {
	db      *sql.DB
	library *LibraryService
}

func NewStatsService(database *sql.DB, library *LibraryService) *StatsService {
	return &StatsService{db: database, library: library}
}

// sqliteTime is how SQLite's CURRENT_TIMESTAMP writes play_history.played_at:
// UTC, space separated, no offset. Bounds must be formatted the same way or
// the string comparison silently drops rows.
const sqliteTime = "2006-01-02 15:04:05"

// Period is a half-open listening window [Start, End).
type Period struct {
	Name  string
	Label string
	Start time.Time
	End   time.Time
}

func (p Period) bounds() (string, string) {
	return p.Start.UTC().Format(sqliteTime), p.End.UTC().Format(sqliteTime)
}

// Range renders the period for the API.
func (p Period) Range() models.PeriodRange {
	return models.PeriodRange{
		Start: p.Start.Format(time.RFC3339),
		End:   p.End.Format(time.RFC3339),
	}
}

var monthNames = [...]string{"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}

// nowBound rounds up to the next whole second. played_at is stored with
// one-second granularity, so a half-open window ending at the current instant
// would drop a play recorded in this very second.
func nowBound(t time.Time) time.Time {
	return t.Truncate(time.Second).Add(time.Second)
}

// ResolvePeriod maps the API's period names onto concrete windows, defaulting
// to the last 30 days.
func ResolvePeriod(name string) Period {
	now := time.Now()
	end := nowBound(now)
	switch name {
	case "hour":
		return Period{Name: name, Label: "the last hour", Start: now.Add(-time.Hour), End: end}
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return Period{Name: name, Label: "today", Start: start, End: start.AddDate(0, 0, 1)}
	case "week":
		return Period{Name: name, Label: "the last week", Start: now.AddDate(0, 0, -7), End: end}
	case "month":
		return Period{Name: name, Label: "the last month", Start: now.AddDate(0, -1, 0), End: end}
	case "year":
		return Period{Name: name, Label: "the last year", Start: now.AddDate(-1, 0, 0), End: end}
	case "all_time":
		return Period{Name: name, Label: "all time", Start: time.Time{}, End: end}
	default:
		return Period{Name: "30d", Label: "the last 30 days", Start: now.AddDate(0, 0, -30), End: end}
	}
}

// CalendarPeriod resolves a named calendar month ("2026-08") or year ("2026").
func CalendarPeriod(periodType, key string) (Period, error) {
	loc := time.Now().Location()
	switch periodType {
	case "year":
		var y int
		if _, err := fmt.Sscanf(key, "%d", &y); err != nil || y < 1970 {
			return Period{}, fmt.Errorf("%w: bad year %q", ErrInvalid, key)
		}
		start := time.Date(y, time.January, 1, 0, 0, 0, 0, loc)
		return Period{Name: "year", Label: fmt.Sprintf("%d", y), Start: start, End: start.AddDate(1, 0, 0)}, nil
	case "month":
		var y, m int
		if _, err := fmt.Sscanf(key, "%d-%d", &y, &m); err != nil || m < 1 || m > 12 || y < 1970 {
			return Period{}, fmt.Errorf("%w: bad month %q", ErrInvalid, key)
		}
		start := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, loc)
		return Period{
			Name:  "month",
			Label: fmt.Sprintf("%s %d", monthNames[m-1], y),
			Start: start,
			End:   start.AddDate(0, 1, 0),
		}, nil
	}
	return Period{}, fmt.Errorf("%w: unknown period type %q", ErrInvalid, periodType)
}

// PlayCount is a cheap emptiness check, far lighter than a full report.
func (s *StatsService) PlayCount(ctx context.Context, userID int64, p Period) int {
	start, end := p.bounds()
	var n int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM play_history
		WHERE user_id = ? AND played_at >= ? AND played_at < ?
	`, userID, start, end).Scan(&n)
	return n
}

// Overview totals one period. unique_artists counts song-level performers, so
// compilation tracks credit their real artist rather than "Various Artists".
func (s *StatsService) Overview(ctx context.Context, userID int64, p Period) (models.StatsOverview, error) {
	start, end := p.bounds()
	var o models.StatsOverview
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(ph.duration_listened), 0), COUNT(DISTINCT ph.song_id),
		       (SELECT COUNT(DISTINCT sa.artist_id)
		        FROM play_history ph2
		        JOIN song_artists sa ON sa.song_id = ph2.song_id AND sa.role = 'primary'
		        WHERE ph2.user_id = ? AND ph2.played_at >= ? AND ph2.played_at < ?),
		       COUNT(DISTINCT s.album_id), COALESCE(AVG(ph.completion_rate), 0)
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		WHERE ph.user_id = ? AND ph.played_at >= ? AND ph.played_at < ?
	`, userID, start, end, userID, start, end).
		Scan(&o.TotalPlays, &o.TotalTime, &o.UniqueSongs, &o.UniqueArtists, &o.UniqueAlbums, &o.AvgCompletionRate)
	return o, err
}

// ranked is one row of an aggregate-then-hydrate query: an entity id, its play
// totals, and one metric whose meaning is set by the query (average completion
// for songs and albums, distinct song count for artists). The entity itself is
// fetched from the library, so stats show the same shapes as the rest of the API.
type ranked struct {
	id        int64
	plays     int
	totalTime int
	metric    float64
}

func (s *StatsService) rank(ctx context.Context, query string, args ...any) ([]ranked, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ranked{}
	for rows.Next() {
		var r ranked
		if err := rows.Scan(&r.id, &r.plays, &r.totalTime, &r.metric); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func ids(rows []ranked) []int64 {
	out := make([]int64, len(rows))
	for i, r := range rows {
		out[i] = r.id
	}
	return out
}

// TopSongs ranks songs by play count, most played first.
func (s *StatsService) TopSongs(ctx context.Context, userID int64, p Period, limit int) ([]models.RankedSong, error) {
	start, end := p.bounds()
	rows, err := s.rank(ctx, `
		SELECT ph.song_id, COUNT(*), COALESCE(SUM(ph.duration_listened), 0), COALESCE(AVG(ph.completion_rate), 0)
		FROM play_history ph
		WHERE ph.user_id = ? AND ph.played_at >= ? AND ph.played_at < ?
		GROUP BY ph.song_id
		ORDER BY COUNT(*) DESC
		LIMIT ?
	`, userID, start, end, limit)
	if err != nil {
		return nil, err
	}
	songs, err := s.library.Songs(ctx, ids(rows))
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]models.Song, len(songs))
	for _, song := range songs {
		byID[song.ID] = song
	}
	out := make([]models.RankedSong, 0, len(rows))
	for _, r := range rows {
		song, ok := byID[r.id]
		if !ok {
			continue
		}
		out = append(out, models.RankedSong{
			Song: song, PlayCount: r.plays, TotalTime: r.totalTime, AvgCompletion: r.metric,
		})
	}
	return out, nil
}

// TopArtists ranks artists by plays of songs they perform on.
func (s *StatsService) TopArtists(ctx context.Context, userID int64, p Period, limit int) ([]models.RankedArtist, error) {
	start, end := p.bounds()
	rows, err := s.rank(ctx, `
		SELECT sa.artist_id, COUNT(*), COALESCE(SUM(ph.duration_listened), 0), COUNT(DISTINCT ph.song_id)
		FROM play_history ph
		JOIN song_artists sa ON sa.song_id = ph.song_id AND sa.role = 'primary'
		WHERE ph.user_id = ? AND ph.played_at >= ? AND ph.played_at < ?
		GROUP BY sa.artist_id
		ORDER BY COUNT(*) DESC
		LIMIT ?
	`, userID, start, end, limit)
	if err != nil {
		return nil, err
	}
	artists, err := s.library.ArtistsByIDs(ctx, ids(rows))
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]models.Artist, len(artists))
	for _, a := range artists {
		byID[a.ID] = a
	}
	out := make([]models.RankedArtist, 0, len(rows))
	for _, r := range rows {
		artist, ok := byID[r.id]
		if !ok {
			continue
		}
		out = append(out, models.RankedArtist{
			Artist: artist, PlayCount: r.plays, TotalTime: r.totalTime, UniqueSongs: int(r.metric),
		})
	}
	return out, nil
}

// TopAlbums ranks albums by plays of their tracks.
func (s *StatsService) TopAlbums(ctx context.Context, userID int64, p Period, limit int) ([]models.RankedAlbum, error) {
	start, end := p.bounds()
	rows, err := s.rank(ctx, `
		SELECT s.album_id, COUNT(*), COALESCE(SUM(ph.duration_listened), 0), COALESCE(AVG(ph.completion_rate), 0)
		FROM play_history ph
		JOIN songs s ON s.id = ph.song_id
		WHERE ph.user_id = ? AND ph.played_at >= ? AND ph.played_at < ?
		GROUP BY s.album_id
		ORDER BY COUNT(*) DESC
		LIMIT ?
	`, userID, start, end, limit)
	if err != nil {
		return nil, err
	}
	albums, err := s.library.AlbumsByIDs(ctx, ids(rows))
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]models.Album, len(albums))
	for _, al := range albums {
		byID[al.ID] = al
	}
	out := make([]models.RankedAlbum, 0, len(rows))
	for _, r := range rows {
		album, ok := byID[r.id]
		if !ok {
			continue
		}
		out = append(out, models.RankedAlbum{
			Album: album, PlayCount: r.plays, TotalTime: r.totalTime, CompletionRate: r.metric,
		})
	}
	return out, nil
}

// Patterns buckets plays by hour of day, day of week and month.
func (s *StatsService) Patterns(ctx context.Context, userID int64, p Period) (models.ListeningPatterns, error) {
	var patterns models.ListeningPatterns
	var err error
	if patterns.ByHour, err = s.bucket(ctx, userID, p, "%H"); err != nil {
		return patterns, err
	}
	if patterns.ByDay, err = s.bucket(ctx, userID, p, "%w"); err != nil {
		return patterns, err
	}
	patterns.ByMonth, err = s.bucket(ctx, userID, p, "%m")
	return patterns, err
}

func (s *StatsService) bucket(ctx context.Context, userID int64, p Period, format string) ([]models.Bucket, error) {
	start, end := p.bounds()
	rows, err := s.db.QueryContext(ctx, `
		SELECT strftime(?, played_at) AS bucket, COUNT(*)
		FROM play_history
		WHERE user_id = ? AND played_at >= ? AND played_at < ?
		GROUP BY bucket
		ORDER BY bucket
	`, format, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buckets := []models.Bucket{}
	for rows.Next() {
		var b models.Bucket
		if err := rows.Scan(&b.Label, &b.Value); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

// Discovery counts what the user heard for the first time in this period.
func (s *StatsService) Discovery(ctx context.Context, userID int64, p Period, totalPlays int) (models.DiscoveryStats, error) {
	start, end := p.bounds()
	var d models.DiscoveryStats

	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT song_id) FROM play_history
		WHERE user_id = ? AND played_at >= ? AND played_at < ?
		  AND song_id NOT IN (SELECT song_id FROM play_history WHERE user_id = ? AND played_at < ?)
	`, userID, start, end, userID, start).Scan(&d.NewSongs)
	if err != nil {
		return d, err
	}

	if d.NewArtists, err = s.newArtists(ctx, userID, p); err != nil {
		return d, err
	}
	if totalPlays > 0 {
		d.ExplorationRate = float64(d.NewSongs) / float64(totalPlays)
	}
	return d, nil
}

func (s *StatsService) newArtists(ctx context.Context, userID int64, p Period) (int, error) {
	start, end := p.bounds()
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT sa.artist_id)
		FROM play_history ph
		JOIN song_artists sa ON sa.song_id = ph.song_id AND sa.role = 'primary'
		WHERE ph.user_id = ? AND ph.played_at >= ? AND ph.played_at < ?
		  AND sa.artist_id NOT IN (
		      SELECT sa2.artist_id FROM play_history ph2
		      JOIN song_artists sa2 ON sa2.song_id = ph2.song_id AND sa2.role = 'primary'
		      WHERE ph2.user_id = ? AND ph2.played_at < ?
		  )
	`, userID, start, end, userID, start).Scan(&n)
	return n, err
}

// DaysListened counts distinct calendar days with at least one play.
func (s *StatsService) DaysListened(ctx context.Context, userID int64, p Period) (int, error) {
	start, end := p.bounds()
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT DATE(played_at)) FROM play_history
		WHERE user_id = ? AND played_at >= ? AND played_at < ?
	`, userID, start, end).Scan(&n)
	return n, err
}

// Report assembles the full /stats payload.
func (s *StatsService) Report(ctx context.Context, userID int64, p Period) (models.StatsReport, error) {
	report := models.StatsReport{Period: p.Range(), TopGenres: []models.RankedGenre{}}

	overview, err := s.Overview(ctx, userID, p)
	if err != nil {
		return report, err
	}
	report.TotalPlays = overview.TotalPlays
	report.TotalDuration = overview.TotalTime
	report.UniqueSongs = overview.UniqueSongs
	report.UniqueArtists = overview.UniqueArtists
	report.UniqueAlbums = overview.UniqueAlbums

	if report.TopSongs, err = s.TopSongs(ctx, userID, p, 10); err != nil {
		return report, err
	}
	if report.TopArtists, err = s.TopArtists(ctx, userID, p, 10); err != nil {
		return report, err
	}
	if report.TopAlbums, err = s.TopAlbums(ctx, userID, p, 10); err != nil {
		return report, err
	}
	if report.ListeningPatterns, err = s.Patterns(ctx, userID, p); err != nil {
		return report, err
	}
	report.Discovery, err = s.Discovery(ctx, userID, p, overview.TotalPlays)
	return report, err
}

// Summary is the headline view of a period, shared by /stats/wrapped and the
// assistant.
func (s *StatsService) Summary(ctx context.Context, userID int64, p Period, songLimit, artistLimit int) (models.ListeningSummary, error) {
	summary := models.ListeningSummary{
		Period:     p.Label,
		TopSongs:   []models.PlayedSong{},
		TopArtists: []models.PlayedArtist{},
		TopAlbums:  []models.PlayedAlbum{},
	}

	overview, err := s.Overview(ctx, userID, p)
	if err != nil {
		return summary, err
	}
	summary.TotalPlays = overview.TotalPlays
	summary.TotalMinutes = overview.TotalTime / 60
	summary.UniqueSongs = overview.UniqueSongs
	summary.UniqueArtists = overview.UniqueArtists

	if summary.DaysListened, err = s.DaysListened(ctx, userID, p); err != nil {
		return summary, err
	}
	if summary.DaysListened > 0 {
		summary.AvgPlaysPerDay = float64(summary.TotalPlays) / float64(summary.DaysListened)
	}
	if summary.NewArtists, err = s.newArtists(ctx, userID, p); err != nil {
		return summary, err
	}

	songs, err := s.TopSongs(ctx, userID, p, songLimit)
	if err != nil {
		return summary, err
	}
	for _, r := range songs {
		played := models.PlayedSong{ID: r.Song.ID, Title: r.Song.Title, Plays: r.PlayCount}
		if len(r.Song.Artists) > 0 {
			played.Artist = &r.Song.Artists[0]
		} else if r.Song.Album != nil {
			played.Artist = r.Song.Album.Artist
		}
		summary.TopSongs = append(summary.TopSongs, played)
	}

	artists, err := s.TopArtists(ctx, userID, p, artistLimit)
	if err != nil {
		return summary, err
	}
	for _, r := range artists {
		summary.TopArtists = append(summary.TopArtists, models.PlayedArtist{
			ID: r.Artist.ID, Name: r.Artist.Name, ImagePath: r.Artist.ImagePath, Plays: r.PlayCount,
		})
	}

	albums, err := s.TopAlbums(ctx, userID, p, artistLimit)
	if err != nil {
		return summary, err
	}
	for _, r := range albums {
		summary.TopAlbums = append(summary.TopAlbums, models.PlayedAlbum{
			ID: r.Album.ID, Title: r.Album.Title, Artist: r.Album.Artist, Plays: r.PlayCount,
		})
	}
	return summary, nil
}

// Streaks returns the user's current and longest runs of consecutive days with
// a play. The current streak only counts if they listened today or yesterday.
func (s *StatsService) Streaks(ctx context.Context, userID int64) (models.Insights, error) {
	var insights models.Insights

	rows, err := s.db.QueryContext(ctx, `
		SELECT DATE(played_at) AS day FROM play_history WHERE user_id = ? GROUP BY day ORDER BY day DESC
	`, userID)
	if err != nil {
		return insights, err
	}
	defer rows.Close()

	days := []time.Time{}
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return insights, err
		}
		day, err := time.Parse("2006-01-02", text)
		if err != nil {
			continue
		}
		days = append(days, day)
	}
	if err := rows.Err(); err != nil {
		return insights, err
	}
	if len(days) == 0 {
		return insights, nil
	}

	// days descends, so each unbroken run of consecutive dates is one streak.
	runs := []int{}
	run := 1
	for i := 1; i < len(days); i++ {
		if days[i-1].AddDate(0, 0, -1).Equal(days[i]) {
			run++
			continue
		}
		runs = append(runs, run)
		run = 1
	}
	runs = append(runs, run)

	for _, length := range runs {
		if length > insights.LongestStreak {
			insights.LongestStreak = length
		}
	}

	// The newest run only counts as current if it reaches today or yesterday.
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if days[0].Equal(today) || days[0].Equal(today.AddDate(0, 0, -1)) {
		insights.CurrentStreak = runs[0]
	}
	return insights, nil
}
