package wal

import (
	"fmt"
	"laguna/internal/query"
	"strings"
)

type row struct {
	lsnID    uint64
	methodID query.MethodID
	args     []string
}

const (
	lsnIdLen  = 8
	methodLen = 1
)

func (r *row) Marshal() []byte {
	respLen := lsnIdLen + methodLen + (len(r.args) - 1)
	for _, a := range r.args {
		respLen += len(a)
	}

	lsnBytes := r.encodeUint64(r.lsnID)

	resp := make([]byte, 0, respLen)
	resp = append(resp, lsnBytes...)
	resp = append(resp, byte(r.methodID))

	for i, a := range r.args {
		resp = append(resp, a...)

		if i != len(r.args)-1 {
			resp = append(resp, ' ')
		}
	}

	return resp
}

func (r *row) Unmarshal(buf []byte) error {
	if len(buf) < lsnIdLen+methodLen {
		return fmt.Errorf("invalid length %d", len(buf))
	}

	lsnID := r.decodeUint64(buf[:lsnIdLen])

	methodID := buf[lsnIdLen]

	args := string(buf[lsnIdLen+methodLen:])

	argsArr := strings.Split(args, " ")

	r.lsnID = lsnID
	r.methodID = query.MethodID(methodID)
	r.args = argsArr

	return nil
}

func (r *row) encodeUint64(v uint64) []byte {
	b := make([]byte, lsnIdLen)
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
	return b
}

func (r *row) decodeUint64(b []byte) uint64 {
	if len(b) < lsnIdLen {
		return 0
	}
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}
