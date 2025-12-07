package replication

import (
	"bytes"
	"context"
	"fmt"
	"laguna/common/logger"
	"laguna/internal/database/wal"
	"laguna/internal/query"

	"time"
)

type Storage interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}

type Slave struct {
	cl        TCPClient
	st        Storage
	logReader LogReader
	logWriter LogWriter

	syncInterval   time.Duration
	lastLsnWritten uint64

	log logger.Logger
}

const (
	defaultSyncInterval = 1 * time.Second
)

func NewSlaveWithWal(cl TCPClient, lr LogReader, lw LogWriter, st Storage, syncInterval time.Duration, lg logger.Logger) *Slave {
	s := &Slave{cl: cl,
		syncInterval: syncInterval,
		log:          lg,
		logReader:    lr,
		logWriter:    lw,

		st: st,
	}

	if s.syncInterval == 0 {
		s.syncInterval = defaultSyncInterval
	}

	return s
}

func NewSlave(cl TCPClient, lr LogReader, st Storage, syncInterval time.Duration, lg logger.Logger) *Slave {
	s := &Slave{cl: cl,
		syncInterval: syncInterval,
		log:          lg,
		logReader:    lr,

		st: st,
	}

	if s.syncInterval == 0 {
		s.syncInterval = defaultSyncInterval
	}

	return s
}

func (s *Slave) Start(ctx context.Context) {
	go s.startReplication(ctx)
}

func (s *Slave) startReplication(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.log.Error("context deadline exceeded", logger.Error(ctx.Err()))
			return
		default:
		}

		select {
		case <-ctx.Done():
			s.log.Error("context deadline exceeded", logger.Error(ctx.Err()))
			return
		case <-time.After(s.syncInterval):
			_ = s.getUpdates(ctx)
		}
	}
}

func (s *Slave) getUpdates(ctx context.Context) error {
	if ctx.Err() != nil {
		s.log.Error("context deadline exceeded", logger.Error(ctx.Err()))
		return ctx.Err()
	}

	resp, err := s.sendReq(ctx, s.lastLsnWritten+1)
	if err != nil {
		return err
	}

	if len(resp.Data) == 0 {
		s.log.Info("response data is empty")
		return nil
	}

	s.log.Info("replication get logs", logger.Integer("len", int64(len(resp.Data))))
	rowsStream := s.logReader.ReadStream(bytes.NewReader(resp.Data))

	for {
		rows, err, ok := rowsStream.Next()
		if err != nil {
			return err
		}

		if !ok {
			break
		}

		err = s.processRows(ctx, rows)
		if err != nil {
			s.log.Error("process rows failed", logger.Error(err))
			return err
		}

	}

	return nil
}

func (s *Slave) processRows(ctx context.Context, rows []*wal.Row) error {
	if ctx.Err() != nil {
		s.log.Error("context deadline exceeded", logger.Error(ctx.Err()))
		return ctx.Err()
	}

	if s.walEnable() {
		err := s.logWriter.Write(rows)
		if err != nil {
			s.log.Error("write response failed", logger.Error(err))
			return err
		}
	}

	err := s.execStorage(ctx, rows)
	if err != nil {
		s.log.Error("exec storage failed", logger.Error(err))
		return err
	}

	s.lastLsnWritten = rows[len(rows)-1].GetLsnID()

	return nil
}

func (s *Slave) execStorage(ctx context.Context, rows []*wal.Row) error {
	for _, r := range rows {
		args := r.GetArgs()
		method := r.GetMethodID()

		switch method {

		case query.SetMethodID:
			err := s.st.Set(ctx, args[query.SetKeyIdx], args[query.SetValueIdx])
			return err

		case query.DelMethodID:
			err := s.st.Delete(ctx, args[query.DelKeyIdx])
			return err

		default:
			return fmt.Errorf("unknown command ID: %d", method)
		}
	}
	return nil
}

func (s *Slave) sendReq(ctx context.Context, lsnID uint64) (Response, error) {
	var resp Response

	req := Request{
		LsnID: lsnID, //todo ??
	}

	body, err := req.Marshal()
	if err != nil {
		s.log.Error("marshal request failed", logger.Error(err))
		return resp, err
	}

	b, err := s.cl.Send(ctx, body)
	if err != nil {
		s.log.Error("send request failed", logger.Error(err), logger.Bytes("body", b))
		return resp, err
	}

	err = resp.Unmarshal(b)
	if err != nil {
		s.log.Error("unmarshal response failed", logger.Error(err))
		return resp, err
	}

	if resp.Err != nil {
		s.log.Error("response failed", logger.Error(resp.Err))
		return resp, resp.Err
	}

	return resp, nil
}
func (s *Slave) walEnable() bool {
	return s.logWriter != nil
}
