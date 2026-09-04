// Package bot is the Korus Discord client: slash commands, Components V2 views
// and one voice session per guild.
package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo"
	disgobot "github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/handler/middleware"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/godave/golibdave"
	"github.com/disgoorg/snowflake/v2"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/player"
	"github.com/Aunali321/korus/internal/bot/store"
	"github.com/Aunali321/korus/internal/bot/ui"
)

// userError carries a message meant for the person who ran the command.
// Anything else is a fault: logged here, shown to the user as a generic failure.
type userError string

func (e userError) Error() string { return string(e) }

func failf(format string, a ...any) error {
	return userError(fmt.Sprintf(format, a...))
}

// caller is the part of an interaction the handlers need, shared by slash
// commands, autocomplete requests and component clicks.
type caller interface {
	GuildID() *snowflake.ID
	User() discord.User
	Member() *discord.ResolvedMember
}

// displayName prefers the caller's nickname in this server.
func displayName(e caller) string {
	if member := e.Member(); member != nil && member.Nick != nil && *member.Nick != "" {
		return *member.Nick
	}
	return e.User().EffectiveName()
}

// queueNoticeTTL is how long a "queued at #n" confirmation stays in the channel.
const queueNoticeTTL = 10 * time.Second

type Bot struct {
	cfg      Config
	log      *slog.Logger
	store    *store.Store
	accounts *accounts
	client   *disgobot.Client
	players  *player.Manager
	captions *captions
}

func New(cfg Config, log *slog.Logger) (*Bot, error) {
	linkStore, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		cfg:      cfg,
		log:      log,
		store:    linkStore,
		accounts: newAccounts(linkStore),
	}

	client, err := disgo.New(cfg.Token,
		disgobot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentGuilds, gateway.IntentGuildVoiceStates)),
		disgobot.WithCacheConfigOpts(cache.WithCaches(cache.FlagGuilds, cache.FlagChannels, cache.FlagVoiceStates)),
		// Discord closes the voice gateway with 4017 unless DAVE is negotiated,
		// so the noop session disgo defaults to cannot carry audio any more.
		disgobot.WithVoiceManagerConfigOpts(voice.WithDaveSessionCreateFunc(golibdave.NewSession)),
		disgobot.WithEventListeners(b.routes(),
			disgobot.NewListenerFunc(b.onVoiceLeave),
			disgobot.NewListenerFunc(b.onVoiceMove),
		),
	)
	if err != nil {
		linkStore.Close()
		return nil, fmt.Errorf("build discord client: %w", err)
	}

	b.client = client
	b.players = player.NewManager(client, cfg.FFmpeg, log)
	b.captions = newCaptions(b)
	return b, nil
}

// Start syncs commands and opens the gateway.
func (b *Bot) Start(ctx context.Context) error {
	var guilds []snowflake.ID
	if b.cfg.GuildID != 0 {
		guilds = append(guilds, b.cfg.GuildID)
	}
	if err := handler.SyncCommands(b.client, commands, guilds); err != nil {
		return fmt.Errorf("sync commands: %w", err)
	}
	if err := b.client.OpenGateway(ctx); err != nil {
		return fmt.Errorf("open gateway: %w", err)
	}
	return nil
}

func (b *Bot) Close(ctx context.Context) {
	b.captions.StopAll()
	b.players.StopAll()
	b.client.Close(ctx)
	b.store.Close()
}

