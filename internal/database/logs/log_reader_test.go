package logs

import (
	"laguna/internal/database/wal"
	"laguna/internal/mocks"
	"laguna/internal/query"
	"laguna/internal/segment"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockSegmentManager struct {
	files map[string][]byte
}

func (m *mockSegmentManager) ReadSegment(name string) ([]byte, error) {
	return m.files[name], nil
}

func (m *mockSegmentManager) AddBatchIndex(entries map[uint64]segment.IndexEntry) {
	// Mock implementation
}

func TestLogReader_Read(t *testing.T) {
	dir := t.TempDir()

	row1 := wal.NewRow(1, query.SetMethodID, []string{"user", "foo"})
	row2 := wal.NewRow(2, query.GetMethodID, []string{"user"})

	data1, _ := row1.Marshal()
	data2, _ := row2.Marshal()
	fileData := append(data1, data2...)

	filePath := filepath.Join(dir, "wal1.log")
	err := os.WriteFile(filePath, fileData, 0644)
	if err != nil {
		t.Fatalf("failed to write wal1: %v", err)
	}

	mockSM := &mockSegmentManager{
		files: map[string][]byte{
			filePath: fileData,
		},
	}

	reader := NewLogReader(mockSM, &mocks.MockLogger{})
	rows, err := reader.ReadFrom(dir, "wal1.log", 0)
	if err != nil {
		t.Fatalf("readFrom failed: %v", err)
	}

	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}

	if !assert.Equal(t, row1, rows[0]) {
		t.Errorf("unexpected row1: %+v", rows[0])
	}
	if !assert.Equal(t, row2, rows[1]) {
		t.Errorf("unexpected row2: %+v", rows[1])
	}
}
