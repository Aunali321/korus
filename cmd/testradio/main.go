package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/services"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/testradio/main.go <song_id>")
		fmt.Println("Example: go run cmd/testradio/main.go 4")
		os.Exit(1)
	}

	songID, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Fatalf("Invalid song ID: %v", err)
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable not set")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./korus.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	model := os.Getenv("RADIO_MODEL")
	if model == "" {
		model = "google/gemini-3-flash-preview"
	}

	radio := services.NewRadioService(database, apiKey, model)

	// Get seed song info
	var title, artist string
	err = database.QueryRow(`
		SELECT s.title, ar.name
		FROM songs s
		JOIN albums al ON s.album_id = al.id
		JOIN artists ar ON al.artist_id = ar.id
		WHERE s.id = ?
	`, songID).Scan(&title, &artist)
	if err != nil {
		log.Fatalf("Song not found: %v", err)
	}

	fmt.Printf("Seed song: [%d] %s - %s\n", songID, title, artist)
	fmt.Printf("Model: %s\n", model)
	fmt.Println("Getting recommendations...")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	ids, err := radio.GetRecommendations(ctx, songID, 20)
	elapsed := time.Since(start)

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Got %d recommendations in %v:\n\n", len(ids), elapsed)

	for i, id := range ids {
		var recTitle, recArtist string
		err := database.QueryRow(`
			SELECT s.title, ar.name
			FROM songs s
			JOIN albums al ON s.album_id = al.id
			JOIN artists ar ON al.artist_id = ar.id
			WHERE s.id = ?
		`, id).Scan(&recTitle, &recArtist)
		if err != nil {
			fmt.Printf("%2d. [%d] NOT FOUND\n", i+1, id)
		} else {
			fmt.Printf("%2d. [%d] %s - %s\n", i+1, id, recTitle, recArtist)
		}
	}
}