func (b *Bot) routes() *handler.Mux {
	router := handler.New()
	router.Use(middleware.Go)

	// Personal data stays private to the caller; library and playback are shared.
	router.Group(func(r handler.Router) {
		r.Use(middleware.Defer(discord.InteractionTypeApplicationCommand, false, true))
		r.SlashCommand("/login", b.slash(b.login))
		r.SlashCommand("/logout", b.slash(b.logout))
		r.SlashCommand("/whoami", b.slash(b.whoami))
		r.SlashCommand("/wrapped", b.slash(b.wrapped))
		r.SlashCommand("/playlists", b.slash(b.playlists))
		r.SlashCommand("/playlist/view", b.slash(b.playlistView))
		r.SlashCommand("/playlist/create", b.slash(b.playlistCreate))
		r.SlashCommand("/playlist/add", b.slash(b.playlistAdd))
		r.SlashCommand("/playlist/remove", b.slash(b.playlistRemove))
	})

	// Stats are about one person, so they stay private unless shared.
	router.Group(func(r handler.Router) {
		r.Use(b.deferShared(false))
		r.SlashCommand("/stats", b.slash(b.stats))
	})

	router.Group(func(r handler.Router) {
		r.Use(middleware.Defer(discord.InteractionTypeApplicationCommand, false, false))
		r.SlashCommand("/search", b.slash(b.search))
		r.SlashCommand("/album", b.slash(b.album))
		r.SlashCommand("/artist", b.slash(b.artist))
		r.SlashCommand("/play", b.slashQueue(b.play))
		r.SlashCommand("/radio", b.slashQueue(b.radio))
		r.SlashCommand("/pause", b.slash(b.pause))
		r.SlashCommand("/resume", b.slash(b.resume))
		r.SlashCommand("/skip", b.slash(b.skip))
		r.SlashCommand("/stop", b.slash(b.stop))
		r.SlashCommand("/queue", b.slash(b.queue))
		r.SlashCommand("/nowplaying", b.slash(b.nowPlaying))
		r.SlashCommand("/lyrics", b.slash(b.lyrics))
		r.SlashCommand("/captions", b.slash(b.captionsCommand))
	})

	router.Group(func(r handler.Router) {
		r.Autocomplete("/album", b.complete(b.libraryClient, b.completeAlbums))
		r.Autocomplete("/artist", b.complete(b.libraryClient, b.completeArtists))
		r.Autocomplete("/lyrics", b.complete(b.personalClient, b.completeSongs))
		r.Autocomplete("/play", b.complete(b.libraryClient, b.completeSongs))
		r.Autocomplete("/radio", b.complete(b.libraryClient, b.completeSongs))
		r.Autocomplete("/playlist/view", b.complete(b.personalClient, b.completePlaylists))
		r.Autocomplete("/playlist/add", b.complete(b.personalClient, b.completePlaylistAdd))
		r.Autocomplete("/playlist/remove", b.complete(b.personalClient, b.completePlaylistRemove))
	})

	// Player buttons rewrite the message they live on.
	router.Group(func(r handler.Router) {
		r.Use(middleware.Defer(discord.InteractionTypeComponent, true, false))
		r.Component(ui.IDToggle, b.component(b.toggleButton))
		r.Component(ui.IDSkip, b.component(b.skipButton))
		r.Component(ui.IDStop, b.component(b.stopButton))
		r.Component(ui.IDQueue, b.component(b.queueButton))
		r.Component(ui.IDNowPlaying, b.component(b.nowPlayingButton))
	})

	router.Component(ui.IDCaptionsStop, func(e *handler.ComponentEvent) error {
		if e.GuildID() != nil {
			b.captions.Stop(*e.GuildID())
		}
		return e.DeferUpdateMessage()
	})

	// Queueing from a result list posts its own confirmation.
	router.Group(func(r handler.Router) {
		r.Use(middleware.Defer(discord.InteractionTypeComponent, false, false))
		r.Component(ui.IDSearchPlay, b.componentQueue(b.searchPlaySelect))
		r.Component(ui.IDAlbumPlay+"{album}", b.componentQueue(b.albumPlayButton))
	})

	return router
}

type commandFunc func(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error)

func (b *Bot) slash(fn commandFunc) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		update, err := fn(e.Ctx, e, data)
		if err != nil {
			update = b.render(err)
		}
		_, err = e.UpdateInteractionResponse(update)
		return err
	}
}

// deferShared defers a command privately unless its share option opts the reply
// into the channel. Discord fixes visibility at defer time, so this has to run
// before the handler.
func (b *Bot) deferShared(shareByDefault bool) handler.Middleware {
	return func(next handler.Handler) handler.Handler {
		return func(e *handler.InteractionEvent) error {
			share := shareByDefault
			if command, ok := e.Interaction.(discord.ApplicationCommandInteraction); ok {
				if choice, set := command.SlashCommandInteractionData().OptBool(optShare); set {
					share = choice
				}
			}
			var data discord.InteractionResponseData
			if !share {
				data = discord.MessageCreate{Flags: discord.MessageFlagEphemeral}
			}
			if err := e.Respond(discord.InteractionResponseTypeDeferredCreateMessage, data); err != nil {
				return err
			}
			return next(e)
		}
	}
}

