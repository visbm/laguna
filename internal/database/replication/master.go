package replication

import (
	"bufio"
	"context"
	"errors"
	"io"
	"laguna/common/logger"
	"laguna/internal/fs"
	"laguna/utils/retry"
	"time"
)

type SegmentManager interface {
	GetIndex(lsn uint64) (fs.IndexEntry, error)
}

type Master struct {
	lR     LogReader
	lW     LogWriter
	im     SegmentManager
	walDir string

	log logger.Logger
}

func NewMaster(lR LogReader, lW LogWriter, im SegmentManager, log logger.Logger, walDir string) *Master {
	return &Master{
		lR:     lR,
		lW:     lW,
		log:    log,
		im:     im,
		walDir: walDir,
	}
}

func (m *Master) Handle(ctx context.Context, r io.Reader, w io.Writer) error {
	if ctx.Err() != nil {
		m.log.Error("context canceled", logger.Error(ctx.Err()))
		return ctx.Err()
	}

	bufR := bufio.NewReader(r)
	bufW := bufio.NewWriter(w)

	data := make([]byte, reqSize)
	n, err := bufR.Read(data)
	if err != nil {
		m.log.Error("failed to read input", logger.Error(err))
		return m.writeError(err, bufW)
	}

	req := &Request{}
	err = req.Unmarshal(data[:n])
	if err != nil {
		m.log.Error("failed to unmarshal request", logger.Error(err))
		return m.writeError(err, bufW)
	}

	if req.LsnID == 0 {
		m.log.Error("lsnID is 0")
		return m.writeError(errors.New("lsnID is 0"), bufW)
	}

	index, err := retry.WithRetryValue(ctx, 3, 200*time.Millisecond, func() (fs.IndexEntry, error) {
		index, errInd := m.im.GetIndex(req.LsnID)
		if errInd != nil {
			return index, errInd
		}

		return index, nil
	})

	if err != nil {
		if errors.Is(err, fs.ErrNotFound) {
			m.log.Info("index not found", logger.Integer("lsnID", int64(req.LsnID)))
			return m.writeError(err, bufW)
		}
		m.log.Error("failed to fetch index", logger.Error(err))
		return m.writeError(err, bufW)
	}

	rows, err := m.lR.ReadFrom(m.walDir, index.FileName, index.Offset)
	if err != nil {
		m.log.Error("failed to read rows", logger.Error(err))
		return m.writeError(err, bufW)
	}

	conTg := NewConnTarget(bufW)

	err = m.lW.WriteTo(rows, conTg)
	if err != nil {
		m.log.Error("failed to write rows", logger.Error(err))
		return err
	}
	err = bufW.Flush()
	if err != nil {
		m.log.Error("failed to flush buffer", logger.Error(err))
	}

	m.log.Info("Successfully sent logs to slave", logger.Integer("len", int64(len(rows))))

	return nil
}

func (m *Master) writeError(err error, bufW *bufio.Writer) error {
	var resp Response
	resp.Err = err
	data, err := resp.WriteMessage()
	if err != nil {
		m.log.Error("failed to marshal response", logger.Error(err))
		return err
	}

	_, err = bufW.Write(data)
	if err != nil {
		m.log.Error("failed to write response", logger.Error(err))
		return err
	}

	err = bufW.Flush()
	if err != nil {
		m.log.Error("failed to flush buffer", logger.Error(err))
		return err
	}

	return nil
}
