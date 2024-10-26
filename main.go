package main

import (
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var pingMsg = []byte(`p`)

type backend struct {
	addr    *net.TCPAddr
	conns   int
	latency time.Duration
	isAlive bool

	mu *sync.Mutex
}

func (b *backend) monitor() {
beginning:
	b.isAlive = false

	conn, err := net.DialTCP("tcp", nil, b.addr)
	if err != nil {
		time.Sleep(1 * time.Second)
		goto beginning
	}

	b.isAlive = true

	for {
		conn.SetWriteDeadline(time.Now().Add(15 * time.Second))

		start := time.Now()
		if _, err := conn.Write(pingMsg); err != nil {
			conn.Close()
			goto beginning
		}
		b.latency = time.Now().Sub(start)

		time.Sleep(1 * time.Second)
	}
}

func (b *backend) pipe(frontConn net.Conn) error {

	backConn, err := net.DialTCP("tcp", nil, b.addr)
	if err != nil {
		return errors.Wrap(err, "cannot dial backend")
	}

	b.mu.Lock()
	b.conns++
	b.mu.Unlock()

	defer func() {
		backConn.Close()

		b.mu.Lock()
		b.conns--
		b.mu.Unlock()
	}()

	doneChan := make(chan any, 1)

	go func() {
		io.Copy(frontConn, backConn)
		doneChan <- nil
	}()

	go func() {
		io.Copy(backConn, frontConn)
		doneChan <- nil
	}()

	<-doneChan
	return nil
}

var backends = []*backend{}

func main() {
	backendAddrs := []string{
		"localhost:8001",
		"localhost:8002",
	}
	useTLS := true

	for _, addr := range backendAddrs {

		addr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			slog.Error("Cannot resolve backend-address", "backend_address", addr, "err", err.Error())
			return
		}

		b := &backend{addr: addr, mu: &sync.Mutex{}}
		go b.monitor()
		backends = append(backends, b)
	}

	listenAddr, err := net.ResolveTCPAddr("tcp", "localhost:8000")
	if err != nil {
		slog.Error("Cannot resolve listen-address", "listen_address", listenAddr, "err", err.Error())
		return
	}

	var listener net.Listener
	if useTLS {

		crt, err := tls.LoadX509KeyPair("public.crt", "private.key")
		if err != nil {
			slog.Error("Cannot load x509 key-pair", "err", err.Error())
			return
		}

		listener, err = tls.Listen("tcp", listenAddr.String(), &tls.Config{Certificates: []tls.Certificate{crt}})
		if err != nil {
			slog.Error("Cannot create listener", "err", err.Error())
			return
		}
	} else {
		listener, err = net.ListenTCP("tcp", listenAddr)
		if err != nil {
			slog.Error("Cannot create listener", "err", err.Error())
			return
		}
	}

	for {
		frontConn, err := listener.Accept()
		if err != nil {
			slog.Error("Cannot accept connection", "err", err.Error())
			continue
		}

		go handleConn(frontConn)
	}
}

func handleConn(frontConn net.Conn) {
	defer frontConn.Close()

	chosen := backends[0]
	for _, b := range backends {
		if !b.isAlive {
			continue
		}

		if b.conns == 0 {
			chosen = b
			break
		}

		if chosen == nil {
			chosen = b
			continue
		}

		if b.conns < chosen.conns {
			chosen = b
		} else if b.conns == chosen.conns && b.latency < chosen.latency {
			chosen = b
		}
	}

	if err := chosen.pipe(frontConn); err != nil {
		slog.Error("Pipe failed", "err", err.Error())
	}
}
