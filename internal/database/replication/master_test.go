package replication

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"laguna/internal/database/logs"
	"laguna/internal/database/wal"
	"laguna/internal/mocks"
	"laguna/utils/concurrency"
	"testing"
)

type mockLogWriter struct {
	writeToErr error
}

func (m *mockLogWriter) Write(rows []*wal.Row) error {
	return nil
}

func (m *mockLogWriter) WriteTo(rows []*wal.Row, t logs.Target) error {
	return m.writeToErr
}

type mockLogReader struct{}

func (m *mockLogReader) Read(r io.Reader) ([]*wal.Row, error) {
	return nil, nil
}

func (m *mockLogReader) ReadStream(r io.Reader) concurrency.FutureRespWithErr[[]*wal.Row] {
	resp := concurrency.NewFutureRespWithErr[[]*wal.Row]()
	go func() {
		defer resp.Done()
		resp.Put(nil, nil)
	}()
	return resp
}

func (m *mockLogReader) ReadFrom(directory string, target string) ([]*wal.Row, error) {
	return nil, nil
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestMaster_Handle(t *testing.T) {
	tests := []struct {
		name       string
		request    *Request
		logWriter  LogWriter
		reader     io.Reader
		ctx        context.Context
		wantErr    bool
		wantErrMsg string
		checkResp  bool
	}{
		{
			name: "successful handle",
			request: &Request{
				LsnID: 1,
			},
			logWriter: &mockLogWriter{
				writeToErr: nil,
			},
			reader:    nil,
			ctx:       context.Background(),
			wantErr:   false,
			checkResp: true,
		},
		{
			name: "context canceled",
			request: &Request{
				LsnID: 1,
			},
			logWriter: &mockLogWriter{},
			reader:    nil,
			ctx:       func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			wantErr:   true,
		},
		{
			name: "read input error",
			request: &Request{
				LsnID: 1,
			},
			logWriter: &mockLogWriter{},
			reader:    &errorReader{err: errors.New("read error")},
			ctx:       context.Background(),
			wantErr:   false,
			checkResp: true,
		},
		{
			name: "unmarshal request error",
			request: &Request{
				LsnID: 1,
			},
			logWriter: &mockLogWriter{},
			reader:    bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04}),
			ctx:       context.Background(),
			wantErr:   false,
			checkResp: true,
		},
		{
			name: "writeTo error",
			request: &Request{
				LsnID: 1,
			},
			logWriter: &mockLogWriter{
				writeToErr: errors.New("write error"),
			},
			reader:     nil,
			ctx:        context.Background(),
			wantErr:    false,
			wantErrMsg: "write error",
			checkResp:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reader io.Reader
			if tt.reader != nil {
				reader = tt.reader
			} else {
				reqData, err := tt.request.Marshal()
				if err != nil {
					t.Fatalf("Failed to marshal request: %v", err)
				}
				reader = bytes.NewReader(reqData)
			}

			master := NewMaster(&mockLogReader{}, tt.logWriter, &mocks.MockLogger{})
			writer := &bytes.Buffer{}

			err := master.Handle(tt.ctx, reader, writer)

			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.checkResp {
				if writer.Len() == 0 {
					t.Error("Handle() expected response in writer, got empty")
				} else {
					var resp Response
					if err := resp.Unmarshal(writer.Bytes()); err != nil {
						t.Errorf("Handle() produced invalid response: %v", err)
					}

					if tt.wantErrMsg != "" && resp.Err != nil {
						if resp.Err.Error() != tt.wantErrMsg {
							t.Errorf("Handle() error message = %q, want %q", resp.Err.Error(), tt.wantErrMsg)
						}
					}
				}
			}
		})
	}
}

func TestMaster_writeError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantErr    bool
		checkError bool
	}{
		{
			name:       "write error response",
			err:        errors.New("test error"),
			wantErr:    false,
			checkError: true,
		},
		{
			name:       "nil error",
			err:        nil,
			wantErr:    false,
			checkError: false,
		},
		{
			name:       "unsupported error",
			err:        errors.ErrUnsupported,
			wantErr:    false,
			checkError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			master := NewMaster(nil, nil, &mocks.MockLogger{})
			writer := &bytes.Buffer{}
			bufWriter := bufio.NewWriter(writer)

			err := master.writeError(tt.err, bufWriter)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeError() error = %v, wantErr %v", err, tt.wantErr)
			}

			if writer.Len() == 0 {
				t.Error("writeError() expected output, got empty")
				return
			}

			if tt.checkError {
				var resp Response
				if err := resp.Unmarshal(writer.Bytes()); err != nil {
					t.Errorf("writeError() produced invalid response: %v", err)
					return
				}

				if resp.Err == nil {
					t.Error("writeError() expected error in response, got nil")
				} else if resp.Err.Error() != tt.err.Error() {
					t.Errorf("writeError() error message = %q, want %q", resp.Err.Error(), tt.err.Error())
				}
			}
		})
	}
}
