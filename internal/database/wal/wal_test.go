package wal

import (
	"context"
	"errors"
	"laguna/internal/config"
	"laguna/internal/database/logs"
	"laguna/internal/mocks"
	"laguna/internal/query"
	"reflect"
	"sync"
	"testing"
	"time"
)

type MockWriterWithError struct {
	fail bool
}

func (w *MockWriterWithError) Write(_ []*logs.Row) error {
	if w.fail {
		return errors.New("writeInSeg error")
	}
	return nil
}

type mockReader struct {
	val []*logs.Row
	err error
}

func (m *mockReader) Read() ([]*logs.Row, error) {
	return m.val, m.err
}

func TestWALBasic(t *testing.T) {
	wr := &MockWriterWithError{}

	const walDeadline = 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), walDeadline)
	defer cancel()

	conf := config.WAL{
		Enable:         true,
		FlushBatchSize: 2,
		FlushInterval:  10 * time.Millisecond,
	}

	wal := NewWAL(conf, &mocks.MockLogger{}, wr, &mockReader{})
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
				t.Error(err)
			}

			resp, err := wal.Write(ctx, q)
			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			errV, err := resp.GetResponseWithDeadline(ctx)
			if err != nil {
				t.Error(err)
				return
			}

			if errV != nil {
				t.Error(err)
				return
			}
		})
	}
}

func TestWALConcurrently(t *testing.T) {
	wr := &MockWriterWithError{}
	const walDeadline = 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), walDeadline)
	defer cancel()

	conf := config.WAL{
		Enable:         true,
		FlushBatchSize: 100,
		FlushInterval:  10 * time.Millisecond,
	}
	runs := 100000

	wal := NewWAL(conf, &mocks.MockLogger{}, wr, &mockReader{})
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

			resp, err := wal.Write(ctx, q)
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Error(err)
				return
			}

			errV, err := resp.GetResponseWithDeadline(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				t.Error(err)
			}

			if errV != nil {
				t.Error(err)
				return
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
	const walDeadline = 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), walDeadline)
	defer cancel()

	conf := config.WAL{
		Enable:         true,
		FlushBatchSize: 2,
		FlushInterval:  10 * time.Millisecond,
	}

	wal := NewWAL(conf, &mocks.MockLogger{}, wr, &mockReader{})

	b := query.NewBuilder()

	go wal.Start(ctx)

	q, _ := b.Parse([]byte("SET key value"))
	resp, err := wal.Write(ctx, q)
	if err != nil {
		t.Error(err)
	}

	errV, err := resp.GetResponseWithDeadline(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	if errV != nil {
		if errV.Error() != "writeInSeg error" {
			t.Error(err)
			return
		}
	}
}
func TestReadWal_Success(t *testing.T) {
	const walDeadline = 1 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), walDeadline)
	defer cancel()

	conf := config.WAL{
		Enable:         true,
		FlushBatchSize: 2,
		FlushInterval:  10 * time.Millisecond,
	}

	tests := []struct {
		name    string
		records []*logs.Row
		want    []query.Query
		wantErr bool
	}{
		{
			name: "normal SET and DEL",
			records: []*logs.Row{
				logs.NewRow(42, query.SetMethodID, []string{"user", "1"}),
				logs.NewRow(99, query.DelMethodID, []string{"user"}),
			},
			want: []query.Query{
				query.NewQuery(query.SetMethodID, []string{"user", "1"}),
				query.NewQuery(query.DelMethodID, []string{"user"}),
			},
			wantErr: false,
		},
		{
			name: "SET with multiple args",
			records: []*logs.Row{
				logs.NewRow(101, query.SetMethodID, []string{"config", "hello", "world", "!"}),
			},
			want: []query.Query{
				query.NewQuery(query.SetMethodID, []string{"config", "hello", "world", "!"}),
			},
			wantErr: false,
		},
		{
			name:    "empty WAL",
			records: []*logs.Row{},
			want:    []query.Query{},
			wantErr: false,
		},
		{
			name:    "reader error",
			records: nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "mixed SET, GET, DEL",
			records: []*logs.Row{
				logs.NewRow(200, query.SetMethodID, []string{"k1", "v1"}),
				logs.NewRow(201, query.GetMethodID, []string{"k1"}),
				logs.NewRow(202, query.DelMethodID, []string{"k1"}),
			},
			want: []query.Query{
				query.NewQuery(query.SetMethodID, []string{"k1", "v1"}),
				query.NewQuery(query.GetMethodID, []string{"k1"}),
				query.NewQuery(query.DelMethodID, []string{"k1"}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r := &mockReader{
				val: tt.records,
				err: nil,
			}
			if tt.wantErr {
				r.err = errors.New("read error")
			}

			w := NewWAL(conf, &mocks.MockLogger{}, &MockWriterWithError{}, r)
			go w.Start(ctx)

			got, err := w.Restore()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Restore() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Fatalf("Restore() len = %d, want %d", len(got), len(tt.want))
				}

				for i := range got {
					if !reflect.DeepEqual(got[i], tt.want[i]) {
						t.Fatalf("Restore() got[%d] = %v, want %v", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}
