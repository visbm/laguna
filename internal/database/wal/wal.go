package wal

import (
	"context"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/query"
	"laguna/utils"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultFlushInterval  = 10 * time.Millisecond
	defaultFlushBatchSize = 100
	defaultMaxSegmentSize = 1000000
	defaultDirectory      = "/data/laguna/wal"
)

type Writer interface {
	Write(ctx context.Context, q [][]byte) error
}

type WAL struct {
	log logger.Logger

	writer   Writer
	m        *sync.Mutex
	waitCh   []chan error
	queryCh  chan []byte
	queryBuf [][]byte

	lsnGen *utils.Generator

	flushInterval  time.Duration  `yaml:"flush_interval"`
	flushBatchSize int64          `yaml:"flush_batch_size"`
	maxSegmentSize utils.ByteSize `yaml:"max_segment_size"`
	directory      string         `yaml:"directory"`
}

func NewWAL(c config.WAL, log logger.Logger, w Writer) *WAL {
	log.Info("initializing WAL")

	wal := &WAL{
		log:    log,
		m:      &sync.Mutex{},
		writer: w,

		flushInterval:  c.FlushInterval,
		flushBatchSize: c.FlushBatchSize,
		maxSegmentSize: c.MaxSegmentSize,
		directory:      c.Directory,
		lsnGen:         utils.NewIDGenerator(),
	}
	if wal.flushInterval == 0 {
		wal.flushInterval = defaultFlushInterval
	}
	if wal.flushBatchSize == 0 {
		wal.flushBatchSize = defaultFlushBatchSize
	}
	if wal.maxSegmentSize == 0 {
		wal.maxSegmentSize = defaultMaxSegmentSize
	}
	if wal.directory == "" {
		wal.directory = defaultDirectory
	}

	wal.queryCh = make(chan []byte, wal.flushBatchSize)
	wal.waitCh = make([]chan error, 0, wal.flushBatchSize)
	wal.queryBuf = make([][]byte, 0, wal.flushBatchSize)
	return wal
}

func (w *WAL) Start(ctx context.Context) {
	w.startInfileWALWorker(ctx)
}

func (w *WAL) Write(ctx context.Context, q query.Query) (<-chan error, error) {
	if ctx.Err() != nil {
		w.log.Error("context canceled")
		return nil, ctx.Err()
	}

	newLsnID := w.lsnGen.NextID()
	r := &row{
		lsnID:    newLsnID,
		methodID: q.MethodID(),
		args:     q.GetArs(),
	}
	body := r.Marshal()

	w.queryCh <- body

	ch := make(chan error, 1)
	utils.WithLock(w.m, func() {
		w.waitCh = append(w.waitCh, ch)
	})

	return ch, nil
}

func (w *WAL) startInfileWALWorker(ctx context.Context) {
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			for {
				select {
				case q := <-w.queryCh:
					w.queryBuf = append(w.queryBuf, q)
				default:
					w.flush(ctx)
					return
				}
			}

		case <-ticker.C:
			if len(w.queryBuf) > 0 {
				w.flush(ctx)
			}
		case q := <-w.queryCh:
			w.queryBuf = append(w.queryBuf, q)
			if len(w.queryBuf) >= int(w.flushBatchSize) {
				w.flush(ctx)
				ticker.Reset(w.flushInterval)
			}
		}
	}
}

func (w *WAL) flush(ctx context.Context) {
	err := w.writer.Write(ctx, w.queryBuf)
	if err != nil {
		w.log.Error("failed to flush WAL files", zap.Error(err))
	}

	w.queryBuf = w.queryBuf[:0]

	var waitChCopy []chan error

	utils.WithLock(w.m, func() {
		waitChCopy = make([]chan error, len(w.waitCh))
		copy(waitChCopy, w.waitCh)
		w.waitCh = w.waitCh[:0]
	})

	for _, ch := range waitChCopy {
		ch <- err
		close(ch)
	}
}
