package inmemory

import (
	"context"
	"errors"
	"laguna/internal/mocks"
	"testing"
)

func TestEngine_SetGetDel(t *testing.T) {
	ctx := context.Background()
	lg := mocks.MockLogger{}
	engine := NewEngine(&lg, 20)

	err := engine.Set(ctx, "key1", "value1")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := engine.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("GetIndex failed: %v", err)
	}
	if val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}

	_, err = engine.Get(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	err = engine.Del(ctx, "key1")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	_, err = engine.Get(ctx, "key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}

	err = engine.Del(ctx, "missing")
	if err != nil {
		t.Errorf("expected no error when deleting missing key, got %v", err)
	}
}

func TestInMemory_ConcurrentAccess(t *testing.T) {
	mem := NewInMemory(10)

	const n = 1000000
	done := make(chan struct{})

	for i := 0; i < n; i++ {
		go func(i int) {
			_ = mem.Set(string(rune(i)), "val")
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < n; i++ {
		<-done
	}

	for i := 0; i < n; i++ {
		go func(i int) {
			_, _ = mem.Get(string(rune(i)))
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < n; i++ {
		<-done
	}

	for i := 0; i < n; i++ {
		go func(i int) {
			_ = mem.Del(string(rune(i)))
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < n; i++ {
		<-done
	}

	for _, sh := range mem.shards {
		if len(sh.store) != 0 {
			t.Errorf("expected store to be empty, got %d items", len(sh.store))
		}
	}

}
