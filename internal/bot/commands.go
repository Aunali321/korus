package bot

import "github.com/disgoorg/disgo/discord"

const (
	radioMinLimit = 1
	radioMaxLimit = 25

	optShare = "share"
)

func autocompleteOption(name, description string, required bool) discord.ApplicationCommandOptionString {
	return discord.ApplicationCommandOptionString{
		Name:         name,
		Description:  description,
		Required:     required,
		Autocomplete: true,
	}
}

func periodOption(choices ...discord.ApplicationCommandOptionChoiceString) discord.ApplicationCommandOptionString {
	return discord.ApplicationCommandOptionString{
		Name:        "period",
		Description: "Window to summarise",
		Choices:     choices,
	}
}

// shareOption lets a caller override whether a reply is private to them. The
// per-command default lives in the route, since Discord has no default for
// boolean options.
func shareOption(description string) discord.ApplicationCommandOptionBool {
	return discord.ApplicationCommandOptionBool{
		Name:        optShare,
		Description: description,
	}
}

func choice(name, value string) discord.ApplicationCommandOptionChoiceString {
	return discord.ApplicationCommandOptionChoiceString{Name: name, Value: value}
}

var commands = []discord.ApplicationCommandCreate{
	discord.SlashCommandCreate{
		Name:        "login",
		Description: "Link your Korus account to Discord",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "url", Description: "Korus server URL", Required: true},
			discord.ApplicationCommandOptionString{Name: "username", Description: "Korus username", Required: true},
			discord.ApplicationCommandOptionString{Name: "password", Description: "Korus password", Required: true},
		},
	},
	discord.SlashCommandCreate{Name: "logout", Description: "Unlink your Korus account"},
	discord.SlashCommandCreate{Name: "whoami", Description: "Show the Korus account you are linked to"},

	discord.SlashCommandCreate{
		Name:        "search",
		Description: "Search songs, albums, artists and playlists",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{Name: "query", Description: "What to look for", Required: true},
		},
	},
	discord.SlashCommandCreate{
		Name:        "album",
		Description: "Show an album and its tracklist",
		Options:     []discord.ApplicationCommandOption{autocompleteOption("title", "Album title", true)},
	},
	discord.SlashCommandCreate{
		Name:        "artist",
		Description: "Show an artist, their albums and top tracks",
		Options:     []discord.ApplicationCommandOption{autocompleteOption("name", "Artist name", true)},
	},
	discord.SlashCommandCreate{
		Name:        "lyrics",
		Description: "Show the lyrics of a song, or of whatever is playing",
		Options: []discord.ApplicationCommandOption{
			autocompleteOption("song", "Song title, defaults to the current track", false),
		},
	},

	discord.SlashCommandCreate{
		Name:        "stats",
		Description: "Your listening stats",
		Options: []discord.ApplicationCommandOption{
			periodOption(
				choice("Last 30 days", "30d"),
				choice("Today", "today"),
				choice("This week", "week"),
				choice("This month", "month"),
				choice("This year", "year"),
				choice("All time", "all_time"),
			),
			shareOption("Post to the channel instead of only to you (default false)"),
		},
	},
	discord.SlashCommandCreate{
		Name:        "captions",
		Description: "Follow the current track's lyrics line by line, or stop following",
	},
	discord.SlashCommandCreate{
		Name:        "wrapped",
		Description: "Your listening recap",
		Options: []discord.ApplicationCommandOption{periodOption(
			choice("This week", "week"),
			choice("This month", "month"),
			choice("This year", "year"),
			choice("All time", "all_time"),
		)},
	},

	discord.SlashCommandCreate{Name: "playlists", Description: "List your playlists"},
	discord.SlashCommandCreate{
		Name:        "playlist",
		Description: "Manage a playlist",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "view",
				Description: "Show a playlist and its tracks",
				Options:     []discord.ApplicationCommandOption{autocompleteOption("name", "Playlist", true)},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "create",
				Description: "Create a playlist",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{Name: "name", Description: "Playlist name", Required: true},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "add",
				Description: "Add a song to a playlist",
				Options: []discord.ApplicationCommandOption{
					autocompleteOption("playlist", "Playlist", true),
					autocompleteOption("song", "Song to add", true),
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "remove",
				Description: "Remove a song from a playlist",
				Options: []discord.ApplicationCommandOption{
					autocompleteOption("playlist", "Playlist", true),
					autocompleteOption("song", "Song to remove", true),
				},
			},
		},
	},

	discord.SlashCommandCreate{
		Name:        "play",
		Description: "Play a song, or queue it behind the current one",
		Options:     []discord.ApplicationCommandOption{autocompleteOption("query", "Song to play", true)},
	},
	discord.SlashCommandCreate{
		Name:        "radio",
		Description: "Queue a station built from a song",
		Options: []discord.ApplicationCommandOption{
			autocompleteOption("seed", "Song to build from, defaults to the last one you played", false),
			discord.ApplicationCommandOptionInt{
				Name:        "limit",
				Description: "How many tracks to queue",
				MinValue:    new(radioMinLimit),
				MaxValue:    new(radioMaxLimit),
			},
		},
	},
	discord.SlashCommandCreate{Name: "pause", Description: "Pause playback"},
	discord.SlashCommandCreate{Name: "resume", Description: "Resume playback"},
	discord.SlashCommandCreate{Name: "skip", Description: "Skip the current track"},
	discord.SlashCommandCreate{Name: "stop", Description: "Stop playback and leave the voice channel"},
	discord.SlashCommandCreate{Name: "queue", Description: "Show what is playing and what is next"},
	discord.SlashCommandCreate{Name: "nowplaying", Description: "Show the current track"},
}
