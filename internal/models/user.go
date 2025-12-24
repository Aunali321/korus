package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Onboarded    bool      `json:"onboarded"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
}
