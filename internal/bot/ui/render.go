package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/player"
)

// CoverName is the attachment filename cover art is uploaded under.
const (
	CoverName = "cover.jpg"
	CoverRef  = "attachment://" + CoverName
)

const (
	maxTitle   = 80
	maxBio     = 1000
	maxChoices = 25
)

// header renders a title block, adding a cover thumbnail when one is attached.
func header(thumbnail string, lines ...string) discord.ContainerSubComponent {
	texts := make([]discord.SectionSubComponent, len(lines))
	for i, line := range lines {
		texts[i] = discord.NewTextDisplay(line)
	}
	if thumbnail == "" {
		return discord.NewTextDisplay(strings.Join(lines, "\n"))
	}
	return discord.NewSection(texts...).WithAccessory(discord.NewThumbnail(thumbnail))
}

// SongLine is the one-line form of a song used across list views.
func SongLine(song korus.Song) string {
	return fmt.Sprintf("**%s** — %s `%s`",
		Escape(Truncate(song.Title, maxTitle)),
		Escape(Truncate(song.ArtistNames(), maxTitle)),
		Duration(song.Seconds()))
}

func albumLine(album korus.Album) string {
	year := album.YearText()
	if year != "" {
		year = " (" + year + ")"
	}
	return fmt.Sprintf("**%s**%s — %s `#%d`", Escape(Truncate(album.Title, maxTitle)), year, Escape(album.ArtistName()), album.ID)
}

func Search(result korus.SearchResult) discord.MessageUpdate {
	components := []discord.ContainerSubComponent{discord.NewTextDisplay("## Search results")}

	section := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		components = append(components,
			discord.NewSmallSeparator(),
			discord.NewTextDisplay("**"+title+"**\n"+Join(lines)))
	}

	songs := make([]string, len(result.Songs))
	for i, song := range result.Songs {
		songs[i] = fmt.Sprintf("%s `#%d`", SongLine(song), song.ID)
	}
	albums := make([]string, len(result.Albums))
	for i, album := range result.Albums {
		albums[i] = albumLine(album)
	}
	artists := make([]string, len(result.Artists))
	for i, artist := range result.Artists {
		artists[i] = fmt.Sprintf("**%s** `#%d`", Escape(Truncate(artist.Name, maxTitle)), artist.ID)
	}
	playlists := make([]string, len(result.Playlists))
	for i, playlist := range result.Playlists {
		playlists[i] = fmt.Sprintf("**%s** — %d tracks `#%d`", Escape(Truncate(playlist.Name, maxTitle)), playlist.SongCount, playlist.ID)
	}

	section("Songs", songs)
	section("Albums", albums)
	section("Artists", artists)
	section("Playlists", playlists)

	if len(components) == 1 {
		return Fail("Nothing in the library matched that.")
	}
	if len(result.Songs) > 0 {
		options := make([]discord.StringSelectMenuOption, 0, maxChoices)
		for _, song := range result.Songs {
			if len(options) == maxChoices {
				break
			}
			options = append(options, discord.NewStringSelectMenuOption(
				Truncate(song.Title, 90), strconv.FormatInt(song.ID, 10),
			).WithDescription(Truncate(song.ArtistNames(), 90)))
		}
		components = append(components,
			discord.NewSmallSeparator(),
			discord.NewActionRow(discord.NewStringSelectMenu(IDSearchPlay, "Play one of these", options...)))
	}
	return Reply(ColorContent, components...)
}

func Album(album korus.AlbumDetail, thumbnail string) discord.MessageUpdate {
	artist := "Unknown artist"
	if album.Artist != nil {
		artist = album.Artist.Name
	}
	year := ""
	if album.Year != nil && *album.Year > 0 {
		year = fmt.Sprintf(" · %d", *album.Year)
	}

	total := 0
	tracks := make([]string, len(album.Songs))
	for i, song := range album.Songs {
		total += song.Seconds()
		tracks[i] = fmt.Sprintf("`%2d.` %s", i+1, SongLine(song))
	}

	components := []discord.ContainerSubComponent{
		header(thumbnail,
			"## "+Escape(Truncate(album.Title, maxTitle)),
			"-# "+Escape(artist)+year,
		),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay(Join(tracks)),
		discord.NewSmallSeparator(),
		Text("-# %d tracks · %s · album `#%d`", len(album.Songs), Duration(total), album.ID),
		discord.NewActionRow(discord.NewPrimaryButton("Queue album", IDAlbumPlay+strconv.FormatInt(album.ID, 10))),
	}
	return Reply(ColorContent, components...)
}

