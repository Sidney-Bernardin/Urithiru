package src

import (
	"log/slog"
	"net"

	"github.com/pkg/errors"
)

func StartProxy(logger *slog.Logger, urithiruCfg *UrithiruConfig, proxyCfg *ProxyConfig) error {
	backends := make([]*backend, len(proxyCfg.Backends))

	for i, backendCfg := range proxyCfg.Backends {

		b, err := newBackend(logger, urithiruCfg, proxyCfg, &backendCfg)
		if err != nil {
			return errors.Wrap(err, "cannot create backend")
		}

		backends[i] = b
	}

	addr, err := net.ResolveTCPAddr("tcp", proxyCfg.Addr)
	if err != nil {
		return errors.Wrap(err, "cannot resolve address")
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "cannot create listener")
	}

	logger.Info("Proxy listening", "name", proxyCfg.Name, "address", proxyCfg.Addr)

	for {

		conn, _ := listener.AcceptTCP()
		if conn == nil {
			continue
		}

		go func() {
			defer conn.Close()
			if best := bestBackend(backends); best != nil {
				best.pipe(conn)
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
