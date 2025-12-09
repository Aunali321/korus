package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type MusicBrainzService struct {
	client    *http.Client
	userAgent string
	baseURL   string
}

type ListenBrainzService struct {
	client  *http.Client
	token   string
	user    string
	baseURL string
}

func NewMusicBrainzService(agent string) *MusicBrainzService {
	return &MusicBrainzService{
		client:    &http.Client{Timeout: 10 * time.Second},
		userAgent: agent,
		baseURL:   "https://musicbrainz.org/ws/2",
	}
}

func NewListenBrainzService(token, user string) *ListenBrainzService {
	return &ListenBrainzService{
		client:  &http.Client{Timeout: 10 * time.Second},
		token:   token,
		user:    user,
		baseURL: "https://api.listenbrainz.org/1",
	}
}

func (m *MusicBrainzService) Enrich(ctx context.Context, entity, id string) (string, error) {
	// Minimal stub: fetch by MBID if provided.
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s/%s?fmt=json", m.baseURL, entity, id), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", m.userAgent)
	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("musicbrainz status %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if idVal, ok := payload["id"].(string); ok {
		return idVal, nil
	}
	return "", fmt.Errorf("id not found")
}

func (l *ListenBrainzService) SubmitListen(ctx context.Context, track string, listenedAt int64) error {
	if l.token == "" || l.user == "" {
		return nil
	}
	body := fmt.Sprintf(`{"listen_type":"single","payload":[{"listened_at":%d,"track_metadata":{"artist_name":"","track_name":"%s","additional_info":{"media_player":"korus"}}}]}`, listenedAt, escape(track))
	req, err := http.NewRequestWithContext(ctx, "POST", l.baseURL+"/submit-listens", strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+l.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := l.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("listenbrainz status %d", resp.StatusCode)
	}
	return nil
}

func escape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
