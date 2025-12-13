package models

import "time"

type Playlist struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Public      bool      `json:"public"`
	CreatedAt   time.Time `json:"created_at"`
	Owner       *User     `json:"owner,omitempty"`
	SongCount   int       `json:"song_count,omitempty"`
	Songs       []Song    `json:"songs,omitempty"`
}
