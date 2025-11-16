package wal

import (
	"bytes"
	"laguna/internal/config"
	"laguna/internal/fs"
	"laguna/internal/mocks"
	"os"
	"path/filepath"
	"testing"
)

func readAllLines(t *testing.T, dir string) [][]byte {
	t.Helper()
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}
	var lines [][]byte
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		path := filepath.Join(dir, f.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file %s failed: %v", path, err)
		}
		parts := bytes.Split(content, []byte("\n"))
		// drop trailing empty element if file ends with newline
		end := len(parts)
		if end > 0 && len(parts[end-1]) == 0 {
			end--
		}
		for _, p := range parts[:end] {
			// skip empty lines just in case
			if len(p) == 0 {
				continue
			}
			// copy to avoid holding reference
			cpy := make([]byte, len(p))
			copy(cpy, p)
			lines = append(lines, cpy)
		}
	}
	return lines
}

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

	batch := [][]byte{
		[]byte("hello world"),
		[]byte("golang"),
		[]byte("laguna"),
		[]byte("infile writer"),
	}

	if err := lw.Write(batch); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	lines := readAllLines(t, dir)
	if len(lines) != len(batch) {
		t.Fatalf("expected %d lines, got %d", len(batch), len(lines))
	}

	// check all messages
	for _, want := range batch {
		found := false
		for _, got := range lines {
			if bytes.Equal(got, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected line %q not found in WAL files", string(want))
		}
	}
}

func TestLogWriter_Write_manyBatches_segmentRotation(t *testing.T) {
	dir := t.TempDir()
	conf := config.WAL{
		Directory:      dir,
		MaxSegmentSize: 64,
	}

	seg, err := fs.NewFileSegment(conf.Directory, conf.MaxSegmentSize.Int64())
	if err != nil {
		t.Fatalf("NewFileSegment failed: %v", err)
	}

	logger := &mocks.MockLogger{}
	lw := NewLogWriter(conf, logger, seg)

	perBatch := [][]byte{
		[]byte("Hello World"),
		[]byte("Hello World"),
		[]byte("Hello World"),
	}
	const runs = 10

	totalExpected := 0
	for i := 0; i < runs; i++ {
		if err := lw.Write(perBatch); err != nil {
			t.Fatalf("Write failed on run %d: %v", i, err)
		}
		totalExpected += len(perBatch)
	}

	lines := readAllLines(t, dir)
	if len(lines) != totalExpected {
		t.Fatalf("expected total %d lines, got %d", totalExpected, len(lines))
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}
	if len(files) <= 1 {
		t.Fatalf("expected more than 1 WAL file after multiple writes, got %d", len(files))
	}
}

func TestLogWriter_Write_oversizedEntry(t *testing.T) {
	dir := t.TempDir()
	const runs = 5
	const shortLen = 6

	conf := config.WAL{
		Directory:      dir,
		MaxSegmentSize: runs*shortLen + runs*sepLen,
	}

	seg, err := fs.NewFileSegment(conf.Directory, conf.MaxSegmentSize.Int64())
	if err != nil {
		t.Fatalf("NewFileSegment failed: %v", err)
	}
	logger := &mocks.MockLogger{}
	lw := NewLogWriter(conf, logger, seg)

	data := [][]byte{
		[]byte("Hello1"),
		[]byte("Hello2"),
		[]byte("Hello3"),
		[]byte("Hello4"),
		[]byte("this entry is definitely longer than the configured max segment size and should be handled specially"),
	}

	totalExpected := 0
	for i := 0; i < runs; i++ {
		if err := lw.Write(data); err != nil {
			t.Fatalf("Write failed on run %d: %v", i, err)
		}
		totalExpected += len(data)
	}

	lines := readAllLines(t, dir)
	if len(lines) != totalExpected {
		t.Fatalf("expected total %d lines, got %d", totalExpected, len(lines))
	}

	longFound := false
	for _, l := range lines {
		if bytes.HasPrefix(l, []byte("this entry is definitely")) {
			longFound = true
			break
		}
	}
	if !longFound {
		t.Errorf("oversized entry not found in WAL files")
	}
}
