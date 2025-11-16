package main

import (
	"bufio"
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"syscall"
	"time"
)

var (
	address     = flag.String("address", "localhost:8080", "http service address")
	size        = flag.Int("size", 1024, "max response size")
	idleTimeout = flag.Duration("deadline", time.Minute*5, "deadline for client connection")
)

type Client struct {
	conn      net.Conn
	address   string
	respSizeB int
	deadline  time.Duration
}

func NewClient(addr string, respS int, deadline time.Duration) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	err = conn.SetDeadline(time.Now().Add(deadline))
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn, address: addr, respSizeB: respS, deadline: deadline}, nil
}

func (c *Client) Close() {
	err := c.conn.Close()
	if err != nil {
		return
	}
}

func (c *Client) Send(req string) (string, error) {
	_, err := c.conn.Write([]byte(req))
	if err != nil {
		return "", err
	}

	resp := make([]byte, c.respSizeB)
	n, err := c.conn.Read(resp)
	if err != nil {
		return "", err
	}

	return string(resp[:n]), nil
}

// go run client.go -address=localhost:8080
func main() {
	flag.Parse()

	cli, err := NewClient(*address, *size, *idleTimeout)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)

	log.Print("New laguna cli")

	for {
		_ = write(w, "> ")

		line, err := r.ReadString('\n')
		if err != nil {
			writeError(w, err)
			continue
		}

		resp, err := cli.Send(line)
		if err != nil {
			if errors.Is(err, syscall.EPIPE) {
				log.Print("connection closed")
				break
			}
			writeError(w, err)
			continue
		}

		err = write(w, resp)
		if err != nil {
			writeError(w, err)
			continue
		}
	}

	log.Print("Exiting")
}

func write(w *bufio.Writer, p string) error {
	_, err := w.Write([]byte(p))
	if err != nil {
		return err
	}
	err = w.Flush()
	if err != nil {
		return err
	}
	return nil
}

func writeError(w *bufio.Writer, err error) {
	_, err = w.Write([]byte(err.Error()))
	if err != nil {
		return
	}

	err = w.Flush()
	if err != nil {
		return
	}
}
