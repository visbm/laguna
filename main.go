package main

import (
	"context"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/database"
	"laguna/internal/handlers"
	"laguna/internal/query"
	"laguna/internal/transport"
	"laguna/internal/transport/cli"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conf := config.NewConfig("config.yaml")

	log := logger.New(conf.Logger)
	log.Info("Starting application")

	db, repl, wal, err := database.InitDB(conf, log)
	if err != nil {
		log.Fatal("Failed to create database", logger.Error(err))
	}

	if repl != nil {
		repl.Start(ctx)
	}
	wal.Start(ctx)

	qb := query.NewBuilder()
	handler := handlers.NewUniversalHandler(qb, db, log)

	{ // additional cli listener
		cliListener := cli.NewListener(log, handler)
		cliListener.Listen(ctx)
		cliListener.Close()
	}

	ls := transport.NewListener(log, handler, conf.Transport)
	ls.Listen(ctx)

	if conf.Profiler.Enable {
		startProfiler(conf.Profiler.Address, log)
	}

	<-ctx.Done()

	ls.Close()
	repl.Close()

	err = log.Sync()
	if err != nil {
		log.Error("failed to sync logger", logger.Error(err))
	}

	log.Info("Shutting down...")
}

func startProfiler(addr string, log logger.Logger) {
	go func() {
		log.Info("Starting profiler server")
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatal("pprof server failed:", logger.Error(err))
		}
	}()
}
