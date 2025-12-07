package logs

import (
	"laguna/internal/mocks"
	"laguna/internal/query"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogReader_Read(t *testing.T) {
	dir := t.TempDir()

	row1 := &Row{lsnID: 1, methodID: query.SetMethodID, args: []string{"user", "foo"}}
	row2 := &Row{lsnID: 2, methodID: query.GetMethodID, args: []string{"user"}}

	data1, _ := row1.Marshal()
	data2, _ := row2.Marshal()

	err := os.WriteFile(filepath.Join(dir, "wal1.log"), append(data1, data2...), 0644)
	if err != nil {
		t.Fatalf("failed to write wal1: %v", err)
	}

	// write wrong data
	err = os.WriteFile(filepath.Join(dir, "wal2.log"), []byte{0x00, 0x01, 0x02, 0x03}, 0644)
	if err != nil {
		t.Fatalf("failed to write wal2: %v", err)
	}

	reader := NewLogReader(&mocks.MockLogger{})
	rows, err := reader.ReadFromFiles(dir)
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