type queueFunc func(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, bool, error)

// slashQueue is slash for commands that confirm a queue action. A confirmation
// that nothing started playing is noise once it has been read, so it is dropped
// shortly after.
func (b *Bot) slashQueue(fn queueFunc) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		update, transient, err := fn(e.Ctx, e, data)
		if err != nil {
			update, transient = b.render(err), false
		}
		if _, err := e.UpdateInteractionResponse(update); err != nil {
			return err
		}
		b.expire(e.DeleteInteractionResponse, transient)
		return nil
	}
}

func (b *Bot) componentQueue(fn func(ctx context.Context, e *handler.ComponentEvent) (discord.MessageUpdate, bool, error)) handler.ComponentHandler {
	return func(e *handler.ComponentEvent) error {
		update, transient, err := fn(e.Ctx, e)
		if err != nil {
			update, transient = b.render(err), false
		}
		if _, err := e.UpdateInteractionResponse(update); err != nil {
			return err
		}
		b.expire(e.DeleteInteractionResponse, transient)
		return nil
	}
}

// expire removes a queue confirmation once it has served its purpose. The
// message is cosmetic, so a failed delete is logged and forgotten.
func (b *Bot) expire(delete func(...rest.RequestOpt) error, transient bool) {
	if !transient {
		return
	}
	go func() {
		time.Sleep(queueNoticeTTL)
		if err := delete(); err != nil {
			b.log.Debug("cannot drop queue confirmation", "err", err)
		}
	}()
}

func (b *Bot) component(fn func(ctx context.Context, e *handler.ComponentEvent) (discord.MessageUpdate, error)) handler.ComponentHandler {
	return func(e *handler.ComponentEvent) error {
		update, err := fn(e.Ctx, e)
		if err != nil {
			update = b.render(err)
		}
		_, err = e.UpdateInteractionResponse(update)
		return err
	}
}

// render turns an error into the message the user sees.
func (b *Bot) render(err error) discord.MessageUpdate {
	if user, ok := errors.AsType[userError](err); ok {
		return ui.Fail(string(user))
	}
	if errors.Is(err, store.ErrNoLink) {
		return ui.Fail("Link your Korus account first with /login.")
	}
	if errors.Is(err, korus.ErrUnlinked) {
		return ui.Fail("Your Korus session expired. Run /login again.")
	}
	if apiErr, ok := errors.AsType[*korus.APIError](err); ok && apiErr.NotFound() {
		return ui.Fail("Korus has nothing matching that.")
	}
	b.log.Error("command failed", "err", err)
	return ui.Fail("Something went wrong. Try again in a moment.")
}

// onVoiceLeave ends a session when the bot is disconnected or its channel empties.
func (b *Bot) onVoiceLeave(e *events.GuildVoiceLeave) {
	active := b.players.Get(e.VoiceState.GuildID)
	if active == nil {
		return
	}
	if e.VoiceState.UserID == b.client.ID() {
		active.Stop()
		return
	}
	b.stopIfEmpty(active, e.OldVoiceState.ChannelID)
}

// onVoiceMove follows the bot when a moderator drags it, and treats anyone else
// moving away as leaving.
func (b *Bot) onVoiceMove(e *events.GuildVoiceMove) {
	active := b.players.Get(e.VoiceState.GuildID)
	if active == nil {
		return
	}
	if e.VoiceState.UserID == b.client.ID() {
		if e.VoiceState.ChannelID != nil {
			active.SetChannel(*e.VoiceState.ChannelID)
		}
		return
	}
	b.stopIfEmpty(active, e.OldVoiceState.ChannelID)
}

func (b *Bot) stopIfEmpty(active *player.Player, left *snowflake.ID) {
	if left == nil || *left != active.ChannelID() {
		return
	}
	for state := range b.client.Caches.VoiceStates(active.GuildID()) {
		if state.ChannelID != nil && *state.ChannelID == active.ChannelID() && state.UserID != b.client.ID() {
			return
		}
	}
	active.Stop()
}
