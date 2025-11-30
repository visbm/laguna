package utils

import (
	"context"
)

type FutureResp[T any] struct {
	resp chan T
}

func NewFutureResp[T any]() FutureResp[T] {
	return FutureResp[T]{
		resp: make(chan T, 1),
	}
}

func (f FutureResp[T]) Put(v T) {
	f.resp <- v
}

func (f FutureResp[T]) GetResponse() T {
	return <-f.resp
}

func (f FutureResp[T]) Done() {
	close(f.resp)
}

func (f FutureResp[T]) GetResponseWithDeadline(ctx context.Context) (T, error) {
	var zero T

	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	default:
	}

	select {

	case <-ctx.Done():
		return zero, ctx.Err()

	case resp := <-f.resp:
		return resp, nil
	}
}
