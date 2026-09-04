package bot

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/Aunali321/korus/internal/bot/ui"
)

// periodLabels name the windows Korus resolves. "30d" is not a Korus period, so
// it falls through to the server's own default of the last 30 days.
var periodLabels = map[string]string{
	"30d":      "the last 30 days",
	"today":    "today",
	"week":     "the last week",
	"month":    "the last month",
	"year":     "the last year",
	"all_time": "all time",
}

func period(data discord.SlashCommandInteractionData, fallback string) (string, string) {
	value, ok := data.OptString("period")
	if !ok {
		value = fallback
	}
	return value, periodLabels[value]
}

func (b *Bot) stats(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	account, err := b.accounts.Get(ctx, e.User().ID.String())
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	name, label := period(data, "30d")
	stats, err := account.Client.Stats(ctx, name)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if stats.TotalPlays == 0 {
		return ui.Note(ui.ColorContent, "Nothing played in %s yet.", label), nil
	}
	return ui.Stats(stats, label), nil
}

func (b *Bot) wrapped(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	account, err := b.accounts.Get(ctx, e.User().ID.String())
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	name, label := period(data, "year")
	summary, err := account.Client.Wrapped(ctx, name)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if summary.TotalPlays == 0 {
		return ui.Note(ui.ColorWrapped, "Nothing played in %s yet.", label), nil
	}
	return ui.Wrapped(summary, label), nil
}
