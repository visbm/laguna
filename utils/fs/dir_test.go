package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDir(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(string) error
		wantErr bool
		wantLen int
	}{
		{
			name: "read empty directory",
			setup: func(dir string) error {
				return nil
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "read directory with files",
			setup: func(dir string) error {
				for i := 0; i < 3; i++ {
					file, err := os.Create(filepath.Join(dir, "file"+string(rune('0'+i))+".log"))
					if err != nil {
						return err
					}
					_ = file.Close()
				}
				return nil
			},
			wantErr: false,
			wantLen: 3,
		},
		{
			name: "create non-existent directory automatically",
			setup: func(dir string) error {
				return nil
			},
			wantErr: false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.name == "create non-existent directory automatically" {
				dir = filepath.Join(dir, "nonexistent")
			} else {
				if err := tt.setup(dir); err != nil {
					t.Fatalf("setup error: %v", err)
				}
			}

			names, err := ReadDir(dir)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(names) != tt.wantLen {
				t.Errorf("ReadDir() len = %d, want %d", len(names), tt.wantLen)
			}
		})
	}
}

func TestOpenFile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(string) (string, error)
		wantErr bool
	}{
		{
			name: "create new file",
			setup: func(dir string) (string, error) {
				return filepath.Join(dir, "test.log"), nil
			},
			wantErr: false,
		},
		{
			name: "create file in nested directory",
			setup: func(dir string) (string, error) {
				nestedDir := filepath.Join(dir, "nested", "subdir")
				return filepath.Join(nestedDir, "test.log"), nil
			},
			wantErr: false,
		},
		{
			name: "open existing file",
			setup: func(dir string) (string, error) {
				path := filepath.Join(dir, "existing.log")
				file, err := os.Create(path)
				if err != nil {
					return "", err
				}
				_ = file.Close()
				return path, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			filePath, err := tt.setup(dir)
			if err != nil {
				t.Fatalf("setup error: %v", err)
			}

			file, err := OpenFile(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("OpenFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if file == nil {
					t.Error("OpenFile() returned nil file")
					return
				}
				_ = file.Close()

				if _, err := os.Stat(filePath); err != nil {
					t.Errorf("OpenFile() file does not exist: %v", err)
				}
			}
		})
	}
}

func TestReadDirsForm(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		target    string
		wantLen   int
		wantErr   bool
		wantFirst string
	}{
		{
			name:    "empty directory",
			files:   []string{},
			target:  "file.log",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:      "all files after target",
			files:     []string{"file1.log", "file2.log", "file3.log"},
			target:    "file1.log",
			wantLen:   3,
			wantErr:   false,
			wantFirst: "file1.log",
		},
		{
			name:      "target at beginning",
			files:     []string{"file1.log", "file2.log", "file3.log"},
			target:    "",
			wantLen:   3,
			wantErr:   false,
			wantFirst: "file1.log",
		},
		{
			name:    "target at end",
			files:   []string{"file1.log", "file2.log", "file3.log"},
			target:  "file3.log",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:      "target in middle",
			files:     []string{"a.log", "b.log", "c.log", "d.log"},
			target:    "b.log",
			wantLen:   3,
			wantErr:   false,
			wantFirst: "b.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			for _, fileName := range tt.files {
				file, err := os.Create(filepath.Join(dir, fileName))
				if err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
				_ = file.Close()
			}

			result, err := ReadDirsFrom(dir, tt.target)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadDirsFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(result) != tt.wantLen {
					t.Errorf("ReadDirsFrom() len = %d, want %d", len(result), tt.wantLen)
				}

				if tt.wantFirst != "" && len(result) > 0 && result[0] != tt.wantFirst {
					t.Errorf("ReadDirsFrom() first = %q, want %q", result[0], tt.wantFirst)
				}
			}
		})
	}
}

func TestBinarySearch(t *testing.T) {
	tests := []struct {
		name   string
		array  []string
		target string
		want   int
	}{
		{
			name:   "empty array",
			array:  []string{},
			target: "b",
			want:   -1,
		},
		{
			name:   "target before all",
			array:  []string{"c", "d", "e"},
			target: "a",
			want:   -1,
		},
		{
			name:   "target after all",
			array:  []string{"a", "b", "c"},
			target: "d",
			want:   -1,
		},
		{
			name:   "target in middle",
			array:  []string{"a", "b", "c", "d", "e"},
			target: "c",
			want:   2,
		},
		{
			name:   "target at beginning",
			array:  []string{"a", "b", "c"},
			target: "a",
			want:   0,
		},
		{
			name:   "target at end",
			array:  []string{"a", "b", "c"},
			target: "c",
			want:   2,
		},
		{
			name:   "single element before",
			array:  []string{"b"},
			target: "a",
			want:   -1,
		},
		{
			name:   "single element after",
			array:  []string{"a"},
			target: "b",
			want:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := binarySearch(tt.array, tt.target)
			if got != tt.want {
				t.Errorf("binarySearch() = %d, want %d", got, tt.want)
			}
		})
	}
}
