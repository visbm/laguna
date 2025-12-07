package fs

import (
	"testing"
)

func TestIndexManagerMutex_Add(t *testing.T) {
	tests := []struct {
		name     string
		lsn      uint64
		fileName string
		offset   int64
	}{
		{
			name:     "add single entry",
			lsn:      1,
			fileName: "file1.log",
			offset:   100,
		},
		{
			name:     "add with zero lsn",
			lsn:      0,
			fileName: "file0.log",
			offset:   0,
		},
		{
			name:     "add with large lsn",
			lsn:      18446744073709551615,
			fileName: "file_large.log",
			offset:   999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewIndexManagerMutex()
			im.Add(tt.lsn, tt.fileName, tt.offset)

			entry, err := im.Get(tt.lsn)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if entry.FileName != tt.fileName {
				t.Errorf("Get() FileName = %q, want %q", entry.FileName, tt.fileName)
			}

			if entry.Offset != tt.offset {
				t.Errorf("Get() Offset = %d, want %d", entry.Offset, tt.offset)
			}
		})
	}
}

func TestIndexManagerMutex_Get(t *testing.T) {
	tests := []struct {
		name    string
		lsn     uint64
		wantErr bool
	}{
		{
			name:    "get existing entry",
			lsn:     1,
			wantErr: false,
		},
		{
			name:    "get non-existing entry",
			lsn:     999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewIndexManagerMutex()
			if !tt.wantErr {
				im.Add(tt.lsn, "test.log", 100)
			}

			_, err := im.Get(tt.lsn)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIndexManagerMutex_AddBatch(t *testing.T) {
	tests := []struct {
		name    string
		entries map[uint64]IndexEntry
	}{
		{
			name: "add single batch entry",
			entries: map[uint64]IndexEntry{
				1: {FileName: "file1.log", Offset: 100},
			},
		},
		{
			name: "add multiple batch entries",
			entries: map[uint64]IndexEntry{
				1: {FileName: "file1.log", Offset: 100},
				2: {FileName: "file2.log", Offset: 200},
				3: {FileName: "file3.log", Offset: 300},
			},
		},
		{
			name:    "add empty batch",
			entries: map[uint64]IndexEntry{},
		},
		{
			name: "add large batch",
			entries: func() map[uint64]IndexEntry {
				entries := make(map[uint64]IndexEntry, 100)
				for i := uint64(0); i < 100; i++ {
					entries[i] = IndexEntry{FileName: "file.log", Offset: int64(i * 100)}
				}
				return entries
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewIndexManagerMutex()
			im.AddBatch(tt.entries)

			for lsn, wantEntry := range tt.entries {
				gotEntry, err := im.Get(lsn)
				if err != nil {
					t.Errorf("Get(%d) error = %v", lsn, err)
					continue
				}

				if gotEntry.FileName != wantEntry.FileName {
					t.Errorf("Get(%d) FileName = %q, want %q", lsn, gotEntry.FileName, wantEntry.FileName)
				}

				if gotEntry.Offset != wantEntry.Offset {
					t.Errorf("Get(%d) Offset = %d, want %d", lsn, gotEntry.Offset, wantEntry.Offset)
				}
			}
		})
	}
}

func TestIndexManagerMutex_Overwrite(t *testing.T) {
	im := NewIndexManagerMutex()
	im.Add(1, "file1.log", 100)

	entry, err := im.Get(1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if entry.FileName != "file1.log" {
		t.Errorf("Get() FileName = %q, want %q", entry.FileName, "file1.log")
	}

	im.Add(1, "file2.log", 200)

	entry, err = im.Get(1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if entry.FileName != "file2.log" {
		t.Errorf("Get() FileName after overwrite = %q, want %q", entry.FileName, "file2.log")
	}

	if entry.Offset != 200 {
		t.Errorf("Get() Offset after overwrite = %d, want %d", entry.Offset, 200)
	}
}

func TestIndexManagerCOW_Add(t *testing.T) {
	tests := []struct {
		name     string
		lsn      uint64
		fileName string
		offset   int64
	}{
		{
			name:     "add single entry",
			lsn:      1,
			fileName: "file1.log",
			offset:   100,
		},
		{
			name:     "add with zero lsn",
			lsn:      0,
			fileName: "file0.log",
			offset:   0,
		},
		{
			name:     "add multiple entries",
			lsn:      5,
			fileName: "file5.log",
			offset:   500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewIndexManagerCOW()
			im.Add(tt.lsn, tt.fileName, tt.offset)

			entry, err := im.Get(tt.lsn)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if entry.FileName != tt.fileName {
				t.Errorf("Get() FileName = %q, want %q", entry.FileName, tt.fileName)
			}

			if entry.Offset != tt.offset {
				t.Errorf("Get() Offset = %d, want %d", entry.Offset, tt.offset)
			}
		})
	}
}

func TestIndexManagerCOW_Get(t *testing.T) {
	tests := []struct {
		name    string
		lsn     uint64
		wantErr bool
	}{
		{
			name:    "get existing entry",
			lsn:     1,
			wantErr: false,
		},
		{
			name:    "get non-existing entry",
			lsn:     999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewIndexManagerCOW()
			if !tt.wantErr {
				im.Add(tt.lsn, "test.log", 100)
			}

			_, err := im.Get(tt.lsn)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIndexManagerCOW_AddBatch(t *testing.T) {
	tests := []struct {
		name    string
		entries map[uint64]IndexEntry
	}{
		{
			name: "add single batch entry",
			entries: map[uint64]IndexEntry{
				1: {FileName: "file1.log", Offset: 100},
			},
		},
		{
			name: "add multiple batch entries",
			entries: map[uint64]IndexEntry{
				1: {FileName: "file1.log", Offset: 100},
				2: {FileName: "file2.log", Offset: 200},
				3: {FileName: "file3.log", Offset: 300},
			},
		},
		{
			name:    "add empty batch",
			entries: map[uint64]IndexEntry{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im := NewIndexManagerCOW()
			im.AddBatch(tt.entries)

			for lsn, wantEntry := range tt.entries {
				gotEntry, err := im.Get(lsn)
				if err != nil {
					t.Errorf("Get(%d) error = %v", lsn, err)
					continue
				}

				if gotEntry.FileName != wantEntry.FileName {
					t.Errorf("Get(%d) FileName = %q, want %q", lsn, gotEntry.FileName, wantEntry.FileName)
				}

				if gotEntry.Offset != wantEntry.Offset {
					t.Errorf("Get(%d) Offset = %d, want %d", lsn, gotEntry.Offset, wantEntry.Offset)
				}
			}
		})
	}
}

func TestIndexManagerCOW_Overwrite(t *testing.T) {
	im := NewIndexManagerCOW()
	im.Add(1, "file1.log", 100)

	entry, err := im.Get(1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if entry.FileName != "file1.log" {
		t.Errorf("Get() FileName = %q, want %q", entry.FileName, "file1.log")
	}

	im.Add(1, "file2.log", 200)

	entry, err = im.Get(1)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if entry.FileName != "file2.log" {
		t.Errorf("Get() FileName after overwrite = %q, want %q", entry.FileName, "file2.log")
	}

	if entry.Offset != 200 {
		t.Errorf("Get() Offset after overwrite = %d, want %d", entry.Offset, 200)
	}
}

func TestIndexManagerCOW_AddThenBatch(t *testing.T) {
	im := NewIndexManagerCOW()
	im.Add(1, "file1.log", 100)
	im.AddBatch(map[uint64]IndexEntry{
		2: {FileName: "file2.log", Offset: 200},
		3: {FileName: "file3.log", Offset: 300},
	})

	entry1, err := im.Get(1)
	if err != nil {
		t.Fatalf("Get(1) error = %v", err)
	}
	if entry1.FileName != "file1.log" {
		t.Errorf("Get(1) FileName = %q, want %q", entry1.FileName, "file1.log")
	}

	entry2, err := im.Get(2)
	if err != nil {
		t.Fatalf("Get(2) error = %v", err)
	}
	if entry2.FileName != "file2.log" {
		t.Errorf("Get(2) FileName = %q, want %q", entry2.FileName, "file2.log")
	}
}
