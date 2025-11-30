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
	Write(rows []*Row) error
}

type Reader interface {
	Read() ([]*Row, error)
}

type WAL struct {
	disable bool

	log logger.Logger

	writer Writer
	reader Reader

	m          *sync.Mutex
	queryCh    chan *Row
	queryBuf   []*Row
	futureResp []utils.FutureResp[error]

	lsnGen *utils.Generator

	flushInterval  time.Duration  `yaml:"flush_interval"`
	flushBatchSize int64          `yaml:"flush_batch_size"`
	maxSegmentSize utils.ByteSize `yaml:"max_segment_size"`
	directory      string         `yaml:"directory"`
}

func NewWAL(c config.WAL, log logger.Logger, w Writer, r Reader) *WAL {
	log.Info("initializing WAL")
	if c.Disable {
		log.Info("disabling WAL")
		return &WAL{
			disable: true,
		}
	}

	wal := &WAL{
		log:    log,
		m:      &sync.Mutex{},
		writer: w,
		reader: r,

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

	wal.queryCh = make(chan *Row, wal.flushBatchSize)
	wal.queryBuf = make([]*Row, 0, wal.flushBatchSize)

	wal.futureResp = make([]utils.FutureResp[error], 0, wal.flushBatchSize)
	return wal
}

func (w *WAL) Start(ctx context.Context) {
	if w.disable {
		return
	}
	w.startInfileWALWorker(ctx)
}

func (w *WAL) Write(ctx context.Context, q query.Query) (utils.FutureResp[error], error) {
	fr := utils.NewFutureResp[error]()

	if ctx.Err() != nil {
		w.log.Error("context canceled")
		return fr, ctx.Err()
	}

	r := &Row{
		lsnID:    w.lsnGen.NextID(),
		methodID: q.MethodID(),
		args:     q.GetArgs(),
	}

	w.queryCh <- r

	utils.WithLock(w.m, func() {
		w.futureResp = append(w.futureResp, fr)
	})

	return fr, nil
}

func (w *WAL) startInfileWALWorker(ctx context.Context) {
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("context canceled")
			for {
				select {
				case q := <-w.queryCh:
					w.queryBuf = append(w.queryBuf, q)
				default:
					w.flush()
					return
				}
			}

		default:
		}

		select {
		case <-ctx.Done():
			w.log.Info("context canceled")
			for {
				select {
				case q := <-w.queryCh:
					w.queryBuf = append(w.queryBuf, q)
				default:
					w.flush()
					return
				}
			}

		case <-ticker.C:
			if len(w.queryBuf) > 0 {
				w.flush()
			}
		case q := <-w.queryCh:
			w.queryBuf = append(w.queryBuf, q)
			if len(w.queryBuf) >= int(w.flushBatchSize) {
				w.flush()
				ticker.Reset(w.flushInterval)
			}
		}
	}
}

func (w *WAL) flush() {
	err := w.writer.Write(w.queryBuf)
	if err != nil {
		w.log.Error("failed to flush WAL files", zap.Error(err))
	}

	w.queryBuf = w.queryBuf[:0]

	var resp []utils.FutureResp[error]

	utils.WithLock(w.m, func() {
		resp = make([]utils.FutureResp[error], len(w.futureResp))
		copy(resp, w.futureResp)
		w.futureResp = w.futureResp[:0]
	})

	for _, r := range resp {
		r.Put(err)
		r.Done()
	}
}

func (w *WAL) ReadWal() ([]query.Query, error) {
	val, err := w.reader.Read()
	if err != nil {
		w.log.Error("failed to read WAL files", zap.Error(err))
		return nil, err
	}
	resp := make([]query.Query, 0, len(val))

	for _, rec := range val {
		resp = append(resp, query.NewQuery(rec.methodID, rec.args))
	}

	return resp, nil
}
