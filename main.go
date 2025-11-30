package main

import (
	"context"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/database"
	"laguna/internal/database/wal"
	"laguna/internal/fs"
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

	db, newWal := initDB(conf, lg)

	qb := query.NewBuilder()

	handler := handlers.NewUniversalHandler(qb, db, lg)

	cliListener := cli.NewListener(lg, handler) // todo delete
	go cliListener.Listen(ctx)

	ls := transport.NewListener(lg, handler, conf.Transport)
	go ls.Listen(ctx)

	go newWal.Start(ctx)

	<-ctx.Done()

	ls.Close()
	err := lg.Sync()
	if err != nil {
		lg.Error("failed to sync logger", logger.Error(err))
	}

	lg.Info("Shutting down...")
}

func initDB(conf *config.Config, lg logger.Logger) (*database.Database, *wal.WAL) {
	fsWriter, err := fs.NewFileSegment(conf.WAL.Directory, conf.WAL.MaxSegmentSize.Int64())
	if err != nil {
		lg.Fatal("Failed to create file segment", logger.Error(err))
	}

	lw := wal.NewLogWriter(conf.WAL, lg, fsWriter)
	lr := wal.NewLogReader(conf.WAL, lg)
	newWAL := wal.NewWAL(conf.WAL, lg, lw, lr)

	eng := engine.NewEngine(conf.Engine, lg)
	st := storage.New(eng, lg)
	db, err := database.NewDatabase(st, newWAL, lg)
	if err != nil {
		lg.Fatal("Failed to create database", logger.Error(err))
	}

	return db, newWAL
}
