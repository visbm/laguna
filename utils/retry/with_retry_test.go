package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithRetry(t *testing.T) {
	static := 0
	type args struct {
		ctx       context.Context
		retry     int
		baseDelay time.Duration
		f         func() error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "always error → must return error",
			args: args{
				ctx:   context.Background(),
				retry: 3,
				f: func() error {
					return errors.New("a")
				},
				baseDelay: 1 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "success on second attempt → must return nil",
			args: args{
				ctx:       context.Background(),
				retry:     3,
				baseDelay: 1 * time.Millisecond,
				f: func() error {
					if static == 1 {
						return nil
					}
					static++
					return errors.New("fail")
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := WithRetry(tt.args.ctx, tt.args.retry, tt.args.baseDelay, tt.args.f); (err != nil) != tt.wantErr {
				t.Errorf("WithRetry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWithRetryValue(t *testing.T) {
	static := 0
	type args[T any] struct {
		ctx       context.Context
		retry     int
		baseDelay time.Duration
		f         func() (T, error)
	}
	tests := []struct {
		name    string
		args    args[int]
		want    int
		wantErr bool
	}{
		{
			name: "always error → must return error",
			args: args[int]{
				ctx:       context.Background(),
				retry:     3,
				baseDelay: 1 * time.Millisecond,
				f: func() (int, error) {
					return 0, errors.New("fail")
				},
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "success on second attempt → must return value",
			args: args[int]{
				ctx:       context.Background(),
				retry:     3,
				baseDelay: 1 * time.Millisecond,
				f: func() (int, error) {
					if static == 1 {
						return 42, nil
					}
					static++
					return 0, errors.New("fail")
				},
			},
			want:    42,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WithRetryValue(tt.args.ctx, tt.args.retry, tt.args.baseDelay, tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("WithRetryValue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("WithRetryValue() got = %v, want %v", got, tt.want)
			}
		})
	}
}
