package bot

import (
	"bytes"
	"context"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

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
	client, err := b.personalClient(ctx, e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	song, err := resolveSong(ctx, client, data.String("song"))
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

// cover uploads artwork as an attachment. Korus is often only reachable on a
// private network, so Discord cannot fetch image URLs from it.
func (b *Bot) cover(ctx context.Context, fetch func(context.Context, int64) ([]byte, error), id int64) ([]*discord.File, string) {
	data, err := fetch(ctx, id)
	if err != nil || len(data) == 0 {
		return nil, ""
	}
	return []*discord.File{discord.NewFile(ui.CoverName, "", bytes.NewReader(data))}, ui.CoverRef
}