func Artist(artist korus.ArtistDetail, thumbnail string) discord.MessageUpdate {
	components := []discord.ContainerSubComponent{
		header(thumbnail,
			"## "+Escape(Truncate(artist.Name, maxTitle)),
			fmt.Sprintf("-# %d albums · %d tracks · artist `#%d`", len(artist.Albums), len(artist.Songs), artist.ID),
		),
	}
	if bio := strings.TrimSpace(artist.Bio); bio != "" {
		components = append(components, discord.NewSmallSeparator(), discord.NewTextDisplay(Escape(Truncate(bio, maxBio))))
	}
	if len(artist.Albums) > 0 {
		lines := make([]string, len(artist.Albums))
		for i, album := range artist.Albums {
			year := album.YearText()
			if year != "" {
				year = " (" + year + ")"
			}
			lines[i] = fmt.Sprintf("**%s**%s `#%d`", Escape(Truncate(album.Title, maxTitle)), year, album.ID)
		}
		components = append(components, discord.NewSmallSeparator(), discord.NewTextDisplay("**Albums**\n"+Join(lines)))
	}
	if len(artist.Songs) > 0 {
		top := artist.Songs[:min(5, len(artist.Songs))]
		lines := make([]string, len(top))
		for i, song := range top {
			lines[i] = fmt.Sprintf("%s `#%d`", SongLine(song), song.ID)
		}
		components = append(components, discord.NewSmallSeparator(), discord.NewTextDisplay("**Tracks**\n"+Join(lines)))
	}
	return Reply(ColorContent, components...)
}

func Lyrics(song korus.Song, text, thumbnail string) discord.MessageUpdate {
	return Reply(ColorContent,
		header(thumbnail,
			"## "+Escape(Truncate(song.Title, maxTitle)),
			"-# "+Escape(song.ArtistNames()),
		),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay(Truncate(text, listBudget)),
	)
}

func Stats(stats korus.Stats, label string) discord.MessageUpdate {
	songs := make([]string, len(stats.TopSongs))
	for i, ranked := range stats.TopSongs {
		songs[i] = fmt.Sprintf("`%d.` **%s** — %s · %d plays", i+1,
			Escape(Truncate(ranked.Song.Title, maxTitle)), Escape(ranked.Song.ArtistNames()), ranked.PlayCount)
	}
	artists := make([]string, len(stats.TopArtists))
	for i, ranked := range stats.TopArtists {
		artists[i] = fmt.Sprintf("`%d.` **%s** · %d plays", i+1, Escape(Truncate(ranked.Artist.Name, maxTitle)), ranked.PlayCount)
	}
	albums := make([]string, len(stats.TopAlbums))
	for i, ranked := range stats.TopAlbums {
		albums[i] = fmt.Sprintf("`%d.` **%s** — %s · %d plays", i+1,
			Escape(Truncate(ranked.Album.Title, maxTitle)), Escape(ranked.Album.ArtistName()), ranked.PlayCount)
	}

	components := []discord.ContainerSubComponent{
		discord.NewTextDisplay("## Listening stats\n-# " + label),
		discord.NewSmallSeparator(),
		Text("**%d** plays · **%s** listened\n**%d** songs · **%d** artists · **%d** albums",
			stats.TotalPlays, Minutes(stats.TotalDuration/60), stats.UniqueSongs, stats.UniqueArtists, stats.UniqueAlbums),
	}
	if len(songs) > 0 {
		components = append(components, discord.NewSmallSeparator(), discord.NewTextDisplay("**Top songs**\n"+Join(songs)))
	}
	if len(artists) > 0 {
		components = append(components, discord.NewSmallSeparator(), discord.NewTextDisplay("**Top artists**\n"+Join(artists)))
	}
	if len(albums) > 0 {
		components = append(components, discord.NewSmallSeparator(), discord.NewTextDisplay("**Top albums**\n"+Join(albums)))
	}
	components = append(components, discord.NewSmallSeparator(), Text("-# %s to %s", date(stats.Period.Start), date(stats.Period.End)))
	return Reply(ColorContent, components...)
}

func Wrapped(summary korus.Summary, label string) discord.MessageUpdate {
	songs := make([]string, len(summary.TopSongs))
	for i, song := range summary.TopSongs {
		songs[i] = fmt.Sprintf("`%d.` **%s** — %s · %d plays", i+1,
			Escape(Truncate(song.Title, maxTitle)), Escape(song.ArtistName()), song.Plays)
	}
	artists := make([]string, len(summary.TopArtists))
	for i, artist := range summary.TopArtists {
		artists[i] = fmt.Sprintf("`%d.` **%s** · %d plays", i+1, Escape(Truncate(artist.Name, maxTitle)), artist.Plays)
	}

	components := []discord.ContainerSubComponent{
		discord.NewTextDisplay("## Wrapped\n-# " + label),
		discord.NewSmallSeparator(),
		Text("**%s** of music across **%d** days\n**%.1f** plays a day · **%d** songs · **%d** new artists",
			Minutes(summary.TotalMinutes), summary.DaysListened, summary.AvgPlaysPerDay, summary.UniqueSongs, summary.NewArtists),
	}
	if len(songs) > 0 {
		components = append(components, discord.NewSmallSeparator(), discord.NewTextDisplay("**On repeat**\n"+Join(songs)))
	}
	if len(artists) > 0 {
		components = append(components, discord.NewSmallSeparator(), discord.NewTextDisplay("**Your artists**\n"+Join(artists)))
	}
	return Reply(ColorWrapped, components...)
}

