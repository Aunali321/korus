package models

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

type StatsResponse struct {
	Period            PeriodRange       `json:"period"`
	Overview          StatsOverview     `json:"overview"`
	TopSongs          []RankedSong      `json:"top_songs"`
	TopArtists        []RankedArtist    `json:"top_artists"`
	TopAlbums         []RankedAlbum     `json:"top_albums"`
	ListeningPatterns ListeningPatterns `json:"listening_patterns"`
	Discovery         DiscoveryStats    `json:"discovery"`
}

type PeriodRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
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
