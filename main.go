package main

import (
	"context"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/database"
	"laguna/internal/handlers"
	"laguna/internal/query"
	"laguna/internal/storage"
	"laguna/internal/storage/engine"
	"laguna/internal/transport"
	"laguna/internal/transport/cli"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conf := config.NewConfig("config.yaml")

	lg := logger.New(conf.Logger)
	lg.Info("Starting application")

	eng := engine.NewEngine(conf.Engine, lg)

	st := storage.New(eng, lg)
	db := database.NewDatabase(st)
	qb := query.NewBuilder()

	handler := handlers.NewUniversalHandler(qb, db, lg)

	cliListener := cli.NewListener(lg, handler) // todo delete
	go cliListener.Listen(ctx)

	ls := transport.NewListener(lg, handler, conf.Transport)
	go ls.Listen(ctx)

	<-ctx.Done()

	ls.Close()
	err := lg.Sync()
	if err != nil {
		lg.Error("failed to sync logger", logger.Error(err))
	}
	lg.Info("Shutting down...")
}
