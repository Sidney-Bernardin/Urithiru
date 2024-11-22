package src

import (
	"io"
	"log/slog"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

var pingMsg = []byte(`p`)

type backend struct {
	urithiruCfg *UrithiruConfig
	proxyCfg    *ProxyConfig
	backendCfg  *BackendConfig

	logger *slog.Logger

	pingTimeout           time.Duration
	pingInterval          time.Duration
	pingReconnectInterval time.Duration

	mu *sync.RWMutex

	isAlive bool
	conns   int
	latency time.Duration
}

func newBackend(logger *slog.Logger, urithiruCfg *UrithiruConfig, proxyCfg *ProxyConfig, backendCfg *BackendConfig) *backend {
	b := &backend{
		urithiruCfg:           urithiruCfg,
		proxyCfg:              proxyCfg,
		backendCfg:            backendCfg,
		logger:                logger,
		mu:                    &sync.RWMutex{},
		pingTimeout:           or(backendCfg.PingTimeout, proxyCfg.PingTimeout, urithiruCfg.PingTimeout),
		pingInterval:          or(backendCfg.PingInterval, proxyCfg.PingInterval, urithiruCfg.PingInterval),
		pingReconnectInterval: or(backendCfg.PingReconnectInterval, proxyCfg.PingReconnectInterval, urithiruCfg.PingReconnectInterval),
	}

	go b.ping()
	return b
}

func (b *backend) ping() {
beginning:

	b.mu.Lock()
	b.isAlive = false
	b.mu.Unlock()

	conn, err := net.Dial("tcp", b.backendCfg.Addr)
	if err != nil {
		time.Sleep(b.pingReconnectInterval)
		goto beginning
	}

	b.logger.Info("Backend connected", "address", b.backendCfg.Addr)

	b.mu.Lock()
	b.isAlive = true
	b.mu.Unlock()

	for {
		conn.SetWriteDeadline(time.Now().Add(b.pingTimeout))

		start := time.Now()
		if _, err := conn.Write(pingMsg); err != nil {
			conn.Close()
			b.logger.Warn("Backend unresponsive: "+err.Error(), "address", b.backendCfg.Addr)
			goto beginning
		}

		b.mu.Lock()
		b.latency = time.Now().Sub(start)
		b.mu.Unlock()

		time.Sleep(b.pingInterval)
	}
}

func (b *backend) pipe(frontConn net.Conn) error {

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
		_, err := io.Copy(frontConn, backConn)
		errChan <- err
	}()

	go func() {
		_, err := io.Copy(backConn, frontConn)
		errChan <- err
	}()

	if err := <-errChan; err != nil {
		if !errors.Is(err, net.ErrClosed) && !errors.Is(err, syscall.ECONNRESET) && !errors.Is(err, syscall.EPIPE) {
			return errors.Wrap(err, "cannot copy")
		}
	}

	return nil
}

func or[T comparable](s ...T) (ret T) {
	for _, v := range s {
		if v != ret {
			return v
		}
	}
	return ret
}
