package handlers

import (
	"bufio"
	"context"
	"errors"
	"io"
	"laguna/common/logger"
	"laguna/internal/query"
)

type Handler interface {
	Handle(ctx context.Context, r io.Reader, w io.Writer) error
}

// mockgen -source=internal/handlers/handlers.go -destination=internal/mocks/handlers_mocks.go -package=mocks
type Database interface {
	Execute(ctx context.Context, q query.Query) (string, error)
}

type QueryBuilder interface {
	Parse(in []byte) (query.Query, error)
}
type UniversalHandler struct {
	bl  QueryBuilder
	db  Database
	log logger.Logger
}

func NewUniversalHandler(qb QueryBuilder, db Database, log logger.Logger) *UniversalHandler {
	return &UniversalHandler{
		bl:  qb,
		db:  db,
		log: log,
	}
}

func (h *UniversalHandler) Handle(ctx context.Context, r io.Reader, w io.Writer) error {
	if ctx.Err() != nil {
		h.log.Error("context canceled")
		return ctx.Err()
	}

	bufR := bufio.NewReader(r)
	bufW := bufio.NewWriter(w)

	in, err := bufR.ReadBytes('\n')
	if err != nil {
		h.log.Error("failed to read input", logger.Error(err))
		return h.writeError(err, bufW)
	}

	q, err := h.bl.Parse(in)
	if err != nil {
		h.log.Error("failed to parse input",
			logger.Bytes("input", in),
			logger.Error(err),
		)
		return h.writeError(err, bufW)
	}

	v, err := h.db.Execute(ctx, q)
	if err != nil {
		h.log.Error("failed to execute query",
			logger.Bytes("input", in),
			logger.Error(err),
		)

		return h.writeError(err, bufW)
	}

	err = h.write(v, bufW)
	if err != nil {
		h.log.Error("failed to write response",
			logger.Bytes("response", []byte(v)),
			logger.Error(err),
		)

		return h.writeError(err, bufW)
	}

	return nil
}

func (h *UniversalHandler) write(v string, bufW *bufio.Writer) error {
	_, err := bufW.Write(append([]byte(v), '\n'))
	if err != nil {
		h.log.Error("failed to write response", logger.Error(err))
		return err
	}

	err = bufW.Flush()
	if err != nil {
		h.log.Error("failed to flush response", logger.Error(err))
		return err
	}

	return nil
}

func (h *UniversalHandler) writeError(error error, bufW *bufio.Writer) error {
	if errors.Is(error, io.EOF) {
		return error
	}

	_, err := bufW.Write(append([]byte(error.Error()), '\n'))
	if err != nil {
		h.log.Error("failed to write response", logger.Error(err))
		return err
	}

	err = bufW.Flush()
	if err != nil {
		h.log.Error("failed to flush response", logger.Error(err))
		return err
	}

	return nil
}
