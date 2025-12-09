package models

import "time"

type PlayHistory struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	SongID           int64     `json:"song_id"`
	PlayedAt         time.Time `json:"played_at"`
	DurationListened int       `json:"duration_listened"`
	CompletionRate   float64   `json:"completion_rate"`
	Source           string    `json:"source,omitempty"`
	Song             *Song     `json:"song,omitempty"`
}
