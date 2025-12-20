package replication

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"laguna/internal/database/wal"
)

var ErrNoNewLogs = errors.New("no new logs")

type ConnTarget struct {
	conn io.Writer
}

func NewConnTarget(w io.Writer) *ConnTarget {
	return &ConnTarget{
		conn: w,
	}
}

func (ct *ConnTarget) Write(rows []*wal.Row) error {
	data, err := flatten(rows)
	if err != nil {
		return err
	}

	var resp Response
	resp.Data = data

	respBytes, err := resp.WriteMessage()
	if err != nil {
		return err
	}
	_, err = ct.conn.Write(respBytes)
	if err != nil {
		return err
	}
	return nil
}

func flatten(rows []*wal.Row) ([]byte, error) {
	const approxBytesInRow = 8 + 8 + 8

	buf := make([]byte, 0, len(rows)*approxBytesInRow)
	for _, r := range rows {
		b, err := r.Marshal()
		if err != nil {
			return nil, err
		}

		buf = append(buf, b...)
	}
	return buf, nil
}

const reqSize = 8

type Request struct {
	LsnID uint64
}

func (r *Request) Marshal() ([]byte, error) {
	buf := make([]byte, reqSize)
	binary.LittleEndian.PutUint64(buf, r.LsnID)
	return buf, nil
}

func (r *Request) Unmarshal(data []byte) error {
	if len(data) < reqSize {
		return errors.New("not enough bytes to unmarshal Request")
	}

	r.LsnID = binary.LittleEndian.Uint64(data[:8])
	return nil
}

const (
	errFlag = 1
	okFlag  = 0
)

type Response struct {
	Data []byte
	Err  error
}

func (r *Response) WriteMessage() ([]byte, error) {
	var buf bytes.Buffer
	var err error

	if r.Err != nil {
		buf.WriteByte(errFlag)
		err = r.writerError(&buf)
	} else {
		buf.WriteByte(okFlag)
		err = r.writerData(&buf)
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (r *Response) writerData(buf *bytes.Buffer) error {
	if err := binary.Write(buf, binary.BigEndian, int32(len(r.Data))); err != nil {
		return err
	}

	if _, err := buf.Write(r.Data); err != nil {
		return err
	}
	return nil
}

func (r *Response) writerError(buf *bytes.Buffer) error {
	errStr := r.Err.Error()

	if err := binary.Write(buf, binary.BigEndian, int32(len(errStr))); err != nil {
		return err
	}

	if _, err := buf.Write([]byte(errStr)); err != nil {
		return err
	}
	return nil
}

func ReadMessage(r io.Reader) (*Response, error) {
	resp := &Response{}

	buf := bufio.NewReader(r)

	flag, err := buf.ReadByte()
	if err != nil {
		return resp, err
	}

	var dataLen int32
	if err := binary.Read(buf, binary.BigEndian, &dataLen); err != nil {
		return resp, err
	}

	if dataLen < 0 {
		return resp, errors.New("invalid data length")
	}

	data := make([]byte, dataLen)

	if flag == 1 {
		if _, err := io.ReadFull(buf, data); err != nil {
			return resp, err
		}
		resp.Err = errors.New(string(data))
	} else {
		if _, err := io.ReadFull(buf, data); err != nil {
			return resp, err
		}
		resp.Data = data
	}

	return resp, nil
}
