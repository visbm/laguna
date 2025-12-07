package wal

import (
	"context"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/query"
	"laguna/utils/concurrency"
	"laguna/utils/data_type"
	"laguna/utils/id_generator"
	"laguna/utils/retry"
	"sync"
	"time"
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
	ReadFromFiles(directory string) ([]*Row, error)
	ReadFromFilesStream(directory string) concurrency.FutureRespWithErr[[]*Row]
}

type WAL struct {
	enable bool

	log logger.Logger

	writer Writer
	reader Reader

	m          *sync.Mutex
	queryCh    chan *Row
	queryBuf   []*Row
	futureResp []concurrency.FutureResp[error]

	lsnGen *id_generator.Generator

	flushInterval  time.Duration      `yaml:"flush_interval"`
	flushBatchSize int64              `yaml:"flush_batch_size"`
	maxSegmentSize data_type.ByteSize `yaml:"max_segment_size"`
	directory      string             `yaml:"directory"`
}

func NewWAL(c config.WAL, log logger.Logger, w Writer, r Reader) *WAL {
	log.Info("initializing WAL")
	if !c.Enable {
		log.Info("disabling WAL")
		return &WAL{
			enable: false,
		}
	}

	wal := &WAL{
		enable: true,
		log:    log,
		m:      &sync.Mutex{},
		writer: w,
		reader: r,

		flushInterval:  c.FlushInterval,
		flushBatchSize: c.FlushBatchSize,
		maxSegmentSize: c.MaxSegmentSize,
		directory:      c.Directory,
		lsnGen:         id_generator.NewIDGenerator(),
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

	wal.futureResp = make([]concurrency.FutureResp[error], 0, wal.flushBatchSize)
	return wal
}

func (w *WAL) Start(ctx context.Context) {
	if !w.enable {
		return
	}
	go w.startInfileWALWorker(ctx)
}

func (w *WAL) Write(ctx context.Context, q query.Query) (concurrency.FutureResp[error], error) {
	if !w.enable {
		return concurrency.FutureResp[error]{}, nil
	}

	fr := concurrency.NewFutureResp[error]()

	if ctx.Err() != nil {
		w.log.Error("context canceled")
		return fr, ctx.Err()
	}

	r := NewRow(w.lsnGen.NextID(), q.MethodID(), q.GetArgs())

	w.queryCh <- r

	concurrency.WithLock(w.m, func() {
		w.futureResp = append(w.futureResp, fr)
	})

	return fr, nil
}

func (w *WAL) Restore() ([]query.Query, error) {
	val, err := w.reader.ReadFromFiles(w.directory)
	if err != nil {
		w.log.Error("failed to read WAL files", logger.Error(err))
		return nil, err
	}

	resp := make([]query.Query, 0, len(val))

	var lastLSNID uint64
	for _, rec := range val {
		if rec.GetLsnID() > lastLSNID {
			lastLSNID = rec.GetLsnID()
		}
		resp = append(resp, query.NewQuery(rec.GetMethodID(), rec.GetArgs()))
	}

	w.lsnGen = id_generator.NewIDGeneratorWithStart(lastLSNID)

	return resp, nil
}

func (w *WAL) RestoreStream() (concurrency.FutureRespWithErr[[]query.Query], error) {
	resp := concurrency.NewFutureRespWithErr[[]query.Query]()

	rowsStream := w.reader.ReadFromFilesStream(w.directory)

	go func() {
		defer resp.Done()
		var lastLSNID uint64
		for {
			rows, err, ok := rowsStream.Next()
			if err != nil {
				resp.Put(nil, err)
				return
			}

			if !ok {
				break
			}

			q := make([]query.Query, 0, len(rows))

			for _, rec := range rows {
				if rec.GetLsnID() > lastLSNID {
					lastLSNID = rec.GetLsnID()
				}
				q = append(q, query.NewQuery(rec.GetMethodID(), rec.GetArgs()))
			}
			resp.Put(q, nil)
		}

		w.lsnGen = id_generator.NewIDGeneratorWithStart(lastLSNID)

	}()

	return resp, nil
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
	const (
		retryCount = 2
		delay      = 100 * time.Millisecond
	)

	err := retry.WithRetry(context.Background(), retryCount, delay, func() error {
		return w.writer.Write(w.queryBuf)
	})
	if err != nil {
		w.log.Error("failed to flush WAL files", logger.Error(err))
	} else {

		w.log.Info("successfully flushed WAL files")
		w.queryBuf = w.queryBuf[:0]
	}

	var resp []concurrency.FutureResp[error]

	concurrency.WithLock(w.m, func() {
		resp = make([]concurrency.FutureResp[error], len(w.futureResp))
		copy(resp, w.futureResp)
		w.futureResp = w.futureResp[:0]
	})

	for _, r := range resp {
		r.Put(err)
		r.Done()
	}
}
