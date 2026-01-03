package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-pdf/fpdf"
)

type RadioMode string

const (
	RadioModeCurator    RadioMode = "curator"
	RadioModeMainstream RadioMode = "mainstream"
)

type RadioService struct {
	db     *sql.DB
	apiKey string
	model  string
}

func NewRadioService(db *sql.DB, apiKey, model string) *RadioService {
	if model == "" {
		model = "google/gemini-3-flash-preview"
	}
	return &RadioService{
		db:     db,
		apiKey: apiKey,
		model:  model,
	}
}

func (r *RadioService) SetModel(model string) {
	r.model = model
}

type songEntry struct {
	ID     int64
	Title  string
	Artist string
}

func (r *RadioService) getSongs() ([]songEntry, error) {
	rows, err := r.db.Query(`
		SELECT s.id, s.title, ar.name
		FROM songs s
		JOIN albums al ON s.album_id = al.id
		JOIN artists ar ON al.artist_id = ar.id
		ORDER BY s.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []songEntry
	for rows.Next() {
		var s songEntry
		if err := rows.Scan(&s.ID, &s.Title, &s.Artist); err != nil {
			continue
		}
		songs = append(songs, s)
	}
	return songs, nil
}

func (r *RadioService) getSongByID(id int64) (*songEntry, error) {
	var s songEntry
	err := r.db.QueryRow(`
		SELECT s.id, s.title, ar.name
		FROM songs s
		JOIN albums al ON s.album_id = al.id
		JOIN artists ar ON al.artist_id = ar.id
		WHERE s.id = ?
	`, id).Scan(&s.ID, &s.Title, &s.Artist)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

type openRouterRequest struct {
	Model          string           `json:"model"`
	Messages       []message        `json:"messages"`
	Temperature    float64          `json:"temperature"`
	Provider       providerOptions  `json:"provider"`
	Plugins        []plugin         `json:"plugins,omitempty"`
	ResponseFormat *responseFormat  `json:"response_format,omitempty"`
	Reasoning      *reasoningConfig `json:"reasoning,omitempty"`
}

type reasoningConfig struct {
	Effort  string `json:"effort,omitempty"`
	Exclude bool   `json:"exclude,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type contentPart struct {
	Type         string        `json:"type"`
	Text         string        `json:"text,omitempty"`
	File         *fileObject   `json:"file,omitempty"`
	CacheControl *cacheControl `json:"cache_control,omitempty"`
}

type cacheControl struct {
	Type string `json:"type"`
}

type fileObject struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"`
}

type providerOptions struct {
	AllowFallbacks bool     `json:"allow_fallbacks"`
	Only           []string `json:"only"`
}

type plugin struct {
	ID  string       `json:"id"`
	PDF *pdfSettings `json:"pdf,omitempty"`
}

type pdfSettings struct {
	Engine string `json:"engine"`
}

type responseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *jsonSchema `json:"json_schema,omitempty"`
}

type jsonSchema struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
	Schema any    `json:"schema"`
}

type openRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type recommendationResponse struct {
	SongIDs []int64 `json:"song_ids"`
}

