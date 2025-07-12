package internal

import (
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var pingMsg = []byte(`p`)

type backend struct {
	urithiruCfg *UrithiruConfig
	proxyCfg    *ProxyConfig
	backendCfg  *BackendConfig

	pingTimeout  time.Duration
	pingInterval time.Duration

	logger  *slog.Logger
	mu      *sync.Mutex
	isAlive bool
	conns   int
	latency time.Duration
}

func newBackend(logger *slog.Logger, urithiruCfg *UrithiruConfig, proxyCfg *ProxyConfig, backendCfg *BackendConfig) *backend {
	if backendCfg.PingTimeout == 0 {
		backendCfg.PingTimeout = proxyCfg.PingTimeout
	}
	if backendCfg.PingInterval == 0 {
		backendCfg.PingInterval = proxyCfg.PingInterval
	}

	b := &backend{
		urithiruCfg: urithiruCfg,
		proxyCfg:    proxyCfg,
		backendCfg:  backendCfg,
		logger:      logger,
		mu:          &sync.Mutex{},
	}

	go b.ping()
	return b
}

// ping monitors the backend server's health and latency by repeatedly writing it pings messages.
func (b *backend) ping() {
	for {
		b.mu.Lock()
		b.isAlive = false
		b.mu.Unlock()

		// Connect to the backend server.
		conn, err := net.Dial("tcp", b.backendCfg.Addr)
		if err != nil {
			b.logger.Error("Cannot dial backend",
				"err", err.Error(),
				"backend_addr", b.backendCfg.Addr)
			time.Sleep(b.pingInterval)
			continue
		}

		b.logger.Info("Backend connected", "backend_addr", b.backendCfg.Addr)

		b.mu.Lock()
		b.isAlive = true
		b.mu.Unlock()

		// Ping loop.
		for {
			conn.SetWriteDeadline(time.Now().Add(b.pingTimeout))
			start := time.Now()

			if _, err := conn.Write(pingMsg); err != nil {
				conn.Close()
				b.logger.Warn("Cannot ping backend:",
					"err", err.Error(),
					"address", b.backendCfg.Addr)
				break
			}

			b.mu.Lock()
			b.latency = time.Now().Sub(start)
			b.mu.Unlock()

			time.Sleep(b.pingInterval)
		}
	}
}

// pip copies data to and from the backend and frontend connections.
func (b *backend) pipe(frontConn net.Conn) error {

	// Connect to the backend server.
	backConn, _ := net.Dial("tcp", b.backendCfg.Addr)
	if backConn == nil {
		return nil
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

	errChan := make(chan error, 1)

	go func() {
		_, err := io.CopyBuffer(frontConn, backConn, make([]byte, b.proxyCfg.BufferSize))
		errChan <- err
	}()

	go func() {
		_, err := io.CopyBuffer(backConn, frontConn, make([]byte, b.proxyCfg.BufferSize))
		errChan <- err
	}()

	return errors.Wrap(<-errChan, "cannot copy")
}
