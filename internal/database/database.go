package database

import (
	"context"
	"fmt"
	"laguna/internal/query"
	"laguna/utils"
)

type Storage interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}

type Database struct {
	st    Storage
	txGen *utils.Generator
}

func NewDatabase(st Storage) *Database {
	return &Database{st: st, txGen: utils.NewIDGenerator()}
}

func (e *Database) Execute(ctx context.Context, q query.Query) (string, error) {
	ctx = utils.SetTxInContext(ctx, e.txGen.NextID())

	args := q.GetArs()
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