func Playlists(playlists []korus.Playlist) discord.MessageUpdate {
	if len(playlists) == 0 {
		return Note(ColorContent, "No playlists yet. Create one with /playlist create.")
	}
	lines := make([]string, len(playlists))
	for i, playlist := range playlists {
		visibility := "private"
		if playlist.Public {
			visibility = "public"
		}
		lines[i] = fmt.Sprintf("**%s** — %d tracks · %s `#%d`",
			Escape(Truncate(playlist.Name, maxTitle)), playlist.SongCount, visibility, playlist.ID)
	}
	return Reply(ColorContent,
		discord.NewTextDisplay("## Playlists"),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay(Join(lines)),
	)
}

func Playlist(playlist korus.Playlist) discord.MessageUpdate {
	total := 0
	lines := make([]string, len(playlist.Songs))
	for i, song := range playlist.Songs {
		total += song.Seconds()
		lines[i] = fmt.Sprintf("`%2d.` %s", i+1, SongLine(song))
	}
	body := discord.NewTextDisplay("-# This playlist is empty.")
	if len(lines) > 0 {
		body = discord.NewTextDisplay(Join(lines))
	}
	return Reply(ColorContent,
		Text("## %s\n-# %d tracks · %s · playlist `#%d`",
			Escape(Truncate(playlist.Name, maxTitle)), len(playlist.Songs), Duration(total), playlist.ID),
		discord.NewSmallSeparator(),
		body,
	)
}

func NowPlaying(snapshot player.Snapshot, thumbnail string) discord.MessageUpdate {
	song := snapshot.Current.Song
	state := ""
	if snapshot.Paused {
		state = "\n-# Paused"
	}
	return Reply(ColorContent,
		header(thumbnail,
			"## "+Escape(Truncate(song.Title, maxTitle)),
			"-# "+Escape(song.ArtistNames()),
			"-# Requested by "+Escape(snapshot.Current.Requester),
		),
		Text("%s%s", Progress(snapshot.Elapsed, song.Seconds()), state),
		discord.NewSmallSeparator(),
		Controls(snapshot.Paused),
	)
}

func Queue(snapshot player.Snapshot) discord.MessageUpdate {
	song := snapshot.Current.Song
	upNext := discord.NewTextDisplay("-# Nothing queued up next.")
	if len(snapshot.Queue) > 0 {
		lines := make([]string, len(snapshot.Queue))
		for i, track := range snapshot.Queue {
			lines[i] = fmt.Sprintf("`%2d.` %s · %s", i+1, SongLine(track.Song), Escape(track.Requester))
		}
		upNext = discord.NewTextDisplay("**Up next**\n" + Join(lines))
	}
	return Reply(ColorContent,
		Text("## Queue\n**%s** — %s\n%s",
			Escape(Truncate(song.Title, maxTitle)), Escape(song.ArtistNames()),
			Progress(snapshot.Elapsed, song.Seconds())),
		discord.NewSmallSeparator(),
		upNext,
		discord.NewSmallSeparator(),
		Controls(snapshot.Paused),
	)
}

// Queued confirms one track being added, or started when position is zero.
func Queued(track player.Track, position int, paused bool, thumbnail string) discord.MessageUpdate {
	title := "Now playing"
	if position > 0 {
		title = fmt.Sprintf("Queued at #%d", position)
	}
	return Reply(ColorSuccess,
		header(thumbnail,
			"### "+title,
			"**"+Escape(Truncate(track.Song.Title, maxTitle))+"** — "+Escape(track.Song.ArtistNames()),
			fmt.Sprintf("-# %s · requested by %s", Duration(track.Song.Seconds()), Escape(track.Requester)),
		),
		discord.NewSmallSeparator(),
		Controls(paused),
	)
}

// QueuedBatch confirms a multi-track add such as radio, an album or a playlist.
func QueuedBatch(title string, tracks []player.Track, paused bool) discord.MessageUpdate {
	lines := make([]string, len(tracks))
	total := 0
	for i, track := range tracks {
		total += track.Song.Seconds()
		lines[i] = fmt.Sprintf("`%2d.` %s", i+1, SongLine(track.Song))
	}
	return Reply(ColorSuccess,
		Text("## %s\n-# %d tracks · %s", Escape(title), len(tracks), Duration(total)),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay(Join(lines)),
		discord.NewSmallSeparator(),
		Controls(paused),
	)
}

func date(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	if value == "" {
		return "the beginning"
	}
	return value
}
