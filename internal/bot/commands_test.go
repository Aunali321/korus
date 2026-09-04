package bot

import (
	"testing"

	"github.com/disgoorg/disgo/discord"
)

// Discord rejects the whole command sync when one definition is malformed, and
// that only shows up at startup, so the shapes are checked here instead.
func TestCommandDefinitions(t *testing.T) {
	for _, command := range commands {
		slash, ok := command.(discord.SlashCommandCreate)
		if !ok {
			t.Fatalf("%s is not a slash command", command.CommandName())
		}
		t.Run(slash.Name, func(t *testing.T) {
			checkName(t, slash.Name)
			checkDescription(t, slash.Description)
			checkOptions(t, slash.Options)
		})
	}
}

func checkOptions(t *testing.T, options []discord.ApplicationCommandOption) {
	t.Helper()
	if len(options) > 25 {
		t.Errorf("%d options, limit is 25", len(options))
	}
	seenOptional := false
	for _, option := range options {
		checkName(t, option.OptionName())
		required := false
		switch typed := option.(type) {
		case discord.ApplicationCommandOptionSubCommand:
			checkDescription(t, typed.Description)
			checkOptions(t, typed.Options)
			continue
		case discord.ApplicationCommandOptionString:
			checkDescription(t, typed.Description)
			if typed.Autocomplete && len(typed.Choices) > 0 {
				t.Errorf("option %q has both autocomplete and choices", typed.Name)
			}
			for _, choice := range typed.Choices {
				checkDescription(t, choice.Name)
			}
			required = typed.Required
		case discord.ApplicationCommandOptionInt:
			checkDescription(t, typed.Description)
			required = typed.Required
		case discord.ApplicationCommandOptionBool:
			checkDescription(t, typed.Description)
			required = typed.Required
		default:
			t.Errorf("option %q has an unhandled type", option.OptionName())
			continue
		}
		if required && seenOptional {
			t.Errorf("required option %q follows an optional one", option.OptionName())
		}
		seenOptional = seenOptional || !required
	}
}

func checkName(t *testing.T, name string) {
	t.Helper()
	if name == "" || len(name) > 32 {
		t.Errorf("name %q must be 1 to 32 characters", name)
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r == '-' || r == '_' || r >= '0' && r <= '9' {
			continue
		}
		t.Errorf("name %q must be lowercase", name)
		break
	}
}

func checkDescription(t *testing.T, description string) {
	t.Helper()
	if description == "" || len(description) > 100 {
		t.Errorf("description %q must be 1 to 100 characters", description)
	}
}

func findCommand(t *testing.T, name string) discord.SlashCommandCreate {
	t.Helper()
	for _, command := range commands {
		if slash, ok := command.(discord.SlashCommandCreate); ok && slash.Name == name {
			return slash
		}
	}
	t.Fatalf("no %q command", name)
	return discord.SlashCommandCreate{}
}

func option(t *testing.T, name string, options []discord.ApplicationCommandOption) discord.ApplicationCommandOption {
	t.Helper()
	for _, o := range options {
		if o.OptionName() == name {
			return o
		}
	}
	t.Fatalf("no %q option", name)
	return nil
}

// Lyrics must accept being called bare so it can fall back to the current
// track, and it is always public, so it carries no share option.
func TestLyricsOptions(t *testing.T) {
	lyrics := findCommand(t, "lyrics")
	song, ok := option(t, "song", lyrics.Options).(discord.ApplicationCommandOptionString)
	if !ok || song.Required {
		t.Error("lyrics song must be an optional string")
	}
	for _, o := range lyrics.Options {
		if o.OptionName() == optShare {
			t.Error("lyrics is public only and must not offer share")
		}
	}
}

// Stats visibility is chosen at defer time from this option.
func TestStatsShareOption(t *testing.T) {
	stats := findCommand(t, "stats")
	if _, ok := option(t, optShare, stats.Options).(discord.ApplicationCommandOptionBool); !ok {
		t.Error("stats share option must be a bool")
	}
}
