package tcp

import (
	"context"
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

func (c *Client) Send(ctx context.Context, req []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	resp, err := retry.WithRetryValue(ctx, int(c.retryCount), 100*time.Millisecond, func() ([]byte, error) {
		v, err := c.send(req)
		if err != nil {
			errReconnect := c.setConn()
			if errReconnect != nil {
				c.log.Error("error read from connection", logger.Error(errReconnect))
				return nil, errReconnect
			}
		}

		return v, nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) send(req []byte) ([]byte, error) {
	_, err := c.conn.Write(req)
	if err != nil {
		c.log.Error("error write to connection", logger.Error(err))
		return nil, err
	}

	buf := make([]byte, c.respSizeB)
	n, err := c.conn.Read(buf)
	if err != nil {
		c.log.Error("error read from connection", logger.Error(err))
		return nil, err
	}

	return buf[:n], nil
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
