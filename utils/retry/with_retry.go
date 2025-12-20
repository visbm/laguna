package retry

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

func WithRetry(ctx context.Context, retry int, baseDelay time.Duration, f func() error) error {
	if retry <= 0 {
		retry = 1
	}

	if baseDelay <= 0 {
		baseDelay = time.Millisecond * 50
	}

	if f == nil {
		return errors.New("func is nil")
	}
	var err error

	for i := 0; i < retry; i++ {
		err = f()
		if err == nil {
			return nil
		}

		if i == retry {
			return err
		}

		delay := baseDelay << i
		if delay <= 0 {
			delay = baseDelay
		}

		jitter := time.Duration(rand.Int63n(int64(delay / 3)))
		delay = delay + jitter

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return err
}

func WithRetryValue[T any](ctx context.Context, retry int, baseDelay time.Duration, f func() (T, error)) (T, error) {
	var val T
	if retry <= 0 {
		retry = 1
	}

	if baseDelay <= 0 {
		baseDelay = time.Millisecond * 50
	}

	if f == nil {
		return val, errors.New("func is nil")
	}
	var err error

	for i := 0; i < retry; i++ {
		val, err = f()
		if err == nil {
			return val, nil
		}

		if i == retry {
			return val, err
		}

		delay := baseDelay << i
		if delay <= 0 {
			delay = baseDelay
		}

		jitter := time.Duration(rand.Int63n(int64(delay / 3)))
		delay = delay + jitter

		select {
		case <-ctx.Done():
			return val, ctx.Err()
		case <-time.After(delay):
		}
	}

	return val, err
}
