package tcp

import (
	"context"
	"io"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/utils/retry"
	"net"
	"time"
)

type Client struct {
	conn       net.Conn
	address    string
	respSizeB  int64
	deadline   time.Duration
	retryCount int64
	log        logger.Logger
}

const (
	defaultRespSizeB  = 4096
	defaultDeadline   = 5 * time.Second
	defaultRetryCount = 3
)

func NewClient(conf config.Client, log logger.Logger) (*Client, error) {
	c := &Client{
		address:    conf.Address,
		respSizeB:  conf.MaxRespSize.Int64(),
		deadline:   conf.Deadline,
		retryCount: conf.RetryCount,
		log:        log,
	}

	if c.deadline == 0 {
		c.deadline = defaultDeadline
	}
	if c.retryCount == 0 {
		c.retryCount = defaultRetryCount
	}
	if c.respSizeB == 0 {
		c.respSizeB = defaultRespSizeB
	}

	err := c.setConn()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) Close() {
	err := c.conn.Close()
	if err != nil {
		return
	}
}

func (c *Client) Send(ctx context.Context, req []byte) (io.Reader, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	err := retry.WithRetry(ctx, int(c.retryCount), 200*time.Millisecond, func() error {
		err := c.send(req)
		if err != nil {
			errReconnect := c.setConn()
			if errReconnect != nil {
				return errReconnect
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return c.conn, nil
}

func (c *Client) send(req []byte) error {
	_, err := c.conn.Write(req)
	if err != nil {
		c.log.Error("error write to connection", logger.Error(err))
		return err
	}

	return nil
}

func (c *Client) setConn() error {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return err
	}

	err = conn.SetDeadline(time.Now().Add(c.deadline))
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}
