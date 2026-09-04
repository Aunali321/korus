package bot

import (
	"context"
	"strconv"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/ui"
)

const (
	minQueryLength = 2
	searchLimit    = 10
	choiceLabel    = 90
	maxChoices     = 25
)

// resolveSong accepts either an autocomplete value, which is a numeric id, or
// free text to search for. Every resolver below follows the same rule.
func resolveSong(ctx context.Context, client *korus.Client, query string) (korus.Song, error) {
	if id, err := strconv.ParseInt(query, 10, 64); err == nil {
		return client.Song(ctx, id)
	}
	result, err := client.Search(ctx, query, 1)
	if err != nil {
		return korus.Song{}, err
	}
	if len(result.Songs) == 0 {
		return korus.Song{}, failf("No song matched %q.", query)
	}
	return result.Songs[0], nil
}

func resolveAlbum(ctx context.Context, client *korus.Client, query string) (korus.AlbumDetail, error) {
	if id, err := strconv.ParseInt(query, 10, 64); err == nil {
		return client.Album(ctx, id)
	}
	result, err := client.Search(ctx, query, 1)
	if err != nil {
		return korus.AlbumDetail{}, err
	}
	if len(result.Albums) == 0 {
		return korus.AlbumDetail{}, failf("No album matched %q.", query)
	}
	return client.Album(ctx, result.Albums[0].ID)
}

func resolveArtist(ctx context.Context, client *korus.Client, query string) (korus.ArtistDetail, error) {
	if id, err := strconv.ParseInt(query, 10, 64); err == nil {
		return client.Artist(ctx, id)
	}
	result, err := client.Search(ctx, query, 1)
	if err != nil {
		return korus.ArtistDetail{}, err
	}
	if len(result.Artists) == 0 {
		return korus.ArtistDetail{}, failf("No artist matched %q.", query)
	}
	return client.Artist(ctx, result.Artists[0].ID)
}

// resolvePlaylist matches against the caller's own playlists so a name never
// resolves to somebody else's public list.
func resolvePlaylist(ctx context.Context, client *korus.Client, query string) (korus.Playlist, error) {
	playlists, err := client.Playlists(ctx)
	if err != nil {
		return korus.Playlist{}, err
	}
	if id, err := strconv.ParseInt(query, 10, 64); err == nil {
		for _, playlist := range playlists {
			if playlist.ID == id {
				return playlist, nil
			}
		}
		return korus.Playlist{}, failf("You have no playlist with id %d.", id)
	}
	needle := strings.ToLower(query)
	for _, playlist := range playlists {
		if strings.Contains(strings.ToLower(playlist.Name), needle) {
			return playlist, nil
		}
	}
	return korus.Playlist{}, failf("No playlist matched %q.", query)
}

// matchPlaylistSong finds a track inside one playlist, by id or substring.
func matchPlaylistSong(playlist korus.Playlist, query string) (korus.Song, error) {
	if id, err := strconv.ParseInt(query, 10, 64); err == nil {
		for _, song := range playlist.Songs {
			if song.ID == id {
				return song, nil
			}
		}
		return korus.Song{}, failf("%s has no song with id %d.", playlist.Name, id)
	}
	needle := strings.ToLower(query)
	for _, song := range playlist.Songs {
		if strings.Contains(strings.ToLower(song.Title), needle) {
			return song, nil
		}
	}
	return korus.Song{}, failf("No song in %s matched %q.", playlist.Name, query)
}

type clientFunc func(ctx context.Context, e caller) (*korus.Client, error)

type completeFunc func(ctx context.Context, client *korus.Client, query string, data discord.AutocompleteInteractionData) ([]discord.AutocompleteChoice, error)

// complete answers an autocomplete request, staying silent on any failure so a
// slow or unlinked lookup never blocks typing.
func (b *Bot) complete(clientFor clientFunc, fn completeFunc) handler.AutocompleteHandler {
	return func(e *handler.AutocompleteEvent) error {
		query := strings.TrimSpace(e.Data.String(e.Data.Focused().Name))
		if len(query) < minQueryLength {
			return e.AutocompleteResult(nil)
		}
		client, err := clientFor(e.Ctx, e)
		if err != nil {
			return e.AutocompleteResult(nil)
		}
		choices, err := fn(e.Ctx, client, query, e.Data)
		if err != nil {
			b.log.Debug("autocomplete lookup failed", "err", err)
			return e.AutocompleteResult(nil)
		}
		return e.AutocompleteResult(choices)
	}
}

