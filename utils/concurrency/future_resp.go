package concurrency

import (
	"context"
	"sync"
)

type FutureResp[T any] struct {
	resp chan T
}

type FutureRespWithErr[T any] struct {
	resp chan T
	m    *sync.Mutex
	err  error
}

func NewFutureResp[T any]() FutureResp[T] {
	return FutureResp[T]{
		resp: make(chan T, 1),
	}
}

func NewFutureRespWithErr[T any]() FutureRespWithErr[T] {
	return FutureRespWithErr[T]{
		resp: make(chan T, 1),
		m:    &sync.Mutex{},
	}
}

func (f *FutureResp[T]) Put(v T) {
	f.resp <- v
}

func (f *FutureRespWithErr[T]) Put(v T, err error) {
	if err != nil {
		f.m.Lock()
		f.err = err
		f.m.Unlock()
		return
	}
	f.resp <- v
}

func (f *FutureResp[T]) GetResponse() T {
	return <-f.resp
}

func (f *FutureRespWithErr[T]) GetResponse() (T, error) {
	f.m.Lock()
	if f.err != nil {
		defer f.m.Unlock()
		var zero T
		return zero, f.err
	}
	f.m.Unlock()

	v, ok := <-f.resp
	if !ok {
		var zero T
		return zero, nil
	}
	return v, nil
}

func (f *FutureRespWithErr[T]) Next() (T, error, bool) {
	f.m.Lock()
	if f.err != nil {
		defer f.m.Unlock()
		var zero T
		return zero, f.err, false
	}
	f.m.Unlock()

	r, ok := <-f.resp
	if !ok {
		var zero T
		return zero, nil, false
	}
	return r, nil, true

}

func (f *FutureResp[T]) Done() {
	close(f.resp)
}

func (f *FutureRespWithErr[T]) Done() {
	close(f.resp)
}

func (f *FutureResp[T]) GetResponseWithDeadline(ctx context.Context) (T, error) {
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
