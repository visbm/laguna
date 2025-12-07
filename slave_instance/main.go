package main

import (
	"context"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/database"
	"laguna/internal/database/logs"
	"laguna/internal/database/replication"
	"laguna/internal/database/storage"
	"laguna/internal/database/storage/engine"
	"laguna/internal/database/wal"
	"laguna/internal/fs"
	"laguna/internal/handlers"
	"laguna/internal/query"
	"laguna/internal/transport"
	"laguna/internal/transport/cli"
	"laguna/internal/transport/tcp"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	conf := config.NewConfig("/Users/nick/goSelfEducation/laguna/slave_instance/config.yaml")

	log := logger.New(conf.Logger)
	log.Info("Starting application")

	db := initDB(ctx, conf, log)

	qb := query.NewBuilder()
	handler := handlers.NewUniversalHandler(qb, db, log)

	{ // todo delete
		cliListener := cli.NewListener(log, handler)
		cliListener.Listen(ctx)
	}

	ls := transport.NewListener(log, handler, conf.Transport)
	ls.Listen(ctx)

	<-ctx.Done()

	ls.Close()
	err := log.Sync()
	if err != nil {
		log.Error("failed to sync logger", logger.Error(err))
	}

	log.Info("Shutting down...")
}

// todo make proper init
func initDB(ctx context.Context, conf *config.Config, log logger.Logger) *database.Database {
	im := fs.NewIndexManagerMutex()

	eng := engine.NewEngine(conf.Engine, log)
	st := storage.New(eng, log)

	newWAL, err := initWal(ctx, conf, log, im)

	db, err := database.NewDatabase(st, conf.Replication.IsMaster, newWAL, log)
	if err != nil {
		log.Fatal("Failed to create database", logger.Error(err))
	}

	initReplication(ctx, st, conf, im, log)

	return db
}

func initWal(ctx context.Context, conf *config.Config, log logger.Logger, im *fs.IndexManagerMutex) (database.WAL, error) {
	fileSegment, err := fs.NewFileSegment(log, conf.WAL.Directory, conf.WAL.MaxSegmentSize.Int64())
	if err != nil {
		log.Fatal("Failed to create file segment", logger.Error(err))
	}

	sm := fs.NewSegmentManager(log, im, fileSegment)

	lw := logs.NewLogWriterWithTarget(log, sm)
	lr := logs.NewLogReader(im, log)

	newWAL := wal.NewWAL(conf.WAL, log, lw, lr)

	newWAL.Start(ctx)

	return newWAL, nil
}

func initReplication(ctx context.Context, st replication.Storage, conf *config.Config, im *fs.IndexManagerMutex, lg logger.Logger) {
	if !conf.Replication.Enable {
		return
	}

	if conf.Replication.IsMaster {
		initMaster(ctx, conf, lg, im)
	} else {
		initSlave(ctx, st, conf, lg, im)
	}

}

func initMaster(ctx context.Context, conf *config.Config, log logger.Logger, im *fs.IndexManagerMutex) {
	log.Info("Initializing master...")

	connWriter := logs.NewLogWriter(log)
	connReader := logs.NewLogReader(im, log)
	master := replication.NewMaster(connReader, connWriter, im, log, conf.WAL.Directory)
	masterTcpServer := tcp.NewListener(log, master, conf.Replication.Server)

	masterTcpServer.Listen(ctx)

	log.Info("Master initialized")

	go func() {
		<-ctx.Done()
		masterTcpServer.Close()
	}()

}

func initSlave(ctx context.Context, st replication.Storage, conf *config.Config, log logger.Logger, im *fs.IndexManagerMutex) {
	log.Info("Initializing slave")

	syncInterval := conf.Replication.SyncInterval

	slaveCl, err := tcp.NewClient(conf.Replication.Client, log)
	if err != nil {
		log.Fatal("Failed to create client", logger.Error(err))
	}
	lgReader := logs.NewLogReader(im, log)

	var slave *replication.Slave
	if conf.WAL.Enable {
		fileSegment, err := fs.NewFileSegment(log, conf.WAL.Directory, conf.WAL.MaxSegmentSize.Int64())
		if err != nil {
			log.Fatal("Failed to create file segment", logger.Error(err))
		}
		sm := fs.NewSegmentManager(log, im, fileSegment)

		lgWriter := logs.NewLogWriterWithTarget(log, sm)
		slave = replication.NewSlaveWithWal(slaveCl, lgReader, lgWriter, st, syncInterval, log)

	} else {
		slave = replication.NewSlave(slaveCl, lgReader, st, syncInterval, log)
	}

	log.Info("Slave initialized")

	slave.Start(ctx)
}
