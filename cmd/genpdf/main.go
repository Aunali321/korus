package main

import (
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/services"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./korus.db"
	}

	output := "songs.pdf"
	if len(os.Args) > 1 {
		output = os.Args[1]
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()

	radio := services.NewRadioService(database, "", "")
	pdfBytes, err := radio.GeneratePDF()
	if err != nil {
		log.Fatalf("generate pdf: %v", err)
	}

	if err := os.WriteFile(output, pdfBytes, 0644); err != nil {
		log.Fatalf("write file: %v", err)
	}

	log.Printf("Saved: %s (%d bytes)", output, len(pdfBytes))
}
