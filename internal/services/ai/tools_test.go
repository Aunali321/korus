package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/aunali321/pi-go/agent"
	"github.com/aunali321/pi-go/llm"
	_ "modernc.org/sqlite"

	"github.com/Aunali321/korus/internal/db"
	"github.com/Aunali321/korus/internal/services"
)

// newTestService builds a Service wired to real services over a seeded
// in-memory database, with no model attached: these tests drive the tools the
// way the agent does, through Execute with raw JSON arguments.
func newTestService(t *testing.T) (*Service, int64, []int64) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	database.SetMaxOpenConns(1)
	if err := db.RunMigrations(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	exec := func(query string, args ...any) sql.Result {
		t.Helper()
		res, err := database.ExecContext(ctx, query, args...)
		if err != nil {
			t.Fatalf("seed (%s): %v", query, err)
		}
		return res
	}

	userID, _ := exec(`INSERT INTO users(username, password_hash, email) VALUES('tester', 'x', 't@example.com')`).LastInsertId()
	artistID, _ := exec(`INSERT INTO artists(name, bio) VALUES('Nadia Reid', 'A songwriter from Port Chalmers.')`).LastInsertId()
	albumID, _ := exec(`INSERT INTO albums(artist_id, title, year) VALUES(?, 'Preservation', 2017)`, artistID).LastInsertId()

	songIDs := []int64{}
	for i, title := range []string{"Richard", "Preservation", "The Arrow & The Aim"} {
		id, _ := exec(`INSERT INTO songs(album_id, title, track_number, duration_ms, file_path, lyrics) VALUES(?, ?, ?, ?, ?, ?)`,
			albumID, title, i+1, (i+3)*60000, "/music/"+title+".flac", "lyric line for "+title).LastInsertId()
		exec(`INSERT INTO song_artists(song_id, artist_id, role, position) VALUES(?, ?, 'primary', 0)`, id, artistID)
		exec(`INSERT INTO songs_fts(rowid, song_id, title, artist_name, album_title) VALUES(?, ?, ?, 'Nadia Reid', 'Preservation')`,
			id, id, title)
		songIDs = append(songIDs, id)
	}

	svc := &Service{
		db:        database,
		library:   services.NewLibraryService(database),
		playlists: services.NewPlaylistService(database),
		search:    services.NewSearchService(database),
	}
	return svc, userID, songIDs
}

// call runs a tool the way the agent does and returns the text the model sees
// plus the typed Details payload.
func call(t *testing.T, tool agent.Tool, args string) (string, any) {
	t.Helper()
	res, err := tool.Execute(context.Background(), "call-1", json.RawMessage(args), nil)
	if err != nil {
		t.Fatalf("%s: %v", tool.Name(), err)
	}
	var text strings.Builder
	for _, c := range res.Content {
		if txt, ok := c.(*llm.Text); ok {
			text.WriteString(txt.Text)
		}
	}
	return text.String(), res.Details
}

func toolNamed(tools []agent.Tool, name string) agent.Tool {
	for _, tool := range tools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}

func decode(t *testing.T, text string, into any) {
	t.Helper()
	if err := json.Unmarshal([]byte(text), into); err != nil {
		t.Fatalf("decode %q: %v", text, err)
	}
}

func TestChatToolSet(t *testing.T) {
	svc, userID, _ := newTestService(t)
	tools := append(svc.readTools(userID), svc.playerTool(userID, PlayerContext{}), svc.listeningStatsTool(userID), svc.uiTool())
	tools = append(tools, svc.writeTools(userID)...)

	want := []string{
		"search_library", "get_details", "my_library", "get_player", "get_listening_stats",
		"render_ui", "play", "set_queue", "playback_control", "create_playlist",
		"update_playlist", "set_favorite",
	}
	if len(tools) != len(want) {
		t.Fatalf("chat exposes %d tools, want %d", len(tools), len(want))
	}
	for _, name := range want {
		if toolNamed(tools, name) == nil {
			t.Fatalf("missing tool %q", name)
		}
	}
}

