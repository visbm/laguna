package replication

import (
	"context"
	"io"
	"laguna/internal/database/logs"
	"laguna/internal/database/wal"
	"laguna/utils/concurrency"
)

type TCPClient interface {
	Send(ctx context.Context, req []byte) ([]byte, error)
}

type LogReader interface {
	Read(r io.Reader) ([]*wal.Row, error)
	ReadStream(r io.Reader) concurrency.FutureRespWithErr[[]*wal.Row]
	ReadFrom(directory string, target string) ([]*wal.Row, error)
}

type LogWriter interface {
	Write([]*wal.Row) error
	WriteTo(rows []*wal.Row, t logs.Target) error
}
