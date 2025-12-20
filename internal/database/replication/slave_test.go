package replication

import (
	"bytes"
	"context"
	"errors"
	"io"
	"laguna/internal/database/logs"
	"laguna/internal/database/wal"
	"laguna/internal/mocks"
	"laguna/internal/query"
	"laguna/utils/concurrency"
	"testing"
	"time"
)

type mockTCPClient struct {
	sendResp []byte
	sendErr  error
}

func (m *mockTCPClient) Send(ctx context.Context, req []byte) (io.Reader, error) {
	if m.sendErr != nil {
		return nil, m.sendErr
	}
	return bytes.NewReader(m.sendResp), nil
}

func (m *mockTCPClient) Close() {
	// Mock implementation, no-op
}

type mockStorage struct {
	getErr    error
	setErr    error
	deleteErr error
	getVal    string
}

func (m *mockStorage) Get(ctx context.Context, key string) (string, error) {
	return m.getVal, m.getErr
}

func (m *mockStorage) Set(ctx context.Context, key, value string) error {
	return m.setErr
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	return m.deleteErr
}

type mockSlaveLogReader struct {
	readStreamResp concurrency.FutureRespWithErr[[]*wal.Row]
}

func (m *mockSlaveLogReader) Read(r io.Reader) ([]*wal.Row, error) {
	return nil, nil
}

func (m *mockSlaveLogReader) ReadStream(r io.Reader) concurrency.FutureRespWithErr[[]*wal.Row] {
	return m.readStreamResp
}

func (m *mockSlaveLogReader) ReadFrom(directory string, target string, firstOffset int64) ([]*wal.Row, error) {
	return nil, nil
}

type mockSlaveLogWriter struct {
	writeErr error
}

func (m *mockSlaveLogWriter) Write(rows []*wal.Row) error {
	return m.writeErr
}

func (m *mockSlaveLogWriter) WriteTo(rows []*wal.Row, t logs.Target) error {
	return nil
}

