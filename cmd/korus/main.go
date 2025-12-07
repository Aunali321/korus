package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"korus/internal/auth"
	"korus/internal/config"
	"korus/internal/database"
	"korus/internal/handlers"
	"korus/internal/indexer"
	"korus/internal/middleware"
	"korus/internal/scanner"
	"korus/internal/search"
	"korus/internal/services"
	"korus/internal/streaming"
	"korus/internal/transcoding"
	"korus/migrations"

	"github.com/gin-gonic/gin"
)

type indexerAdapter struct {
	svc *indexer.Service
}

func (a *indexerAdapter) StartScanAsync(ctx context.Context, force bool) (string, error) {
	if a == nil || a.svc == nil {
		return "", fmt.Errorf("indexer not available")
	}
	return a.svc.StartScanAsync(ctx, indexer.Options{Force: force})
}

func (a *indexerAdapter) GetJob(jobID string) (*services.LibraryScanJob, error) {
	if a == nil || a.svc == nil {
		return nil, fmt.Errorf("indexer not available")
	}

	job, err := a.svc.GetJob(jobID)
	if err != nil {
		return nil, err
	}

	return toLibraryScanJob(job), nil
}

func (a *indexerAdapter) Status() services.LibraryIndexerStatus {
	status := a.svc.Status()
	converted := services.LibraryIndexerStatus{
		Running:   status.Running,
		LastError: status.LastError,
	}
	if status.LastRun != nil {
		converted.LastRun = toLibraryScanResult(status.LastRun)
	}
	return converted
}

func toLibraryScanJob(job *indexer.Job) *services.LibraryScanJob {
	if job == nil {
		return nil
	}
	result := &services.LibraryScanJob{
		ID:        job.ID,
		Status:    string(job.Status),
		Phase:     job.Phase,
		Progress:  job.Progress,
		Total:     job.Total,
		Force:     job.Force,
		StartedAt: job.StartedAt,
	}
	if job.CompletedAt != nil {
		result.CompletedAt = job.CompletedAt
	}
	if job.Result != nil {
		result.Result = toLibraryScanResult(job.Result)
	}
	if job.Error != "" {
		result.Error = job.Error
	}
	return result
}

func toLibraryScanResult(res *indexer.Result) *services.LibraryScanResult {
	if res == nil {
		return nil
	}
	copy := &services.LibraryScanResult{
		StartedAt:       res.StartedAt,
		CompletedAt:     res.CompletedAt,
		Duration:        res.Duration,
		FilesDiscovered: res.FilesDiscovered,
		FilesQueued:     res.FilesQueued,
		FilesNew:        res.FilesNew,
		FilesUpdated:    res.FilesUpdated,
		FilesRemoved:    res.FilesRemoved,
		Ingested:        res.Ingested,
	}
	if len(res.Errors) > 0 {
		copy.Errors = append([]error(nil), res.Errors...)
	}
	return copy
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run database migrations
	migrator := migrations.NewMigrator(db.Pool)
	if err := migrator.Migrate(context.Background()); err != nil {
		log.Fatal("Failed to run database migrations:", err)
	}

	// Initialize services
	tokenManager := auth.NewTokenManager(&cfg.Auth)
	authService := auth.NewService(db, tokenManager)
	libraryService := services.NewLibraryService(db)
	searchService, err := search.NewSearchService(db, &cfg.Library)
	if err != nil {
		log.Fatal("Failed to initialize search service:", err)
	}
	defer searchService.Close()

	streamingService := streaming.NewStreamingService(libraryService, transcoding.New())

	// Check FFmpeg availability
	if tc := transcoding.New(); !tc.IsAvailable() {
		log.Println("Warning: FFmpeg not found, transcoding will be disabled")
	}

	batchMetadataService := services.NewBatchMetadataService(db, cfg.Library.ExtractLyrics)
	indexerService := indexer.NewService(db, &cfg.Library, batchMetadataService)

	// Create initial admin user if no users exist
	if err := createInitialAdminUser(authService, cfg); err != nil {
		log.Fatal("Failed to create initial admin user:", err)
	}

	// Initialize file watcher scanner
	fileScanner, err := scanner.New(db, indexerService, &cfg.Library)
	if err != nil {
		log.Printf("Warning: Failed to create file watcher: %v", err)
	} else {
		if err := fileScanner.Start(context.Background()); err != nil {
			log.Printf("Warning: Failed to start file watcher: %v", err)
		}
		defer fileScanner.Stop()
	}

	// Initialize additional services
	playlistService := services.NewPlaylistService(db)
	userLibraryService := services.NewUserLibraryService(db)
	historyService := services.NewHistoryService(db)
	adminService := services.NewAdminService(db, &indexerAdapter{svc: indexerService})

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db)
	authHandler := handlers.NewAuthHandler(authService)
	libraryHandler := handlers.NewLibraryHandler(libraryService, searchService)
	playlistHandler := handlers.NewPlaylistHandler(playlistService, authService)
	userLibraryHandler := handlers.NewUserLibraryHandler(userLibraryService)
	historyHandler := handlers.NewHistoryHandler(historyService)
	adminHandler := handlers.NewAdminHandler(adminService)

	// Setup router
	router := setupRouter(cfg, authService, healthHandler, authHandler, libraryHandler, playlistHandler, userLibraryHandler, historyHandler, adminHandler, streamingService)

	// Setup HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		fmt.Printf("🎵 Korus server starting on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
		fmt.Printf("Environment: %s\n", cfg.Server.Environment)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("🛑 Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("✅ Server shutdown complete")
}

