package tests

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Aunali321/korus/internal/services"
)

func newStatsService(t *testing.T, database *sql.DB) *services.StatsService {
	t.Helper()
	return services.NewStatsService(database, services.NewLibraryService(database))
}

// play records one listen at a given time, the way the app records it.
func play(t *testing.T, database *sql.DB, userID, songID int64, at time.Time, listened int, completion float64) {
	t.Helper()
	_, err := database.ExecContext(context.Background(),
		`INSERT INTO play_history(user_id, song_id, played_at, duration_listened, completion_rate) VALUES(?, ?, ?, ?, ?)`,
		userID, songID, at.UTC().Format("2006-01-02 15:04:05"), listened, completion)
	if err != nil {
		t.Fatalf("seed play: %v", err)
	}
}

// A play made earlier today has to appear in period=today. The old handler
// compared SQLite's "2026-08-15 20:32:15" against an RFC3339 local-offset
// string, which silently excluded it.
func TestTodayIncludesTodaysPlays(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	stats := newStatsService(t, database)

	play(t, database, userID, songIDs[0], time.Now(), 120, 1.0)

	for _, name := range []string{"today", "hour", "week", "month", "year", "all_time", ""} {
		period := services.ResolvePeriod(name)
		overview, err := stats.Overview(ctx, userID, period)
		if err != nil {
			t.Fatalf("overview %q: %v", name, err)
		}
		if overview.TotalPlays != 1 {
			t.Fatalf("period %q counted %d plays, want 1", name, overview.TotalPlays)
		}
	}
}

func TestPeriodExcludesOlderPlays(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	stats := newStatsService(t, database)

	play(t, database, userID, songIDs[0], time.Now().AddDate(0, 0, -10), 60, 1.0)

	week, err := stats.Overview(ctx, userID, services.ResolvePeriod("week"))
	if err != nil {
		t.Fatalf("week: %v", err)
	}
	if week.TotalPlays != 0 {
		t.Fatalf("week counted %d plays, want 0", week.TotalPlays)
	}

	month, err := stats.Overview(ctx, userID, services.ResolvePeriod("month"))
	if err != nil {
		t.Fatalf("month: %v", err)
	}
	if month.TotalPlays != 1 {
		t.Fatalf("month counted %d plays, want 1", month.TotalPlays)
	}
}

func TestStatsReport(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	stats := newStatsService(t, database)

	now := time.Now()
	for i := 0; i < 3; i++ {
		play(t, database, userID, songIDs[0], now.Add(-time.Duration(i)*time.Hour), 180, 1.0)
	}
	play(t, database, userID, songIDs[2], now, 200, 0.8)

	report, err := stats.Report(ctx, userID, services.ResolvePeriod("week"))
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if report.TotalPlays != 4 {
		t.Fatalf("total_plays = %d, want 4", report.TotalPlays)
	}
	if report.TotalDuration != 3*180+200 {
		t.Fatalf("total_duration = %d", report.TotalDuration)
	}
	if report.UniqueSongs != 2 || report.UniqueArtists != 2 || report.UniqueAlbums != 2 {
		t.Fatalf("uniques = %d songs, %d artists, %d albums", report.UniqueSongs, report.UniqueArtists, report.UniqueAlbums)
	}
	if len(report.TopSongs) != 2 || report.TopSongs[0].Song.ID != songIDs[0] || report.TopSongs[0].PlayCount != 3 {
		t.Fatalf("top_songs = %+v", report.TopSongs)
	}
	// The nested song must be the same shape the rest of the API returns.
	top := report.TopSongs[0].Song
	if top.Album == nil || len(top.Artists) == 0 || top.Duration == nil {
		t.Fatalf("top song is not fully hydrated: %+v", top)
	}
	if *top.Duration != 180 {
		t.Fatalf("duration = %d seconds, want 180 (not milliseconds)", *top.Duration)
	}
	if len(report.TopArtists) != 2 || report.TopArtists[0].PlayCount != 3 || report.TopArtists[0].UniqueSongs != 1 {
		t.Fatalf("top_artists = %+v", report.TopArtists)
	}
	if len(report.TopAlbums) != 2 || report.TopAlbums[0].PlayCount != 3 {
		t.Fatalf("top_albums = %+v", report.TopAlbums)
	}
	if len(report.ListeningPatterns.ByHour) == 0 {
		t.Fatalf("expected hourly buckets")
	}
	if report.Discovery.NewSongs != 2 || report.Discovery.NewArtists != 2 {
		t.Fatalf("discovery = %+v", report.Discovery)
	}
	if report.TopGenres == nil {
		t.Fatalf("top_genres must serialise as [] rather than null")
	}
}

func TestDiscoveryExcludesPreviouslyHeard(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	stats := newStatsService(t, database)

	// Heard long ago, then again this week: not a discovery.
	play(t, database, userID, songIDs[0], time.Now().AddDate(0, 0, -60), 60, 1.0)
	play(t, database, userID, songIDs[0], time.Now(), 60, 1.0)
	// First time ever this week: a discovery.
	play(t, database, userID, songIDs[2], time.Now(), 60, 1.0)

	week := services.ResolvePeriod("week")
	overview, err := stats.Overview(ctx, userID, week)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	discovery, err := stats.Discovery(ctx, userID, week, overview.TotalPlays)
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	if discovery.NewSongs != 1 || discovery.NewArtists != 1 {
		t.Fatalf("discovery = %+v, want 1 new song and 1 new artist", discovery)
	}
}

