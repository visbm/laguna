package storage

import (
	"context"
	"errors"
	"laguna/common/logger"
	"laguna/internal/storage/engine/inmemory"
)

type Engine interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Del(ctx context.Context, key string) error
}

type Storage struct {
	log logger.Logger
	en  Engine
}

func New(en Engine, log logger.Logger) *Storage {
	log.Info("Initializing storage")
	return &Storage{
		log: log,
		en:  en,
	}
}

func (s *Storage) Get(ctx context.Context, key string) (string, error) {
	if ctx.Err() != nil {
		s.log.Error("context canceled")
		return "", ctx.Err()
	}

	v, err := s.en.Get(ctx, key)
	if err != nil {
		if errors.Is(err, inmemory.ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	return v, err
}
func (s *Storage) Set(ctx context.Context, key, value string) error {
	if ctx.Err() != nil {
		s.log.Error("context canceled")
		return ctx.Err()
	}

	return s.en.Set(ctx, key, value)
}
func (s *Storage) Delete(ctx context.Context, key string) error {
	if ctx.Err() != nil {
		s.log.Error("context canceled")
		return ctx.Err()
	}

	return s.en.Del(ctx, key)
}
