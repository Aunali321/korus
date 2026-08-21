package models

// StatsReport is the /stats payload.
type StatsReport struct {
	Period        PeriodRange    `json:"period"`
	TotalPlays    int            `json:"total_plays"`
	TotalDuration int            `json:"total_duration"`
	UniqueSongs   int            `json:"unique_songs"`
	UniqueArtists int            `json:"unique_artists"`
	UniqueAlbums  int            `json:"unique_albums"`
	TopSongs      []RankedSong   `json:"top_songs"`
	TopArtists    []RankedArtist `json:"top_artists"`
	TopAlbums     []RankedAlbum  `json:"top_albums"`
	// TopGenres is always empty: the library has no genre column to rank by.
	// The field stays because clients read it.
	TopGenres         []RankedGenre     `json:"top_genres"`
	ListeningPatterns ListeningPatterns `json:"listening_patterns"`
	Discovery         DiscoveryStats    `json:"discovery"`
}

type PeriodRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type StatsOverview struct {
	TotalPlays        int     `json:"total_plays"`
	TotalTime         int     `json:"total_time"`
	UniqueSongs       int     `json:"unique_songs"`
	UniqueArtists     int     `json:"unique_artists"`
	UniqueAlbums      int     `json:"unique_albums"`
	AvgCompletionRate float64 `json:"avg_completion_rate"`
}

type RankedSong struct {
	Song          Song    `json:"song"`
	PlayCount     int     `json:"play_count"`
	TotalTime     int     `json:"total_time"`
	AvgCompletion float64 `json:"avg_completion"`
}

type RankedArtist struct {
	Artist      Artist `json:"artist"`
	PlayCount   int    `json:"play_count"`
	TotalTime   int    `json:"total_time"`
	UniqueSongs int    `json:"unique_songs"`
}

type RankedAlbum struct {
	Album          Album   `json:"album"`
	PlayCount      int     `json:"play_count"`
	TotalTime      int     `json:"total_time"`
	CompletionRate float64 `json:"completion_rate"`
}

type RankedGenre struct {
	Genre     string `json:"genre"`
	PlayCount int    `json:"play_count"`
}

type ListeningPatterns struct {
	ByHour  []Bucket `json:"by_hour"`
	ByDay   []Bucket `json:"by_day"`
	ByMonth []Bucket `json:"by_month"`
}

type Bucket struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type DiscoveryStats struct {
	NewArtists      int     `json:"new_artists"`
	NewSongs        int     `json:"new_songs"`
	ExplorationRate float64 `json:"exploration_rate"`
}

// ListeningSummary is the headline view of one period. It backs /stats/wrapped
// and the assistant's get_listening_stats tool, so both quote the same numbers.
type ListeningSummary struct {
	Period         string         `json:"period"`
	TotalPlays     int            `json:"total_plays"`
	TotalMinutes   int            `json:"total_minutes"`
	DaysListened   int            `json:"days_listened"`
	AvgPlaysPerDay float64        `json:"avg_plays_per_day"`
	UniqueSongs    int            `json:"unique_songs"`
	UniqueArtists  int            `json:"unique_artists"`
	NewArtists     int            `json:"new_artists_discovered"`
	TopSongs       []PlayedSong   `json:"top_songs"`
	TopArtists     []PlayedArtist `json:"top_artists"`
	TopAlbums      []PlayedAlbum  `json:"top_albums"`
}

type PlayedSong struct {
	ID     int64   `json:"id"`
	Title  string  `json:"title"`
	Artist *Artist `json:"artist"`
	Plays  int     `json:"plays"`
}

type PlayedArtist struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ImagePath string `json:"image_path,omitempty"`
	Plays     int    `json:"plays"`
}

type PlayedAlbum struct {
	ID     int64   `json:"id"`
	Title  string  `json:"title"`
	Artist *Artist `json:"artist"`
	Plays  int     `json:"plays"`
}

type Insights struct {
	CurrentStreak int `json:"current_streak"`
	LongestStreak int `json:"longest_streak"`
}
