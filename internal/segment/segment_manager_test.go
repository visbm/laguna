package segment

/*

func TestSegmentManager_Write(t *testing.T) {
	tests := []struct {
		name    string
		rows    []*wal.Row
		maxSize int64
		wantErr bool
	}{
		{
			name: "write single row",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key", "value"}),
			},
			maxSize: 1024,
			wantErr: false,
		},
		{
			name: "write multiple rows",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key1", "value1"}),
				wal.NewRow(2, query.DelMethodID, []string{"key2"}),
				wal.NewRow(3, query.SetMethodID, []string{"key3", "value3"}),
			},
			maxSize: 1024,
			wantErr: false,
		},
		{
			name:    "write empty rows",
			rows:    []*wal.Row{},
			maxSize: 1024,
			wantErr: false,
		},
		{
			name: "write rows requiring rotation",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key1", "value1"}),
				wal.NewRow(2, query.SetMethodID, []string{"key2", "value2"}),
			},
			maxSize: 50,
			wantErr: false,
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

			sm := NewSegmentManager(&mocks.MockLogger{}, seg)

			err = sm.Write(tt.rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSegmentManager_processBatches(t *testing.T) {
	tests := []struct {
		name      string
		batch     [][]byte
		rows      []*wal.Row
		maxSize   int64
		wantErr   bool
		wantFiles int
	}{
		{
			name: "single batch fits",
			batch: [][]byte{
				[]byte("test1"),
				[]byte("test2"),
			},
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key1", "value1"}),
				wal.NewRow(2, query.SetMethodID, []string{"key2", "value2"}),
			},
			maxSize:   1024,
			wantErr:   false,
			wantFiles: 1,
		},
		{
			name: "batch requires rotation",
			batch: [][]byte{
				[]byte("large data that exceeds max size"),
				[]byte("more data"),
			},
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key1", "value1"}),
				wal.NewRow(2, query.SetMethodID, []string{"key2", "value2"}),
			},
			maxSize:   20,
			wantErr:   false,
			wantFiles: 2,
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

			sm := NewSegmentManager(&mocks.MockLogger{}, seg)

			err = sm.processBatches(tt.batch, tt.rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("processBatches() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				files, err := ReadDir(dir)
				if err != nil {
					t.Fatalf("ReadDir() error = %v", err)
				}

				if len(files) < tt.wantFiles {
					t.Errorf("processBatches() created %d files, want at least %d", len(files), tt.wantFiles)
				}
			}
		})
	}
}

func TestSegmentManager_flatten(t *testing.T) {
	tests := []struct {
		name    string
		batch   [][]byte
		bufSize int
		wantLen int
	}{
		{
			name:    "flatten single element",
			batch:   [][]byte{[]byte("test")},
			bufSize: 10,
			wantLen: 4,
		},
		{
			name:    "flatten multiple elements",
			batch:   [][]byte{[]byte("test1"), []byte("test2"), []byte("test3")},
			bufSize: 20,
			wantLen: 15,
		},
		{
			name:    "flatten empty batch",
			batch:   [][]byte{},
			bufSize: 10,
			wantLen: 0,
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

			sm := NewSegmentManager(&mocks.MockLogger{}, seg)

			result := sm.flatten(tt.batch, tt.bufSize)
			if len(result) != tt.wantLen {
				t.Errorf("flatten() len = %d, want %d", len(result), tt.wantLen)
			}

			offset := 0
			for i, b := range tt.batch {
				if offset+len(b) > len(result) {
					t.Errorf("flatten() batch[%d] not found in result", i)
					break
				}

				got := result[offset : offset+len(b)]
				if string(got) != string(b) {
					t.Errorf("flatten() batch[%d] = %q, want %q", i, got, b)
				}
				offset += len(b)
			}
		})
	}
}

func TestSegmentManager_createButch(t *testing.T) {
	tests := []struct {
		name    string
		rows    []*wal.Row
		wantErr bool
		wantLen int
	}{
		{
			name: "create batch from single row",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key", "value"}),
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "create batch from multiple rows",
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key1", "value1"}),
				wal.NewRow(2, query.DelMethodID, []string{"key2"}),
				wal.NewRow(3, query.SetMethodID, []string{"key3", "value3"}),
			},
			wantErr: false,
			wantLen: 3,
		},
		{
			name:    "create batch from empty rows",
			rows:    []*wal.Row{},
			wantErr: false,
			wantLen: 0,
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

			sm := NewSegmentManager(&mocks.MockLogger{}, seg)

			batch, err := sm.createButch(tt.rows)
			if (err != nil) != tt.wantErr {
				t.Errorf("createButch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(batch) != tt.wantLen {
					t.Errorf("createButch() len = %d, want %d", len(batch), tt.wantLen)
				}

				for i, row := range tt.rows {
					if i >= len(batch) {
						break
					}

					expectedData, _ := row.Marshal()
					if len(batch[i]) != len(expectedData) {
						t.Errorf("createButch() batch[%d] len = %d, want %d", i, len(batch[i]), len(expectedData))
					}
				}
			}
		})
	}
}

func TestSegmentManager_writeBatch(t *testing.T) {
	tests := []struct {
		name    string
		batch   [][]byte
		rows    []*wal.Row
		bufSize int
		wantErr bool
	}{
		{
			name: "write single batch",
			batch: [][]byte{
				[]byte("test"),
			},
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key", "value"}),
			},
			bufSize: 10,
			wantErr: false,
		},
		{
			name: "write multiple batches",
			batch: [][]byte{
				[]byte("test1"),
				[]byte("test2"),
			},
			rows: []*wal.Row{
				wal.NewRow(1, query.SetMethodID, []string{"key1", "value1"}),
				wal.NewRow(2, query.SetMethodID, []string{"key2", "value2"}),
			},
			bufSize: 20,
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

			sm := NewSegmentManager(&mocks.MockLogger{}, seg)

			initialOffset := seg.CurrentOffset()
			err = sm.writeBatch(tt.batch, tt.rows, tt.bufSize)

			if (err != nil) != tt.wantErr {
				t.Errorf("writeBatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				newOffset := seg.CurrentOffset()
				if newOffset <= initialOffset {
					t.Errorf("writeBatch() offset did not increase: %d -> %d", initialOffset, newOffset)
				}
			}
		})
	}
}

func TestSegmentManager_writeInSeg(t *testing.T) {
	tests := []struct {
		name    string
		batch   []byte
		wantErr bool
	}{
		{
			name:    "write non-empty batch",
			batch:   []byte("test data"),
			wantErr: false,
		},
		{
			name:    "write empty batch",
			batch:   []byte{},
			wantErr: false,
		},
		{
			name:    "write large batch",
			batch:   make([]byte, 1000),
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

			sm := NewSegmentManager(&mocks.MockLogger{}, seg)

			initialSize := seg.size
			err = sm.writeInSeg(tt.batch)

			if (err != nil) != tt.wantErr {
				t.Errorf("writeInSeg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(tt.batch) > 0 {
				expectedSize := initialSize + int64(len(tt.batch))
				if seg.size != expectedSize {
					t.Errorf("writeInSeg() size = %d, want %d", seg.size, expectedSize)
				}
			}
		})
	}
}

func TestNewSegmentManager(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewFileSegment(&mocks.MockLogger{}, dir, 1024)
	if err != nil {
		t.Fatalf("NewFileSegment() error = %v", err)
	}
	defer seg.Close()

	sm := NewSegmentManager(&mocks.MockLogger{}, seg)

	if sm.curSeg != seg {
		t.Error("NewSegmentManager() curSeg not set correctly")
	}

	if sm.im == nil {
		t.Error("NewSegmentManager() ind is nil")
	}
}

func TestSegmentManager_Rotation(t *testing.T) {
	dir := t.TempDir()
	seg, err := NewFileSegment(&mocks.MockLogger{}, dir, 50)
	if err != nil {
		t.Fatalf("NewFileSegment() error = %v", err)
	}
	defer seg.Close()

	sm := NewSegmentManager(&mocks.MockLogger{}, seg)

	rows := []*wal.Row{
		wal.NewRow(1, query.SetMethodID, []string{"key1", "very long value that will cause rotation"}),
		wal.NewRow(2, query.SetMethodID, []string{"key2", "another long value"}),
		wal.NewRow(3, query.SetMethodID, []string{"key3", "third long value"}),
	}

	err = sm.Write(rows)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	files, err := ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	if len(files) < 2 {
		t.Errorf("Write() created %d files, expected at least 2 due to rotation", len(files))
	}
}
*/
