package bot

import (
	"context"
	"sync"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/ui"
)

const (
	captionTick = time.Second
	// Discord rate limits message edits per channel, and a fast song can change
	// line twice a second, so edits are spaced out even when the line has moved.
	captionMinEdit = 2 * time.Second
	// A caption message a listener deleted would otherwise error every tick.
	captionMaxErrors = 5
	captionLoadWait  = 10 * time.Second
)

// captions keeps one live lyric message per guild, rewriting it as the track
// plays.
type captions struct {
	bot *Bot

	mu       sync.Mutex
	sessions map[snowflake.ID]*captionSession
}

type captionSession struct {
	guildID   snowflake.ID
	channelID snowflake.ID
	messageID snowflake.ID
	stop      chan struct{}
	once      sync.Once
}

func (s *captionSession) close() { s.once.Do(func() { close(s.stop) }) }

func newCaptions(b *Bot) *captions {
	return &captions{bot: b, sessions: map[snowflake.ID]*captionSession{}}
}

// Start posts a caption message and follows playback until it ends. Any caption
// already running in the guild is replaced.
func (c *captions) Start(guildID, channelID snowflake.ID) error {
	active := c.bot.players.Get(guildID)
	if active == nil {
		return userError("Nothing is playing.")
	}
	c.Stop(guildID)

	message, err := c.bot.client.Rest.CreateMessage(channelID,
		ui.Create(ui.CaptionsWaiting(active.Snapshot().Current.Song)))
	if err != nil {
		return err
	}

	session := &captionSession{
		guildID:   guildID,
		channelID: channelID,
		messageID: message.ID,
		stop:      make(chan struct{}),
	}
	c.mu.Lock()
	c.sessions[guildID] = session
	c.mu.Unlock()

	go c.follow(session)
	return nil
}

// Stop ends a guild's captions, reporting whether any were running. The message
// is removed as the loop exits.
func (c *captions) Stop(guildID snowflake.ID) bool {
	c.mu.Lock()
	session, running := c.sessions[guildID]
	c.mu.Unlock()
	if running {
		session.close()
	}
	return running
}

func (c *captions) StopAll() {
	c.mu.Lock()
	sessions := make([]*captionSession, 0, len(c.sessions))
	for _, session := range c.sessions {
		sessions = append(sessions, session)
	}
	c.mu.Unlock()
	for _, session := range sessions {
		session.close()
	}
}

func (c *captions) follow(session *captionSession) {
	defer c.finish(session)

	ticker := time.NewTicker(captionTick)
	defer ticker.Stop()

	var (
		songID   int64
		lines    []korus.Line
		shown    = -2
		lastEdit time.Time
		failures int
	)
	for {
		select {
		case <-session.stop:
			return
		case <-ticker.C:
		}

		active := c.bot.players.Get(session.guildID)
		if active == nil {
			return
		}
		snapshot := active.Snapshot()
		if !snapshot.Playing {
			return
		}

		if snapshot.Current.Song.ID != songID {
			songID, lines, shown = snapshot.Current.Song.ID, c.timed(active.Source(), snapshot.Current.Song.ID), -2
		}
		index := korus.LineAt(lines, snapshot.Elapsed)
		if index == shown || time.Since(lastEdit) < captionMinEdit {
			continue
		}

		update := ui.Captions(snapshot.Current.Song, lines, index)
		if len(lines) == 0 {
			update = ui.CaptionsWaiting(snapshot.Current.Song)
		}
		if _, err := c.bot.client.Rest.UpdateMessage(session.channelID, session.messageID, update); err != nil {
			if failures++; failures >= captionMaxErrors {
				c.bot.log.Debug("giving up on captions", "err", err)
				return
			}
			continue
		}
		shown, lastEdit, failures = index, time.Now(), 0
	}
}

func (c *captions) timed(source *korus.Client, songID int64) []korus.Line {
	ctx, cancel := context.WithTimeout(context.Background(), captionLoadWait)
	defer cancel()
	lyrics, err := source.Lyrics(ctx, songID)
	if err != nil {
		return nil
	}
	return lyrics.Timed()
}

func (c *captions) finish(session *captionSession) {
	c.mu.Lock()
	if c.sessions[session.guildID] == session {
		delete(c.sessions, session.guildID)
	}
	c.mu.Unlock()
	session.close()

	if err := c.bot.client.Rest.DeleteMessage(session.channelID, session.messageID); err != nil {
		c.bot.log.Debug("cannot remove caption message", "err", err)
	}
}

// captionsCommand toggles the live lyric display for the caller's session.
func (b *Bot) captionsCommand(_ context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	active, err := b.controlled(e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if b.captions.Stop(active.GuildID()) {
		return ui.Note(ui.ColorSuccess, "Captions stopped."), nil
	}
	if err := b.captions.Start(active.GuildID(), e.Channel().ID()); err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Note(ui.ColorSuccess, "Captions started."), nil
}
