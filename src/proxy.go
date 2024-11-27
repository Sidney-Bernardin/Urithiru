package src

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"syscall"

	"github.com/pkg/errors"
)

func StartProxy(ctx context.Context, logger *slog.Logger, urithiruCfg *UrithiruConfig, proxyCfg *ProxyConfig) error {

	// Create backends.
	backends := make([]*backend, len(proxyCfg.Backends))
	for i, backendCfg := range proxyCfg.Backends {
		backends[i] = newBackend(logger, urithiruCfg, proxyCfg, &backendCfg)
	}

	// Create a TCP listener.
	listener, isTLS, err := newListener(proxyCfg)
	if err != nil {
		return errors.Wrap(err, "cannot create listener")
	}
	defer listener.Close()

	logger.Info("Proxy listening", "name", proxyCfg.Name, "address", proxyCfg.Addr, "TLS", isTLS)

	for {

		// Accept next connection.
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				logger.Error("Cannot accept connection: " + err.Error())
				continue
			}
		}

		// Get the best (least busy) backend.
		best := bestBackend(backends)
		if best == nil {
			conn.Close()
			continue
		}

		go func() {
			defer conn.Close()

			// Pipe messages to and from the backend.
			if err := best.pipe(conn); err != nil {
				if !errors.Is(err, net.ErrClosed) && !errors.Is(err, syscall.ECONNRESET) && !errors.Is(err, syscall.EPIPE) {
					logger.Error("Cannot pipe connection to backend: "+err.Error(), "address", best.backendCfg.Addr)
				}
			}
		}()
	}
}

// newListener returns a basic TCP or TLS listener depending on the configuration's certificate fields.
func newListener(proxyCfg *ProxyConfig) (net.Listener, bool, error) {
	if proxyCfg.TLSCert == "" && proxyCfg.TLSKey == "" {

		// Create a basic TCP listener.
		ln, err := net.Listen("tcp", proxyCfg.Addr)
		return ln, false, errors.Wrap(err, "cannot create listener")
	}

	// Load public/private key pair.
	cert, err := tls.LoadX509KeyPair(proxyCfg.TLSCert, proxyCfg.TLSKey)
	if err != nil {
		return nil, false, errors.Wrap(err, "cannot load x509 key pair")
	}

	// Create a TCP listener with TLS.
	ln, err := tls.Listen("tcp", proxyCfg.Addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
	})

	return ln, ln != nil, errors.Wrap(err, "cannot create TLS listener")
}

// bestBackend returns the given backend with the least number of connections.
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
			continue
		}
	}

	return best
}
