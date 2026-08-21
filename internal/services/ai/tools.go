package ai

import (
	"context"
	"fmt"

	"github.com/aunali321/pi-go/agent"

	"github.com/Aunali321/korus/internal/services"
)

type searchArgs struct {
	Query string   `json:"query"`
	Types []string `json:"types"`
	Limit int      `json:"limit"`
}

type detailsArgs struct {
	Type string  `json:"type"`
	IDs  []int64 `json:"ids"`
}

type libraryArgs struct {
	Source string `json:"source"`
	Days   int    `json:"days"`
	Limit  int    `json:"limit"`
}

// readTools are the library reads shared by every surface. They never mutate
// anything, so radio and chat can both hold them.
func (s *Service) readTools(userID int64) []agent.Tool {
	return []agent.Tool{
		s.searchTool(userID),
		s.detailsTool(userID),
		s.myLibraryTool(userID),
	}
}

func (s *Service) searchTool(userID int64) agent.Tool {
	return agent.NewTool(agent.ToolDef[searchArgs]{
		Name: "search_library",
		Description: "Find music in the user's library by name. Searches song titles, album titles, artist names and playlist names. " +
			"Returns ids you can pass to get_details or to any playback tool. " +
			"Songs matching every word come first; if none do, the closest partial matches are returned instead, so an empty result means the library has nothing under that spelling.",
		Label: "search",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "Free text: a title, artist, album or playlist name."},
				"types": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string", "enum": []string{"song", "album", "artist", "playlist"}},
					"description": "Which kinds to search. Defaults to all of them.",
				},
				"limit": map[string]any{"type": "integer", "description": "Max results per kind (default 20, max 100)."},
			},
			"required": []string{"query"},
		},
		Run: func(ctx context.Context, _ string, a searchArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
			kinds := services.AllSearchKinds
			if len(a.Types) > 0 {
				kinds = make([]services.SearchKind, len(a.Types))
				for i, t := range a.Types {
					kinds[i] = services.SearchKind(t)
				}
			}
			res, err := s.search.Search(ctx, userID, a.Query, kinds, clampLimit(a.Limit, 20, 100), 0)
			if err != nil {
				return agent.ToolResult{}, err
			}
			return jsonResult(map[string]any{
				"songs":     briefs(res.Songs),
				"albums":    albumsOf(res.Albums),
				"artists":   artistsOf(res.Artists),
				"playlists": playlistsOf(res.Playlists, userID),
			}), nil
		},
	})
}

func (s *Service) detailsTool(userID int64) agent.Tool {
	return agent.NewTool(agent.ToolDef[detailsArgs]{
		Name: "get_details",
		Description: "Expand ids into full records. " +
			"song: lyrics and track number. album: its tracks in order. artist: their biography and albums. playlist: its songs in order. " +
			"Use this when you need what is inside something, not just its name.",
		Label: "details",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{"type": "string", "enum": []string{"song", "album", "artist", "playlist"}},
				"ids":  map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Ids of that kind, from an earlier tool result."},
			},
			"required": []string{"type", "ids"},
		},
		Run: func(ctx context.Context, _ string, a detailsArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
			items := make([]any, 0, len(a.IDs))
			for _, id := range a.IDs {
				item, err := s.detail(ctx, userID, a.Type, id)
				if err != nil {
					continue
				}
				items = append(items, item)
			}
			if len(items) == 0 {
				return textResult(fmt.Sprintf("No %s found for those ids.", a.Type)), nil
			}
			return jsonResult(items), nil
		},
	})
}

