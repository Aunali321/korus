// Package korus is a client for the Korus HTTP API, scoped to one linked account.
package korus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrUnlinked reports that the stored session is dead and the user must log in again.
var ErrUnlinked = errors.New("korus session expired")

const (
	requestTimeout = 30 * time.Second
	maxArtworkSize = 8 << 20
	maxErrorBody   = 4 << 10
)

// Tokens is one Korus session's credential pair.
type Tokens struct {
	Access  string
	Refresh string
}

// TokenSink persists rotated tokens. It is called with zero Tokens when the
// session dies, which is the signal to drop the link.
type TokenSink func(context.Context, Tokens) error

// APIError is a non-2xx response from Korus.
type APIError struct {
	Status  int    `json:"-"`
	Message string `json:"error"`
	Code    string `json:"code"`
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("korus: http %d", e.Status)
	}
	return "korus: " + e.Message
}

func (e *APIError) NotFound() bool { return e.Status == http.StatusNotFound }

// Client talks to one Korus server as one user, rotating tokens as they expire.
type Client struct {
	http    *http.Client
	baseURL string
	sink    TokenSink

	mu        sync.RWMutex
	tokens    Tokens
	refreshMu sync.Mutex
}

func New(baseURL string, tokens Tokens, sink TokenSink) *Client {
	return &Client{
		http:    &http.Client{Timeout: requestTimeout},
		baseURL: baseURL,
		sink:    sink,
		tokens:  tokens,
	}
}

func (c *Client) BaseURL() string { return c.baseURL }

// NormalizeURL turns pasted input such as "music.example.com/api/" into an origin.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("server url is required")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("%q is not a valid server url", raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported url scheme %q", u.Scheme)
	}
	u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), "/api")
	u.RawQuery, u.Fragment = "", ""
	return u.String(), nil
}

type sessionResponse struct {
	User    User   `json:"user"`
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
}

// Login authenticates against a Korus server and returns a fresh session.
func Login(ctx context.Context, baseURL, username, password string) (User, Tokens, error) {
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		return User{}, Tokens{}, err
	}
	hc := &http.Client{Timeout: requestTimeout}
	res, err := send(ctx, hc, http.MethodPost, baseURL+"/api/auth/login", "", body)
	if err != nil {
		return User{}, Tokens{}, err
	}
	defer res.Body.Close()

	var out sessionResponse
	if err := decode(res, &out); err != nil {
		return User{}, Tokens{}, err
	}
	return out.User, Tokens{Access: out.Access, Refresh: out.Refresh}, nil
}

func (c *Client) access() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tokens.Access
}

// Bearer returns an access token known to be valid, refreshing it if needed.
// FFmpeg gets this token in an Authorization header, so it must not be stale.
func (c *Client) Bearer(ctx context.Context) (string, error) {
	if err := c.do(ctx, http.MethodGet, "/api/auth/me", nil, nil); err != nil {
		return "", err
	}
	return c.access(), nil
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body []byte
	if in != nil {
		var err error
		if body, err = json.Marshal(in); err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}

	token := c.access()
	if token == "" {
		return ErrUnlinked
	}
	res, err := send(ctx, c.http, method, c.baseURL+path, token, body)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusUnauthorized {
		res.Body.Close()
		if err := c.refresh(ctx, token); err != nil {
			return err
		}
		if res, err = send(ctx, c.http, method, c.baseURL+path, c.access(), body); err != nil {
			return err
		}
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		return c.drop(ctx)
	}
	return decode(res, out)
}

// refresh rotates the session once. stale is the token that just failed, so
// concurrent callers that lost the race return without spending the new token.
func (c *Client) refresh(ctx context.Context, stale string) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()
	if c.access() != stale {
		return nil
	}

	c.mu.RLock()
	refreshToken := c.tokens.Refresh
	c.mu.RUnlock()

	body, err := json.Marshal(map[string]string{"refresh_token": refreshToken})
	if err != nil {
		return err
	}
	res, err := send(ctx, c.http, http.MethodPost, c.baseURL+"/api/auth/refresh", "", body)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return c.drop(ctx)
	}

	var out sessionResponse
	if err := decode(res, &out); err != nil {
		return err
	}
	tokens := Tokens{Access: out.Access, Refresh: out.Refresh}

	c.mu.Lock()
	c.tokens = tokens
	c.mu.Unlock()
	return c.sink(ctx, tokens)
}

func (c *Client) drop(ctx context.Context) error {
	c.Revoke()
	if err := c.sink(ctx, Tokens{}); err != nil {
		return err
	}
	return ErrUnlinked
}

// Revoke retires the client. Anything still holding it, such as a queued track
// waiting to record a listen, stops writing to the account.
func (c *Client) Revoke() {
	c.mu.Lock()
	c.tokens = Tokens{}
	c.mu.Unlock()
}

func send(ctx context.Context, hc *http.Client, method, url, bearer string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	res, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, url, err)
	}
	return res, nil
}

