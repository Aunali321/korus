package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
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
	mode := flag.String("mode", "curator", "Radio mode: curator or mainstream")
	benchmark := flag.Int("benchmark", 0, "Run N benchmarks with random songs")
	flag.Parse()

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

	if *benchmark > 0 {
		runBenchmark(database, radio, model, *benchmark)
		return
	}

	if flag.NArg() < 1 {
		fmt.Println("Usage: go run cmd/testradio/main.go [-mode curator|mainstream] [-benchmark N] <song_id>")
		fmt.Println("Example: go run cmd/testradio/main.go 4")
		fmt.Println("         go run cmd/testradio/main.go -mode mainstream 4")
		os.Exit(1)
	}

	songID, err := strconv.ParseInt(flag.Arg(0), 10, 64)
	if err != nil {
		log.Fatalf("Invalid song ID: %v", err)
	}

	radioMode := services.RadioModeCurator
	if *mode == "mainstream" {
		radioMode = services.RadioModeMainstream
	}

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
	fmt.Printf("Mode: %s\n", *mode)
	fmt.Println("Getting recommendations...")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	start := time.Now()
	ids, err := radio.GetRecommendations(ctx, songID, 20, radioMode)
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

type recSong struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

type modeResult struct {
	TimeMs int64     `json:"time_ms"`
	Recs   []recSong `json:"results"`
	Err    string    `json:"error,omitempty"`
}

type benchmarkResult struct {
	SongID     int64      `json:"song_id"`
	Title      string     `json:"title"`
	Artist     string     `json:"artist"`
	Curator    modeResult `json:"curator"`
	Mainstream modeResult `json:"mainstream"`
}

func runBenchmark(database *sql.DB, radio *services.RadioService, model string, count int) {
	const limit = 20

	rows, err := database.Query(`
		SELECT s.id, s.title, ar.name
		FROM songs s
		JOIN albums al ON s.album_id = al.id
		JOIN artists ar ON al.artist_id = ar.id
		ORDER BY RANDOM()
		LIMIT ?
	`, count)
	if err != nil {
		log.Fatalf("Failed to get random songs: %v", err)
	}
	defer rows.Close()

	type testSong struct {
		id     int64
		title  string
		artist string
	}
	var songs []testSong
	for rows.Next() {
		var s testSong
		if err := rows.Scan(&s.id, &s.title, &s.artist); err != nil {
			log.Fatalf("Failed to scan song: %v", err)
		}
		songs = append(songs, s)
	}

	getSongInfo := func(id int64) recSong {
		var title, artist string
		err := database.QueryRow(`
			SELECT s.title, ar.name
			FROM songs s
			JOIN albums al ON s.album_id = al.id
			JOIN artists ar ON al.artist_id = ar.id
			WHERE s.id = ?
		`, id).Scan(&title, &artist)
		if err != nil {
			return recSong{ID: id, Title: "NOT FOUND", Artist: ""}
		}
		return recSong{ID: id, Title: title, Artist: artist}
	}

	var results []benchmarkResult

	fmt.Printf("Benchmarking %d songs with model: %s\n\n", count, model)

	for i, song := range songs {
		fmt.Printf("[%d/%d] %s - %s\n", i+1, count, song.title, song.artist)

		r := benchmarkResult{
			SongID: song.id,
			Title:  song.title,
			Artist: song.artist,
		}

		// Curator mode
		ctx1, cancel1 := context.WithTimeout(context.Background(), 120*time.Second)
		start := time.Now()
		ids1, err := radio.GetRecommendations(ctx1, song.id, limit, services.RadioModeCurator)
		r.Curator.TimeMs = time.Since(start).Milliseconds()
		cancel1()
		if err != nil {
			r.Curator.Err = err.Error()
			fmt.Printf("  Curator:    ERROR - %v\n", err)
		} else {
			for _, id := range ids1 {
				r.Curator.Recs = append(r.Curator.Recs, getSongInfo(id))
			}
			fmt.Printf("  Curator:    %d results in %dms\n", len(ids1), r.Curator.TimeMs)
		}

		// Mainstream mode
		ctx2, cancel2 := context.WithTimeout(context.Background(), 120*time.Second)
		start = time.Now()
		ids2, err := radio.GetRecommendations(ctx2, song.id, limit, services.RadioModeMainstream)
		r.Mainstream.TimeMs = time.Since(start).Milliseconds()
		cancel2()
		if err != nil {
			r.Mainstream.Err = err.Error()
			fmt.Printf("  Mainstream: ERROR - %v\n", err)
		} else {
			for _, id := range ids2 {
				r.Mainstream.Recs = append(r.Mainstream.Recs, getSongInfo(id))
			}
			fmt.Printf("  Mainstream: %d results in %dms\n", len(ids2), r.Mainstream.TimeMs)
		}

		results = append(results, r)
	}

	output := struct {
		Model     string            `json:"model"`
		Timestamp string            `json:"timestamp"`
		Results   []benchmarkResult `json:"results"`
	}{
		Model:     model,
		Timestamp: time.Now().Format(time.RFC3339),
		Results:   results,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal results: %v", err)
	}

	filename := fmt.Sprintf("benchmark_%s.json", time.Now().Format("2006-01-02_15-04-05"))
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Fatalf("Failed to write results: %v", err)
	}

	fmt.Printf("\nResults saved to %s\n", filename)
}
