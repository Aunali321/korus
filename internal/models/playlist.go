package models

import "time"

// PlaylistOwner is the public identity of a playlist's owner. Playlists are
// visible to other users when public, so this carries no private user fields.
type PlaylistOwner struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Playlist struct {
	ID          int64          `json:"id"`
	UserID      int64          `json:"user_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Public      bool           `json:"public"`
	CreatedAt   time.Time      `json:"created_at"`
	CoverPath   string         `json:"cover_path,omitempty"`
	Owner       *PlaylistOwner `json:"owner,omitempty"`
	SongCount   int            `json:"song_count"`
	// FirstSongID lets a client fall back to the first track's artwork when the
	// playlist has no cover of its own.
	FirstSongID *int64 `json:"first_song_id,omitempty"`
	Songs       []Song `json:"songs,omitempty"`
}
