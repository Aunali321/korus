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
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	LastLogin    *time.Time `json:"lastLogin,omitempty" db:"last_login"`
}

type UserSession struct {
	ID           int       `json:"id" db:"id"`
	UserID       int       `json:"userId" db:"user_id"`
	RefreshToken string    `json:"refreshToken" db:"refresh_token"`
	ExpiresAt    time.Time `json:"expiresAt" db:"expires_at"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
}

type Artist struct {
	ID            int     `json:"id" db:"id"`
	Name          string  `json:"name" db:"name"`
	SortName      *string `json:"sortName,omitempty" db:"sort_name"`
	MusicBrainzID *string `json:"musicbrainzId,omitempty" db:"musicbrainz_id"`
	AlbumCount    int     `json:"albumCount,omitempty"`
	SongCount     int     `json:"songCount,omitempty"`
	Albums        []Album `json:"albums,omitempty"`
	TopTracks     []Song  `json:"topTracks,omitempty"`
}

type Album struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	ArtistID      *int      `json:"artistId,omitempty" db:"artist_id"`
	AlbumArtistID *int      `json:"albumArtistId,omitempty" db:"album_artist_id"`
	Year          *int      `json:"year,omitempty" db:"year"`
	MusicBrainzID *string   `json:"musicbrainzId,omitempty" db:"musicbrainz_id"`
	CoverPath     *string   `json:"coverPath" db:"cover_path"`
	DateAdded     time.Time `json:"dateAdded" db:"date_added"`
	Artist        *Artist   `json:"artist,omitempty"`
	AlbumArtist   *Artist   `json:"albumArtist,omitempty"`
	SongCount     int       `json:"songCount,omitempty"`
	Duration      int       `json:"duration,omitempty"`
	Songs         []Song    `json:"songs,omitempty"`
}

type Song struct {
	ID           int       `json:"id" db:"id"`
	Title        string    `json:"title" db:"title"`
	AlbumID      *int      `json:"albumId,omitempty" db:"album_id"`
	ArtistID     *int      `json:"artistId,omitempty" db:"artist_id"`
	TrackNumber  *int      `json:"trackNumber,omitempty" db:"track_number"`
	DiscNumber   int       `json:"discNumber" db:"disc_number"`
	Duration     int       `json:"duration" db:"duration"`
	FilePath     string    `json:"filePath" db:"file_path"`
	FileSize     int64     `json:"fileSize" db:"file_size"`
	FileModified time.Time `json:"fileModified" db:"file_modified"`
	Bitrate      *int      `json:"bitrate,omitempty" db:"bitrate"`
	Format       *string   `json:"format,omitempty" db:"format"`
	CoverPath    *string   `json:"coverPath" db:"cover_path"`
	DateAdded    time.Time `json:"dateAdded" db:"date_added"`
	Artist       *Artist   `json:"artist,omitempty"`
	Album        *Album    `json:"album,omitempty"`
	Lyrics       []Lyrics  `json:"lyrics,omitempty"`
}

type Lyrics struct {
	ID        int       `json:"id" db:"id"`
	SongID    int       `json:"songId" db:"song_id"`
	Content   string    `json:"content" db:"content"`
	Type      string    `json:"type" db:"type"`         // "synced" or "unsynced"
	Source    string    `json:"source" db:"source"`     // "embedded", "external_lrc", "external_txt"
	Language  string    `json:"language" db:"language"` // ISO 639-2 language codes
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type Playlist struct {
	ID          int       `json:"id" db:"id"`
	UserID      int       `json:"userId" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	Visibility  string    `json:"visibility" db:"visibility"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
	SongCount   int       `json:"songCount,omitempty"`
	Duration    int       `json:"duration,omitempty"`
}

type PlaylistSong struct {
	ID         int       `json:"id" db:"id"`
	PlaylistID int       `json:"playlistId" db:"playlist_id"`
	SongID     int       `json:"songId" db:"song_id"`
	Position   int       `json:"position" db:"position"`
	AddedAt    time.Time `json:"addedAt" db:"added_at"`
	Song       *Song     `json:"song,omitempty"`
}

type PlayHistory struct {
	ID           int       `json:"id" db:"id"`
	UserID       int       `json:"userId" db:"user_id"`
	SongID       int       `json:"songId" db:"song_id"`
	PlayedAt     time.Time `json:"playedAt" db:"played_at"`
	PlayDuration *int      `json:"playDuration,omitempty" db:"play_duration"`
	IPAddress    *string   `json:"ipAddress,omitempty" db:"ip_address"`
	Song         *Song     `json:"song,omitempty"`
}

type ScanHistory struct {
	ID           int        `json:"id" db:"id"`
	StartedAt    time.Time  `json:"startedAt" db:"started_at"`
	CompletedAt  *time.Time `json:"completedAt,omitempty" db:"completed_at"`
	SongsAdded   int        `json:"songsAdded" db:"songs_added"`
	SongsUpdated int        `json:"songsUpdated" db:"songs_updated"`
	SongsRemoved int        `json:"songsRemoved" db:"songs_removed"`
}

type Job struct {
	ID          int        `json:"id" db:"id"`
	JobType     string     `json:"jobType" db:"job_type"`
	Payload     []byte     `json:"payload,omitempty" db:"payload"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   *time.Time `json:"createdAt,omitempty" db:"created_at"`
	ProcessedAt *time.Time `json:"processedAt,omitempty" db:"processed_at"`
	Attempts    int        `json:"attempts" db:"attempts"`
	LastError   *string    `json:"lastError,omitempty" db:"last_error"`
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
