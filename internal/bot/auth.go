package bot

import (
	"context"
	"errors"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/Aunali321/korus/internal/bot/korus"
	"github.com/Aunali321/korus/internal/bot/ui"
)

func (b *Bot) login(ctx context.Context, e *handler.CommandEvent, data discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	baseURL, err := korus.NormalizeURL(data.String("url"))
	if err != nil {
		return discord.MessageUpdate{}, userError(err.Error())
	}

	user, tokens, err := korus.Login(ctx, baseURL, data.String("username"), data.String("password"))
	if err != nil {
		if apiErr, ok := errors.AsType[*korus.APIError](err); ok {
			return discord.MessageUpdate{}, failf("Korus rejected the login: %s", apiErr.Message)
		}
		return discord.MessageUpdate{}, failf("Could not reach %s.", baseURL)
	}

	discordID := e.User().ID.String()
	if err := b.accounts.Link(ctx, discordID, baseURL, user, tokens); err != nil {
		return discord.MessageUpdate{}, err
	}
	account, err := b.accounts.Get(ctx, discordID)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	if _, err := account.Client.Me(ctx); err != nil {
		if unlinkErr := b.accounts.Unlink(ctx, discordID); unlinkErr != nil {
			b.log.Error("cannot drop a link that failed verification", "err", unlinkErr)
		}
		return discord.MessageUpdate{}, failf("Logged in, but %s rejected the token.", baseURL)
	}

	return ui.Reply(ui.ColorSuccess,
		ui.Text("### Linked to Korus\n**%s** · %s\n-# %s", ui.Escape(user.Username), user.Role, baseURL),
		discord.NewSmallSeparator(),
		discord.NewTextDisplay("-# Your plays are recorded to this account, and only to this account."),
	), nil
}

func (b *Bot) logout(ctx context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	discordID := e.User().ID.String()
	if _, err := b.accounts.Get(ctx, discordID); err != nil {
		return discord.MessageUpdate{}, err
	}
	if e.GuildID() != nil {
		if active := b.players.Get(*e.GuildID()); active != nil && active.Host() == e.User().ID {
			active.Stop()
		}
	}
	if err := b.accounts.Unlink(ctx, discordID); err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Note(ui.ColorSuccess, "Unlinked. Run /login to connect again."), nil
}

func (b *Bot) whoami(ctx context.Context, e *handler.CommandEvent, _ discord.SlashCommandInteractionData) (discord.MessageUpdate, error) {
	account, err := b.accounts.Get(ctx, e.User().ID.String())
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	user, err := account.Client.Me(ctx)
	if err != nil {
		return discord.MessageUpdate{}, err
	}
	return ui.Reply(ui.ColorContent,
		ui.Text("### %s\n**%s** · %s\n-# %s", ui.Escape(e.User().EffectiveName()), ui.Escape(user.Username), user.Role, account.BaseURL),
	), nil
}
