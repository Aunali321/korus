package bot

import (
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"

	"github.com/Aunali321/korus/internal/bot/player"
	"github.com/Aunali321/korus/internal/bot/ui"
)

// liveKind names the displays that can follow one guild at the same time.
type liveKind int

const (
	livePanel liveKind = iota
	liveCaptions
)

const (
	liveTick = time.Second
	// Discord rate limits edits per channel and two displays can share one, so
	// edits are spaced even when there is something new to show.
	liveMinEdit = 2 * time.Second
	// A message a listener deleted would otherwise fail on every tick.
	liveMaxErrors = 5
)

// draw renders a display along with a fingerprint of what it shows, and the
// cover to attach when the artwork has changed. The message is only edited when
// the fingerprint moves, so a paused session costs nothing.
type draw func(active *player.Player, snapshot player.Snapshot) (discord.MessageUpdate, string, []*discord.File)

// live keeps self-updating messages in step with playback. They are ordinary
// channel messages: an interaction response expires after fifteen minutes, and
// these outlive a song.
type live struct {
	bot *Bot

	mu    sync.Mutex
	views map[liveKey]*liveView
}

type liveKey struct {
	guild snowflake.ID
	kind  liveKind
}

type liveView struct {
	key       liveKey
	channelID snowflake.ID
	messageID snowflake.ID
	stop      chan struct{}
	once      sync.Once
}

func (v *liveView) close() { v.once.Do(func() { close(v.stop) }) }

func newLive(b *Bot) *live {
	return &live{bot: b, views: map[liveKey]*liveView{}}
}

// channelOf reports where a display is running, if it is.
func (l *live) channelOf(guild snowflake.ID, kind liveKind) (snowflake.ID, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	view, running := l.views[liveKey{guild, kind}]
	if !running {
		return 0, false
	}
	return view.channelID, true
}

// start posts a display and follows playback until it ends, replacing any of the
// same kind already running in the guild.
func (l *live) start(guild, channel snowflake.ID, kind liveKind, render draw) error {
	active := l.bot.players.Get(guild)
	if active == nil {
		return userError("Nothing is playing.")
	}
	l.stop(guild, kind)

	first, _, files := render(active, active.Snapshot())
	message, err := l.bot.client.Rest.CreateMessage(channel, ui.Create(ui.Attach(first, files...)))
	if err != nil {
		return err
	}

	view := &liveView{
		key:       liveKey{guild, kind},
		channelID: channel,
		messageID: message.ID,
		stop:      make(chan struct{}),
	}
	l.mu.Lock()
	l.views[view.key] = view
	l.mu.Unlock()

	go l.follow(view, render)
	return nil
}

// stop ends a display, reporting whether it was running. The message goes as the
// loop unwinds.
func (l *live) stop(guild snowflake.ID, kind liveKind) bool {
	l.mu.Lock()
	view, running := l.views[liveKey{guild, kind}]
	l.mu.Unlock()
	if running {
		view.close()
	}
	return running
}

func (l *live) stopAll() {
	l.mu.Lock()
	views := make([]*liveView, 0, len(l.views))
	for _, view := range l.views {
		views = append(views, view)
	}
	l.mu.Unlock()
	for _, view := range views {
		view.close()
	}
}

func (l *live) follow(view *liveView, render draw) {
	defer l.finish(view)

	ticker := time.NewTicker(liveTick)
	defer ticker.Stop()

	var (
		shown    string
		lastEdit time.Time
		failures int
		// Artwork is offered once, on the tick the track changes. That tick is
		// often throttled, so it is held until an edit actually lands.
		pending []*discord.File
	)
	for {
		select {
		case <-view.stop:
			return
		case <-ticker.C:
		}

		active := l.bot.players.Get(view.key.guild)
		if active == nil {
			return
		}
		snapshot := active.Snapshot()
		if !snapshot.Playing {
			return
		}

		update, fingerprint, files := render(active, snapshot)
		if len(files) > 0 {
			pending = files
		}
		if fingerprint == shown || time.Since(lastEdit) < liveMinEdit {
			continue
		}
		// An update that leaves attachments alone keeps the one already on the
		// message, so artwork only rides along when it has changed.
		if len(pending) > 0 {
			update = ui.Attach(update, pending...)
		}
		if _, err := l.bot.client.Rest.UpdateMessage(view.channelID, view.messageID, update); err != nil {
			if failures++; failures >= liveMaxErrors {
				l.bot.log.Debug("giving up on a live display", "kind", view.key.kind, "err", err)
				return
			}
			continue
		}
		shown, lastEdit, failures, pending = fingerprint, time.Now(), 0, nil
	}
}

func (l *live) finish(view *liveView) {
	l.mu.Lock()
	if l.views[view.key] == view {
		delete(l.views, view.key)
	}
	l.mu.Unlock()
	view.close()

	if err := l.bot.client.Rest.DeleteMessage(view.channelID, view.messageID); err != nil {
		l.bot.log.Debug("cannot remove a live display", "kind", view.key.kind, "err", err)
	}
}
