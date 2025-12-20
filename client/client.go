package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"io"
	"laguna/common/logger"
	"laguna/internal/config"
	"laguna/internal/transport/tcp"
	"laguna/utils/data_type"
	"log"
	"os"
	"syscall"
	"time"
)

var (
	address     = flag.String("address", "localhost:8080", "http service address")
	size        = flag.Int("size", 1024, "max response size")
	idleTimeout = flag.Duration("deadline", time.Minute*5, "deadline for client connection")
)

// go run client.go -address=localhost:8080
func main() {
	flag.Parse()

	conf := config.Client{
		Address:     *address,
		MaxRespSize: data_type.ByteSize(*size),
		Deadline:    *idleTimeout,
	}

	loggerConf := config.Logger{
		Level: "info",
		Out:   "stdout",
	}
	lg := logger.New(loggerConf)

	cli, err := tcp.NewClient(conf, lg)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)

	log.Print("New laguna cli")

	ctx, cancel := context.WithTimeout(context.Background(), conf.Deadline)
	defer ctx.Done()
	defer cancel()

	for {
		_ = write(w, []byte("> "))

		line, err := readLine(r)
		if err != nil {
			writeError(w, err)
			continue
		}

		resp, err := cli.Send(ctx, line)
		if err != nil {
			if errors.Is(err, syscall.EPIPE) {
				log.Print("connection closed")
				break
			}
			writeError(w, err)
			continue
		}

		l, err := readLine(resp)
		if err != nil {
			writeError(w, err)
			continue
		}

		err = write(w, l)
		if err != nil {
			writeError(w, err)
			continue
		}
	}

	log.Print("Exiting")
}

func readLine(r io.Reader) ([]byte, error) {
	br := bufio.NewReader(r)

	line, err := br.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	return line, nil
}

func write(w *bufio.Writer, p []byte) error {
	_, err := w.Write(p)
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
