package database

import (
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/database/logs"
	"laguna/internal/database/replication"
	"laguna/internal/database/storage"
	"laguna/internal/database/storage/engine"
	"laguna/internal/database/wal"
	"laguna/internal/segment"
)

func InitDB(conf *config.Config, log logger.Logger) (*Database, replication.Replication, *wal.WAL, error) {
	sm, err := initSegmentManager(log, conf)
	if err != nil {
		return nil, nil, nil, err
	}

	eng := engine.NewEngine(conf.Engine, log)
	st := storage.New(eng, log)

	walObj, err := initWal(conf, log, sm)
	if err != nil {
		return nil, nil, nil, err
	}

	db, err := NewDatabase(st, conf.Replication.IsMaster, walObj, log)
	if err != nil {
		return nil, nil, nil, err
	}

	repl, err := replication.InitReplication(log, st, conf, sm)
	if err != nil {
		return nil, nil, nil, err
	}

	return db, repl, walObj, nil
}

func initWal(conf *config.Config, log logger.Logger, sm *segment.Manager) (*wal.WAL, error) {
	lw := logs.NewLogWriter(log, sm)
	lr := logs.NewLogReader(sm, log)
	newWAL := wal.NewWAL(conf.WAL, log, lw, lr)

	return newWAL, nil
}

func initSegmentManager(log logger.Logger, conf *config.Config) (*segment.Manager, error) {
	im := segment.NewIndexManagerMutex()

	fileSegment, err := segment.NewFileSegment(log, conf.WAL.Directory, conf.WAL.MaxSegmentSize.Int64())
	if err != nil {
		return nil, err
	}

	return segment.NewSegmentManager(log, im, fileSegment), nil
}
