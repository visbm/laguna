package logs

import (
	"bytes"
	"laguna/internal/config"
	"laguna/internal/database/wal"
	"laguna/internal/fs"
	"laguna/internal/mocks"
	"laguna/internal/query"
	"laguna/utils/data_type"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var mockLg = &mocks.MockLogger{}

func TestLogWriter_Write_basicBatches(t *testing.T) {
	dir := t.TempDir()
	conf := config.WAL{
		Directory:      dir,
		MaxSegmentSize: 1024,
	}

	seg, err := fs.NewFileSegment(mockLg, conf.Directory, conf.MaxSegmentSize.Int64())
	if err != nil {
		t.Fatalf("NewFileSegment failed: %v", err)
	}

	sm := fs.NewSegmentManager(mockLg, seg)

	lw := NewLogWriterWithTarget(mockLg, sm)

	batch := []*wal.Row{
		wal.NewRow(1, query.GetMethodID, []string{"user"}),
		wal.NewRow(2, query.SetMethodID, []string{"user", "1"}),
		wal.NewRow(3, query.DelMethodID, []string{"user"}),
	}

	var batchBytes []byte
	for _, b := range batch {
		rB, _ := b.Marshal()
		batchBytes = append(batchBytes, rB...)
	}

	if err := lw.Write(batch); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	got := readAllLines(t, dir)
	if len(got) != len(batchBytes) {
		t.Errorf("expected %d lines, got %d", len(batchBytes), len(got))
		return
	}
}

func TestLogWriter_Write_manyBatches_segmentRotation(t *testing.T) {
	dir := t.TempDir()
	const (
		runs           = 10
		maxSegmentSize = 64
	)

	conf := config.WAL{
		Directory:      dir,
		MaxSegmentSize: maxSegmentSize,
	}

	seg, err := fs.NewFileSegment(mockLg, conf.Directory, conf.MaxSegmentSize.Int64())
	if err != nil {
		t.Fatalf("NewFileSegment failed: %v", err)
	}

	sm := fs.NewSegmentManager(mockLg, seg)

	lw := NewLogWriterWithTarget(mockLg, sm)

	rows := []*wal.Row{
		wal.NewRow(1, query.SetMethodID, []string{"a", "1"}),
		wal.NewRow(2, query.SetMethodID, []string{"b", "2"}),
		wal.NewRow(3, query.SetMethodID, []string{"c", "3"}),
	}

	var wantPerBatch []byte
	for _, r := range rows {
		b, _ := r.Marshal()
		wantPerBatch = append(wantPerBatch, b...)
	}

	var want []byte
	for i := 0; i < runs; i++ {
		require.NoError(t, lw.Write(rows), "Write failed on run %d", i)
		want = append(want, wantPerBatch...)
	}

	got := readAllLines(t, dir)
	require.Equal(t, len(want), len(got), "unexpected total bytes written")

	files, err := os.ReadDir(dir)
	require.NoError(t, err, "readFrom dir failed")

	expectedFiles := 10
	require.Equal(t, expectedFiles, len(files), "unexpected number of WAL files")

	require.True(t, bytes.Equal(got, want), "written bytes do not match expected")
}

func TestLogWriter_Write_oversizedEntry(t *testing.T) {
	dir := t.TempDir()
	const (
		runs     = 5
		shortLen = 6
	)
	maxSegmentSize := runs*shortLen + runs

	conf := config.WAL{
		Directory:      dir,
		MaxSegmentSize: data_type.ByteSize(maxSegmentSize),
	}

	seg, err := fs.NewFileSegment(mockLg, conf.Directory, conf.MaxSegmentSize.Int64())
	if err != nil {
		t.Fatalf("NewFileSegment failed: %v", err)
	}

	sm := fs.NewSegmentManager(mockLg, seg)

	lw := NewLogWriterWithTarget(mockLg, sm)

	rows := []*wal.Row{
		wal.NewRow(1, query.SetMethodID, []string{"k1", "Hello1"}),
		wal.NewRow(2, query.SetMethodID, []string{"k2", "Hello2"}),
		wal.NewRow(3, query.SetMethodID, []string{"k3", "Hello3"}),
		wal.NewRow(4, query.SetMethodID, []string{"k4", "Hello4"}),
		wal.NewRow(5, query.SetMethodID, []string{"k5", "this entry is definitely longer than the configured max segment size and should be handled specially"}),
	}

	var wantPerBatch []byte
	for _, r := range rows {
		b, _ := r.Marshal()
		wantPerBatch = append(wantPerBatch, b...)
	}

	var want []byte
	for i := 0; i < runs; i++ {
		require.NoError(t, lw.Write(rows), "Write failed on run %d", i)
		want = append(want, wantPerBatch...)
	}

	got := readAllLines(t, dir)
	require.True(t, bytes.Equal(got, want), "written bytes do not match expected")

	files, err := os.ReadDir(dir)
	require.NoError(t, err, "readFrom dir failed")

	expectedFiles := runs * len(rows)
	require.Equal(t, expectedFiles, len(files), "unexpected number of WAL files")
}

func readAllLines(t *testing.T, dir string) []byte {
	t.Helper()
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readFrom dir failed: %v", err)
	}
	var lines []byte
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		path := filepath.Join(dir, f.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("readFrom file %s failed: %v", path, err)
		}

		lines = append(lines, content...)
	}
	return lines
}
