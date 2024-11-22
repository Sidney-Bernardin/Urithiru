package src

import (
	"log/slog"
	"net"

	"github.com/pkg/errors"
)

func StartProxy(logger *slog.Logger, urithiruCfg *UrithiruConfig, proxyCfg *ProxyConfig) error {

	backends := make([]*backend, len(proxyCfg.Backends))
	for i, backendCfg := range proxyCfg.Backends {
		backends[i] = newBackend(logger, urithiruCfg, proxyCfg, &backendCfg)
	}

	listener, err := net.Listen("tcp", proxyCfg.Addr)
	if err != nil {
		return errors.Wrap(err, "cannot create listener")
	}

	logger.Info("Proxy listening", "name", proxyCfg.Name, "address", proxyCfg.Addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("Cannot accept connection: "+err.Error())
			continue
		}

		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			conn.Close()
			continue
		}

		best := bestBackend(backends)
		if best == nil {
			conn.Close()
			continue
		}

		go func() {
			defer tcpConn.Close()


			if err := best.pipe(tcpConn); err != nil {
				logger.Error("Cannot pipe connection to backend: "+err.Error(), "address", best.backendCfg.Addr)
			}
		}()
	}
}

func bestBackend(backends []*backend) (best *backend) {
	for _, b := range backends {
		b.mu.RLock()

		if !b.isAlive {
			b.mu.RUnlock()
			continue
		}

		if b.conns == 0 {
			b.mu.RUnlock()
			best = b
			break
		}

		if best == nil {
			b.mu.RUnlock()
			best = b
			continue
		}

		if (b.conns < best.conns) || (b.conns == best.conns && b.latency < best.latency) {
			b.mu.RUnlock()
			best = b
			continue
		}

		b.mu.RUnlock()
	}

	return best
}
