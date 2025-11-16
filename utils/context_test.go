package utils

import (
	"context"
	"testing"
)

func TestTxInContext(t *testing.T) {
	type testCase struct {
		name   string
		ctx    context.Context
		txID   uint64
		expect uint64
		setup  func() context.Context
	}

	tests := []testCase{
		{
			name:   "basic set and get",
			ctx:    context.Background(),
			txID:   123,
			expect: 123,
		},
		{
			name:   "nil context on set",
			ctx:    nil,
			txID:   42,
			expect: 42,
		},
		{
			name:   "nil context on get",
			ctx:    nil,
			txID:   0,
			expect: 0,
		},
		{
			name: "wrong type stored in context",
			setup: func() context.Context {
				return context.WithValue(context.Background(), txKey, "not_uint64")
			},
			expect: 0,
		},
		{
			name: "independent contexts",
			setup: func() context.Context {
				ctx1 := SetTxInContext(context.Background(), 100)
				ctx2 := SetTxInContext(context.Background(), 200)
				if GetTxInContext(ctx1) != 100 {
					t.Fatalf("ctx1 expected 100, got %d", GetTxInContext(ctx1))
				}
				if GetTxInContext(ctx2) != 200 {
					t.Fatalf("ctx2 expected 200, got %d", GetTxInContext(ctx2))
				}
				return ctx1
			},
			expect: 100,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var ctx context.Context
			if tc.setup != nil {
				ctx = tc.setup()
			} else {
				ctx = SetTxInContext(tc.ctx, tc.txID)
			}

			got := GetTxInContext(ctx)
			if got != tc.expect {
				t.Fatalf("[%s] expected %d, got %d", tc.name, tc.expect, got)
			}
		})
	}
}
