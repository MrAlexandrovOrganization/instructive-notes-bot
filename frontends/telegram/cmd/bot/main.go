package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/bot"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/client"
	"github.com/mrralexandrov/instructive-notes-bot/frontends/telegram/internal/config"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	clients, err := client.New(cfg.CoreGRPCAddr)
	if err != nil {
		slog.Error("create gRPC clients", "error", err)
		os.Exit(1)
	}
	defer clients.Close()

	b, err := bot.New(cfg.BotToken, clients, cfg.RootTelegramID)
	if err != nil {
		slog.Error("create bot", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	b.Run(ctx)
	slog.Info("bot stopped")
}
