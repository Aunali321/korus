package bot

import (
	"bytes"
	"context"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/ui"
)

func (b *Bot) search(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	client, err := b.libraryClient(ctx, e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	result, err := client.Search(ctx, data.String("query"), searchLimit)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Search(result), nil
}

func (b *Bot) album(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	client, err := b.libraryClient(ctx, e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	album, err := resolveAlbum(ctx, client, data.String("title"))
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	cover, ref := b.cover(ctx, client.AlbumArtwork, album.ID)
	return ui.Attach(ui.Album(album, ref), cover...), nil
}

func (b *Bot) artist(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	client, err := b.libraryClient(ctx, e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	artist, err := resolveArtist(ctx, client, data.String("name"))
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	image, ref := b.cover(ctx, client.ArtistImage, artist.ID)
	return ui.Attach(ui.Artist(artist, ref), image...), nil
}

func (b *Bot) lyrics(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	client, song, err := b.lyricsTarget(ctx, e, data)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	lyrics, err := client.Lyrics(ctx, song.ID)
	if err != nil {
		return discord.MessageUpdate{}, failf("No lyrics stored for %s.", song.Title)
	}
	text := strings.TrimSpace(lyrics.Lyrics)
	if text == "" {
		text = strings.TrimSpace(lyrics.Synced)
	}
	if text == "" {
		return discord.MessageUpdate{}, failf("No lyrics stored for %s.", song.Title)
	}
	cover, ref := b.cover(ctx, client.Artwork, song.ID)
	return ui.Attach(ui.Lyrics(song, text, ref), cover...), nil
}

// lyricsTarget picks the song to look up. Without one named it falls back to
// what the guild is playing, read from the library that track streams from
// rather than the caller's own, since a session may run on someone else's.
func (b *Bot) lyricsTarget(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (*korus.Client, korus.Song, error) {
	if query := strings.TrimSpace(data.String("song")); query != "" {
		client, err := b.personalClient(ctx, e)
		if err != nil {
			return nil, korus.Song{}, err
		}
		song, err := resolveSong(ctx, client, query)
		return client, song, err
	}

	active, err := b.viewed(e)
	if err != nil {
		return nil, korus.Song{}, userError("Name a song, or start playback and I will use the current track.")
	}
	snapshot := active.Snapshot()
	if !snapshot.Playing {
		return nil, korus.Song{}, userError("Name a song, or start playback and I will use the current track.")
	}
	return active.Source(), snapshot.Current.Song, nil
}

// cover uploads artwork as an attachment. Korus is often only reachable on a
// private network, so Discord cannot fetch image URLs from it.
func (b *Bot) cover(ctx context.Context, fetch func(context.Context, int64) ([]byte, error), id int64) ([]*discord.File, string) {
	data, err := fetch(ctx, id)
	if err != nil || len(data) == 0 {
		return nil, ""
	}
	return []*discord.File{discord.NewFile(ui.CoverName, "", bytes.NewReader(data))}, ui.CoverRef
}
