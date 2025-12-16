package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/joho/godotenv/autoload"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "github.com/Aunali321/korus/docs"
	"github.com/Aunali321/korus/internal/api"
	"github.com/Aunali321/korus/internal/config"
	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/services"
)

// @title Korus API
// @version 0.1
// @description Self-hosted music streaming server API.
// @BasePath /api
func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()

	ctx := context.Background()
	if err := db.Migrate(ctx, database); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "changeme"
	}
	adminUser := os.Getenv("ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@example.com"
	}
	authSvc := services.NewAuthService(database, []byte(cfg.JWTSecret), cfg.TokenTTL, cfg.RefreshTTL)
	if err := seedAdmin(ctx, authSvc, database, adminUser, adminEmail, adminPass); err != nil {
		log.Fatalf("seed admin: %v", err)
	}

	if _, err := exec.LookPath(cfg.FFmpegPath); err != nil {
		log.Fatalf("ffmpeg not found at %s: %v", cfg.FFmpegPath, err)
	}
	if _, err := exec.LookPath(cfg.FFprobePath); err != nil {
		log.Fatalf("ffprobe not found at %s: %v", cfg.FFprobePath, err)
	}

	scanner := services.NewScannerService(database, cfg.MediaRoot, cfg.FFprobePath, cfg.FFmpegPath, cfg.ScanExcludePattern, cfg.ScanEmbeddedCover, cfg.ScanWatch, cfg.ScanWorkers, cfg.CoverCachePath, cfg.ScanAutoPlaylists)
	if cfg.ScanWatch {
		go func() {
			if err := scanner.Watch(context.Background()); err != nil {
				log.Printf("scanner watch stopped: %v", err)
			}
		}()
	}
	search := services.NewSearchService(database)
	transcoder := services.NewTranscoder(cfg.FFmpegPath)
	var mb *services.MusicBrainzService
	var lb *services.ListenBrainzService
	if cfg.EnableMusicBrainz {
		mb = services.NewMusicBrainzService(cfg.MusicBrainzAgent)
	}
	if cfg.EnableListenBrainz {
		lb = services.NewListenBrainzService(cfg.ListenBrainzToken, cfg.ListenBrainzUser)
	}

	e := api.New(api.Deps{
		DB:           database,
		Auth:         authSvc,
		Scanner:      scanner,
		Search:       search,
		Transcoder:   transcoder,
		MusicBrainz:  mb,
		ListenBrainz: lb,
		MediaRoot:    cfg.MediaRoot,
		AuthRate:     cfg.RateLimitAuthCount,
		AuthWindow:   cfg.RateLimitAuthWindow,
		WebDistPath:  "web/dist",
	})
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	go func() {
		if err := e.Start(cfg.Addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(ctxShutdown); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}

func seedAdmin(ctx context.Context, auth *services.AuthService, dbConn *sql.DB, username, email, password string) error {
	var count int
	if err := dbConn.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = dbConn.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, email, role)
		VALUES (?, ?, ?, 'admin')
	`, username, string(hash), email)
	if err != nil {
		return err
	}
	// issue tokens via login to create session
	_, tokens, err := auth.Login(ctx, username, password)
	if err != nil {
		log.Printf("seeded admin user %s (login token failed: %v)", username, err)
		return nil
	}
	log.Printf("seeded admin user %s with token %s", username, tokens.Access)
	return nil
}
