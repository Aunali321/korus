package bot

import (
	"cmp"
	"errors"
	"os"

	"github.com/disgoorg/snowflake/v2"
)

// Config is the bot's runtime configuration.
type Config struct {
	Token   string
	GuildID snowflake.ID
	DBPath  string
	FFmpeg  string
}

// ConfigFromEnv reads configuration from the environment.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		Token:  os.Getenv("DISCORD_TOKEN"),
		DBPath: cmp.Or(os.Getenv("BOT_DB_PATH"), "./bot.db"),
		FFmpeg: cmp.Or(os.Getenv("FFMPEG_PATH"), "ffmpeg"),
	}
	if cfg.Token == "" {
		return Config{}, errors.New("DISCORD_TOKEN is required")
	}
	if raw := os.Getenv("DISCORD_GUILD_ID"); raw != "" {
		id, err := snowflake.Parse(raw)
		if err != nil {
			return Config{}, errors.New("DISCORD_GUILD_ID is not a snowflake")
		}
		cfg.GuildID = id
	}
	return cfg, nil
}
