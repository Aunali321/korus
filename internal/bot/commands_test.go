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