func TestSummaryMatchesReport(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	stats := newStatsService(t, database)

	now := time.Now()
	for i := 0; i < 4; i++ {
		play(t, database, userID, songIDs[0], now.Add(-time.Duration(i)*time.Hour), 150, 1.0)
	}

	period := services.ResolvePeriod("month")
	report, err := stats.Report(ctx, userID, period)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	summary, err := stats.Summary(ctx, userID, period, 8, 6)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	// The whole point of one service: the assistant and the app agree.
	if summary.TotalPlays != report.TotalPlays {
		t.Fatalf("plays differ: summary %d, report %d", summary.TotalPlays, report.TotalPlays)
	}
	if summary.TotalMinutes != report.TotalDuration/60 {
		t.Fatalf("minutes = %d, want %d", summary.TotalMinutes, report.TotalDuration/60)
	}
	if summary.UniqueSongs != report.UniqueSongs || summary.UniqueArtists != report.UniqueArtists {
		t.Fatalf("uniques differ")
	}
	if summary.DaysListened != 1 {
		t.Fatalf("days_listened = %d, want 1", summary.DaysListened)
	}
	if summary.AvgPlaysPerDay != 4 {
		t.Fatalf("avg_plays_per_day = %v, want 4", summary.AvgPlaysPerDay)
	}
	if len(summary.TopSongs) != 1 || summary.TopSongs[0].Artist == nil {
		t.Fatalf("top_songs = %+v, want one entry with an artist", summary.TopSongs)
	}
	if summary.TopSongs[0].Title != report.TopSongs[0].Song.Title {
		t.Fatalf("summary and report disagree on the top song")
	}
}

func TestCalendarPeriod(t *testing.T) {
	month, err := services.CalendarPeriod("month", "2026-08")
	if err != nil {
		t.Fatalf("month: %v", err)
	}
	if month.Label != "August 2026" {
		t.Fatalf("label = %q", month.Label)
	}
	if month.Start.Month() != time.August || month.End.Month() != time.September {
		t.Fatalf("bounds = %v..%v", month.Start, month.End)
	}

	year, err := services.CalendarPeriod("year", "2026")
	if err != nil {
		t.Fatalf("year: %v", err)
	}
	if year.Label != "2026" || year.End.Year() != 2027 {
		t.Fatalf("year = %+v", year)
	}

	for _, bad := range [][2]string{{"month", "2026-13"}, {"month", "nonsense"}, {"year", "12"}, {"decade", "2020"}} {
		if _, err := services.CalendarPeriod(bad[0], bad[1]); err == nil {
			t.Fatalf("CalendarPeriod(%q, %q) should fail", bad[0], bad[1])
		}
	}
}

func TestStreaks(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	stats := newStatsService(t, database)

	now := time.Now().UTC()
	// Three days up to today, then a gap, then a four-day run.
	for _, back := range []int{0, 1, 2, 10, 11, 12, 13} {
		play(t, database, userID, songIDs[0], now.AddDate(0, 0, -back), 60, 1.0)
	}

	insights, err := stats.Streaks(ctx, userID)
	if err != nil {
		t.Fatalf("streaks: %v", err)
	}
	if insights.CurrentStreak != 3 {
		t.Fatalf("current streak = %d, want 3", insights.CurrentStreak)
	}
	if insights.LongestStreak != 4 {
		t.Fatalf("longest streak = %d, want 4", insights.LongestStreak)
	}
}

func TestStreaksBrokenCurrent(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, songIDs := seedLibrary(t, database)
	stats := newStatsService(t, database)

	now := time.Now().UTC()
	for _, back := range []int{5, 6} {
		play(t, database, userID, songIDs[0], now.AddDate(0, 0, -back), 60, 1.0)
	}

	insights, err := stats.Streaks(ctx, userID)
	if err != nil {
		t.Fatalf("streaks: %v", err)
	}
	if insights.CurrentStreak != 0 {
		t.Fatalf("current streak = %d, want 0 (last play was 5 days ago)", insights.CurrentStreak)
	}
	if insights.LongestStreak != 2 {
		t.Fatalf("longest streak = %d, want 2", insights.LongestStreak)
	}
}

func TestStatsWithNoHistory(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()
	ctx := context.Background()
	userID, _ := seedLibrary(t, database)
	stats := newStatsService(t, database)

	report, err := stats.Report(ctx, userID, services.ResolvePeriod("week"))
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if report.TotalPlays != 0 || len(report.TopSongs) != 0 {
		t.Fatalf("expected an empty report, got %+v", report)
	}
	insights, err := stats.Streaks(ctx, userID)
	if err != nil {
		t.Fatalf("streaks: %v", err)
	}
	if insights.CurrentStreak != 0 || insights.LongestStreak != 0 {
		t.Fatalf("streaks = %+v, want zeroes", insights)
	}
}
