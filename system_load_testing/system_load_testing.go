package main

import (
	"bufio"
	"flag"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

var (
	rps      = flag.Int("rps", 1000, "commands per second")
	address  = flag.String("address", "localhost:8080", "http service address")
	duration = flag.Duration("duration", 600*time.Second, "test duration")
)

var commands = []string{
	"GET user\n",
	"SET user 1\n",
	"SET user 2\n",
	"SET user hello world\n",
	"DEL user\n",
	"DEL user 1\n",
	"GET key123\n",
	"SET key123 foobar\n",
}

const connectionCount = 10

func main() {
	flag.Parse()

	log.Printf("connecting to %s with %d connections", *address, connectionCount)

	conns := make([]net.Conn, connectionCount)
	writers := make([]*bufio.Writer, connectionCount)

	for i := 0; i < connectionCount; i++ {
		conn, err := net.Dial("tcp", *address)
		if err != nil {
			log.Fatalf("dial error on connection %d: %v", i, err)
		}
		conns[i] = conn
		writers[i] = bufio.NewWriter(conn)
	}

	defer func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}()

	log.Printf("starting command stream, rps=%d", *rps)

	ticker := time.NewTicker(time.Second / time.Duration(*rps))
	defer ticker.Stop()
	stop := time.After(*duration)

	var mu sync.Mutex
	connIndex := 0

	for {
		select {
		case <-stop:
			log.Printf("test finished after %s", duration.String())
			return

		case <-ticker.C:
			cmd := commands[rand.Intn(len(commands))]

			mu.Lock()
			w := writers[connIndex]
			connIndex = (connIndex + 1) % connectionCount
			mu.Unlock()

			if _, err := w.WriteString(cmd); err != nil {
				log.Printf("write error: %v", err)
				return
			}
			if err := w.Flush(); err != nil {
				log.Printf("flush error: %v", err)
				return
			}
		}
	}
}
