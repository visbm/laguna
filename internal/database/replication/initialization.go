package replication

import (
	"context"
	"laguna/internal/segment"

	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/database/logs"

	"laguna/internal/transport/tcp"
)

type Replication interface {
	Start(ctx context.Context)
	Close()
}

func InitReplication(log logger.Logger, st Storage, conf *config.Config, sm *segment.Manager) (Replication, error) {
	if !conf.Replication.Enable {
		return nil, nil
	}
	return initReplication(log, st, conf, sm)
}

func initReplication(log logger.Logger, st Storage, conf *config.Config, sm *segment.Manager) (Replication, error) {

	if conf.Replication.IsMaster {
		return initMaster(log, conf, sm), nil
	} else {
		return initSlave(log, st, conf, sm)
	}
}

func initMaster(log logger.Logger, conf *config.Config, sm *segment.Manager) Replication {
	log.Info("Initializing master...")

	connWriter := logs.NewLogWriter(log, sm)
	connReader := logs.NewLogReader(sm, log)

	master := NewMaster(log, connReader, connWriter, sm, conf.WAL.Directory)

	tcpLister := tcp.NewListener(log, master, conf.Replication.Server)

	master.setTransport(tcpLister)

	log.Info("Master initialized")

	return master
}

func initSlave(log logger.Logger, st Storage, conf *config.Config, sm *segment.Manager) (Replication, error) {
	log.Info("Initializing slave")

	syncInterval := conf.Replication.SyncInterval

	slaveCl, err := tcp.NewClient(conf.Replication.Client, log)
	if err != nil {
		return nil, err
	}

	lgReader := logs.NewLogReader(sm, log)

	var slave *Slave
	if conf.WAL.Enable {
		lgWriter := logs.NewLogWriter(log, sm)
		slave = NewSlaveWithWal(log, slaveCl, lgReader, lgWriter, st, syncInterval)

	} else {
		slave = NewSlave(log, slaveCl, lgReader, st, syncInterval)
	}

	log.Info("Slave initialized")

	return slave, nil
}
