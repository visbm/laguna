package tcp

import (
	"bufio"
	"context"
	"io"
	"laguna/internal/config"
	"laguna/internal/mocks"
	"net"
	"sync"
	"testing"
	"time"
)

type MockHandler struct{}

func (h *MockHandler) Handle(_ context.Context, r io.Reader, w io.Writer) error {
	rb := bufio.NewReader(r)
	wb := bufio.NewWriter(w)

	line, err := rb.ReadString('\n')
	if err != nil {
		return err
	}
	_, _ = wb.WriteString("echo: " + line)
	_ = wb.Flush()
	return nil
}

func TestTCPListener_MultiConn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockHandler := &MockHandler{}
	mockLogger := mocks.MockLogger{}

	addr := "127.0.0.1:6060"

	conf := config.Transport{
		Address:        addr,
		MaxConn:        5,
		MaxMessageSize: 1024,
		SemWaitTimeout: 100 * time.Millisecond,
		IdleTimeout:    500 * time.Minute,
	}

	listener := NewListener(&mockLogger, mockHandler, conf)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		listener.Listen(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	conns := make([]net.Conn, 0, conf.MaxConn)
	for i := 0; i < int(conf.MaxConn); i++ {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		conns = append(conns, c)
	}

	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect extra conn: %v", err)
	}
	buf := make([]byte, 1024)
	n, _ := c.Read(buf)
	if string(buf[:n]) != "too many connections" {
		t.Errorf("expected too many connections, got %q", string(buf[:n]))
	}
	_ = c.Close()

	for _, c := range conns {
		_, _ = c.Write([]byte("hello\n"))
		resp := make([]byte, 1024)
		n, _ := c.Read(resp)
		if string(resp[:n]) != "echo: hello\n" {
			t.Errorf("expected echo, got %q", string(resp[:n]))
		}
		_ = c.Close()
	}

	// Закрываем сервер
	listener.Close()
	cancel()

	wg.Wait()
}
