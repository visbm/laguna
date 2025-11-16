package query

import "errors"

type Method string

const (
	SET Method = "SET"
	GET Method = "GET"
	DEL Method = "DEL"
)

var Methods = map[Method]struct{}{
	SET: {},
	GET: {},
	DEL: {},
}

type MethodID uint8

const (
	SetMethodID MethodID = iota
	GetMethodID MethodID = iota
	DelMethodID MethodID = iota
)

var MethodNames = map[Method]MethodID{
	SET: SetMethodID,
	GET: GetMethodID,
	DEL: DelMethodID,
}

var MethodIDs = map[MethodID]Method{
	SetMethodID: SET,
	GetMethodID: GET,
	DelMethodID: DEL,
}

var (
	ErrUnknowMethod = errors.New("unknow method")
	ErrWrongQuery   = errors.New("wrong query")
	ErrWrongArgsLen = errors.New("wrong args length")
)

const (
	SetArgsLen = 2
	GetArgsLen = 1
	DelArgsLen = 1
)

var MethodArgs = map[MethodID]int{
	SetMethodID: SetArgsLen,
	GetMethodID: GetArgsLen,
	DelMethodID: DelArgsLen,
}

const (
	SetKeyIdx   = 0
	SetValueIdx = 1

	GetKeyIdx = 0

	DelKeyIdx = 0
)

type Query struct {
	methodID MethodID
	args     []string
}

func (q Query) GetArs() []string {
	return q.args
}
func (q Query) MethodID() MethodID {
	return q.methodID
}
