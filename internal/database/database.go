package database

import (
	"context"
	"errors"
	"fmt"
	"laguna/common/logger"
	"laguna/internal/query"
	"laguna/utils/concurrency"
	"time"
)

var errNonReadOpSlave = errors.New("non read operation on slave")

type Storage interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}

type WAL interface {
	Write(ctx context.Context, q query.Query) (concurrency.FutureResp[error], error)
	RestoreStream() (concurrency.FutureRespWithErr[[]query.Query], error)
	IsEnable() bool
}

type Database struct {
	log logger.Logger
	st  Storage

	wal WAL

	isMaster bool
}

func NewDatabase(st Storage, isMaster bool, wal WAL, log logger.Logger) (*Database, error) {
	db := &Database{
		log:      log,
		st:       st,
		wal:      wal,
		isMaster: isMaster,
	}

	err := db.restoreFromWal()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (e *Database) Execute(ctx context.Context, q query.Query) (string, error) {
	if e.wal.IsEnable() {
		err := e.setInWal(ctx, q)
		if err != nil {
			return "", err
		}
	}

	if !e.isMaster && q.MethodID() != query.GetMethodID {
		e.log.Error("non read operation on slave", logger.Error(errNonReadOpSlave))
		return "", errNonReadOpSlave
	}

	ans, err := e.exec(ctx, q)
	if err != nil {
		return "", err
	}
	return ans, nil
}

func (e *Database) setInWal(ctx context.Context, q query.Query) error {
	if q.MethodID() == query.GetMethodID {
		return nil
	}

	resp, err := e.wal.Write(ctx, q)
	if err != nil {
		return err
	}

	const walDeadline = 1 * time.Second
	ctx, cancel := context.WithTimeout(ctx, walDeadline)
	defer cancel()

	errV, err := resp.GetResponseWithDeadline(ctx)
	if errV != nil || err != nil {
		if errV != nil {
			return errV
		}
		return err
	}

	return nil
}

func (e *Database) exec(ctx context.Context, q query.Query) (string, error) {
	args := q.GetArgs()
	method := q.MethodID()

	switch method {
	case query.GetMethodID:
		value, err := e.st.Get(ctx, args[query.GetKeyIdx])
		if err != nil {
			return "", err
		}
		return value, nil

	case query.SetMethodID:
		err := e.st.Set(ctx, args[query.SetKeyIdx], args[query.SetValueIdx])
		return "OK", err

	case query.DelMethodID:
		err := e.st.Delete(ctx, args[query.DelKeyIdx])
		return "OK", err

	default:
		return "", fmt.Errorf("unknown command ID: %d", method)
	}
}

func (e *Database) restoreFromWal() error {
	if !e.wal.IsEnable() {
		return nil
	}
	e.log.Info("restoring from WAL")

	qStream, err := e.wal.RestoreStream()
	if err != nil {
		e.log.Error("read wal failed", logger.Error(err))
		return fmt.Errorf("read wal: %w", err)
	}

	for {
		queries, err, ok := qStream.Next()
		if err != nil {
			e.log.Error("read wal failed", logger.Error(err))
			return fmt.Errorf("read wal: %w", err)
		}
		if !ok {
			break
		}

		for _, q := range queries {
			_, err := e.exec(context.Background(), q)
			if err != nil {
				e.log.Error("execute failed", logger.Error(err))
				return fmt.Errorf("execute query: %w", err)
			}
		}

	}

	e.log.Info("restoring from WAL finished")

	return nil
}
