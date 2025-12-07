package replication

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"laguna/internal/database/wal"
)

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

	respBytes, err := resp.Marshal()
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

	buf := make([]byte, 0)
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

type Response struct {
	Data []byte
	Err  error
}

func (r *Response) Marshal() ([]byte, error) {
	var buf bytes.Buffer

	if r.Err != nil {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}

	if err := binary.Write(&buf, binary.BigEndian, int32(len(r.Data))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(r.Data); err != nil {
		return nil, err
	}

	if r.Err != nil {
		errStr := r.Err.Error()
		if _, err := buf.Write([]byte(errStr)); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (r *Response) Unmarshal(b []byte) error {
	buf := bytes.NewReader(b)

	flag, err := buf.ReadByte()
	if err != nil {
		return err
	}

	var dataLen int32
	if err := binary.Read(buf, binary.BigEndian, &dataLen); err != nil {
		return err
	}
	if dataLen < 0 {
		return errors.New("invalid data length")
	}
	r.Data = make([]byte, dataLen)
	if _, err := io.ReadFull(buf, r.Data); err != nil {
		return err
	}

	if flag == 1 {
		errBytes, _ := io.ReadAll(buf)
		r.Err = errors.New(string(errBytes))
	} else {
		r.Err = nil
	}

	return nil
}
