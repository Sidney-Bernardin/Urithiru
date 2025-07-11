package internal

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"syscall"

	"github.com/pkg/errors"
)

type Proxy struct {
	urithiruCfg *UrithiruConfig
	proxyCfg    *ProxyConfig

	backends []*backend
	logger   *slog.Logger
	listener net.Listener
}

func NewProxy(ctx context.Context, logger *slog.Logger, urithiruCfg *UrithiruConfig, proxyCfg *ProxyConfig) (*Proxy, error) {

	// Create backends.
	backends := make([]*backend, len(proxyCfg.Backends))
	for i, backendCfg := range proxyCfg.Backends {
		backends[i] = newBackend(logger, urithiruCfg, proxyCfg, &backendCfg)
	}

	// Create a TCP listener.
	listener, err := newListener(proxyCfg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create listener")
	}

	return &Proxy{
		urithiruCfg: urithiruCfg,
		proxyCfg:    proxyCfg,
		backends:    backends,
		logger:      logger,
		listener:    listener,
	}, nil
}

// Close closes the proxies listener.
func (p *Proxy) Close() error {
	err := p.listener.Close()
	return errors.Wrap(err, "Cannot close listener")
}

// Run continuously accepts connections to be forwarded to a configured backend.
func (p *Proxy) Run() {
	for {

		// Accept next connection.
		conn, err := p.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			p.logger.Error("Cannot accept connection", "err", err.Error())
			continue
		}

		// Choose the best backend.
		best := p.bestBackend()
		if best == nil {
			conn.Close()
			continue
		}

		go func() {
			defer conn.Close()

			if err := best.pipe(conn); err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, syscall.ECONNRESET) && !errors.Is(err, syscall.EPIPE) {
				p.logger.Error("Cannot pipe",
					"err", err.Error(),
					"frontend_addr", conn.RemoteAddr,
					"backend_addr", best.backendCfg.Addr)
			}
		}()
	}
}

// newListener returns a TCP or TLS listener depending on the configuration's certificate fields.
func newListener(proxyCfg *ProxyConfig) (net.Listener, error) {
	if proxyCfg.TLSCert == "" && proxyCfg.TLSKey == "" {

		// Create a basic TCP listener.
		ln, err := net.Listen("tcp", proxyCfg.Addr)
		return ln, errors.Wrap(err, "cannot create listener")
	}

	// Load public/private key pair.
	cert, err := tls.LoadX509KeyPair(proxyCfg.TLSCert, proxyCfg.TLSKey)
	if err != nil {
		return nil, errors.Wrap(err, "cannot load x509 key pair")
	}

	// Create a TCP listener with TLS.
	ln, err := tls.Listen("tcp", proxyCfg.Addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
	})

	return ln, errors.Wrap(err, "cannot create TLS listener")
}

// bestBackend returns the given backend with the least number of connections.
func (p *Proxy) bestBackend() *backend {
	ret := p.backends[0]

	for _, b := range p.backends[1:] {
		if !b.isAlive {
			continue
		}

		if b.conns == 0 {
			return b
		}

		if (b.conns < ret.conns) || (b.conns == ret.conns && b.latency < ret.latency) {
			ret = b
			continue
		}
	}

	return ret
}
