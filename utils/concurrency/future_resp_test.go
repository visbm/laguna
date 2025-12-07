package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFutureResp_PutAndGet(t *testing.T) {
	f := NewFutureResp[int]()
	f.Put(42)

	got := f.GetResponse()
	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestFutureResp_GetResponseWithDeadline_Success(t *testing.T) {
	f := NewFutureResp[string]()
	go func() {
		time.Sleep(50 * time.Millisecond)
		f.Put("hello")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := f.GetResponseWithDeadline(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected 'hello', got %s", got)
	}
}

func TestFutureResp_GetResponseWithDeadline_Timeout(t *testing.T) {
	f := NewFutureResp[string]()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	got, err := f.GetResponseWithDeadline(ctx)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if got != "" {
		t.Fatalf("expected zero value, got %s", got)
	}
}

func TestFutureRespWithErr_PutSuccess(t *testing.T) {
	f := NewFutureRespWithErr[int]()
	f.Put(99, nil)

	got, err := f.GetResponse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 99 {
		t.Fatalf("expected 99, got %d", got)
	}
}

func TestFutureRespWithErr_PutError(t *testing.T) {
	f := NewFutureRespWithErr[int]()
	testErr := errors.New("test error")
	f.Put(0, testErr)

	got, err := f.GetResponse()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, testErr) {
		t.Fatalf("expected test error, got %v", err)
	}
	if got != 0 {
		t.Fatalf("expected zero value, got %d", got)
	}
}

func TestFutureRespWithErr_Next(t *testing.T) {
	f := NewFutureRespWithErr[string]()
	f.Put("first", nil)
	testErr := errors.New("test error")
	f.Put("", testErr)
	f.Done()

	v, err, ok := f.Next()
	if ok {
		t.Fatalf("expected ok=false")
	}

	if !errors.Is(err, testErr) {
		t.Fatalf("unexpected error: %v", err)
	}

	if v == "first" {
		t.Fatalf("expected empty', got %s", v)
	}

	// второй вызов
	v, err, ok = f.Next()
	if ok {
		t.Fatalf("expected ok=false")
	}
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// третий вызов — каналы закрыты
	_, _, ok = f.Next()
	if ok {
		t.Fatalf("expected ok=false after Done")
	}
}

func TestFutureRespWithErr_Done(t *testing.T) {
	f := NewFutureRespWithErr[int]()
	f.Put(1, nil)
	f.Done()

	v, ok := <-f.resp
	if v != 1 {
		t.Fatalf("expected ok=true, got %d", v)
	}

	if !ok {
		t.Fatalf("expected ok=true,")
	}

	v, ok = <-f.resp
	if v != 0 {
		t.Fatalf("expected ok=true, got %d", v)
	}
	if ok {
		t.Fatalf("expected ok=false,")
	}
}
