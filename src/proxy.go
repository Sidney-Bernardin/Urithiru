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
		conn, _ := listener.Accept()

		tcpConn, ok := conn.(*net.TCPConn)
		if conn != nil && !ok {
			continue
		}

		go func() {
			defer tcpConn.Close()

			best := bestBackend(backends)
			if best == nil {
				return
			}

			if err := best.pipe(tcpConn); err != nil {
				slog.Error("Cannot pipe connection to backend: "+err.Error(), "address", best.backendCfg.Addr)
			}
		}()
	}
}

func bestBackend(backends []*backend) (best *backend) {
	for _, b := range backends {
		if !b.isAlive {
			continue
		}

		if b.conns == 0 {
			best = b
			break
		}

		if best == nil {
			best = b
			continue
		}

		if (b.conns < best.conns) || (b.conns == best.conns && b.latency < best.latency) {
			best = b
		}
	}

	return best
}
