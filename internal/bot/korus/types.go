package korus

import (
	"fmt"
	"strings"
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type Artist struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Bio       string `json:"bio,omitempty"`
	ImagePath string `json:"image_path,omitempty"`
}

type Album struct {
	ID     int64   `json:"id"`
	Title  string  `json:"title"`
	Year   *int    `json:"year,omitempty"`
	Artist *Artist `json:"artist,omitempty"`
}

// YearText renders the release year, or an empty string when unknown.
func (a Album) YearText() string {
	if a.Year == nil || *a.Year == 0 {
		return ""
	}
	return fmt.Sprint(*a.Year)
}

func (a Album) ArtistName() string {
	if a.Artist != nil && a.Artist.Name != "" {
		return a.Artist.Name
	}
	return "Unknown artist"
}

type Song struct {
	ID          int64    `json:"id"`
	AlbumID     int64    `json:"album_id"`
	Title       string   `json:"title"`
	TrackNumber *int     `json:"track_number,omitempty"`
	Duration    *int     `json:"duration,omitempty"`
	Album       *Album   `json:"album,omitempty"`
	Artists     []Artist `json:"artists,omitempty"`
}

// Seconds is the track length, 0 when the library has no duration for it.
func (s Song) Seconds() int {
	if s.Duration == nil {
		return 0
	}
	return *s.Duration
}

// ArtistNames joins the credited artists, falling back to the album artist.
func (s Song) ArtistNames() string {
	if len(s.Artists) > 0 {
		names := make([]string, len(s.Artists))
		for i, a := range s.Artists {
			names[i] = a.Name
		}
		return strings.Join(names, ", ")
	}
	if s.Album != nil {
		return s.Album.ArtistName()
	}
	return "Unknown artist"
}

func (s Song) AlbumTitle() string {
	if s.Album == nil {
		return ""
	}
	return s.Album.Title
}

type Playlist struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Public      bool   `json:"public"`
	SongCount   int    `json:"song_count"`
	Songs       []Song `json:"songs,omitempty"`
}

type SearchResult struct {
	Songs     []Song     `json:"songs"`
	Albums    []Album    `json:"albums"`
	Artists   []Artist   `json:"artists"`
	Playlists []Playlist `json:"playlists"`
}

type AlbumDetail struct {
	ID     int64   `json:"id"`
	Title  string  `json:"title"`
	Year   *int    `json:"year"`
	Artist *Artist `json:"artist"`
	Songs  []Song  `json:"songs"`
}

type ArtistDetail struct {
	ID     int64   `json:"id"`
	Name   string  `json:"name"`
	Bio    string  `json:"bio"`
	Albums []Album `json:"albums"`
	Songs  []Song  `json:"songs"`
}

type Lyrics struct {
	Lyrics string `json:"lyrics"`
	Synced string `json:"synced"`
}

type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type RankedSong struct {
	Song      Song `json:"song"`
	PlayCount int  `json:"play_count"`
}

type RankedArtist struct {
	Artist    Artist `json:"artist"`
	PlayCount int    `json:"play_count"`
}

type RankedAlbum struct {
	Album     Album `json:"album"`
	PlayCount int   `json:"play_count"`
}

type Stats struct {
	Period        Period         `json:"period"`
	TotalPlays    int            `json:"total_plays"`
	TotalDuration int            `json:"total_duration"`
	UniqueSongs   int            `json:"unique_songs"`
	UniqueArtists int            `json:"unique_artists"`
	UniqueAlbums  int            `json:"unique_albums"`
	TopSongs      []RankedSong   `json:"top_songs"`
	TopArtists    []RankedArtist `json:"top_artists"`
	TopAlbums     []RankedAlbum  `json:"top_albums"`
}

type PlayedSong struct {
	ID     int64   `json:"id"`
	Title  string  `json:"title"`
	Artist *Artist `json:"artist"`
	Plays  int     `json:"plays"`
}

func (p PlayedSong) ArtistName() string {
	if p.Artist != nil && p.Artist.Name != "" {
		return p.Artist.Name
	}
	return "Unknown artist"
}

type PlayedArtist struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Plays int    `json:"plays"`
}

type Summary struct {
	TotalPlays     int            `json:"total_plays"`
	TotalMinutes   int            `json:"total_minutes"`
	DaysListened   int            `json:"days_listened"`
	AvgPlaysPerDay float64        `json:"avg_plays_per_day"`
	UniqueSongs    int            `json:"unique_songs"`
	UniqueArtists  int            `json:"unique_artists"`
	NewArtists     int            `json:"new_artists_discovered"`
	TopSongs       []PlayedSong   `json:"top_songs"`
	TopArtists     []PlayedArtist `json:"top_artists"`
}

type HistoryEntry struct {
	SongID   int64  `json:"song_id"`
	PlayedAt string `json:"played_at"`
	Song     struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	} `json:"song"`
}

// Listen is one recorded play, posted back to Korus after a track finishes.
type Listen struct {
	SongID           int64   `json:"song_id"`
	DurationListened int     `json:"duration_listened"`
	CompletionRate   float64 `json:"completion_rate"`
	Source           string  `json:"source"`
}
