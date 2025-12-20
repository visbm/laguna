package query

import (
	"strings"
)

type Builder struct {
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Parse(in []byte) (Query, error) {
	inputStr := string(in)
	inputStr = strings.Trim(inputStr, "\r\n")
	inputStr = strings.TrimSpace(inputStr)

	tokens := strings.Fields(inputStr)

	err := b.validate(tokens)
	if err != nil {
		return Query{}, err
	}

	q, err := b.getQuery(Method(tokens[0]), tokens[1:])
	if err != nil {
		return Query{}, err
	}

	return q, nil
}

func (b *Builder) validate(tokens []string) error {
	if len(tokens) < 2 {
		return ErrWrongQuery
	}

	_, ok := Methods[Method(tokens[0])]
	if !ok {
		return ErrUnknowMethod
	}

	methodID, err := b.getMethodID(Method(tokens[0]))
	if err != nil {
		return err
	}

	if len(tokens[1:]) < MethodArgs[methodID] {
		return ErrWrongArgsLen
	}

	return nil
}

func (b *Builder) getQuery(method Method, args []string) (Query, error) {
	methodID, err := b.getMethodID(method)
	if err != nil {
		return Query{}, err
	}

	switch methodID {
	case SetMethodID:
		key := args[SetKeyIdx]
		value := strings.Join(args[SetValueIdx:], " ")
		return Query{
			methodID: methodID,
			args:     []string{key, value},
		}, err
	default:
		return Query{
			methodID: methodID,
			args:     args,
		}, nil
	}

}

func (b *Builder) getMethodID(method Method) (MethodID, error) {
	methodID, ok := MethodNames[method]
	if !ok {
		return 0, ErrUnknowMethod
	}

	return methodID, nil
}
