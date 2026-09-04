package bot

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/player"
	"github.com/Aunali321/korus/internal/bot/store"
	"github.com/Aunali321/korus/internal/bot/ui"
)

const defaultRadioLimit = 20

// session is the playback context a caller may queue into.
//
// source is the library the audio comes from, always the host's account.
// listener is the account this caller's plays are written to, which is set only
// when they own an account on that same server.
type session struct {
	guildID   snowflake.ID
	channelID snowflake.ID
	host      snowflake.ID
	source    *korus.Client
	listener  *korus.Client
	requester string
}

func (b *Bot) session(ctx context.Context, e caller) (session, error) {
	if e.GuildID() == nil {
		return session{}, userError("Playback only works in a server.")
	}
	guildID, user := *e.GuildID(), e.User()
	channelID, err := b.voiceChannel(guildID, user.ID)
	if err != nil {
		return session{}, err
	}

	current := session{
		guildID:   guildID,
		channelID: channelID,
		host:      user.ID,
		requester: displayName(e),
	}
	// A guest with no account can still queue from the host's library; they just
	// have nowhere to record the play.
	account, accountErr := b.accounts.Get(ctx, user.ID.String())
	if accountErr != nil && !errors.Is(accountErr, store.ErrNoLink) {
		return session{}, accountErr
	}

	if active := b.players.Get(guildID); active != nil {
		if active.ChannelID() != channelID && active.Host() != user.ID {
			return session{}, failf("Korus is already playing in <#%d>. Join it to queue.", active.ChannelID())
		}
		current.host = active.Host()
		current.source = active.Source()
	} else if accountErr == nil {
		current.source = account.Client
	} else {
		host, source, err := b.borrowLibrary(ctx, guildID, channelID, user.ID)
		if err != nil {
			return session{}, err
		}
		current.host, current.source = host, source
	}

	if accountErr == nil && account.BaseURL == current.source.BaseURL() {
		current.listener = account.Client
	}
	return current, nil
}

func (b *Bot) enqueue(ctx context.Context, current session, songs []korus.Song) (*player.Player, []player.Track, int, error) {
	if len(songs) == 0 {
		return nil, nil, 0, userError("Nothing to queue.")
	}
	tracks := make([]player.Track, len(songs))
	for i, song := range songs {
		tracks[i] = player.Track{Song: song, Listener: current.listener, Requester: current.requester}
	}
	active, err := b.players.Join(ctx, current.guildID, current.channelID, current.host, current.source)
	if err != nil {
		if errors.Is(err, player.ErrBusy) {
			return nil, nil, 0, userError("Korus is already playing in another channel.")
		}
		return nil, nil, 0, err
	}
	return active, tracks, active.Enqueue(tracks...), nil
}

func (b *Bot) voiceChannel(guildID, userID snowflake.ID) (snowflake.ID, error) {
	state, ok := b.client.Caches.VoiceState(guildID, userID)
	if !ok || state.ChannelID == nil {
		return 0, userError("Join a voice channel first.")
	}
	return *state.ChannelID, nil
}

// borrowLibrary finds a linked listener in the caller's voice channel, so a guest
// can start a session from a library that is already in the room. That listener
// hosts the session, which leaves channel moves and /logout teardown unchanged.
// Candidates are ordered by id so the same library wins every time.
func (b *Bot) borrowLibrary(ctx context.Context, guildID, channelID, guest snowflake.ID) (snowflake.ID, *korus.Client, error) {
	var listeners []snowflake.ID
	for state := range b.client.Caches.VoiceStates(guildID) {
		if state.UserID != guest && state.ChannelID != nil && *state.ChannelID == channelID {
			listeners = append(listeners, state.UserID)
		}
	}
	slices.Sort(listeners)

	for _, id := range listeners {
		account, err := b.accounts.Get(ctx, id.String())
		if err == nil {
			return id, account.Client, nil
		}
		if !errors.Is(err, store.ErrNoLink) {
			return 0, nil, err
		}
	}
	return 0, nil, store.ErrNoLink
}

