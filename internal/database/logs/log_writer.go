package logs

import (
	"errors"
	"laguna/common/logger"
	"laguna/internal/database/wal"
)

type Target interface {
	Write(rows []*wal.Row) error
}

type LogWriter struct {
	target Target
	log    logger.Logger
}

func NewLogWriter(log logger.Logger, t Target) *LogWriter {
	return &LogWriter{
		target: t,
		log:    log,
	}
}

func (lw *LogWriter) WriteTo(rows []*wal.Row, t Target) error {
	return lw.write(rows, t)
}

func (lw *LogWriter) Write(rows []*wal.Row) error {
	return lw.write(rows, lw.target)
}

func (lw *LogWriter) write(rows []*wal.Row, t Target) error {
	if t == nil {
		return errors.New("no target")
	}

	if len(rows) == 0 {
		return nil
	}

	err := t.Write(rows)
	if err != nil {
		lw.log.Error("failed to write", logger.Error(err))
		return err
	}

	return nil
}