func TestSearchLibraryTool(t *testing.T) {
	svc, userID, songIDs := newTestService(t)
	tool := svc.searchTool(userID)

	text, _ := call(t, tool, `{"query":"richard"}`)
	var res struct {
		Songs   []songBrief   `json:"songs"`
		Artists []artistBrief `json:"artists"`
	}
	decode(t, text, &res)
	if len(res.Songs) != 1 || res.Songs[0].ID != songIDs[0] {
		t.Fatalf("songs = %+v, want Richard", res.Songs)
	}
	// The brief must carry the ids the UI and the write tools need.
	if res.Songs[0].AlbumID == 0 || res.Songs[0].ArtistID == 0 {
		t.Fatalf("brief missing ids: %+v", res.Songs[0])
	}
	if res.Songs[0].Seconds != 180 {
		t.Fatalf("seconds = %d, want 180", res.Songs[0].Seconds)
	}

	text, _ = call(t, tool, `{"query":"nadia","types":["artist"]}`)
	decode(t, text, &res)
	if len(res.Artists) != 1 || res.Artists[0].Bio == "" {
		t.Fatalf("artists = %+v, want Nadia Reid with a bio", res.Artists)
	}
	if len(res.Songs) != 0 {
		t.Fatalf("expected only artists to be searched")
	}
}

func TestGetDetailsTool(t *testing.T) {
	svc, userID, songIDs := newTestService(t)
	tool := svc.detailsTool(userID)

	text, _ := call(t, tool, `{"type":"song","ids":[`+itoa(songIDs[0])+`]}`)
	var songs []struct {
		Song   songBrief `json:"song"`
		Lyrics string    `json:"lyrics"`
		Track  int       `json:"track_number"`
	}
	decode(t, text, &songs)
	if len(songs) != 1 || songs[0].Lyrics == "" {
		t.Fatalf("song details = %+v, want lyrics", songs)
	}
	if songs[0].Track != 1 {
		t.Fatalf("track_number = %d, want 1", songs[0].Track)
	}

	albumID := songs[0].Song.AlbumID
	text, _ = call(t, tool, `{"type":"album","ids":[`+itoa(albumID)+`]}`)
	var albums []struct {
		Album  albumBrief  `json:"album"`
		Tracks []songBrief `json:"tracks"`
	}
	decode(t, text, &albums)
	if len(albums) != 1 || len(albums[0].Tracks) != 3 {
		t.Fatalf("album details = %+v, want 3 tracks", albums)
	}

	artistID := songs[0].Song.ArtistID
	text, _ = call(t, tool, `{"type":"artist","ids":[`+itoa(artistID)+`]}`)
	var artists []struct {
		Artist artistBrief  `json:"artist"`
		Albums []albumBrief `json:"albums"`
		Songs  []songBrief  `json:"songs"`
	}
	decode(t, text, &artists)
	if len(artists) != 1 || artists[0].Artist.Bio == "" || len(artists[0].Albums) != 1 || len(artists[0].Songs) != 3 {
		t.Fatalf("artist details = %+v, want bio, 1 album, 3 songs", artists)
	}

	text, _ = call(t, tool, `{"type":"song","ids":[9999]}`)
	if !strings.Contains(text, "No song found") {
		t.Fatalf("missing id = %q, want a plain explanation", text)
	}
}

func TestMyLibraryTool(t *testing.T) {
	svc, userID, songIDs := newTestService(t)
	tool := svc.myLibraryTool(userID)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := svc.db.ExecContext(ctx,
			`INSERT INTO play_history(user_id, song_id, completion_rate) VALUES(?, ?, 1.0)`, userID, songIDs[0]); err != nil {
			t.Fatalf("seed play: %v", err)
		}
	}
	if _, err := svc.db.ExecContext(ctx,
		`INSERT INTO favorites_songs(user_id, song_id) VALUES(?, ?)`, userID, songIDs[1]); err != nil {
		t.Fatalf("seed favorite: %v", err)
	}

	var res struct {
		Songs     []songBrief     `json:"songs"`
		Playlists []playlistBrief `json:"playlists"`
		Artists   []artistBrief   `json:"artists"`
	}

	text, _ := call(t, tool, `{"source":"top"}`)
	decode(t, text, &res)
	if len(res.Songs) != 1 || res.Songs[0].ID != songIDs[0] {
		t.Fatalf("top = %+v, want the played song", res.Songs)
	}

	text, _ = call(t, tool, `{"source":"favorites"}`)
	decode(t, text, &res)
	if len(res.Songs) != 1 || res.Songs[0].ID != songIDs[1] {
		t.Fatalf("favorites = %+v", res.Songs)
	}

	text, _ = call(t, tool, `{"source":"unplayed"}`)
	decode(t, text, &res)
	if len(res.Songs) != 2 {
		t.Fatalf("unplayed = %+v, want the 2 never-played songs", res.Songs)
	}

	if _, err := svc.playlists.Create(ctx, userID, "Evening", "", false, songIDs); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	text, _ = call(t, tool, `{"source":"playlists"}`)
	decode(t, text, &res)
	if len(res.Playlists) != 1 || res.Playlists[0].SongCount != 3 || !res.Playlists[0].Mine {
		t.Fatalf("playlists = %+v", res.Playlists)
	}

	text, _ = call(t, tool, `{"source":"followed_artists"}`)
	decode(t, text, &res)
	if len(res.Artists) != 0 {
		t.Fatalf("followed_artists = %+v, want none yet", res.Artists)
	}
}

