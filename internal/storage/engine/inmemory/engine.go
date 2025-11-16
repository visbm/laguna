package inmemory

import (
	"context"
	"errors"
	"laguna/common/logger"
	"laguna/utils"
)

var ErrNotFound = errors.New("not found")

type Engine struct {
	im  *InMemory
	log logger.Logger
}

func NewEngine(log logger.Logger) *Engine {
	return &Engine{
		im:  NewInMemory(),
		log: log,
	}
}

func (e *Engine) Set(ctx context.Context, key, value string) error {
	txId := utils.GetTxInContext(ctx)

	err := e.im.Set(key, value)
	if err != nil {
		return err
	}

	e.log.Info("saved with txID", logger.UInteger("txID", txId))
	return nil
}

func (e *Engine) Get(ctx context.Context, key string) (string, error) {
	txId := utils.GetTxInContext(ctx)
	v, ok := e.im.Get(key)
	if !ok {
		return "", ErrNotFound
	}

	e.log.Info("got with txID", logger.UInteger("txID", txId))

	return v, nil
}

func (e *Engine) Del(ctx context.Context, key string) error {
	txId := utils.GetTxInContext(ctx)
	err := e.im.Del(key)
	if err != nil {
		return err
	}

	e.log.Info("saved with txID", logger.UInteger("txID", txId))
	return nil
}
