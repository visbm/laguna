package cli

import (
	"context"
	"fmt"
	"io"
	"laguna/common/logger"
	"laguna/internal/handlers"
	"os"
)

type Listener struct {
	log logger.Logger
	hd  handlers.Handler
}

func NewListener(log logger.Logger, hd handlers.Handler) *Listener {
	return &Listener{
		log: log,
		hd:  hd,
	}
}

func (l *Listener) Listen(ctx context.Context) {
	l.log.Info("starting cli listener")
	defer l.log.Info("stopped listener")

	r := io.Reader(os.Stdin)
	w := io.Writer(os.Stdout)

	_, err := w.Write([]byte("laguna db > "))
	if err != nil {
		fmt.Println(err)
	}

	for {
		select {
		case <-ctx.Done():
			l.log.Info("context canceled")
			return
		default:
		}

		if err := l.hd.Handle(ctx, r, w); err != nil {
			l.log.Error("handler error", logger.Error(err))
		}
	}
}

func (l *Listener) Close() {
	l.log.Info("closing listener")
}
