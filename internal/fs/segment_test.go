package fs

import (
	"laguna/internal/mocks"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileSegment(t *testing.T) {
	tests := []struct {
		name      string
		maxSize   int64
		setup     func(string) error
		wantErr   bool
		checkFile bool
	}{
		{
			name:    "create in empty directory",
			maxSize: 1024,
			setup: func(dir string) error {
				return nil
			},
			wantErr:   false,
			checkFile: true,
		},
		{
			name:    "open existing file",
			maxSize: 2048,
			setup: func(dir string) error {
				file, err := os.Create(filepath.Join(dir, "1000000.bin"))
				if err != nil {
					return err
				}
				file.Close()
				return nil
			},
			wantErr:   false,
			checkFile: true,
		},
		{
			name:    "invalid directory",
			maxSize: 1024,
			setup: func(dir string) error {
				return nil
			},
			wantErr:   true,
			checkFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.name == "invalid directory" {
				dir = filepath.Join(dir, "nonexistent", "subdir")
			} else {
				if err := tt.setup(dir); err != nil {
					t.Fatalf("setup error: %v", err)
				}
			}

			seg, err := NewFileSegment(&mocks.MockLogger{}, dir, tt.maxSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileSegment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if seg == nil {
					t.Error("NewFileSegment() returned nil segment")
					return
				}

				if seg.maxSize != tt.maxSize {
					t.Errorf("NewFileSegment() maxSize = %d, want %d", seg.maxSize, tt.maxSize)
				}

				if seg.path != dir {
					t.Errorf("NewFileSegment() path = %q, want %q", seg.path, dir)
				}

				if tt.checkFile && seg.file == nil {
					t.Error("NewFileSegment() file is nil")
				}

				seg.Close()
			}
		})
	}
}

func TestFileSegment_Write(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "write single byte",
			data:    []byte{0x01},
			wantErr: false,
		},
		{
			name:    "write multiple bytes",
			data:    []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			wantErr: false,
		},
		{
			name:    "write empty data",
			data:    []byte{},
			wantErr: false,
		},
		{
			name:    "write large data",
			data:    make([]byte, 1000),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			seg, err := NewFileSegment(&mocks.MockLogger{}, dir, 10240)
			if err != nil {
				t.Fatalf("NewFileSegment() error = %v", err)
			}
			defer seg.Close()

			initialSize := seg.size
			err = seg.Write(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				expectedSize := initialSize + int64(len(tt.data))
				if seg.size != expectedSize {
					t.Errorf("Write() size = %d, want %d", seg.size, expectedSize)
				}
			}
		})
	}
}

func TestFileSegment_Fits(t *testing.T) {
	tests := []struct {
		name     string
		maxSize  int64
		fileSize int64
		want     bool
	}{
		{
			name:     "fits exactly",
			maxSize:  100,
			fileSize: 100,
			want:     true,
		},
		{
			name:     "fits with space",
			maxSize:  100,
			fileSize: 50,
			want:     true,
		},
		{
			name:     "does not fit",
			maxSize:  100,
			fileSize: 101,
			want:     false,
		},
		{
			name:     "fits after write",
			maxSize:  100,
			fileSize: 50,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			seg, err := NewFileSegment(&mocks.MockLogger{}, dir, tt.maxSize)
			if err != nil {
				t.Fatalf("NewFileSegment() error = %v", err)
			}
			defer seg.Close()

			if tt.name == "fits after write" {
				seg.Write(make([]byte, 30))
			}

			got := seg.Fits(tt.fileSize)
			if got != tt.want {
				t.Errorf("Fits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileSegment_Rotate(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "rotate successfully",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			seg, err := NewFileSegment(&mocks.MockLogger{}, dir, 1024)
			if err != nil {
				t.Fatalf("NewFileSegment() error = %v", err)
			}
			defer seg.Close()

			oldName := seg.GetName()
			oldSize := seg.size

			seg.Write([]byte("test data"))
			if seg.size == oldSize {
				t.Error("Write() did not update size")
			}

			err = seg.Rotate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Rotate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				newName := seg.GetName()
				if newName == oldName {
					t.Error("Rotate() did not change file name")
				}

				if seg.size != 0 {
					t.Errorf("Rotate() size = %d, want 0", seg.size)
				}
			}
		})
	}
}

func TestFileSegment_GetName(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewFileSegment(&mocks.MockLogger{}, dir, 1024)
	if err != nil {
		t.Fatalf("NewFileSegment() error = %v", err)
	}
	defer seg.Close()

	name := seg.GetName()
	if name == "" {
		t.Error("GetName() returned empty string")
	}

	if !filepath.IsAbs(name) && !filepath.IsAbs(filepath.Join(dir, name)) {
		t.Error("GetName() returned invalid path")
	}
}

func TestFileSegment_CurrentOffset(t *testing.T) {
	tests := []struct {
		name       string
		writeData  []byte
		wantOffset int64
	}{
		{
			name:       "initial offset",
			writeData:  nil,
			wantOffset: 0,
		},
		{
			name:       "offset after write",
			writeData:  []byte("test"),
			wantOffset: 4,
		},
		{
			name:       "offset after multiple writes",
			writeData:  []byte("test data"),
			wantOffset: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			seg, err := NewFileSegment(&mocks.MockLogger{}, dir, 1024)
			if err != nil {
				t.Fatalf("NewFileSegment() error = %v", err)
			}
			defer seg.Close()

			if tt.writeData != nil {
				seg.Write(tt.writeData)
			}

			offset := seg.CurrentOffset()
			if offset != tt.wantOffset {
				t.Errorf("CurrentOffset() = %d, want %d", offset, tt.wantOffset)
			}
		})
	}
}

func TestFileSegment_Close(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewFileSegment(&mocks.MockLogger{}, dir, 1024)
	if err != nil {
		t.Fatalf("NewFileSegment() error = %v", err)
	}

	err = seg.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	err = seg.Close()
	if err == nil {
		t.Error("Close() on closed file should return error")
	}
}
