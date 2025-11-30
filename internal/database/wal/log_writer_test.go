package wal

import (
	"bytes"
	"laguna/internal/config"
	"laguna/internal/fs"
	"laguna/internal/mocks"
	"laguna/internal/query"
	"laguna/utils"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogWriter_Write_basicBatches(t *testing.T) {
	dir := t.TempDir()
	conf := config.WAL{
		Directory:      dir,
		MaxSegmentSize: 1024,
	}

	seg, err := fs.NewFileSegment(conf.Directory, conf.MaxSegmentSize.Int64())
	if err != nil {
		t.Fatalf("NewFileSegment failed: %v", err)
	}

	logger := &mocks.MockLogger{}
	lw := NewLogWriter(conf, logger, seg)

	batch := []*Row{
		{
			lsnID:    1,
			methodID: query.GetMethodID,
			args:     []string{"user"},
		},
		{
			lsnID:    2,
			methodID: query.SetMethodID,
			args:     []string{"user", "1"},
		},
		{
			lsnID:    3,
			methodID: query.DelMethodID,
			args:     []string{"user"},
		},
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

	seg, err := fs.NewFileSegment(conf.Directory, conf.MaxSegmentSize.Int64())
	require.NoError(t, err, "NewFileSegment failed")

	logger := &mocks.MockLogger{}
	lw := NewLogWriter(conf, logger, seg)

	rows := []*Row{
		{lsnID: 1, methodID: query.SetMethodID, args: []string{"a", "1"}},
		{lsnID: 2, methodID: query.SetMethodID, args: []string{"b", "2"}},
		{lsnID: 3, methodID: query.SetMethodID, args: []string{"c", "3"}},
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
	require.NoError(t, err, "read dir failed")

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
		MaxSegmentSize: utils.ByteSize(maxSegmentSize),
	}

	seg, err := fs.NewFileSegment(conf.Directory, conf.MaxSegmentSize.Int64())
	require.NoError(t, err, "NewFileSegment failed")

	logger := &mocks.MockLogger{}
	lw := NewLogWriter(conf, logger, seg)

	rows := []*Row{
		{lsnID: 1, methodID: query.SetMethodID, args: []string{"k1", "Hello1"}},
		{lsnID: 2, methodID: query.SetMethodID, args: []string{"k2", "Hello2"}},
		{lsnID: 3, methodID: query.SetMethodID, args: []string{"k3", "Hello3"}},
		{lsnID: 4, methodID: query.SetMethodID, args: []string{"k4", "Hello4"}},
		{lsnID: 5, methodID: query.SetMethodID, args: []string{"k5", "this entry is definitely longer than the configured max segment size and should be handled specially"}},
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
	require.NoError(t, err, "read dir failed")

	expectedFiles := runs * len(rows)
	require.Equal(t, expectedFiles, len(files), "unexpected number of WAL files")
}

func readAllLines(t *testing.T, dir string) []byte {
	t.Helper()
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}
	var lines []byte
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		path := filepath.Join(dir, f.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file %s failed: %v", path, err)
		}

		lines = append(lines, content...)
	}
	return lines
}
