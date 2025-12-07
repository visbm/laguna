package replication

import (
	"bufio"
	"context"
	"errors"
	"io"
	"laguna/common/logger"
	"laguna/internal/database/wal"

	"laguna/internal/query"
)

type Master struct {
	lR  LogReader
	lW  LogWriter
	log logger.Logger
}

func NewMaster(lR LogReader, lW LogWriter, log logger.Logger) *Master {
	return &Master{
		lR:  lR,
		lW:  lW,
		log: log,
	}
}

var i = 0

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

	/*rows, err := m.lR.ReadFrom("", " ")
	if err != nil {
		m.log.Error("failed to read rows", logger.Error(err))
		return m.writeError(err, bufW)
	}*/
	// todo fix
	rows := []*wal.Row{
		wal.NewRow(32222, query.SetMethodID, []string{"nick", "1"}),
	}

	if i%2 == 0 {
		m.log.Error("failed to read rows", logger.Error(err))
		i++
		return m.writeError(errors.ErrUnsupported, bufW)
	}
	i++

	conTg := NewConnTarget(bufW)

	err = m.lW.WriteTo(rows, conTg)
	if err != nil {
		m.log.Error("failed to write rows", logger.Error(err))
		return m.writeError(err, bufW)
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
	data, err := resp.Marshal()
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
