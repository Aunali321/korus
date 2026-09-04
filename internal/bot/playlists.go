package bot

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/Aunali321/korus/internal/bot/ui"
)

func (b *Bot) playlists(ctx context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	account, err := b.accounts.Get(ctx, e.User().ID.String())
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	playlists, err := account.Client.Playlists(ctx)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Playlists(playlists), nil
}

func (b *Bot) playlistView(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	account, err := b.accounts.Get(ctx, e.User().ID.String())
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	target, err := resolvePlaylist(ctx, account.Client, data.String("name"))
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	playlist, err := account.Client.Playlist(ctx, target.ID)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Playlist(playlist), nil
}

func (b *Bot) playlistCreate(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	account, err := b.accounts.Get(ctx, e.User().ID.String())
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	playlist, err := account.Client.CreatePlaylist(ctx, data.String("name"))
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Note(ui.ColorSuccess, "Created **%s** `#%d`", ui.Escape(playlist.Name), playlist.ID), nil
}

func (b *Bot) playlistAdd(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	account, err := b.accounts.Get(ctx, e.User().ID.String())
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	playlist, err := resolvePlaylist(ctx, account.Client, data.String("playlist"))
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	song, err := resolveSong(ctx, account.Client, data.String("song"))
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if err := account.Client.AddPlaylistSong(ctx, playlist.ID, song.ID); err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Note(ui.ColorSuccess, "Added **%s** to **%s**", ui.Escape(song.Title), ui.Escape(playlist.Name)), nil
}

func (b *Bot) playlistRemove(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	account, err := b.accounts.Get(ctx, e.User().ID.String())
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	target, err := resolvePlaylist(ctx, account.Client, data.String("playlist"))
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	playlist, err := account.Client.Playlist(ctx, target.ID)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	song, err := matchPlaylistSong(playlist, data.String("song"))
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if err := account.Client.RemovePlaylistSong(ctx, playlist.ID, song.ID); err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Note(ui.ColorSuccess, "Removed **%s** from **%s**", ui.Escape(song.Title), ui.Escape(playlist.Name)), nil
}
