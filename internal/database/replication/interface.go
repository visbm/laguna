package replication

import (
	"context"
	"io"
	"laguna/internal/database/logs"
	"laguna/internal/database/wal"
	"laguna/internal/segment"
	"laguna/utils/concurrency"
)

type TCPClient interface {
	Send(ctx context.Context, req []byte) (io.Reader, error)
	Close()
}

type TCPLister interface {
	Listen(ctx context.Context)
	Close()
}

type LogReader interface {
	Read(r io.Reader) ([]*wal.Row, error)
	ReadStream(r io.Reader) concurrency.FutureRespWithErr[[]*wal.Row]
	ReadFrom(directory string, target string, firstOffset int64) ([]*wal.Row, error)
}

type LogWriter interface {
	Write([]*wal.Row) error
	WriteTo(rows []*wal.Row, t logs.Target) error
}

type SegmentManager interface {
	GetIndex(lsn uint64) (segment.IndexEntry, error)
}
