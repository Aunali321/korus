package models

import (
	"time"
)

type User struct {
	ID           int        `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        *string    `json:"email,omitempty" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         string     `json:"role" db:"role"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	LastLogin    *time.Time `json:"last_login,omitempty" db:"last_login"`
}

type UserSession struct {
	ID           int       `json:"id" db:"id"`
	UserID       int       `json:"user_id" db:"user_id"`
	RefreshToken string    `json:"refresh_token" db:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Artist struct {
	ID            int     `json:"id" db:"id"`
	Name          string  `json:"name" db:"name"`
	SortName      *string `json:"sort_name,omitempty" db:"sort_name"`
	MusicBrainzID *string `json:"musicbrainz_id,omitempty" db:"musicbrainz_id"`
	AlbumCount    int     `json:"album_count,omitempty"`
	SongCount     int     `json:"song_count,omitempty"`
	Albums        []Album `json:"albums,omitempty"`
	TopTracks     []Song  `json:"topTracks,omitempty"`
}

type Album struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	ArtistID      *int      `json:"artist_id,omitempty" db:"artist_id"`
	AlbumArtistID *int      `json:"album_artist_id,omitempty" db:"album_artist_id"`
	Year          *int      `json:"year,omitempty" db:"year"`
	MusicBrainzID *string   `json:"musicbrainz_id,omitempty" db:"musicbrainz_id"`
	CoverPath     *string   `json:"cover_path" db:"cover_path"`
	DateAdded     time.Time `json:"date_added" db:"date_added"`
	Artist        *Artist   `json:"artist,omitempty"`
	AlbumArtist   *Artist   `json:"album_artist,omitempty"`
	SongCount     int       `json:"song_count,omitempty"`
	Duration      int       `json:"duration,omitempty"`
	Songs         []Song    `json:"songs,omitempty"`
}

type Song struct {
	ID           int       `json:"id" db:"id"`
	Title        string    `json:"title" db:"title"`
	AlbumID      *int      `json:"album_id,omitempty" db:"album_id"`
	ArtistID     *int      `json:"artist_id,omitempty" db:"artist_id"`
	TrackNumber  *int      `json:"track_number,omitempty" db:"track_number"`
	DiscNumber   int       `json:"disc_number" db:"disc_number"`
	Duration     int       `json:"duration" db:"duration"`
	FilePath     string    `json:"file_path" db:"file_path"`
	FileSize     int64     `json:"file_size" db:"file_size"`
	FileModified time.Time `json:"file_modified" db:"file_modified"`
	Bitrate      *int      `json:"bitrate,omitempty" db:"bitrate"`
	Format       *string   `json:"format,omitempty" db:"format"`
	CoverPath    *string   `json:"cover_path" db:"cover_path"`
	DateAdded    time.Time `json:"date_added" db:"date_added"`
	Artist       *Artist   `json:"artist,omitempty"`
	Album        *Album    `json:"album,omitempty"`
	Lyrics       []Lyrics  `json:"lyrics,omitempty"`
}

type Lyrics struct {
	ID        int       `json:"id" db:"id"`
	SongID    int       `json:"song_id" db:"song_id"`
	Content   string    `json:"content" db:"content"`
	Type      string    `json:"type" db:"type"`         // "synced" or "unsynced"
	Source    string    `json:"source" db:"source"`     // "embedded", "external_lrc", "external_txt"
	Language  string    `json:"language" db:"language"` // ISO 639-2 language codes
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Playlist struct {
	ID          int       `json:"id" db:"id"`
	UserID      int       `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	Visibility  string    `json:"visibility" db:"visibility"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	SongCount   int       `json:"song_count,omitempty"`
	Duration    int       `json:"duration,omitempty"`
}

type PlaylistSong struct {
	ID         int       `json:"id" db:"id"`
	PlaylistID int       `json:"playlist_id" db:"playlist_id"`
	SongID     int       `json:"song_id" db:"song_id"`
	Position   int       `json:"position" db:"position"`
	AddedAt    time.Time `json:"added_at" db:"added_at"`
	Song       *Song     `json:"song,omitempty"`
}

type PlayHistory struct {
	ID           int       `json:"id" db:"id"`
	UserID       int       `json:"user_id" db:"user_id"`
	SongID       int       `json:"song_id" db:"song_id"`
	PlayedAt     time.Time `json:"played_at" db:"played_at"`
	PlayDuration *int      `json:"play_duration,omitempty" db:"play_duration"`
	IPAddress    *string   `json:"ip_address,omitempty" db:"ip_address"`
	Song         *Song     `json:"song,omitempty"`
}

type ScanHistory struct {
	ID           int        `json:"id" db:"id"`
	StartedAt    time.Time  `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	SongsAdded   int        `json:"songs_added" db:"songs_added"`
	SongsUpdated int        `json:"songs_updated" db:"songs_updated"`
	SongsRemoved int        `json:"songs_removed" db:"songs_removed"`
}

type Job struct {
	ID          int        `json:"id" db:"id"`
	JobType     string     `json:"job_type" db:"job_type"`
	Payload     []byte     `json:"payload,omitempty" db:"payload"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   *time.Time `json:"created_at,omitempty" db:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	Attempts    int        `json:"attempts" db:"attempts"`
	LastError   *string    `json:"last_error,omitempty" db:"last_error"`
}

// Response types for API
type LibraryStats struct {
	TotalSongs    int `json:"totalSongs"`
	TotalArtists  int `json:"totalArtists"`
	TotalAlbums   int `json:"totalAlbums"`
	TotalDuration int `json:"totalDuration"`
}

type UserStats struct {
	TotalPlays        int      `json:"totalPlays"`
	TotalTimeListened int      `json:"totalTimeListened"`
	MostPlayedArtist  *Artist  `json:"mostPlayedArtist,omitempty"`
	MostPlayedSong    *Song    `json:"mostPlayedSong,omitempty"`
	TopGenres         []string `json:"topGenres"`
}

type HomeData struct {
	RecentlyAdded     []Album `json:"recentlyAdded"`
	RecentlyPlayed    []Song  `json:"recentlyPlayed"`
	MostPlayed        []Song  `json:"mostPlayed"`
	RecommendedAlbums []Album `json:"recommendedAlbums"`
}

type SearchResults struct {
	Songs   []Song   `json:"songs"`
	Albums  []Album  `json:"albums"`
	Artists []Artist `json:"artists"`
}
