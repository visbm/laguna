package wal

import (
	"context"
	"errors"
	"laguna/internal/config"
	"laguna/internal/mocks"
	"laguna/internal/query"
	"sync"
	"testing"
	"time"
)

type MockWriterWithError struct {
	fail bool
}

func (w *MockWriterWithError) Write(ctx context.Context, q [][]byte) error {
	if w.fail {
		return errors.New("write error")
	}
	return nil
}

func TestWALBasic(t *testing.T) {
	wr := &MockWriterWithError{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := config.WAL{
		FlushBatchSize: 2,
		FlushInterval:  10 * time.Millisecond,
	}

	wal := NewWAL(conf, &mocks.MockLogger{}, wr)
	b := query.NewBuilder()

	go wal.Start(ctx)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"SET simple", "SET key value", false},
		{"SET multi args", "SET key hello world", false},
		{"GET", "GET key", false},
		{"DEL", "DEL key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := b.Parse([]byte(tt.input))
			if err != nil {
				t.Fatal(err)
			}

			ch, err := wal.Write(ctx, q)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}

			select {
			case err := <-ch:
				if (err != nil) != tt.wantErr {
					t.Fatalf("Write() returned error = %v, wantErr %v", err, tt.wantErr)
				}
			case <-time.After(50 * time.Millisecond):
				t.Fatal("timeout waiting for WAL flush")
			}
		})
	}
}

func TestWALConcurrently(t *testing.T) {
	wr := &MockWriterWithError{}
	ctx, cancel := context.WithCancel(context.Background())
	conf := config.WAL{
		FlushBatchSize: 100,
		FlushInterval:  10 * time.Millisecond,
	}
	runs := 100000

	wal := NewWAL(conf, &mocks.MockLogger{}, wr)
	b := query.NewBuilder()

	go wal.Start(ctx)

	wg := sync.WaitGroup{}
	wg.Add(runs)

	for i := 0; i < runs; i++ {
		go func(i int) {
			defer wg.Done()
			q, err := b.Parse([]byte(`SET key value`))
			if err != nil {
				t.Error(err)
				return
			}

			ch, err := wal.Write(ctx, q)
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Error(err)
				return
			}

			select {
			case <-ctx.Done():
				return
			case err := <-ch:
				if err != nil {
					t.Error(err)
				}
			}
		}(i)
	}

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	wg.Wait()
}

func TestWALWriterError(t *testing.T) {
	wr := &MockWriterWithError{fail: true}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := config.WAL{
		FlushBatchSize: 2,
		FlushInterval:  10 * time.Millisecond,
	}

	wal := NewWAL(conf, &mocks.MockLogger{}, wr)
	b := query.NewBuilder()

	go wal.Start(ctx)

	q, _ := b.Parse([]byte("SET key value"))
	ch, err := wal.Write(ctx, q)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-ch:
		if err == nil || err.Error() != "write error" {
			t.Fatalf("expected write error, got %v", err)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("timeout waiting for flush")
	}
}
