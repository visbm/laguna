package query

import (
	"errors"
	"testing"
)

func TestBuilder_Parse(t *testing.T) {
	b := NewBuilder()

	tests := []struct {
		name       string
		input      string
		wantErr    bool
		errValue   error
		wantMethod MethodID
		wantArgs   []string
	}{
		{
			name:       "valid SET",
			input:      "SET key value",
			wantErr:    false,
			wantMethod: SetMethodID,
			wantArgs:   []string{"key", "value"},
		},
		{
			name:       "valid SET multiple args",
			input:      "SET key hello world",
			wantErr:    false,
			wantMethod: SetMethodID,
			wantArgs:   []string{"key", "hello world"},
		},
		{
			name:       "valid GET",
			input:      "GET mykey",
			wantErr:    false,
			wantMethod: GetMethodID,
			wantArgs:   []string{"mykey"},
		},
		{
			name:       "valid DEL",
			input:      "DEL key",
			wantErr:    false,
			wantMethod: DelMethodID,
			wantArgs:   []string{"key"},
		},
		{
			name:     "unknown method",
			input:    "FOO bar",
			wantErr:  true,
			errValue: ErrUnknowMethod,
		},
		{
			name:     "wrong args length for SET",
			input:    "SET key",
			wantErr:  true,
			errValue: ErrWrongArgsLen,
		},
		{
			name:     "wrong args length for GET",
			input:    "GET",
			wantErr:  true,
			errValue: ErrWrongQuery,
		},
		{
			name:     "wrong args length for DEL",
			input:    "DEL",
			wantErr:  true,
			errValue: ErrWrongQuery,
		},
		{
			name:     "empty input",
			input:    "",
			wantErr:  true,
			errValue: ErrWrongQuery,
		},
		{
			name:       "input with extra spaces",
			input:      "  SET   key   value  ",
			wantErr:    false,
			wantMethod: SetMethodID,
			wantArgs:   []string{"key", "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := b.Parse([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}

			if err != nil && tt.errValue != nil && !errors.Is(err, tt.errValue) {
				t.Fatalf("expected error: %v, got: %v", tt.errValue, err)
			}

			if !tt.wantErr {
				if got.MethodID() != tt.wantMethod {
					t.Errorf("methodID: want %v, got %v", tt.wantMethod, got.MethodID())
				}
				if len(got.GetArs()) != len(tt.wantArgs) {
					t.Fatalf("args length: want %v, got %v", len(tt.wantArgs), len(got.GetArs()))
				}
				for i := range got.GetArs() {
					if got.GetArs()[i] != tt.wantArgs[i] {
						t.Errorf("arg[%d]: want %v, got %v", i, tt.wantArgs[i], got.GetArs()[i])
					}
				}
			}
		})
	}
}
