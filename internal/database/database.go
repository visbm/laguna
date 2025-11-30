package database

import (
	"context"
	"fmt"
	"laguna/common/logger"
	"laguna/internal/query"
	"laguna/utils"
	"time"
)

type Storage interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}

type WAL interface {
	Write(ctx context.Context, q query.Query) (utils.FutureResp[error], error)
	ReadWal() ([]query.Query, error)
}

type Database struct {
	lg    logger.Logger
	st    Storage
	txGen *utils.Generator
	wal   WAL
}

func NewDatabase(st Storage, wal WAL, lg logger.Logger) (*Database, error) {
	db := &Database{
		lg:    lg,
		st:    st,
		txGen: utils.NewIDGenerator(),
		wal:   wal,
	}
	err := db.restoreFromWal()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (e *Database) Execute(ctx context.Context, q query.Query) (string, error) {
	if e.walEnable() {
		err := e.setInWal(ctx, q)
		if err != nil {
			return "", err
		}
	}

	ctx = utils.SetTxInContext(ctx, e.txGen.NextID())

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

	err, timeOutErr := resp.GetResponseWithDeadline(ctx)
	if err != nil || timeOutErr != nil {
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
	if !e.walEnable() {
		return nil
	}
	queries, err := e.wal.ReadWal()
	if err != nil {
		e.lg.Error("read wal failed", logger.Error(err))
		return fmt.Errorf("read wal: %w", err)
	}

	for _, q := range queries {
		_, err := e.exec(context.Background(), q)
		if err != nil {
			e.lg.Error("execute failed", logger.Error(err))
			return fmt.Errorf("execute query: %w", err)
		}
	}

	return nil
}

func (e *Database) walEnable() bool {
	return e.wal != nil
}
