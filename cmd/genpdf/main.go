package main

import (
	"flag"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/services"
)

func main() {
	compact := flag.Bool("compact", false, "Generate compact CSV-like PDF with tiny text")
	flag.Parse()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./korus.db"
	}

	output := "songs.pdf"
	if flag.NArg() > 0 {
		output = flag.Arg(0)
	}

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()

	radio := services.NewRadioService(database, "", "")

	var pdfBytes []byte
	if *compact {
		pdfBytes, err = radio.GenerateCompactPDF()
	} else {
		pdfBytes, err = radio.GeneratePDF()
	}
	if err != nil {
		log.Fatalf("generate pdf: %v", err)
	}

	if err := os.WriteFile(output, pdfBytes, 0644); err != nil {
		log.Fatalf("write file: %v", err)
	}

	log.Printf("Saved: %s (%d bytes)", output, len(pdfBytes))
}
