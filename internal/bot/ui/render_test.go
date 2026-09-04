package ui

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/disgoorg/disgo/discord"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/player"
)

// Components V2 limits, enforced here because Discord only reports them at send time.
const (
	maxComponents = 40
	maxText       = 4000
)

func song(id int64, title string) korus.Song {
	duration := 214
	year := 2024
	return korus.Song{
		ID:       id,
		Title:    title,
		Duration: &duration,
		Album:    &korus.Album{ID: 9, Title: "An Album With A Fairly Long Name", Year: &year},
		Artists:  []korus.Artist{{ID: 3, Name: "Some Artist"}, {ID: 4, Name: "Another Artist"}},
	}
}

func songs(n int) []korus.Song {
	out := make([]korus.Song, n)
	for i := range out {
		out[i] = song(int64(i+1), strings.Repeat("Very Long Track Title ", 5))
	}
	return out
}

func tracks(n int) []player.Track {
	out := make([]player.Track, n)
	for i, s := range songs(n) {
		out[i] = player.Track{Song: s, Requester: "a listener with a long display name"}
	}
	return out
}

func TestViewsFitDiscordLimits(t *testing.T) {
	year := 2024
	long := strings.Repeat("biography ", 400)

	views := map[string]discord.MessageUpdate{
		"fail": Fail("something broke"),
		"search": Search(korus.SearchResult{
			Songs:     songs(10),
			Albums:    []korus.Album{{ID: 1, Title: "Album", Year: &year}},
			Artists:   []korus.Artist{{ID: 2, Name: "Artist"}},
			Playlists: []korus.Playlist{{ID: 3, Name: "Playlist", SongCount: 12}},
		}),
		"album": Album(korus.AlbumDetail{
			ID: 1, Title: "Album", Year: &year,
			Artist: &korus.Artist{ID: 2, Name: "Artist"},
			Songs:  songs(120),
		}, CoverRef),
		"artist": Artist(korus.ArtistDetail{
			ID: 2, Name: "Artist", Bio: long,
			Albums: []korus.Album{{ID: 1, Title: "Album", Year: &year}},
			Songs:  songs(40),
		}, CoverRef),
		"lyrics":    Lyrics(song(1, "Track"), strings.Repeat("a lyric line\n", 900), CoverRef),
		"playlists": Playlists([]korus.Playlist{{ID: 1, Name: "Mix", SongCount: 4, Public: true}}),
		"playlist":  Playlist(korus.Playlist{ID: 1, Name: "Mix", Songs: songs(200)}),
		"stats": Stats(korus.Stats{
			Period:     korus.Period{Start: "2026-01-01T00:00:00Z", End: "2026-02-01T00:00:00Z"},
			TotalPlays: 120, TotalDuration: 40000, UniqueSongs: 30, UniqueArtists: 12, UniqueAlbums: 9,
			TopSongs:   []korus.RankedSong{{Song: song(1, "Track"), PlayCount: 9}},
			TopArtists: []korus.RankedArtist{{Artist: korus.Artist{Name: "Artist"}, PlayCount: 8}},
			TopAlbums:  []korus.RankedAlbum{{Album: korus.Album{Title: "Album"}, PlayCount: 7}},
		}, "the last 30 days"),
		"wrapped": Wrapped(korus.Summary{
			TotalPlays: 90, TotalMinutes: 800, DaysListened: 20, AvgPlaysPerDay: 4.5, UniqueSongs: 40, NewArtists: 6,
			TopSongs:   []korus.PlayedSong{{ID: 1, Title: "Track", Plays: 12}},
			TopArtists: []korus.PlayedArtist{{ID: 2, Name: "Artist", Plays: 30}},
		}, "the last year"),
		"nowplaying": NowPlaying(player.Snapshot{
			Current: player.Track{Song: song(1, "Track"), Requester: "listener"},
			Elapsed: 42 * time.Second, Playing: true, Queue: tracks(3),
		}, CoverRef),
		"queue": Queue(player.Snapshot{
			Current: player.Track{Song: song(1, "Track"), Requester: "listener"},
			Elapsed: 42 * time.Second, Playing: true, Queue: tracks(150),
		}),
		"queued":      Queued(player.Track{Song: song(1, "Track"), Requester: "listener"}, 3, false, CoverRef),
		"queuedBatch": QueuedBatch("Radio from Track", tracks(150), false),
	}

	for name, view := range views {
		t.Run(name, func(t *testing.T) {
			if view.Flags == nil || !view.Flags.Has(discord.MessageFlagIsComponentsV2) {
				t.Fatal("view is missing the components v2 flag")
			}
			raw, err := json.Marshal(view.Components)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var payload []map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			count, text := 0, 0
			var walk func(nodes []any)
			walk = func(nodes []any) {
				for _, node := range nodes {
					object, ok := node.(map[string]any)
					if !ok {
						continue
					}
					count++
					if content, ok := object["content"].(string); ok {
						if strings.TrimSpace(content) == "" {
							t.Error("empty text display")
						}
						text += utf8.RuneCountInString(content)
					}
					if children, ok := object["components"].([]any); ok {
						walk(children)
					}
					if accessory, ok := object["accessory"].(map[string]any); ok {
						walk([]any{accessory})
					}
				}
			}
			nodes := make([]any, len(payload))
			for i, object := range payload {
				nodes[i] = object
			}
			walk(nodes)

			if count > maxComponents {
				t.Errorf("uses %d components, limit is %d", count, maxComponents)
			}
			if text > maxText {
				t.Errorf("renders %d characters, limit is %d", text, maxText)
			}
			t.Logf("components=%d characters=%d", count, text)
		})
	}
}
