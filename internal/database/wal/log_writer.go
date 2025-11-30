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

type LogWriter struct {
	log logger.Logger
	buf *bytes.Buffer

	maxSegmentSize utils.ByteSize `yaml:"max_segment_size"`
	directory      string         `yaml:"directory"`

	curSeg Segment
}

func NewLogWriter(conf config.WAL, log logger.Logger, segment Segment) *LogWriter {
	return &LogWriter{
		log:            log,
		directory:      conf.Directory,
		maxSegmentSize: conf.MaxSegmentSize,

		curSeg: segment,
	}
}

func (lw *LogWriter) Write(rows []*Row) error {
	batch := make([][]byte, len(rows))
	for _, row := range rows {
		rB, err := row.Marshal()
		if err != nil {
			return err
		}
		batch = append(batch, rB)
	}

	err := lw.processBatches(batch)
	if err != nil {
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
		size := len(b)

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

func (lw *LogWriter) writeBatch(batch [][]byte, bufSize int) error {
	data := lw.getData(batch, bufSize)
	err := lw.writeInSeg(data)
	if err != nil {
		lw.log.Error("error writing to file", logger.Error(err))
		return err
	}

	return nil
}

func (lw *LogWriter) getData(batch [][]byte, bufSize int) []byte {
	buf := make([]byte, 0, bufSize)
	for _, b := range batch {
		buf = append(buf, b...)
	}

	return buf
}

func (lw *LogWriter) writeInSeg(batch []byte) error {
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
