package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/player"
	"github.com/Aunali321/korus/internal/bot/ui"
)

const captionLoadWait = 10 * time.Second

// captionDraw follows the line being sung. Parsed lyrics are held across ticks,
// so only a track change costs a lookup.
func (b *Bot) captionDraw() draw {
	var (
		songID int64
		lines  []korus.Line
	)
	return func(active *player.Player, snapshot player.Snapshot) (discord.MessageUpdate, string, []*discord.File) {
		song := snapshot.Current.Song
		if song.ID != songID {
			songID, lines = song.ID, b.timedLyrics(active.Source(), song.ID)
		}
		if len(lines) == 0 {
			return ui.CaptionsWaiting(song), fmt.Sprintf("%d/none", song.ID), nil
		}
		index := korus.LineAt(lines, snapshot.Elapsed)
		return ui.Captions(song, lines, index), fmt.Sprintf("%d/%d", song.ID, index), nil
	}
}

func (b *Bot) timedLyrics(source *korus.Client, songID int64) []korus.Line {
	ctx, cancel := context.WithTimeout(context.Background(), captionLoadWait)
	defer cancel()
	lyrics, err := source.Lyrics(ctx, songID)
	if err != nil {
		return nil
	}
	return lyrics.Timed()
}

// captionsCommand toggles the live lyric display for the caller's session.
func (b *Bot) captionsCommand(_ context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	active, err := b.controlled(e)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if b.live.stop(active.GuildID(), liveCaptions) {
		return ui.Note(ui.ColorSuccess, "Captions stopped."), nil
	}
	if err := b.live.start(active.GuildID(), e.Channel().ID(), liveCaptions, b.captionDraw()); err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Note(ui.ColorSuccess, "Captions started."), nil
}
