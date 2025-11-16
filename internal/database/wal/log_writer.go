package wal

import (
	"bytes"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/utils"
)

type Segment interface {
	Rotate() error
	Write(data []byte) error
	Fits(incoming int64) bool
	Close() error
}

const (
	sep    = '\n'
	sepLen = 1
)

type LogWriter struct {
	log logger.Logger
	buf *bytes.Buffer

	maxSegmentSize utils.ByteSize `yaml:"max_segment_size"`
	directory      string         `yaml:"directory"`

	curSeg Segment
}

func NewLogWriter(conf config.WAL, log logger.Logger, segment Segment) *LogWriter {
	return &LogWriter{
		buf:            bytes.NewBuffer(make([]byte, 0)),
		log:            log,
		directory:      conf.Directory,
		maxSegmentSize: conf.MaxSegmentSize,

		curSeg: segment,
	}
}

func (lw *LogWriter) Write(batch [][]byte) error {
	err := lw.processBatches(batch)
	if err != nil {
		return err
	}

	return nil
}

func (lw *LogWriter) write(batch []byte) error {
	if len(batch) == 0 {
		return nil
	}

	err := lw.curSeg.Write(batch)
	if err != nil {
		lw.log.Error("error writing to file", logger.Error(err))
		return err
	}

	return nil
}

func (lw *LogWriter) processBatches(batch [][]byte) error {
	batchSize := 0
	for _, b := range batch {
		batchSize += len(b) + 1
	}

	if lw.curSeg.Fits(int64(batchSize)) {
		err := lw.writeBatch(batch, batchSize)
		if err != nil {
			lw.log.Error("error writing to file", logger.Error(err))
			return err
		}

		return nil
	}

	currentSize := 0
	start := 0

	for i, b := range batch {
		size := len(b) + sepLen

		if !lw.curSeg.Fits(int64(currentSize + size)) {
			err := lw.writeBatch(batch[start:i], currentSize)
			if err != nil {
				lw.log.Error("error writing to file", logger.Error(err))
				return err
			}

			err = lw.curSeg.Rotate()
			if err != nil {
				lw.log.Error("error writing to file", logger.Error(err))
				return err
			}

			start = i
			currentSize = 0
		}

		currentSize += size
	}

	err := lw.writeBatch(batch[start:], currentSize)
	if err != nil {
		lw.log.Error("error writing to file", logger.Error(err))
		return err
	}

	return nil
}

func (lw *LogWriter) writeBatch(bath [][]byte, bufSize int) error {
	data := lw.getData(bath, bufSize)
	err := lw.write(data)
	if err != nil {
		lw.log.Error("error writing to file", logger.Error(err))
		return err
	}

	return nil
}

func (lw *LogWriter) getData(bath [][]byte, bufSize int) []byte {
	lw.buf.Grow(bufSize)
	for _, b := range bath {
		lw.buf.Write(b)
		lw.buf.WriteByte(sep)
	}

	data := lw.buf.Bytes()
	lw.buf.Reset()
	return data
}
