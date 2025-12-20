package replication

import (
	"bufio"
	"context"
	"errors"
	"io"
	"laguna/common/logger"
	"laguna/internal/segment"
	"laguna/utils/retry"
	"time"
)

type Master struct {
	lR     LogReader
	lW     LogWriter
	im     SegmentManager
	ls     TCPLister
	walDir string

	log logger.Logger
}

func NewMaster(log logger.Logger, lR LogReader, lW LogWriter, sm SegmentManager, walDir string) *Master {
	return &Master{
		lR: lR,
		lW: lW,
		im: sm,

		walDir: walDir,

		log: log,
	}
}

func (m *Master) setTransport(ls TCPLister) {
	m.ls = ls
}

func (m *Master) Start(ctx context.Context) {
	if m.ls == nil {
		m.log.Error("TCPLister is nil, cannot start")
		return
	}
	m.ls.Listen(ctx)
}

func (m *Master) Close() {
	if m.ls == nil {
		return
	}
	m.ls.Close()
}

func (m *Master) Handle(ctx context.Context, r io.Reader, w io.Writer) error {
	if ctx.Err() != nil {
		m.log.Error("context canceled", logger.Error(ctx.Err()))
		return ctx.Err()
	}

	bufR := bufio.NewReader(r)
	bufW := bufio.NewWriter(w)

	req, err := m.getReq(bufR)
	if err != nil {
		return m.writeError(err, bufW)
	}

	if req.LsnID == 0 {
		m.log.Error("lsnID is 0")
		return m.writeError(errors.New("lsnID is 0"), bufW)
	}

	index, err := retry.WithRetryValue(ctx, 3, 200*time.Millisecond, func() (segment.IndexEntry, error) {
		index, errInd := m.im.GetIndex(req.LsnID)
		if errInd != nil {
			return index, errInd
		}

		return index, nil
	})

	if err != nil {
		if errors.Is(err, segment.ErrNotFound) {
			m.log.Info("index not found", logger.Integer("lsnID", int64(req.LsnID)))
			return m.writeError(ErrNoNewLogs, bufW)
		}

		m.log.Error("failed to fetch index", logger.Error(err))
		return m.writeError(err, bufW)
	}

	rows, err := m.lR.ReadFrom(m.walDir, index.FileName, index.Offset)
	if err != nil {
		m.log.Error("failed to read rows", logger.Error(err))
		return m.writeError(err, bufW)
	}

	if len(rows) == 0 {
		return m.writeError(ErrNoNewLogs, bufW)
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
		return err
	}

	m.log.Info("Successfully sent logs to slave", logger.Integer("len", int64(len(rows))))

	return nil
}

func (m *Master) getReq(bufR *bufio.Reader) (*Request, error) {
	data := make([]byte, reqSize)
	n, err := bufR.Read(data)
	if err != nil {
		m.log.Error("failed to read input", logger.Error(err))
		return nil, err
	}

	req := &Request{}
	err = req.Unmarshal(data[:n])
	if err != nil {
		m.log.Error("failed to unmarshal request", logger.Error(err))
		return nil, err
	}

	return req, err
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
