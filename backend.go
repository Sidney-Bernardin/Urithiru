package main

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
	urithiruCfg *urithiruConfig
	proxyCfg    *proxyConfig
	backendCfg  *backendConfig

	logger *slog.Logger

	addr *net.TCPAddr
	mu   *sync.Mutex

	isAlive bool
	conns   int
	latency time.Duration

	pingTimeout           time.Duration
	pingInterval          time.Duration
	pingReconnectInterval time.Duration
}

func newBackend(logger *slog.Logger, urithiruCfg *urithiruConfig, proxyCfg *proxyConfig, backendCfg *backendConfig) (*backend, error) {

	addr, err := net.ResolveTCPAddr("tcp", backendCfg.Addr)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve address")
	}

	b := &backend{
		urithiruCfg:           urithiruCfg,
		proxyCfg:              proxyCfg,
		backendCfg:            backendCfg,
		logger:                logger,
		addr:                  addr,
		mu:                    &sync.Mutex{},
		pingTimeout:           or(backendCfg.PingTimeout, proxyCfg.PingTimeout, urithiruCfg.PingTimeout),
		pingInterval:          or(backendCfg.PingInterval, proxyCfg.PingInterval, urithiruCfg.PingInterval),
		pingReconnectInterval: or(backendCfg.PingReconnectInterval, proxyCfg.PingReconnectInterval, urithiruCfg.PingReconnectInterval),
	}

	go b.ping()
	return b, nil
}

func (b *backend) ping() {
beginning:
	b.isAlive = false

	conn, err := net.DialTCP("tcp", nil, b.addr)
	if err != nil {
		time.Sleep(b.pingReconnectInterval)
		goto beginning
	}

	b.logger.Info("Backend connected", "address", b.addr)
	b.isAlive = true

	for {
		conn.SetWriteDeadline(time.Now().Add(b.pingTimeout))

		start := time.Now()
		if _, err := conn.Write(pingMsg); err != nil {
			conn.Close()
			b.logger.Warn("Backend disconnected", "address", b.addr)
			goto beginning
		}
		b.latency = time.Now().Sub(start)

		time.Sleep(b.pingInterval)
	}
}

func (b *backend) pipe(frontConn net.Conn) {

	backConn, _ := net.DialTCP("tcp", nil, b.addr)
	if backConn == nil {
		return
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
}
