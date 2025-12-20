package wal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"laguna/internal/query"
)

type Row struct {
	lsnID    uint64
	methodID query.MethodID
	args     []string
}

func NewRow(lsnID uint64, methodID query.MethodID, args []string) *Row {
	return &Row{
		lsnID:    lsnID,
		methodID: methodID,
		args:     args,
	}
}

func (r *Row) GetLsnID() uint64 {
	return r.lsnID
}

func (r *Row) GetMethodID() query.MethodID {
	return r.methodID
}

func (r *Row) GetArgs() []string {
	return r.args
}

// Marshal [8 bytes LSN] [1 byte methodID] [2 bytes argsCount] +
// for each arg: [2 bytes len] [N bytes content]
func (r *Row) Marshal() ([]byte, error) {
	expectedSize := 8 + 1 + 2
	for _, a := range r.args {
		expectedSize += 2 + len(a)
	}

	buf := bytes.NewBuffer(make([]byte, 0, expectedSize))

	if err := binary.Write(buf, binary.BigEndian, r.lsnID); err != nil {
		return nil, err
	}

	if err := buf.WriteByte(uint8(r.methodID)); err != nil {
		return nil, err
	}

	argsCount := uint16(len(r.args))
	if err := binary.Write(buf, binary.BigEndian, argsCount); err != nil {
		return nil, err
	}

	for _, a := range r.args {
		b := []byte(a)
		if len(b) > 65535 {
			return nil, errors.New("argument too large")
		}

		if err := binary.Write(buf, binary.BigEndian, uint16(len(b))); err != nil {
			return nil, err
		}

		if _, err := buf.Write(b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (r *Row) Unmarshal(data []byte) (int, error) {
	buf := bytes.NewReader(data)
	startLen := buf.Len()

	if err := binary.Read(buf, binary.BigEndian, &r.lsnID); err != nil {
		return 0, io.EOF
	}

	methodID, err := buf.ReadByte()
	if err != nil {
		return 0, io.EOF
	}
	r.methodID = query.MethodID(methodID)

	var argsCount uint16
	if err := binary.Read(buf, binary.BigEndian, &argsCount); err != nil {
		return 0, io.EOF
	}

	r.args = make([]string, 0, argsCount)

	for i := 0; i < int(argsCount); i++ {
		var argLen uint16
		if err := binary.Read(buf, binary.BigEndian, &argLen); err != nil {
			return 0, io.EOF
		}

		argBytes := make([]byte, argLen)
		if err := binary.Read(buf, binary.BigEndian, &argBytes); err != nil {
			return 0, io.EOF
		}

		r.args = append(r.args, string(argBytes))
	}

	readBytes := startLen - buf.Len()
	return readBytes, nil
}