func TestSlave_sendReq(t *testing.T) {
	tests := []struct {
		name      string
		lsnID     uint64
		client    TCPClient
		wantErr   bool
		wantLsnID uint64
	}{
		{
			name:  "successful request",
			lsnID: 123,
			client: &mockTCPClient{
				sendResp: createSuccessResponse([]byte("test data")),
				sendErr:  nil,
			},
			wantErr:   false,
			wantLsnID: 123,
		},
		{
			name:  "client send error",
			lsnID: 456,
			client: &mockTCPClient{
				sendErr: errors.New("network error"),
			},
			wantErr: true,
		},
		{
			name:  "response with error",
			lsnID: 789,
			client: &mockTCPClient{
				sendResp: createErrorResponse(errors.New("server error")),
				sendErr:  nil,
			},
			wantErr: true,
		},
		{
			name:  "response with ErrNoNewLogs",
			lsnID: 1000,
			client: &mockTCPClient{
				sendResp: createErrorResponse(ErrNoNewLogs),
				sendErr:  nil,
			},
			wantErr: true,
		},
		{
			name:  "unmarshal error",
			lsnID: 999,
			client: &mockTCPClient{
				sendResp: []byte{0x01, 0x02}, // invalid response
				sendErr:  nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slave := &Slave{
				cl:  tt.client,
				log: &mocks.MockLogger{},
			}

			ctx := context.Background()
			resp, err := slave.sendReq(ctx, tt.lsnID)

			if (err != nil) != tt.wantErr {
				t.Errorf("sendReq() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resp.Data == nil {
				t.Error("sendReq() expected response data, got nil")
			}

			if tt.name == "response with ErrNoNewLogs" {
				if err == nil {
					t.Error("sendReq() should return ErrNoNewLogs error")
				} else if !errors.Is(err, ErrNoNewLogs) {
					t.Errorf("sendReq() error should be ErrNoNewLogs, got %v", err)
				}
			}
		})
	}
}

func TestSlave_execStorage(t *testing.T) {
	tests := []struct {
		name    string
		rows    []*wal.Row
		storage Storage
		wantErr bool
	}{
		{
			name: "successful SET",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key", "value"}),
			},
			storage: &mockStorage{
				setErr: nil,
			},
			wantErr: false,
		},
		{
			name: "successful DEL",
			rows: []*wal.Row{
				wal.NewRow(2, query.DelMethodID, []string{"key"}),
			},
			storage: &mockStorage{
				deleteErr: nil,
			},
			wantErr: false,
		},
		{
			name: "SET error",
			rows: []*wal.Row{
				wal.NewRow(3, query.SetMethodID, []string{"key", "value"}),
			},
			storage: &mockStorage{
				setErr: errors.New("storage error"),
			},
			wantErr: true,
		},
		{
			name: "DEL error",
			rows: []*wal.Row{
				wal.NewRow(4, query.DelMethodID, []string{"key"}),
			},
			storage: &mockStorage{
				deleteErr: errors.New("delete error"),
			},
			wantErr: true,
		},
		{
			name: "unknown method",
			rows: []*wal.Row{
				wal.NewRow(5, query.GetMethodID, []string{"key"}),
			},
			storage: &mockStorage{},
			wantErr: true,
		},
		{
			name: "multiple rows",
			rows: []*wal.Row{
				wal.NewRow(6, query.SetMethodID, []string{"key1", "value1"}),
				wal.NewRow(7, query.DelMethodID, []string{"key2"}),
			},
			storage: &mockStorage{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slave := &Slave{
				st:  tt.storage,
				log: &mocks.MockLogger{},
			}

			ctx := context.Background()
			err := slave.execStorage(ctx, tt.rows)

			if (err != nil) != tt.wantErr {
				t.Errorf("execStorage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSlave_processRows(t *testing.T) {
	tests := []struct {
		name      string
		rows      []*wal.Row
		logWriter LogWriter
		storage   Storage
		wantErr   bool
		wantLsnID uint64
	}{
		{
			name: "successful process with WAL",
			rows: []*wal.Row{
				wal.NewRow(10, query.SetMethodID, []string{"key", "value"}),
			},
			logWriter: &mockSlaveLogWriter{
				writeErr: nil,
			},
			storage: &mockStorage{
				setErr: nil,
			},
			wantErr:   false,
			wantLsnID: 10,
		},
		{
			name: "successful process without WAL",
			rows: []*wal.Row{
				wal.NewRow(11, query.SetMethodID, []string{"key", "value"}),
			},
			logWriter: nil,
			storage: &mockStorage{
				setErr: nil,
			},
			wantErr:   false,
			wantLsnID: 11,
		},
		{
			name: "WAL write error",
			rows: []*wal.Row{
				wal.NewRow(12, query.SetMethodID, []string{"key", "value"}),
			},
			logWriter: &mockSlaveLogWriter{
				writeErr: errors.New("wal error"),
			},
			storage: &mockStorage{},
			wantErr: true,
		},
		{
			name: "storage error",
			rows: []*wal.Row{
				wal.NewRow(13, query.SetMethodID, []string{"key", "value"}),
			},
			logWriter: &mockSlaveLogWriter{},
			storage: &mockStorage{
				setErr: errors.New("storage error"),
			},
			wantErr: true,
		},
		{
			name: "context canceled",
			rows: []*wal.Row{
				wal.NewRow(14, query.SetMethodID, []string{"key", "value"}),
			},
			logWriter: &mockSlaveLogWriter{},
			storage:   &mockStorage{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slave := &Slave{
				logWriter: tt.logWriter,
				st:        tt.storage,
				log:       &mocks.MockLogger{},
			}

			ctx, done := context.WithCancel(context.Background())
			defer done()
			if tt.name == "context canceled" {
				done()
			}

			err := slave.processRows(ctx, tt.rows)

			if (err != nil) != tt.wantErr {
				t.Errorf("processRows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && slave.lastLsnWritten != tt.wantLsnID {
				t.Errorf("processRows() lastLsnWritten = %v, want %v", slave.lastLsnWritten, tt.wantLsnID)
			}
		})
	}
}

func TestSlave_getUpdates(t *testing.T) {
	tests := []struct {
		name      string
		client    TCPClient
		logReader LogReader
		wantErr   bool
	}{
		{
			name: "successful get updates",
			client: &mockTCPClient{
				sendResp: createSuccessResponse(createRowsData([]*wal.Row{
					wal.NewRow(20, query.SetMethodID, []string{"key", "value"}),
				})),
				sendErr: nil,
			},
			logReader: &mockSlaveLogReader{
				readStreamResp: createMockStream([]*wal.Row{
					wal.NewRow(20, query.SetMethodID, []string{"key", "value"}),
				}),
			},
			wantErr: false,
		},
		{
			name: "empty response data",
			client: &mockTCPClient{
				sendResp: createSuccessResponse([]byte{}),
				sendErr:  nil,
			},
			logReader: &mockSlaveLogReader{},
			wantErr:   false,
		},
		{
			name: "send request error",
			client: &mockTCPClient{
				sendErr: errors.New("network error"),
			},
			logReader: &mockSlaveLogReader{},
			wantErr:   true,
		},
		{
			name: "context canceled",
			client: &mockTCPClient{
				sendResp: createSuccessResponse([]byte("data")),
			},
			logReader: &mockSlaveLogReader{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slave := &Slave{
				cl:             tt.client,
				logReader:      tt.logReader,
				st:             &mockStorage{},
				log:            &mocks.MockLogger{},
				lastLsnWritten: 0,
			}

			ctx, done := context.WithCancel(context.Background())
			defer done()
			if tt.name == "context canceled" {
				done()
			}

			err := slave.getUpdates(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("getUpdates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewSlave(t *testing.T) {
	tests := []struct {
		name         string
		syncInterval time.Duration
		wantInterval time.Duration
	}{
		{
			name:         "custom sync interval",
			syncInterval: 5 * time.Second,
			wantInterval: 5 * time.Second,
		},
		{
			name:         "zero sync interval uses default",
			syncInterval: 0,
			wantInterval: defaultSyncInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slave := NewSlave(
				&mocks.MockLogger{},
				&mockTCPClient{},
				&mockSlaveLogReader{},
				&mockStorage{},
				tt.syncInterval,
			)

			if slave.syncInterval != tt.wantInterval {
				t.Errorf("NewSlave() syncInterval = %v, want %v", slave.syncInterval, tt.wantInterval)
			}
		})
	}
}

func TestNewSlaveWithWal(t *testing.T) {
	tests := []struct {
		name         string
		syncInterval time.Duration
		wantInterval time.Duration
	}{
		{
			name:         "custom sync interval with WAL",
			syncInterval: 3 * time.Second,
			wantInterval: 3 * time.Second,
		},
		{
			name:         "zero sync interval uses default with WAL",
			syncInterval: 0,
			wantInterval: defaultSyncInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slave := NewSlaveWithWal(
				&mocks.MockLogger{},
				&mockTCPClient{},
				&mockSlaveLogReader{},
				&mockSlaveLogWriter{},
				&mockStorage{},
				tt.syncInterval,
			)

			if slave.syncInterval != tt.wantInterval {
				t.Errorf("NewSlaveWithWal() syncInterval = %v, want %v", slave.syncInterval, tt.wantInterval)
			}

			if slave.logWriter == nil {
				t.Error("NewSlaveWithWal() expected logWriter to be set")
			}
		})
	}
}

func TestSlave_walEnable(t *testing.T) {
	tests := []struct {
		name      string
		logWriter LogWriter
		want      bool
	}{
		{
			name:      "WAL enabled",
			logWriter: &mockSlaveLogWriter{},
			want:      true,
		},
		{
			name:      "WAL disabled",
			logWriter: nil,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slave := &Slave{
				logWriter: tt.logWriter,
			}

			got := slave.walEnable()
			if got != tt.want {
				t.Errorf("walEnable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func createSuccessResponse(data []byte) []byte {
	resp := &Response{Data: data, Err: nil}
	result, _ := resp.WriteMessage()
	return result
}

func createErrorResponse(err error) []byte {
	resp := &Response{Data: nil, Err: err}
	result, _ := resp.WriteMessage()
	return result
}

func createRowsData(rows []*wal.Row) []byte {
	var buf bytes.Buffer
	for _, r := range rows {
		data, _ := r.Marshal()
		buf.Write(data)
	}
	return buf.Bytes()
}

func createMockStream(rows []*wal.Row) concurrency.FutureRespWithErr[[]*wal.Row] {
	resp := concurrency.NewFutureRespWithErr[[]*wal.Row]()
	go func() {
		defer resp.Done()
		resp.Put(rows, nil)
	}()
	return resp
}
