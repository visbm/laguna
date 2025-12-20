package tcp

import (
	"bufio"
	"context"
	"errors"
	"io"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/handlers"
	"laguna/utils/concurrency"
	"net"
	"time"
)

type tcpListener struct {
	log logger.Logger
	hd  handlers.Handler
	ls  net.Listener

	address        string
	maxConn        int64
	maxMsgSize     int64
	idleTimeout    time.Duration
	semWaitTimeout time.Duration

	sem *concurrency.Semaphore
}

const (
	defaultAddress        = "0.0.0.0:8080"
	defaultMaxConn        = 100
	defaultMaxMsgSize     = 4096
	defaultSemWaitTimeout = 5 * time.Second
	defaultIdleTimeout    = 5 * time.Minute
)

func NewListener(log logger.Logger, hd handlers.Handler, c config.TCPServer) *tcpListener {
	l := &tcpListener{
		log:            log,
		hd:             hd,
		address:        c.Address,
		maxConn:        c.MaxConn,
		maxMsgSize:     c.MaxMessageSize.Int64(),
		idleTimeout:    c.IdleTimeout,
		semWaitTimeout: c.SemWaitTimeout,
	}

	if l.address == "" {
		l.address = defaultAddress
	}
	if l.maxConn == 0 {
		l.maxConn = defaultMaxConn
	}
	if l.maxMsgSize == 0 {
		l.maxMsgSize = defaultMaxMsgSize
	}
	if l.idleTimeout == 0 {
		l.idleTimeout = defaultIdleTimeout
	}
	if l.semWaitTimeout == 0 {
		l.semWaitTimeout = defaultSemWaitTimeout
	}

	l.sem = concurrency.NewSemaphore(l.maxConn)
	return l
}

func (l *tcpListener) Listen(ctx context.Context) {
	go l.listen(ctx)
}

func (l *tcpListener) listen(ctx context.Context) {
	l.log.Info("starting tcp listener at address", logger.String("address", l.address))

	listener, err := net.Listen("tcp", l.address)
	if err != nil {
		l.log.Fatal("failed to listen: %w", logger.Error(err))
	}
	l.ls = listener

	for {
		select {
		case <-ctx.Done():
			l.log.Info("context canceled")
			return
		default:
		}

		conn, err := l.ls.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				l.log.Info("listener closed")
				return
			}

			l.log.Error("failed to accept connection", logger.Error(err))
			continue
		}

		err = conn.SetDeadline(time.Now().Add(l.idleTimeout))
		if err != nil {
			l.log.Error("failed to set deadline")
			continue
		}

		if !l.sem.AcquireWithDeadline(l.semWaitTimeout) {
			l.log.Info("failed to acquire semaphore")
			l.writeToConn("too many connections", conn)
			l.closeConn(conn)
			continue
		}

		l.log.Info("accepted new connection", logger.String("address", conn.RemoteAddr().String()))
		go l.handleConnection(ctx, conn)
	}

}

func (l *tcpListener) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() {
		if v := recover(); v != nil {
			l.log.Error("panic recovered", logger.String("address", conn.RemoteAddr().String()))
		}

		l.sem.Release()
		l.closeConn(conn)
	}()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		err := l.hd.Handle(ctx, reader, writer)
		if errors.Is(err, io.EOF) {
			break
		}

		var netErr *net.OpError
		if errors.As(err, &netErr) {
			break
		}
	}
}

func (l *tcpListener) Close() {
	l.log.Info("closing tcp listener")
	err := l.ls.Close()
	if err != nil {
		l.log.Warn("failed to close listener", logger.String("address", l.address))
		return
	}
}

func (l *tcpListener) writeToConn(str string, conn net.Conn) {
	_, err := conn.Write([]byte(str))
	if err != nil {
		l.log.Error("failed to write to connection", logger.String("address", conn.RemoteAddr().String()))
	}
}

func (l *tcpListener) closeConn(conn net.Conn) {
	err := conn.Close()
	if err != nil {
		l.log.Warn("failed to close connection", logger.String("address", conn.RemoteAddr().String()))
	}
}