// controlled returns the guild's session, requiring the caller to be listening
// to it before they can change it.
func (b *Bot) controlled(e caller) (*player.Player, error) {
	active, err := b.viewed(e)
	if err != nil {
		return nil, err
	}
	channelID, err := b.voiceChannel(*e.GuildID(), e.User().ID)
	if err != nil {
		return nil, err
	}
	if channelID != active.ChannelID() {
		return nil, failf("Join <#%d> to control playback.", active.ChannelID())
	}
	return active, nil
}

func (b *Bot) viewed(e caller) (*player.Player, error) {
	if e.GuildID() == nil {
		return nil, userError("Playback only works in a server.")
	}
	active := b.players.Get(*e.GuildID())
	if active == nil {
		return nil, userError("Nothing is playing.")
	}
	return active, nil
}

func (b *Bot) play(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	current, err := b.session(ctx, e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	song, err := resolveSong(ctx, current.source, data.String("query"))
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	return b.queueOne(ctx, current, song, e.Channel().ID())
}

func (b *Bot) queueOne(ctx context.Context, current session, song korus.Song, textChannel snowflake.ID) (discord.MessageUpdate, error) {
	active, tracks, position, err := b.enqueue(ctx, current, []korus.Song{song})
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	b.showPanel(current.guildID, textChannel)
	cover, ref := b.cover(ctx, current.source.Artwork, song.ID)
	return ui.Attach(ui.Queued(tracks[0], position, active.Snapshot().Paused, ref), cover...), nil
}

func (b *Bot) radio(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	current, err := b.session(ctx, e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	seed, err := b.radioSeed(ctx, current, data)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	limit, ok := data.OptInt("limit")
	if !ok {
		limit = defaultRadioLimit
	}
	songs, err := current.source.Radio(ctx, seed.ID, limit)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if len(songs) == 0 {
		return discord.MessageUpdate{}, failf("Korus found nothing to play alongside %s.", seed.Title)
	}
	active, tracks, _, err := b.enqueue(ctx, current, songs)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	b.showPanel(current.guildID, e.Channel().ID())
	return ui.QueuedBatch("Radio from "+seed.Title, tracks, active.Snapshot().Paused), nil
}

// radioSeed falls back to the caller's last play on this library.
func (b *Bot) radioSeed(ctx context.Context, current session, data discord.SlashCommandInteractionData) (korus.Song, error) {
	if seed, ok := data.OptString("seed"); ok && seed != "" {
		return resolveSong(ctx, current.source, seed)
	}
	if current.listener == nil {
		return korus.Song{}, userError("Pick a seed song: you have no history on this library.")
	}
	entries, err := current.listener.History(ctx, 1)
	if err != nil {
		return korus.Song{}, err
	}
	if len(entries) == 0 {
		return korus.Song{}, userError("Play something first, or pick a seed song.")
	}
	return current.source.Song(ctx, entries[0].SongID)
}

func (b *Bot) pause(_ context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	active, err := b.controlled(e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if !active.Pause() {
		return ui.Note(ui.ColorContent, "Already paused."), nil
	}
	return ui.Note(ui.ColorSuccess, "Paused."), nil
}

func (b *Bot) resume(_ context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	active, err := b.controlled(e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if !active.Resume() {
		return ui.Note(ui.ColorContent, "Already playing."), nil
	}
	return ui.Note(ui.ColorSuccess, "Resumed."), nil
}

func (b *Bot) skip(_ context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	active, err := b.controlled(e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	result, ok := active.Skip()
	if !ok {
		return discord.MessageUpdate{}, userError("Nothing is playing.")
	}
	if len(result.Upcoming) == 0 {
		return ui.Note(ui.ColorSuccess, "Skipped **%s**. Queue is empty.", ui.Escape(result.Skipped.Song.Title)), nil
	}
	return ui.Note(ui.ColorSuccess, "Skipped **%s**. Up next **%s**.",
		ui.Escape(result.Skipped.Song.Title), ui.Escape(result.Upcoming[0].Song.Title)), nil
}

func (b *Bot) stop(_ context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	active, err := b.controlled(e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	active.Stop()
	return ui.Note(ui.ColorSuccess, "Stopped. Queue cleared."), nil
}

func (b *Bot) queue(_ context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	return b.queueView(e)
}

// nowPlaying brings the live player down to the bottom of the channel, rather
// than posting a second copy that starts going stale immediately.
func (b *Bot) nowPlaying(_ context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	active, err := b.viewed(e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if !active.Snapshot().Playing {
		return discord.MessageUpdate{}, userError("Nothing is playing.")
	}
	b.movePanel(active.GuildID(), e.Channel().ID())
	return ui.Note(ui.ColorSuccess, "Player moved down here."), nil
}

func (b *Bot) queueView(e caller) (discord.MessageUpdate, error) {
	active, err := b.viewed(e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	snapshot := active.Snapshot()
	if !snapshot.Playing {
		return discord.MessageUpdate{}, userError("Nothing is playing.")
	}
	return ui.Queue(snapshot), nil
}

// panelStep coarsens elapsed time so the progress bar looks like it is moving
// without editing the message every second.
const panelStep = 5 * time.Second

// panelDraw renders the live player. Artwork is only re-sent when the track
// changes, so a progress tick leaves the attachment already on the message.
func (b *Bot) panelDraw() draw {
	var (
		songID int64
		ref    string
	)
	return func(active *player.Player, snapshot player.Snapshot) (discord.MessageUpdate, string, []*discord.File) {
		var files []*discord.File
		song := snapshot.Current.Song
		if song.ID != songID {
			songID = song.ID
			ctx, cancel := context.WithTimeout(context.Background(), captionLoadWait)
			files, ref = b.cover(ctx, active.Source().Artwork, song.ID)
			cancel()
		}
		return ui.Panel(snapshot, ref), fmt.Sprintf("%d/%t/%d/%d",
			song.ID, snapshot.Paused, len(snapshot.Queue), snapshot.Elapsed/panelStep), files
	}
}

// showPanel puts the live player in the channel, leaving it alone when it is
// already running there so queueing does not make it jump about.
func (b *Bot) showPanel(guildID, channelID snowflake.ID) {
	if at, running := b.live.channelOf(guildID, livePanel); running && at == channelID {
		return
	}
	b.movePanel(guildID, channelID)
}

func (b *Bot) movePanel(guildID, channelID snowflake.ID) {
	if err := b.live.start(guildID, channelID, livePanel, b.panelDraw()); err != nil {
		b.log.Debug("cannot show the player panel", "err", err)
	}
}

// The panel redraws itself, so a control only has to act and acknowledge.
func (b *Bot) toggleAction(e *handler.ComponentEvent) error {
	active, err := b.controlled(e)
	if err != nil {
		return err
	}
	if !active.Pause() {
		active.Resume()
	}
	return nil
}

func (b *Bot) skipAction(e *handler.ComponentEvent) error {
	active, err := b.controlled(e)
	if err != nil {
		return err
	}
	if _, ok := active.Skip(); !ok {
		return userError("Nothing is playing.")
	}
	return nil
}

func (b *Bot) stopAction(e *handler.ComponentEvent) error {
	active, err := b.controlled(e)
	if err != nil {
		return err
	}
	active.Stop()
	return nil
}

func (b *Bot) albumPlayButton(ctx context.Context, e *handler.ComponentEvent) (discord.MessageUpdate, error) {
	id, err := strconv.ParseInt(e.Vars["album"], 10, 64)
	if err != nil {
		return discord.MessageUpdate{}, userError("That album is no longer valid.")
	}
	current, err := b.session(ctx, e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	album, err := current.source.Album(ctx, id)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	active, tracks, _, err := b.enqueue(ctx, current, album.Songs)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	b.showPanel(current.guildID, e.Channel().ID())
	return ui.QueuedBatch(album.Title, tracks, active.Snapshot().Paused), nil
}

func (b *Bot) searchPlaySelect(ctx context.Context, e *handler.ComponentEvent) (discord.MessageUpdate, error) {
	values := e.StringSelectMenuInteractionData().Values
	if len(values) == 0 {
		return discord.MessageUpdate{}, userError("Nothing selected.")
	}
	id, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return discord.MessageUpdate{}, userError("That result is no longer valid.")
	}
	current, err := b.session(ctx, e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	song, err := current.source.Song(ctx, id)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	return b.queueOne(ctx, current, song, e.Channel().ID())
}
