package wal

import (
	"laguna/internal/query"
	"math"
	"reflect"
	"testing"
)

func TestRowMarshalUnmarshal_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		row  *row
	}{
		{
			name: "simple GET",
			row: &row{
				lsnID:    1,
				methodID: query.GetMethodID,
				args:     []string{"user"},
			},
		},
		{
			name: "simple GET with max lsn",
			row: &row{
				lsnID:    math.MaxUint64,
				methodID: query.GetMethodID,
				args:     []string{"user"},
			},
		},
		{
			name: "SET with multi arg",
			row: &row{
				lsnID:    1,
				methodID: query.GetMethodID,
				args:     []string{"user", "Jone", "Doe"},
			},
		},
		{
			name: "SET with single arg",
			row: &row{
				lsnID:    42,
				methodID: query.SetMethodID,
				args:     []string{"user", "1"},
			},
		},
		{
			name: "DEL with multiple args",
			row: &row{
				lsnID:    99,
				methodID: query.DelMethodID,
				args:     []string{"motosycle", "harrley", "davidson"},
			},
		},
		{
			name: "empty args",
			row: &row{
				lsnID:    7,
				methodID: query.GetMethodID,
				args:     []string{""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.row.Marshal()

			got := &row{}
			if err := got.Unmarshal(data); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if tt.row.lsnID != got.lsnID {
				t.Errorf("lsnID mismatch: got %d, want %d", got.lsnID, tt.row.lsnID)
			}

			if tt.row.methodID != got.methodID {
				t.Errorf("method mismatch: got %v, want %v", got.methodID, tt.row.methodID)
			}

			if !reflect.DeepEqual(tt.row.args, got.args) {
				t.Errorf("args mismatch: got %v, want %v", got.args, tt.row.args)
			}
		})
	}
}
