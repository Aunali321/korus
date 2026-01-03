package api

import (
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/Aunali321/korus/internal/api/handlers"
	"github.com/Aunali321/korus/internal/api/middleware"
	"github.com/Aunali321/korus/internal/api/validators"
	"github.com/Aunali321/korus/internal/services"
)

type Deps struct {
	DB                *sql.DB
	Auth              *services.AuthService
	Scanner           *services.ScannerService
	Search            *services.SearchService
	Transcoder        *services.Transcoder
	MusicBrainz       *services.MusicBrainzService
	ListenBrainz      *services.ListenBrainzService
	Radio             *services.RadioService
	MediaRoot         string
	AuthRate          int
	AuthWindow        time.Duration
	WebDistPath       string
	RadioDefaultLimit int
}

func New(deps Deps) *echo.Echo {
	e := echo.New()
	e.Validator = validators.New()
	e.HideBanner = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if he, ok := err.(*echo.HTTPError); ok {
			c.JSON(he.Code, he.Message)
			return
		}
		c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal", "code": "INTERNAL_ERROR"})
	}

	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.CORS())

	h := handlers.New(deps.DB, deps.Auth, deps.Scanner, deps.Search, deps.Transcoder, deps.MusicBrainz, deps.ListenBrainz, deps.Radio, deps.MediaRoot, deps.RadioDefaultLimit)

	api := e.Group("/api")
	api.GET("/health", h.Health)

	window := deps.AuthWindow
	if window <= 0 {
		window = time.Minute
	}
	rateInt := max(deps.AuthRate, 1)
	authLimiter := echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
		Skipper:             echomw.DefaultSkipper,
		Store:               echomw.NewRateLimiterMemoryStoreWithConfig(echomw.RateLimiterMemoryStoreConfig{Rate: rate.Limit(float64(rateInt) / window.Seconds()), Burst: rateInt}),
		IdentifierExtractor: func(c echo.Context) (string, error) { return c.RealIP(), nil },
	})
	authGroup := api.Group("/auth", authLimiter)
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)
	authGroup.POST("/refresh", h.Refresh)
	authGroup.POST("/logout", h.Logout, middleware.Auth(deps.Auth))
	authGroup.GET("/me", h.Me, middleware.Auth(deps.Auth))
	authGroup.POST("/onboarded", h.CompleteOnboarding, middleware.Auth(deps.Auth))

	api.GET("/library", h.Library, middleware.Auth(deps.Auth))
	api.GET("/artists/:id", h.Artist, middleware.Auth(deps.Auth))
	api.GET("/albums/:id", h.Album, middleware.Auth(deps.Auth))
	api.GET("/songs/:id", h.Song, middleware.Auth(deps.Auth))
	api.GET("/search", h.Search, middleware.Auth(deps.Auth))

	api.GET("/stream/:id", h.Stream, middleware.Auth(deps.Auth))
	api.GET("/streaming/options", h.StreamingOptions, middleware.Auth(deps.Auth))
	api.GET("/artwork/:id", h.Artwork)
	api.GET("/lyrics/:id", h.Lyrics, middleware.Auth(deps.Auth))

	api.GET("/playlists", h.ListPlaylists, middleware.Auth(deps.Auth))
	api.POST("/playlists", h.CreatePlaylist, middleware.Auth(deps.Auth))
	api.GET("/playlists/:id", h.GetPlaylist, middleware.Auth(deps.Auth))
	api.PUT("/playlists/:id", h.UpdatePlaylist, middleware.Auth(deps.Auth))
	api.DELETE("/playlists/:id", h.DeletePlaylist, middleware.Auth(deps.Auth))
	api.POST("/playlists/:id/songs", h.AddPlaylistSong, middleware.Auth(deps.Auth))
	api.DELETE("/playlists/:id/songs/:song_id", h.DeletePlaylistSong, middleware.Auth(deps.Auth))
	api.PUT("/playlists/:id/reorder", h.ReorderPlaylistSongs, middleware.Auth(deps.Auth))
	api.POST("/playlists/:id/cover", h.UploadPlaylistCover, middleware.Auth(deps.Auth))
	api.GET("/playlists/:id/cover", h.GetPlaylistCover)

	api.POST("/favorites/songs/:id", h.FavSong, middleware.Auth(deps.Auth))
	api.DELETE("/favorites/songs/:id", h.UnfavSong, middleware.Auth(deps.Auth))
	api.POST("/favorites/albums/:id", h.FavAlbum, middleware.Auth(deps.Auth))
	api.DELETE("/favorites/albums/:id", h.UnfavAlbum, middleware.Auth(deps.Auth))
	api.POST("/follows/artists/:id", h.FollowArtist, middleware.Auth(deps.Auth))
	api.DELETE("/follows/artists/:id", h.UnfollowArtist, middleware.Auth(deps.Auth))
	api.GET("/favorites", h.ListFavorites, middleware.Auth(deps.Auth))

	api.POST("/history", h.RecordHistory, middleware.Auth(deps.Auth))
	api.GET("/history", h.ListHistory, middleware.Auth(deps.Auth))

	api.GET("/stats", h.Stats, middleware.Auth(deps.Auth))
	api.GET("/stats/wrapped", h.Wrapped, middleware.Auth(deps.Auth))
	api.GET("/stats/insights", h.Insights, middleware.Auth(deps.Auth))
	api.GET("/home", h.Home, middleware.Auth(deps.Auth))
	api.GET("/radio/:id", h.Radio, middleware.Auth(deps.Auth))

	api.GET("/settings", h.GetSettings, middleware.Auth(deps.Auth))
	api.PUT("/settings", h.UpdateSettings, middleware.Auth(deps.Auth))

	api.GET("/player/state", h.GetPlayerState, middleware.Auth(deps.Auth))
	api.PUT("/player/state", h.SavePlayerState, middleware.Auth(deps.Auth))
	api.POST("/player/state", h.SavePlayerState, middleware.Auth(deps.Auth))

	api.POST("/scan", h.StartScan, middleware.Auth(deps.Auth))
	api.GET("/scan/status", h.ScanStatus, middleware.Auth(deps.Auth))

	admin := api.Group("/admin", middleware.Auth(deps.Auth), middleware.AdminOnly)
	admin.GET("/system", h.SystemInfo)
	admin.DELETE("/sessions/cleanup", h.CleanupSessions)
	admin.POST("/musicbrainz/enrich", h.Enrich)
	admin.GET("/settings", h.GetAppSettings)
	admin.PUT("/settings", h.UpdateAppSettings)

	api.POST("/musicbrainz/submit-listen", h.SubmitListen, middleware.Auth(deps.Auth))
	api.GET("/musicbrainz/recommendations", h.Recommendations, middleware.Auth(deps.Auth))

	// SPA fallback: serve static files, fall back to index.html for client-side routing
	if deps.WebDistPath != "" {
		e.Use(spaMiddleware(deps.WebDistPath))
	}

	return e
}

func spaMiddleware(distPath string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path

			// Skip API and swagger routes
			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/swagger") {
				return next(c)
			}

			// Check if file exists in dist folder
			filePath := distPath + path
			if _, err := os.Stat(filePath); err == nil {
				return c.File(filePath)
			}

			// Serve index.html for SPA routes
			return c.File(distPath + "/index.html")
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
