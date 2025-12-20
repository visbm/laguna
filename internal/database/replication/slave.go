package replication

import (
	"bytes"
	"context"
	"errors"
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
	defaultSyncInterval     = 1 * time.Second
	defaultSleepOnNoNewLogs = 3 * time.Second
)

func NewSlaveWithWal(log logger.Logger, cl TCPClient, lr LogReader, lw LogWriter, st Storage, syncInterval time.Duration) *Slave {
	s := &Slave{cl: cl,
		syncInterval: syncInterval,
		logReader:    lr,
		logWriter:    lw,
		st:           st,

		log: log,
	}

	if s.syncInterval == 0 {
		s.syncInterval = defaultSyncInterval
	}

	return s
}

func NewSlave(log logger.Logger, cl TCPClient, lr LogReader, st Storage, syncInterval time.Duration) *Slave {
	s := &Slave{cl: cl,
		syncInterval: syncInterval,
		logReader:    lr,
		st:           st,

		log: log,
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
			err := s.getUpdates(ctx)
			if errors.Is(err, ErrNoNewLogs) {
				time.Sleep(defaultSleepOnNoNewLogs)
			}
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
	if len(rows) == 0 {
		return nil
	}

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

		var err error
		switch method {
		case query.SetMethodID:
			err = s.st.Set(ctx, args[query.SetKeyIdx], args[query.SetValueIdx])
		case query.DelMethodID:
			err = s.st.Delete(ctx, args[query.DelKeyIdx])
		default:
			return fmt.Errorf("unknown command ID: %d", method)
		}

		if err != nil {
			return fmt.Errorf("failed to execute method %d: %w", method, err)
		}
	}
	return nil
}

func (s *Slave) sendReq(ctx context.Context, lsnID uint64) (*Response, error) {
	if ctx.Err() != nil {
		s.log.Error("context deadline exceeded", logger.Error(ctx.Err()))
		return nil, ctx.Err()
	}

	req := Request{
		LsnID: lsnID,
	}

	body, err := req.Marshal()
	if err != nil {
		s.log.Error("marshal request failed", logger.Error(err))
		return nil, err
	}

	r, err := s.cl.Send(ctx, body)
	if err != nil {
		s.log.Error("send request failed", logger.Error(err))
		return nil, err
	}

	resp, err := ReadMessage(r)
	if err != nil {
		s.log.Error("unmarshal response failed", logger.Error(err))
		return nil, err
	}

	if resp.Err != nil {
		if resp.Err.Error() == ErrNoNewLogs.Error() {
			s.log.Info("no new logs for", logger.Integer("lsn id", int64(lsnID)))
			return resp, ErrNoNewLogs
		}

		s.log.Error("response failed with", logger.Integer("lsn ID", int64(lsnID)), logger.Error(resp.Err))
		return resp, resp.Err
	}

	return resp, nil
}

func (s *Slave) walEnable() bool {
	return s.logWriter != nil
}

func (s *Slave) Close() {
	s.cl.Close()
}
