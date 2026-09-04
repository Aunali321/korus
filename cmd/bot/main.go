package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/Aunali321/korus/internal/bot"
)

func main() {
	godotenv.Load()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := bot.ConfigFromEnv()
	if err != nil {
		log.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	korusBot, err := bot.New(cfg, log)
	if err != nil {
		log.Error("cannot build bot", "err", err)
		os.Exit(1)
	}

	if err := korusBot.Start(context.Background()); err != nil {
		log.Error("cannot start bot", "err", err)
		os.Exit(1)
	}
	log.Info("korus bot is running", "guild", cfg.GuildID, "db", cfg.DBPath)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	korusBot.Close(ctx)
	log.Info("korus bot stopped")
}
