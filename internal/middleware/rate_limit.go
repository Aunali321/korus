package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*clientRate
	rate    int           // requests per window
	window  time.Duration // time window
	cleanup time.Duration // cleanup interval
}

type clientRate struct {
	lastSeen time.Time
	count    int
	window   time.Time
}

func NewRateLimiter(rate int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		clients: make(map[string]*clientRate),
		rate:    rate,
		window:  window,
		cleanup: window * 2,
	}

	// Start cleanup goroutine
	go rl.cleanupClients()

	return rl
}

func (rl *rateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[clientID]

	if !exists {
		rl.clients[clientID] = &clientRate{
			lastSeen: now,
			count:    1,
			window:   now,
		}
		return true
	}

	// Check if we're in a new window
	if now.Sub(client.window) >= rl.window {
		client.count = 1
		client.window = now
		client.lastSeen = now
		return true
	}

	// Check if rate limit exceeded
	if client.count >= rl.rate {
		client.lastSeen = now
		return false
	}

	client.count++
	client.lastSeen = now
	return true
}

func (rl *rateLimiter) cleanupClients() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for id, client := range rl.clients {
			if now.Sub(client.lastSeen) > rl.cleanup {
				delete(rl.clients, id)
			}
		}
		rl.mu.Unlock()
	}
}

func RateLimit(rate int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, window)

	return func(c *gin.Context) {
		// Use IP address as client identifier
		clientID := c.ClientIP()

		// For authenticated users, use user ID for more precise limiting
		if userID, exists := c.Get("user_id"); exists {
			clientID = fmt.Sprintf("user:%v", userID)
		}

		if !limiter.Allow(clientID) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Predefined rate limiters for different endpoints
func AuthRateLimit() gin.HandlerFunc {
	return RateLimit(10, time.Minute) // 10 requests per minute for auth endpoints
}

func APIRateLimit() gin.HandlerFunc {
	return RateLimit(1000, time.Hour) // 1000 requests per hour for general API
}

func SearchRateLimit() gin.HandlerFunc {
	return RateLimit(100, time.Minute) // 100 searches per minute
}