// detail builds one record for get_details. Each branch returns its own shape,
// which is the point of the tool: the caller asked for that kind.
func (s *Service) detail(ctx context.Context, userID int64, kind string, id int64) (any, error) {
	switch kind {
	case "song":
		song, err := s.library.Song(ctx, id)
		if err != nil {
			return nil, err
		}
		out := map[string]any{"song": brief(song), "favorited": s.library.IsFavorite(ctx, userID, services.EntitySong, id)}
		if song.TrackNumber != nil {
			out["track_number"] = *song.TrackNumber
		}
		if song.Lyrics != "" {
			out["lyrics"] = song.Lyrics
		}
		return out, nil

	case "album":
		album, err := s.library.Album(ctx, id)
		if err != nil {
			return nil, err
		}
		tracks, err := s.library.SongsByAlbum(ctx, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"album":     albumOf(album),
			"tracks":    briefs(tracks),
			"favorited": s.library.IsFavorite(ctx, userID, services.EntityAlbum, id),
		}, nil

	case "artist":
		artist, err := s.library.Artist(ctx, id)
		if err != nil {
			return nil, err
		}
		albums, err := s.library.AlbumsByArtist(ctx, id)
		if err != nil {
			return nil, err
		}
		songs, err := s.library.SongsByArtist(ctx, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"artist":    artistOf(artist),
			"albums":    albumsOf(albums),
			"songs":     briefs(songs),
			"following": s.library.IsFavorite(ctx, userID, services.EntityArtist, id),
		}, nil

	case "playlist":
		playlist, err := s.playlists.GetWithSongs(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"playlist": playlistOf(playlist, userID),
			"songs":    briefs(playlist.Songs),
		}, nil
	}
	return nil, fmt.Errorf("unknown type %q", kind)
}

func (s *Service) myLibraryTool(userID int64) agent.Tool {
	return agent.NewTool(agent.ToolDef[libraryArgs]{
		Name: "my_library",
		Description: "The user's own relationship to their library. source is one of: " +
			"top (most played), recent (recently played), favorites (songs they liked), " +
			"skipped (songs they repeatedly abandon partway, a signal of dislike), " +
			"unplayed (owned but never played, good for rediscovery), " +
			"playlists (their playlists), followed_artists (artists they follow).",
		Label: "my library",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source": map[string]any{
					"type": "string",
					"enum": []string{"top", "recent", "favorites", "skipped", "unplayed", "playlists", "followed_artists"},
				},
				"days":  map[string]any{"type": "integer", "description": "Only count listening from the last N days. Omit for all time. Applies to top, recent and skipped."},
				"limit": map[string]any{"type": "integer", "description": "Max results (default 20, max 200)."},
			},
			"required": []string{"source"},
		},
		Run: func(ctx context.Context, _ string, a libraryArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
			limit := clampLimit(a.Limit, 20, 200)

			switch a.Source {
			case "playlists":
				playlists, err := s.playlists.Mine(ctx, userID)
				if err != nil {
					return agent.ToolResult{}, err
				}
				return jsonResult(map[string]any{"playlists": playlistsOf(playlists, userID)}), nil

			case "followed_artists":
				artists, err := s.library.FollowedArtists(ctx, userID)
				if err != nil {
					return agent.ToolResult{}, err
				}
				return jsonResult(map[string]any{"artists": artistsOf(artists)}), nil
			}

			songs, err := s.library.UserSongs(ctx, userID, services.Slice(a.Source), a.Days, limit)
			if err != nil {
				return agent.ToolResult{}, err
			}
			return jsonResult(map[string]any{"songs": briefs(songs)}), nil
		},
	})
}

// submitSongIDsTool is a terminal tool: the agent calls it once with its final
// ordered song ids, which are written to *out, ending the run.
func submitSongIDsTool(out *[]int64) agent.Tool {
	type submitArgs struct {
		SongIDs []int64 `json:"song_ids"`
	}
	return agent.NewTool(agent.ToolDef[submitArgs]{
		Name:        "submit_songs",
		Description: "Deliver your final ordered list of chosen song ids. Call this exactly once when finished.",
		Label:       "submit",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"song_ids": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "integer"},
					"description": "Ordered list of chosen song ids drawn from the library.",
				},
			},
			"required": []string{"song_ids"},
		},
		Run: func(_ context.Context, _ string, a submitArgs, _ agent.UpdateFunc) (agent.ToolResult, error) {
			*out = append(*out, a.SongIDs...)
			result := textResult("Recorded.")
			result.Terminate = true
			return result, nil
		},
	})
}