func (r *RadioService) GetRecommendations(ctx context.Context, songID int64, limit int, mode RadioMode) ([]int64, error) {
	if r.apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not configured")
	}

	song, err := r.getSongByID(songID)
	if err != nil {
		return nil, fmt.Errorf("song not found: %w", err)
	}

	var pdfBytes []byte
	if mode == RadioModeCurator {
		pdfBytes, err = r.GenerateCompactPDF()
	} else {
		pdfBytes, err = r.GeneratePDF()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	base64PDF := base64.StdEncoding.EncodeToString(pdfBytes)
	dataURL := "data:application/pdf;base64," + base64PDF

	systemPrompt := `You are a music recommendation expert. Given a song, find similar songs from the provided library based on:
- Genre and subgenre
- Musical style and mood
- Language (prefer same language)

Avoid recommending different versions of the same song (remixes, covers, live versions, acoustic versions, etc.).

Search the ENTIRE library across all pages.`

	userPrompt := fmt.Sprintf(`I'm listening to: [%d] %s - %s

Find %d similar songs from this library. Return song IDs only.`, songID, song.Title, song.Artist, limit)

	reqBody := openRouterRequest{
		Model: r.model,
		Messages: []message{
			{
				Role: "system",
				Content: []contentPart{
					{
						Type:         "text",
						Text:         systemPrompt,
						CacheControl: &cacheControl{Type: "ephemeral"},
					},
				},
			},
			{
				Role: "user",
				Content: []contentPart{
					{
						Type:         "text",
						Text:         "Here is my song library:",
						CacheControl: &cacheControl{Type: "ephemeral"},
					},
					{
						Type: "file",
						File: &fileObject{
							Filename: "songs.pdf",
							FileData: dataURL,
						},
						CacheControl: &cacheControl{Type: "ephemeral"},
					},
				},
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Temperature: 0.1,
		Provider: providerOptions{
			AllowFallbacks: false,
			Only:           []string{"Google AI Studio"},
		},
		Plugins: []plugin{
			{
				ID:  "file-parser",
				PDF: &pdfSettings{Engine: "native"},
			},
		},
		ResponseFormat: &responseFormat{
			Type: "json_schema",
			JSONSchema: &jsonSchema{
				Name:   "recommendations",
				Strict: true,
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"song_ids": map[string]any{
							"type":        "array",
							"items":       map[string]any{"type": "integer"},
							"description": "Array of recommended song IDs from the library",
						},
					},
					"required":             []string{"song_ids"},
					"additionalProperties": false,
				},
			},
		},
		Reasoning: &reasoningConfig{
			Effort:  "low",
			Exclude: true,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result openRouterResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	var recResp recommendationResponse
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &recResp); err != nil {
		return nil, fmt.Errorf("failed to parse recommendations: %w", err)
	}

	// Filter out the seed song and limit results
	var ids []int64
	seen := make(map[int64]bool)
	for _, id := range recResp.SongIDs {
		if id == songID || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		if len(ids) >= limit {
			break
		}
	}

	return ids, nil
}

func (r *RadioService) GeneratePDF() ([]byte, error) {
	songs, err := r.getSongs()
	if err != nil {
		return nil, fmt.Errorf("failed to get songs: %w", err)
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)

	const (
		fontSize   = 8.0
		columns    = 3
		pageWidth  = 210.0
		pageHeight = 297.0
		margin     = 10.0
		colGap     = 5.0
		headerH    = 8.0
	)

	colWidth := (pageWidth - 2*margin - float64(columns-1)*colGap) / float64(columns)
	usableHeight := pageHeight - 2*margin - headerH
	lineHeight := 2.7
	linesPerCol := int(usableHeight / lineHeight)
	songsPerPage := linesPerCol * columns

	// Courier 8pt: ~1.55mm per char
	maxEntryLen := int(colWidth / 1.55)

	entries := make([]string, len(songs))
	for i, s := range songs {
		entry := fmt.Sprintf("[%d] %s - %s", s.ID, s.Title, s.Artist)
		if len(entry) > maxEntryLen {
			entry = entry[:maxEntryLen-3] + "..."
		}
		entries[i] = entry
	}

	numPages := (len(entries) + songsPerPage - 1) / songsPerPage

	for pageIdx := 0; pageIdx < numPages; pageIdx++ {
		pdf.AddPage()

		startIdx := pageIdx * songsPerPage
		endIdx := startIdx + songsPerPage
		if endIdx > len(entries) {
			endIdx = len(entries)
		}
		pageEntries := entries[startIdx:endIdx]

		pdf.SetFont("Courier", "B", 10)
		header := fmt.Sprintf("Song Library - Page %d/%d (Songs %d-%d of %d)",
			pageIdx+1, numPages, startIdx+1, endIdx, len(songs))
		pdf.SetXY(margin, margin)
		pdf.Cell(0, 5, header)

		pdf.SetFont("Courier", "", fontSize)
		yStart := margin + headerH

		for i, entry := range pageEntries {
			col := i / linesPerCol
			row := i % linesPerCol

			x := margin + float64(col)*(colWidth+colGap)
			y := yStart + float64(row)*lineHeight

			pdf.Text(x, y, entry)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateCompactPDF creates a minimal PDF with all songs comma-separated
func (r *RadioService) GenerateCompactPDF() ([]byte, error) {
	songs, err := r.getSongs()
	if err != nil {
		return nil, fmt.Errorf("failed to get songs: %w", err)
	}

	// Build comma-separated string: "1 - Song - Artist, 2 - Song - Artist, ..."
	entries := make([]string, 0, len(songs))
	for _, s := range songs {
		entry := fmt.Sprintf("%d - %s - %s", s.ID, s.Title, s.Artist)
		entries = append(entries, entry)
	}
	content := strings.Join(entries, ", ")

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 5)
	pdf.AddPage()

	const (
		fontSize = 1.5 // Minimum readable size
		margin   = 3.0
	)

	pdf.SetFont("Courier", "", fontSize)
	pdf.SetXY(margin, margin)

	// Use MultiCell for automatic text wrapping
	pageWidth := 210.0 - 2*margin
	pdf.MultiCell(pageWidth, 0.8, content, "", "", false)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}
