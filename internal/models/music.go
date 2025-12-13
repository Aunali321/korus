package models

import "time"

type Artist struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Bio       string    `json:"bio,omitempty"`
	ImagePath string    `json:"image_path,omitempty"`
	MBID      *string   `json:"mbid,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Album struct {
	ID        int64     `json:"id"`
	ArtistID  int64     `json:"artist_id"`
	Title     string    `json:"title"`
	Year      *int      `json:"year,omitempty"`
	CoverPath string    `json:"cover_path,omitempty"`
	MBID      *string   `json:"mbid,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Artist    *Artist   `json:"artist,omitempty"`
}

type Song struct {
	ID           int64   `json:"id"`
	AlbumID      int64   `json:"album_id"`
	Title        string  `json:"title"`
	TrackNumber  *int    `json:"track_number,omitempty"`
	Duration     *int    `json:"duration,omitempty"`
	FilePath     string  `json:"file_path"`
	Lyrics       string  `json:"lyrics,omitempty"`
	LyricsSynced string  `json:"lyrics_synced,omitempty"`
	MBID         *string `json:"mbid,omitempty"`
	Album        *Album  `json:"album,omitempty"`
	Artist       *Artist `json:"artist,omitempty"`
}
