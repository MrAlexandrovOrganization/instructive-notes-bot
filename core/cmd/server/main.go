package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mrralexandrov/instructive-notes-bot/core/grpcserver"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/config"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/db"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/repository"
	"github.com/mrralexandrov/instructive-notes-bot/core/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(pool); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	bunDB := db.NewBunDB(pool)
	defer bunDB.Close()

	// Repositories
	usersRepo := repository.NewUsersRepo(bunDB)
	groupsRepo := repository.NewGroupsRepo(bunDB)
	participantsRepo := repository.NewParticipantsRepo(bunDB)
	notesRepo := repository.NewNotesRepo(bunDB)

	// Services
	usersSvc := service.NewUsersService(usersRepo)
	groupsSvc := service.NewGroupsService(groupsRepo)
	participantsSvc := service.NewParticipantsService(participantsRepo)
	notesSvc := service.NewNotesService(notesRepo)
	mediaSvc := service.NewMediaService(pool, cfg.MediaDir)

	srv := grpcserver.New(
		cfg.GRPCPort,
		cfg.MaxGRPCMsgSize,
		usersSvc,
		groupsSvc,
		participantsSvc,
		notesSvc,
		mediaSvc,
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Run(); err != nil {
			slog.Error("gRPC server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")
	srv.Stop()
}