func TestGetPlayerTool(t *testing.T) {
	svc, userID, songIDs := newTestService(t)
	tool := svc.playerTool(userID, PlayerContext{
		NowPlayingID: songIDs[0],
		QueueIDs:     []int64{songIDs[1], songIDs[2]},
		Shuffle:      true,
		Repeat:       "all",
	})

	text, _ := call(t, tool, `{}`)
	var res struct {
		NowPlaying  *songBrief  `json:"now_playing"`
		Queue       []songBrief `json:"queue"`
		QueueLength int         `json:"queue_length"`
		Shuffle     bool        `json:"shuffle"`
		Repeat      string      `json:"repeat"`
	}
	decode(t, text, &res)
	if res.NowPlaying == nil || res.NowPlaying.ID != songIDs[0] {
		t.Fatalf("now_playing = %+v", res.NowPlaying)
	}
	if len(res.Queue) != 2 || res.Queue[0].ID != songIDs[1] {
		t.Fatalf("queue = %+v, want queue order preserved", res.Queue)
	}
	if !res.Shuffle || res.Repeat != "all" || res.QueueLength != 2 {
		t.Fatalf("player modes = %+v", res)
	}
}

func TestPlaybackToolsEmitEffects(t *testing.T) {
	svc, userID, songIDs := newTestService(t)
	tools := svc.writeTools(userID)

	_, details := call(t, toolNamed(tools, "play"),
		`{"song_ids":[`+itoa(songIDs[0])+`],"mode":"next"}`)
	eff, ok := details.(Effect)
	if !ok || eff.Action != "play" || eff.Mode != "next" || len(eff.SongIDs) != 1 {
		t.Fatalf("play effect = %+v", details)
	}

	// An unknown mode falls back to playing now rather than failing the turn.
	_, details = call(t, toolNamed(tools, "play"), `{"song_ids":[`+itoa(songIDs[0])+`],"mode":"soon"}`)
	if eff = details.(Effect); eff.Mode != "now" {
		t.Fatalf("unknown mode = %q, want now", eff.Mode)
	}

	// An empty queue is how the model clears it, so the effect must still fire.
	_, details = call(t, toolNamed(tools, "set_queue"), `{"song_ids":[]}`)
	if eff = details.(Effect); eff.Action != "set_queue" || len(eff.SongIDs) != 0 {
		t.Fatalf("set_queue effect = %+v", details)
	}

	for _, control := range []string{"pause", "resume", "stop", "next", "previous",
		"shuffle_on", "shuffle_off", "repeat_off", "repeat_one", "repeat_all"} {
		_, details = call(t, toolNamed(tools, "playback_control"), `{"control":"`+control+`"}`)
		eff, ok := details.(Effect)
		if !ok || eff.Action != "playback_control" || eff.Control != control {
			t.Fatalf("playback control %q = %+v", control, details)
		}
	}
}

// Every control the schema advertises must be one the clients implement, or
// the model will confidently emit an action that silently does nothing.
func TestPlaybackControlsAreAllHandled(t *testing.T) {
	svc, userID, _ := newTestService(t)
	tool := toolNamed(svc.writeTools(userID), "playback_control")

	schema := tool.Schema()["properties"].(map[string]any)["control"].(map[string]any)
	advertised := schema["enum"].([]string)

	handled := map[string]bool{
		"pause": true, "resume": true, "stop": true, "next": true, "previous": true,
		"shuffle_on": true, "shuffle_off": true,
		"repeat_off": true, "repeat_one": true, "repeat_all": true,
	}
	for _, control := range advertised {
		if !handled[control] {
			t.Fatalf("control %q is offered to the model but not handled by the clients", control)
		}
	}
	if len(advertised) != len(handled) {
		t.Fatalf("clients handle %d controls, schema offers %d", len(handled), len(advertised))
	}
}

