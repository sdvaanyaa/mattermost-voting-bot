package main

import (
	"fmt"
	"github.com/sdvaanyaa/mattermost-voting-bot/internal/bot"
	"github.com/sdvaanyaa/mattermost-voting-bot/internal/config"
	"github.com/sdvaanyaa/mattermost-voting-bot/internal/storage"
	"log/slog"
	"os"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()
	fmt.Println(cfg)

	log := setupLogger(cfg.Env)

	log.Info("starting mattermost-voting-bot", slog.String("env", cfg.Env))

	log.Debug("debug messages are enabled")

	storage, err := storage.New(cfg)
	if err != nil {
		log.Error("error creating storage", err)
		os.Exit(1)
	}

	bot, err := bot.New(cfg, storage, log)
	if err != nil {
		log.Error("error creating bot", err)
	}

	bot.Run()
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}
	return log
}
