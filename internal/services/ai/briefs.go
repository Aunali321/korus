package ai

import (
	"encoding/json"

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"

	"github.com/Aunali321/korus/internal/models"
)

// The brief types are what the model sees. They carry the ids it needs to act
// and to build UI, and nothing else: full models.Song rows would spend the
// context window on file paths and lyrics the model never asked for.

type songBrief struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	ArtistID int64  `json:"artist_id,omitempty"`
	Album    string `json:"album,omitempty"`
	AlbumID  int64  `json:"album_id,omitempty"`
	Year     int    `json:"year,omitempty"`
	Seconds  int    `json:"seconds,omitempty"`
}

type albumBrief struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist,omitempty"`
	ArtistID int64  `json:"artist_id,omitempty"`
	Year     int    `json:"year,omitempty"`
}

type artistBrief struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Bio  string `json:"bio,omitempty"`
}

type playlistBrief struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SongCount   int    `json:"song_count"`
	Mine        bool   `json:"mine"`
}

// brief projects a song. The artist is the song's own credited performer where
// there is one, since compilation tracks belong to their performer rather than
// to the album's credited artist.
func brief(s models.Song) songBrief {
	b := songBrief{ID: s.ID, Title: s.Title, Artist: "Unknown"}
	if s.Duration != nil {
		b.Seconds = *s.Duration
	}
	if len(s.Artists) > 0 {
		b.Artist = s.Artists[0].Name
		b.ArtistID = s.Artists[0].ID
	}
	if s.Album != nil {
		b.Album = s.Album.Title
		b.AlbumID = s.Album.ID
		if s.Album.Year != nil {
			b.Year = *s.Album.Year
		}
		if b.ArtistID == 0 && s.Album.Artist != nil {
			b.Artist = s.Album.Artist.Name
			b.ArtistID = s.Album.Artist.ID
		}
	}
	return b
}

func briefs(songs []models.Song) []songBrief {
	out := make([]songBrief, len(songs))
	for i, s := range songs {
		out[i] = brief(s)
	}
	return out
}

func albumOf(a models.Album) albumBrief {
	b := albumBrief{ID: a.ID, Title: a.Title}
	if a.Year != nil {
		b.Year = *a.Year
	}
	if a.Artist != nil {
		b.Artist = a.Artist.Name
		b.ArtistID = a.Artist.ID
	}
	return b
}

func albumsOf(albums []models.Album) []albumBrief {
	out := make([]albumBrief, len(albums))
	for i, a := range albums {
		out[i] = albumOf(a)
	}
	return out
}

func artistOf(a models.Artist) artistBrief {
	return artistBrief{ID: a.ID, Name: a.Name, Bio: a.Bio}
}

func artistsOf(artists []models.Artist) []artistBrief {
	out := make([]artistBrief, len(artists))
	for i, a := range artists {
		out[i] = artistOf(a)
	}
	return out
}

func playlistOf(p models.Playlist, userID int64) playlistBrief {
	return playlistBrief{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		SongCount:   p.SongCount,
		Mine:        p.UserID == userID,
	}
}

func playlistsOf(playlists []models.Playlist, userID int64) []playlistBrief {
	out := make([]playlistBrief, len(playlists))
	for i, p := range playlists {
		out[i] = playlistOf(p, userID)
	}
	return out
}

// jsonResult is how every read tool answers: JSON text for the model, and the
// same value in Details for any caller that wants it typed.
func jsonResult(v any) agent.ToolResult {
	b, err := json.Marshal(v)
	if err != nil {
		return textResult("error: " + err.Error())
	}
	return agent.ToolResult{Content: []llm.Content{&llm.Text{Text: string(b)}}, Details: v}
}

func textResult(text string) agent.ToolResult {
	return agent.ToolResult{Content: []llm.Content{&llm.Text{Text: text}}}
}

func clampLimit(n, def, max int) int {
	if n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}