// personalClient reads the caller's own account, for anything private to them.
func (b *Bot) personalClient(ctx context.Context, e caller) (*korus.Client, error) {
	account, err := b.accounts.Get(ctx, e.User().ID.String())
	if err != nil {
		return nil, err
	}
	return account.Client, nil
}

// libraryClient reads the running session's library, so everyone in a group
// session browses the catalogue the host can actually play from.
func (b *Bot) libraryClient(ctx context.Context, e caller) (*korus.Client, error) {
	if e.GuildID() != nil {
		if active := b.players.Get(*e.GuildID()); active != nil {
			return active.Source(), nil
		}
	}
	return b.personalClient(ctx, e)
}

func (b *Bot) completeSongs(ctx context.Context, client *korus.Client, query string, _ discord.AutocompleteInteractionData) ([]discord.AutocompleteChoice, error) {
	result, err := client.Search(ctx, query, searchLimit)
	if err != nil {
		return nil, err
	}
	return songChoices(result.Songs), nil
}

func (b *Bot) completeAlbums(ctx context.Context, client *korus.Client, query string, _ discord.AutocompleteInteractionData) ([]discord.AutocompleteChoice, error) {
	result, err := client.Search(ctx, query, searchLimit)
	if err != nil {
		return nil, err
	}
	choices := make([]discord.AutocompleteChoice, 0, len(result.Albums))
	for _, album := range result.Albums {
		choices = append(choices, choiceOf(album.Title+" - "+album.ArtistName(), album.ID))
	}
	return choices, nil
}

func (b *Bot) completeArtists(ctx context.Context, client *korus.Client, query string, _ discord.AutocompleteInteractionData) ([]discord.AutocompleteChoice, error) {
	result, err := client.Search(ctx, query, searchLimit)
	if err != nil {
		return nil, err
	}
	choices := make([]discord.AutocompleteChoice, 0, len(result.Artists))
	for _, artist := range result.Artists {
		choices = append(choices, choiceOf(artist.Name, artist.ID))
	}
	return choices, nil
}

func (b *Bot) completePlaylists(ctx context.Context, client *korus.Client, query string, _ discord.AutocompleteInteractionData) ([]discord.AutocompleteChoice, error) {
	playlists, err := client.Playlists(ctx)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(query)
	choices := make([]discord.AutocompleteChoice, 0, maxChoices)
	for _, playlist := range playlists {
		if len(choices) == maxChoices {
			break
		}
		if strings.Contains(strings.ToLower(playlist.Name), needle) {
			choices = append(choices, choiceOf(playlist.Name, playlist.ID))
		}
	}
	return choices, nil
}

func (b *Bot) completePlaylistAdd(ctx context.Context, client *korus.Client, query string, data discord.AutocompleteInteractionData) ([]discord.AutocompleteChoice, error) {
	if data.Focused().Name == "playlist" {
		return b.completePlaylists(ctx, client, query, data)
	}
	return b.completeSongs(ctx, client, query, data)
}

// completePlaylistRemove suggests only tracks that are in the chosen playlist.
func (b *Bot) completePlaylistRemove(ctx context.Context, client *korus.Client, query string, data discord.AutocompleteInteractionData) ([]discord.AutocompleteChoice, error) {
	if data.Focused().Name == "playlist" {
		return b.completePlaylists(ctx, client, query, data)
	}
	target, err := resolvePlaylist(ctx, client, data.String("playlist"))
	if err != nil {
		return nil, nil
	}
	playlist, err := client.Playlist(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(query)
	var matched []korus.Song
	for _, song := range playlist.Songs {
		if strings.Contains(strings.ToLower(song.Title), needle) {
			matched = append(matched, song)
		}
	}
	return songChoices(matched), nil
}

func songChoices(songs []korus.Song) []discord.AutocompleteChoice {
	choices := make([]discord.AutocompleteChoice, 0, min(len(songs), maxChoices))
	for _, song := range songs {
		if len(choices) == maxChoices {
			break
		}
		choices = append(choices, choiceOf(song.Title+" - "+song.ArtistNames(), song.ID))
	}
	return choices
}

func choiceOf(label string, id int64) discord.AutocompleteChoice {
	return discord.AutocompleteChoiceString{
		Name:  ui.Truncate(label, choiceLabel),
		Value: strconv.FormatInt(id, 10),
	}
}