func TestPlaylistToolsWriteThroughService(t *testing.T) {
	svc, userID, songIDs := newTestService(t)
	tools := svc.writeTools(userID)
	ctx := context.Background()

	text, details := call(t, toolNamed(tools, "create_playlist"),
		`{"name":"Late Night","description":"quiet","song_ids":[`+itoa(songIDs[0])+`,`+itoa(songIDs[1])+`]}`)
	eff := details.(Effect)
	if eff.PlaylistID == 0 {
		t.Fatalf("create effect = %+v, want a playlist id", eff)
	}
	// The model must be told the new id, or it cannot keep working with it.
	if !strings.Contains(text, itoa(eff.PlaylistID)) {
		t.Fatalf("create text = %q, want the new id in it", text)
	}

	_, _ = call(t, toolNamed(tools, "update_playlist"),
		`{"playlist_id":`+itoa(eff.PlaylistID)+`,"song_ids":[`+itoa(songIDs[2])+`],"mode":"append"}`)
	songs, err := svc.playlists.Songs(ctx, eff.PlaylistID)
	if err != nil {
		t.Fatalf("read playlist: %v", err)
	}
	if len(songs) != 3 || songs[2].ID != songIDs[2] {
		t.Fatalf("after append = %d songs", len(songs))
	}

	// Replace is how the model removes and reorders.
	_, _ = call(t, toolNamed(tools, "update_playlist"),
		`{"playlist_id":`+itoa(eff.PlaylistID)+`,"song_ids":[`+itoa(songIDs[2])+`,`+itoa(songIDs[0])+`],"mode":"replace"}`)
	songs, _ = svc.playlists.Songs(ctx, eff.PlaylistID)
	if len(songs) != 2 || songs[0].ID != songIDs[2] || songs[1].ID != songIDs[0] {
		t.Fatalf("after replace = %+v", songs)
	}

	// A playlist owned by someone else is refused in words, not as a crash.
	other, _ := svc.db.ExecContext(ctx, `INSERT INTO users(username, password_hash, email) VALUES('other','x','o@example.com')`)
	otherID, _ := other.LastInsertId()
	foreign, err := svc.playlists.Create(ctx, otherID, "Theirs", "", false, songIDs[:1])
	if err != nil {
		t.Fatalf("create foreign: %v", err)
	}
	text, details = call(t, toolNamed(tools, "update_playlist"),
		`{"playlist_id":`+itoa(foreign.ID)+`,"song_ids":[`+itoa(songIDs[0])+`],"mode":"replace"}`)
	if details != nil {
		t.Fatalf("expected no effect for a refused write, got %+v", details)
	}
	if !strings.Contains(text, "not yours") {
		t.Fatalf("refusal text = %q", text)
	}
}

func TestSetFavoriteTool(t *testing.T) {
	svc, userID, songIDs := newTestService(t)
	tool := toolNamed(svc.writeTools(userID), "set_favorite")
	ctx := context.Background()

	_, details := call(t, tool, `{"type":"song","id":`+itoa(songIDs[0])+`}`)
	eff := details.(Effect)
	if eff.Action != "favorite_changed" || !eff.On || eff.Entity != "song" {
		t.Fatalf("favorite effect = %+v", eff)
	}
	if !svc.library.IsFavorite(ctx, userID, services.EntitySong, songIDs[0]) {
		t.Fatalf("song was not favorited")
	}

	_, _ = call(t, tool, `{"type":"song","id":`+itoa(songIDs[0])+`,"on":false}`)
	if svc.library.IsFavorite(ctx, userID, services.EntitySong, songIDs[0]) {
		t.Fatalf("song was not unfavorited")
	}

	text, details := call(t, tool, `{"type":"artist","id":9999}`)
	if details != nil {
		t.Fatalf("expected no effect for a missing artist, got %+v", details)
	}
	if !strings.Contains(text, "No artist with id 9999") {
		t.Fatalf("refusal text = %q", text)
	}
}

func TestRenderUIResolvesSongs(t *testing.T) {
	svc, userID, songIDs := newTestService(t)
	tool := svc.uiTool()

	_, details := call(t, tool, `{"spec":{"type":"section","props":{"title":"Picks"},"children":[
		{"type":"song_list","props":{"song_ids":[`+itoa(songIDs[0])+`,`+itoa(songIDs[1])+`]}},
		{"type":"song_card","props":{"song_id":`+itoa(songIDs[2])+`}}
	]}}`)
	_ = userID

	eff, ok := details.(Effect)
	if !ok || eff.Kind != EffectUI {
		t.Fatalf("ui effect = %+v", details)
	}
	children := eff.Spec["children"].([]any)
	list := children[0].(map[string]any)["props"].(map[string]any)
	if songs, ok := list["songs"].([]songBrief); !ok || len(songs) != 2 {
		t.Fatalf("song_list was not resolved: %+v", list)
	}
	card := children[1].(map[string]any)["props"].(map[string]any)
	if _, ok := card["song"].(songBrief); !ok {
		t.Fatalf("song_card was not resolved: %+v", card)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
