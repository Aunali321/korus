package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/services"
	aisvc "github.com/Aunali321/korus/internal/services/ai"
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("Usage: go run ./cmd/testradio <seed_song_id>")
		os.Exit(1)
	}
	seedID, err := strconv.ParseInt(flag.Arg(0), 10, 64)
	if err != nil {
		log.Fatalf("invalid song id: %v", err)
	}

	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		log.Fatal("AI_API_KEY not set")
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./korus.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	var userID int64
	if err := database.QueryRow(`SELECT id FROM users ORDER BY id LIMIT 1`).Scan(&userID); err != nil {
		log.Fatalf("no users in db: %v", err)
	}

	svc := aisvc.New(aisvc.Config{
		DB:        database,
		Library:   services.NewLibraryService(database),
		Playlists: services.NewPlaylistService(database),
		Search:    services.NewSearchService(database),
		APIKey:    apiKey,
		Provider:  os.Getenv("AI_PROVIDER"),
		BaseURL:   os.Getenv("AI_BASE_URL"),
		Model:     os.Getenv("AI_MODEL"),
		Reasoning: os.Getenv("AI_REASONING"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	start := time.Now()
	ids, err := svc.Radio(ctx, userID, seedID, 20)
	if err != nil {
		log.Fatalf("radio: %v", err)
	}

	fmt.Printf("%d songs in %v:\n", len(ids), time.Since(start))
	for i, id := range ids {
		var title, artist string
		database.QueryRow(`
			SELECT s.title, COALESCE(ar.name, '?')
			FROM songs s
			LEFT JOIN song_artists sa ON sa.song_id = s.id AND sa.role = 'primary'
			LEFT JOIN artists ar ON ar.id = sa.artist_id
			WHERE s.id = ?`, id).Scan(&title, &artist)
		fmt.Printf("%2d. [%d] %s — %s\n", i+1, id, title, artist)
	}
}
