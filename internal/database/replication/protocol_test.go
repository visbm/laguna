package replication

import (
	"errors"
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
			}
			if !tt.wantErr && req2.LsnID != tt.lsnID {
				t.Errorf("Unmarshal() got GetLsnID = %v, want %v", req2.LsnID, tt.lsnID)
			}
		})
	}
}

func TestResponse_MarshalUnmarshal(t *testing.T) {
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
			name:    "data and error",
			data:    []byte("payload"),
			err:     errors.New("bad request"),
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
			bin, err := resp.Marshal()
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var resp2 Response
			err = resp2.Unmarshal(bin)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if string(resp2.Data) != string(tt.data) {
					t.Errorf("Unmarshal() Data = %q, want %q", resp2.Data, tt.data)
				}
				if (resp2.Err == nil && tt.err != nil) ||
					(resp2.Err != nil && tt.err == nil) ||
					(resp2.Err != nil && tt.err != nil && resp2.Err.Error() != tt.err.Error()) {
					t.Errorf("Unmarshal() Err = %v, want %v", resp2.Err, tt.err)
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
		{
			name:    "large LSN",
			lsnID:   18446744073709551615,
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

func TestResponse_Unmarshal_InvalidData(t *testing.T) {
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
			var resp Response
			err := resp.Unmarshal(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
