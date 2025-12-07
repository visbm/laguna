package data_type

import "testing"

func TestByteSize_UnmarshalText(t *testing.T) {
	tests := []struct {
		input   string
		want    ByteSize
		wantErr bool
	}{
		{"1024B", 1024, false},
		{"1KB", 1024, false},
		{"1kb", 1024, false},
		{"1.5KB", 1536, false},
		{"2MB", 2 * 1024 * 1024, false},
		{"0.5MB", 524288, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"123", 123, false},
		{"   2kb  ", 2048, false},
		{"", 0, false},
		{"abc", 0, true},
		{"1XB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var b ByteSize
			err := b.UnmarshalText([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr && b != tt.want {
				t.Errorf("expected %d, got %d", tt.want, b)
			}
		})
	}
}