func setupRouter(cfg *config.Config, authService *auth.Service, healthHandler *handlers.HealthHandler, authHandler *handlers.AuthHandler, libraryHandler *handlers.LibraryHandler, playlistHandler *handlers.PlaylistHandler, userLibraryHandler *handlers.UserLibraryHandler, historyHandler *handlers.HistoryHandler, adminHandler *handlers.AdminHandler, streamingService *streaming.StreamingService) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.DefaultCORS())

	// Health endpoints (no auth required)
	router.GET("/api/ping", healthHandler.Ping)
	router.GET("/api/health", healthHandler.Health)

	// API routes
	api := router.Group("/api")
	{
		// Auth endpoints with rate limiting
		auth := api.Group("/auth")
		auth.Use(middleware.AuthRateLimit())
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected endpoints
		protected := api.Group("")
		protected.Use(middleware.AuthRequired(authService))
		{
			protected.GET("/me", authHandler.Me)

			// Library endpoints
			protected.GET("/library/stats", libraryHandler.GetStats)
			protected.GET("/artists", libraryHandler.GetArtists)
			protected.GET("/artists/:id", libraryHandler.GetArtist)
			protected.GET("/albums", libraryHandler.GetAlbums)
			protected.GET("/albums/:id", libraryHandler.GetAlbum)
			protected.GET("/songs", libraryHandler.GetSongs)
			protected.GET("/songs/:id", libraryHandler.GetSong)
			protected.GET("/search", libraryHandler.Search)

			// Streaming endpoints
			protected.GET("/songs/:id/stream", streamingService.StreamSong)

			// Playlist endpoints
			protected.GET("/playlists", playlistHandler.GetUserPlaylists)
			protected.POST("/playlists", playlistHandler.CreatePlaylist)
			protected.GET("/playlists/:id", playlistHandler.GetPlaylist)
			protected.PUT("/playlists/:id", playlistHandler.UpdatePlaylist)
			protected.DELETE("/playlists/:id", playlistHandler.DeletePlaylist)
			protected.POST("/playlists/:id/songs", playlistHandler.AddSongsToPlaylist)
			protected.PUT("/playlists/:id/songs/reorder", playlistHandler.ReorderPlaylistSongs)
			protected.DELETE("/playlists/:id/songs", playlistHandler.RemoveSongsFromPlaylist)

			// User library endpoints
			protected.GET("/me/library/songs", userLibraryHandler.GetLikedSongs)
			protected.GET("/me/library/albums", userLibraryHandler.GetLikedAlbums)
			protected.GET("/me/library/artists", userLibraryHandler.GetFollowedArtists)
			protected.POST("/songs/like", userLibraryHandler.LikeSongs)
			protected.DELETE("/songs/like", userLibraryHandler.UnlikeSongs)
			protected.POST("/albums/:id/like", userLibraryHandler.LikeAlbum)
			protected.DELETE("/albums/:id/like", userLibraryHandler.UnlikeAlbum)
			protected.POST("/artists/:id/follow", userLibraryHandler.FollowArtist)
			protected.DELETE("/artists/:id/follow", userLibraryHandler.UnfollowArtist)

			// History and stats endpoints
			protected.POST("/me/history/scrobble", historyHandler.Scrobble)
			protected.GET("/me/history/recent", historyHandler.GetRecentHistory)
			protected.GET("/me/stats", historyHandler.GetUserStats)
			protected.GET("/me/home", historyHandler.GetHomeData)

			// Admin endpoints
			admin := protected.Group("")
			admin.Use(middleware.AdminRequired())
			{
				admin.POST("/library/scan", adminHandler.TriggerLibraryScan)
				admin.GET("/library/scan/:id", adminHandler.GetScanJob)
				admin.GET("/admin/status", adminHandler.GetSystemStatus)
				admin.GET("/admin/scans", adminHandler.GetScanHistory)
				admin.DELETE("/admin/sessions/cleanup", adminHandler.CleanupSessions)
			}
		}
	}

	// Serve static files (covers are always available)
	router.Static("/static", "./static")

	// Serve cover images with cache headers (1 year cache - filenames are content-hashed)
	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/covers/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	})
	router.Static("/covers", "./static/covers")

	return router
}

func createInitialAdminUser(authService *auth.Service, cfg *config.Config) error {
	ctx := context.Background()

	// Check if any users exist
	hasUsers, err := authService.HasUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if users exist: %w", err)
	}

	if hasUsers {
		return nil // Users already exist, no need to create admin
	}

	// Generate secure password if not provided
	adminPassword := cfg.Auth.AdminPassword
	if adminPassword == "" {
		adminPassword, err = auth.GenerateSecurePassword()
		if err != nil {
			return fmt.Errorf("failed to generate admin password: %w", err)
		}
	}

	// Create admin user
	user, err := authService.CreateAdminUser(ctx, cfg.Auth.AdminUsername, adminPassword)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// Print admin credentials (only on first run)
	fmt.Println("====================KORUS INITIAL SETUP====================")
	fmt.Println("ADMIN ACCOUNT CREATED:")
	fmt.Printf("Username: %s\n", user.Username)
	fmt.Printf("Password: %s (Securely generated - change immediately)\n", adminPassword)
	fmt.Println("==========================================================")

	return nil
}
