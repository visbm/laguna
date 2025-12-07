package ctx_utils

import "context"

type contextKey int

const (
	txKey contextKey = iota
)

func SetTxInContext(ctx context.Context, txID uint64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, txKey, txID)
}

func GetTxFromContext(ctx context.Context) uint64 {
	if ctx == nil {
		return 0
	}

	v := ctx.Value(txKey)
	if id, ok := v.(uint64); ok {
		return id
	}
	return 0
}
