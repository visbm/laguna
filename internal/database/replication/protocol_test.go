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
