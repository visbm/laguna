package engine

import (
	"context"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/database/storage/engine/inmemory"
)

const inMemory = "inmemory"

type Engine interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Del(ctx context.Context, key string) error
}

func NewEngine(c config.Engine, log logger.Logger) Engine {
	log.Info("initializing engine", logger.String("type", c.Type))
	switch c.Type {
	case inMemory:
		return inmemory.NewEngine(log, c.Shards)
	default:
		return inmemory.NewEngine(log, c.Shards)
	}

}
