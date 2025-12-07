package transport

import (
	"context"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/handlers"
	"laguna/internal/transport/cli"
	"laguna/internal/transport/tcp"
)

type Listener interface {
	Listen(ctx context.Context)
	Close()
}

func NewListener(log logger.Logger, hd handlers.Handler, c config.Transport) Listener {
	switch c.Type {
	case "tcp":
		return tcp.NewListener(log, hd, c.Sever)
	case "cli":
		return cli.NewListener(log, hd)
	default:
		return cli.NewListener(log, hd)
	}
}
