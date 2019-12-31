package replication

import (
	"bytes"
	"errors"
	"laguna/internal/database/wal"
	"laguna/internal/query"
	"testing"
)

func TestRequest_MarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		lsnID   uint64
		wantErr bool
	}{
		{
			name:    "valid marshal/unmarshal",
			lsnID:   123456789,
			wantErr: false,
		},
		{
			name:    "unmarshal with not enough bytes",
			lsnID:   0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{LsnID: tt.lsnID}
			data, err := req.Marshal()
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var req2 Request
			var unmarshalErr error
			if tt.wantErr {
				unmarshalErr = req2.Unmarshal(data[:4])
			} else {
				unmarshalErr = req2.Unmarshal(data)
			}

			if (unmarshalErr != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", unmarshalErr, tt.wantErr)
				return
			}
			if !tt.wantErr && req2.LsnID != tt.lsnID {
				t.Errorf("Unmarshal() got LsnID = %v, want %v", req2.LsnID, tt.lsnID)
			}
		})
	}
}

func TestResponse_WriteAndReadMessage(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		err     error
		wantErr bool
	}{
		{
			name:    "only data no error",
			data:    []byte("hello"),
			err:     nil,
			wantErr: false,
		},
		{
			name:    "only error no data",
			data:    nil,
			err:     errors.New("something went wrong"),
			wantErr: false,
		},
		{
			name:    "empty data no error",
			data:    []byte{},
			err:     nil,
			wantErr: false,
		},
		{
			name:    "large data",
			data:    make([]byte, 10000),
			err:     nil,
			wantErr: false,
		},
		{
			name:    "error with empty message",
			data:    nil,
			err:     errors.New(""),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{Data: tt.data, Err: tt.err}
			bin, err := resp.WriteMessage()
			if err != nil {
				t.Fatalf("WriteMessage() error = %v", err)
			}

			resp2, err := ReadMessage(bytes.NewReader(bin))
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMessage() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if string(resp2.Data) != string(tt.data) {
					t.Errorf("ReadMessage() Data = %q, want %q", resp2.Data, tt.data)
				}
				if (resp2.Err == nil && tt.err != nil) ||
					(resp2.Err != nil && tt.err == nil) ||
					(resp2.Err != nil && tt.err != nil && resp2.Err.Error() != tt.err.Error()) {
					t.Errorf("ReadMessage() Err = %v, want %v", resp2.Err, tt.err)
				}
			}
		})
	}
}

func TestRequest_MarshalUnmarshal_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		lsnID   uint64
		wantErr bool
	}{
		{
			name:    "zero LSN",
			lsnID:   0,
			wantErr: false,
		},
		{
			name:    "max uint64 LSN",
			lsnID:   ^uint64(0),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{LsnID: tt.lsnID}
			data, err := req.Marshal()
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			if len(data) != reqSize {
				t.Errorf("Marshal() length = %d, want %d", len(data), reqSize)
			}

			var req2 Request
			err = req2.Unmarshal(data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && req2.LsnID != tt.lsnID {
				t.Errorf("Unmarshal() LsnID = %v, want %v", req2.LsnID, tt.lsnID)
			}
		})
	}
}

func TestResponse_ReadMessage_InvalidData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "too short - only flag",
			data:    []byte{0},
			wantErr: true,
		},
		{
			name:    "too short - flag and partial length",
			data:    []byte{0, 0, 0},
			wantErr: true,
		},
		{
			name:    "negative data length",
			data:    []byte{0, 0xFF, 0xFF, 0xFF, 0xFF},
			wantErr: true,
		},
		{
			name:    "data length exceeds buffer",
			data:    []byte{0, 0, 0, 0, 100, 0, 0, 0, 0}, // length = 100 but only 1 byte after
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadMessage(bytes.NewReader(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConnTarget_Write(t *testing.T) {
	tests := []struct {
		name    string
		rows    []*wal.Row
		wantErr bool
	}{
		{
			name: "successful write single row",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key", "value"}),
			},
			wantErr: false,
		},
		{
			name: "successful write multiple rows",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key1", "value1"}),
				wal.NewRow(2, query.DelMethodID, []string{"key2"}),
				wal.NewRow(3, query.SetMethodID, []string{"key3", "value3"}),
			},
			wantErr: false,
		},
		{
			name:    "empty rows",
			rows:    []*wal.Row{},
			wantErr: false,
		},
		{
			name: "row with multiple args",
			rows: []*wal.Row{
				wal.NewRow(10, query.SetMethodID, []string{"config", "hello", "world", "!"}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &bytes.Buffer{}
			connTarget := NewConnTarget(writer)

			err := connTarget.Write(tt.rows)

			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if writer.Len() == 0 && len(tt.rows) > 0 {
					t.Error("Write() expected output, got empty")
				}

				resp, err := ReadMessage(writer)
				if err != nil {
					t.Errorf("Write() produced invalid response: %v", err)
					return
				}

				if resp.Err != nil {
					t.Errorf("Write() response has error: %v", resp.Err)
				}
				expectedDataLen := 0
				for _, r := range tt.rows {
					data, _ := r.Marshal()
					expectedDataLen += len(data)
				}

				if len(resp.Data) != expectedDataLen {
					t.Errorf("Write() data length = %d, want %d", len(resp.Data), expectedDataLen)
				}
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	tests := []struct {
		name    string
		rows    []*wal.Row
		wantErr bool
	}{
		{
			name: "flatten single row",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key", "value"}),
			},
			wantErr: false,
		},
		{
			name: "flatten multiple rows",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key1", "value1"}),
				wal.NewRow(2, query.DelMethodID, []string{"key2"}),
			},
			wantErr: false,
		},
		{
			name:    "flatten empty rows",
			rows:    []*wal.Row{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := flatten(tt.rows)

			if (err != nil) != tt.wantErr {
				t.Errorf("flatten() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				expectedLen := 0
				for _, r := range tt.rows {
					rowData, _ := r.Marshal()
					expectedLen += len(rowData)
				}

				if len(data) != expectedLen {
					t.Errorf("flatten() length = %d, want %d", len(data), expectedLen)
				}

				offset := 0
				for i, r := range tt.rows {
					expectedData, _ := r.Marshal()
					if offset+len(expectedData) > len(data) {
						t.Errorf("flatten() row %d: not enough data", i)
						break
					}

					gotData := data[offset : offset+len(expectedData)]
					if !bytes.Equal(gotData, expectedData) {
						t.Errorf("flatten() row %d: data mismatch", i)
					}
					offset += len(expectedData)
				}
			}
		})
	}
}
