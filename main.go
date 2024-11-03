package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var configPath = flag.String("config", "~/config/urithiru.toml", "")

func main() {
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	urithiruCfg := urithiruConfig{
		pingConfig: pingConfig{
			PingTimeout:           10 * time.Second,
			PingInterval:          1 * time.Second,
			PingReconnectInterval: 1 * time.Second,
		},
	}

	if _, err := toml.DecodeFile(*configPath, &urithiruCfg); err != nil {
		logger.Error(fmt.Sprintf("Cannot parse config file: %v", err), "config_path", configPath)
		return
	}

	errs := &errgroup.Group{}
	for _, proxyCfg := range urithiruCfg.Proxies {
		errs.Go(func() error {
			return startProxy(logger, &urithiruCfg, &proxyCfg)
		})
	}

	if err := errs.Wait(); err != nil {
		l := logger

		switch errors.Cause(err).(type) {
		case *net.AddrError:
			l = logger.With("config_path", *configPath)
		}

		l.Error(fmt.Sprintf("Proxy error: %v", err))
	}
}

func startProxy(logger *slog.Logger, urithiruCfg *urithiruConfig, proxyCfg *proxyConfig) error {
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
		return errors.Wrap(err, "cannot listen")
	}

	logger.Info(fmt.Sprintf("Proxy %s listening on %s", proxyCfg.Name, proxyCfg.Addr))

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

func or[T comparable](s ...T) (ret T) {
	for _, v := range s {
		if v != ret {
			return v
		}
	}
	return ret
}
