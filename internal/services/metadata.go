package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type MetadataService struct {
	baseURL string
	client  *http.Client
}

func NewMetadataService(baseURL string) *MetadataService {
	return &MetadataService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type MetadataImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type MetadataArtist struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Followers  int             `json:"followers"`
	Popularity int             `json:"popularity"`
	Genres     []string        `json:"genres"`
	Images     []MetadataImage `json:"images"`
}

type MetadataAlbum struct {
	ID                   string           `json:"id"`
	Name                 string           `json:"name"`
	Type                 string           `json:"type"`
	Label                string           `json:"label"`
	ReleaseDate          string           `json:"release_date"`
	ReleaseDatePrecision string           `json:"release_date_precision"`
	UPC                  string           `json:"upc"`
	TotalTracks          int              `json:"total_tracks"`
	Copyright            string           `json:"copyright"`
	CopyrightP           string           `json:"copyright_p"`
	Images               []MetadataImage  `json:"images"`
	Artists              []MetadataArtist `json:"artists"`
}

type MetadataTrack struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	ISRC          string           `json:"isrc"`
	DurationMs    int              `json:"duration_ms"`
	Explicit      bool             `json:"explicit"`
	TrackNumber   int              `json:"track_number"`
	DiscNumber    int              `json:"disc_number"`
	Popularity    int              `json:"popularity"`
	PreviewURL    *string          `json:"preview_url"`
	Album         *MetadataAlbum   `json:"album"`
	Artists       []MetadataArtist `json:"artists"`
	OriginalTitle *string          `json:"original_title"`
	VersionTitle  *string          `json:"version_title"`
	HasLyrics     *bool            `json:"has_lyrics"`
	Languages     []string         `json:"languages"`
	ArtistRoles   []string         `json:"artist_roles"`
}

type BatchRequest struct {
	Tracks  []string `json:"tracks,omitempty"`
	Artists []string `json:"artists,omitempty"`
	Albums  []string `json:"albums,omitempty"`
	ISRCs   []string `json:"isrcs,omitempty"`
}

type BatchResponse struct {
	Tracks  map[string]MetadataTrack   `json:"tracks,omitempty"`
	Artists map[string]MetadataArtist  `json:"artists,omitempty"`
	Albums  map[string]MetadataAlbum   `json:"albums,omitempty"`
	ISRCs   map[string][]MetadataTrack `json:"isrcs,omitempty"`
	Errors  map[string]string          `json:"errors,omitempty"`
}

// BatchLookupISRCs looks up multiple ISRCs in a single request
func (s *MetadataService) BatchLookupISRCs(ctx context.Context, isrcs []string) (*BatchResponse, error) {
	if len(isrcs) == 0 {
		return &BatchResponse{ISRCs: make(map[string][]MetadataTrack)}, nil
	}

	// API limit is 400 items per request
	const batchSize = 400
	result := &BatchResponse{
		ISRCs:  make(map[string][]MetadataTrack),
		Errors: make(map[string]string),
	}

	totalBatches := (len(isrcs) + batchSize - 1) / batchSize
	batchNum := 0

	for i := 0; i < len(isrcs); i += batchSize {
		end := i + batchSize
		if end > len(isrcs) {
			end = len(isrcs)
		}
		batch := isrcs[i:end]
		batchNum++

		log.Printf("metadata: batch %d/%d (%d ISRCs)", batchNum, totalBatches, len(batch))

		resp, err := s.batchLookup(ctx, BatchRequest{ISRCs: batch})
		if err != nil {
			return nil, fmt.Errorf("batch lookup failed: %w", err)
		}

		for k, v := range resp.ISRCs {
			result.ISRCs[k] = v
		}
		for k, v := range resp.Errors {
			result.Errors[k] = v
		}
	}

	return result, nil
}

// BatchLookupArtists looks up multiple artist IDs in a single request
func (s *MetadataService) BatchLookupArtists(ctx context.Context, artistIDs []string) (*BatchResponse, error) {
	if len(artistIDs) == 0 {
		return &BatchResponse{Artists: make(map[string]MetadataArtist)}, nil
	}

	const batchSize = 400
	result := &BatchResponse{
		Artists: make(map[string]MetadataArtist),
		Errors:  make(map[string]string),
	}

	for i := 0; i < len(artistIDs); i += batchSize {
		end := i + batchSize
		if end > len(artistIDs) {
			end = len(artistIDs)
		}
		batch := artistIDs[i:end]

		resp, err := s.batchLookup(ctx, BatchRequest{Artists: batch})
		if err != nil {
			return nil, fmt.Errorf("batch lookup failed: %w", err)
		}

		for k, v := range resp.Artists {
			result.Artists[k] = v
		}
		for k, v := range resp.Errors {
			result.Errors[k] = v
		}
	}

	return result, nil
}

func (s *MetadataService) batchLookup(ctx context.Context, req BatchRequest) (*BatchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	log.Printf("metadata: sending request to %s/batch/lookup with %d bytes", s.baseURL, len(body))

	var resp *http.Response
	var lastErr error

	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * time.Second // 1s, 2s, 4s
			log.Printf("metadata: retry %d/%d after %v", attempt, maxRetries, delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/batch/lookup", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, lastErr = s.client.Do(httpReq)
		if lastErr != nil {
			continue // retry on network error
		}

		// Retry on rate limit or server errors
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			continue
		}

		break // success or non-retryable error
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	log.Printf("metadata: got response status %d, %d bytes", resp.StatusCode, len(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	var result BatchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	log.Printf("metadata: decoded %d ISRCs, %d errors", len(result.ISRCs), len(result.Errors))

	return &result, nil
}

// Health checks if the metadata service is available
func (s *MetadataService) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	return nil
}