func decode(res *http.Response, out any) error {
	if res.StatusCode >= 300 {
		// The body is a bonus; the status alone already describes the failure.
		apiErr := &APIError{Status: res.StatusCode}
		json.NewDecoder(io.LimitReader(res.Body, maxErrorBody)).Decode(apiErr)
		return apiErr
	}
	if out == nil {
		io.Copy(io.Discard, res.Body)
		return nil
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) Me(ctx context.Context) (User, error) {
	var user User
	return user, c.do(ctx, http.MethodGet, "/api/auth/me", nil, &user)
}

func (c *Client) Search(ctx context.Context, query string, limit int) (SearchResult, error) {
	var res SearchResult
	q := url.Values{"q": {query}, "limit": {strconv.Itoa(limit)}}
	return res, c.do(ctx, http.MethodGet, "/api/search?"+q.Encode(), nil, &res)
}

func (c *Client) Song(ctx context.Context, id int64) (Song, error) {
	var song Song
	return song, c.do(ctx, http.MethodGet, "/api/songs/"+strconv.FormatInt(id, 10), nil, &song)
}

func (c *Client) Album(ctx context.Context, id int64) (AlbumDetail, error) {
	var album AlbumDetail
	return album, c.do(ctx, http.MethodGet, "/api/albums/"+strconv.FormatInt(id, 10), nil, &album)
}

func (c *Client) Artist(ctx context.Context, id int64) (ArtistDetail, error) {
	var artist ArtistDetail
	return artist, c.do(ctx, http.MethodGet, "/api/artists/"+strconv.FormatInt(id, 10), nil, &artist)
}

func (c *Client) Lyrics(ctx context.Context, songID int64) (Lyrics, error) {
	var lyrics Lyrics
	return lyrics, c.do(ctx, http.MethodGet, "/api/lyrics/"+strconv.FormatInt(songID, 10), nil, &lyrics)
}

func (c *Client) Stats(ctx context.Context, period string) (Stats, error) {
	var stats Stats
	q := url.Values{"period": {period}}
	return stats, c.do(ctx, http.MethodGet, "/api/stats?"+q.Encode(), nil, &stats)
}

func (c *Client) Wrapped(ctx context.Context, period string) (Summary, error) {
	var summary Summary
	q := url.Values{"period": {period}}
	return summary, c.do(ctx, http.MethodGet, "/api/stats/wrapped?"+q.Encode(), nil, &summary)
}

func (c *Client) Playlists(ctx context.Context) ([]Playlist, error) {
	var playlists []Playlist
	return playlists, c.do(ctx, http.MethodGet, "/api/playlists", nil, &playlists)
}

func (c *Client) Playlist(ctx context.Context, id int64) (Playlist, error) {
	var playlist Playlist
	return playlist, c.do(ctx, http.MethodGet, "/api/playlists/"+strconv.FormatInt(id, 10), nil, &playlist)
}

func (c *Client) CreatePlaylist(ctx context.Context, name string) (Playlist, error) {
	var playlist Playlist
	in := map[string]any{"name": name, "public": false}
	return playlist, c.do(ctx, http.MethodPost, "/api/playlists", in, &playlist)
}

func (c *Client) AddPlaylistSong(ctx context.Context, playlistID, songID int64) error {
	path := "/api/playlists/" + strconv.FormatInt(playlistID, 10) + "/songs"
	return c.do(ctx, http.MethodPost, path, map[string]int64{"song_id": songID}, nil)
}

func (c *Client) RemovePlaylistSong(ctx context.Context, playlistID, songID int64) error {
	path := "/api/playlists/" + strconv.FormatInt(playlistID, 10) + "/songs/" + strconv.FormatInt(songID, 10)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) Radio(ctx context.Context, seedID int64, limit int) ([]Song, error) {
	var out struct {
		Songs []Song `json:"songs"`
	}
	q := url.Values{"limit": {strconv.Itoa(limit)}}
	path := "/api/radio/" + strconv.FormatInt(seedID, 10) + "?" + q.Encode()
	return out.Songs, c.do(ctx, http.MethodGet, path, nil, &out)
}

func (c *Client) History(ctx context.Context, limit int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	q := url.Values{"limit": {strconv.Itoa(limit)}}
	return entries, c.do(ctx, http.MethodGet, "/api/history?"+q.Encode(), nil, &entries)
}

func (c *Client) RecordListen(ctx context.Context, listen Listen) error {
	return c.do(ctx, http.MethodPost, "/api/history", listen, nil)
}

// DownloadURL is the authenticated original-file endpoint FFmpeg reads from.
func (c *Client) DownloadURL(songID int64) string {
	return c.baseURL + "/api/download/" + strconv.FormatInt(songID, 10)
}

// Artwork fetches cover art bytes. Korus serves artwork unauthenticated, but the
// server may not be reachable from Discord, so covers travel as attachments.
func (c *Client) Artwork(ctx context.Context, songID int64) ([]byte, error) {
	return c.fetchImage(ctx, "/api/artwork/"+strconv.FormatInt(songID, 10))
}

func (c *Client) AlbumArtwork(ctx context.Context, albumID int64) ([]byte, error) {
	return c.fetchImage(ctx, "/api/artwork/"+strconv.FormatInt(albumID, 10)+"?type=album")
}

func (c *Client) ArtistImage(ctx context.Context, artistID int64) ([]byte, error) {
	return c.fetchImage(ctx, "/api/artist-image/"+strconv.FormatInt(artistID, 10))
}

func (c *Client) fetchImage(ctx context.Context, path string) ([]byte, error) {
	res, err := send(ctx, c.http, http.MethodGet, c.baseURL+path, c.access(), nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, &APIError{Status: res.StatusCode, Message: "artwork unavailable"}
	}
	return io.ReadAll(io.LimitReader(res.Body, maxArtworkSize))
}
