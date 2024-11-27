package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	_ "net/http/pprof"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"urithiru/src"
)

var (
	configPath = flag.String("config", "/etc/urithiru/config.toml", "Path to configuration file.")
	pprofAddr  = flag.String("pprof_addr", ":6060", "Address for PPROF server to listen on.")
)

var servePPROF func(context.Context, *slog.Logger) error

func main() {
	flag.Parse()
	errs, ctx := errgroup.WithContext(context.Background())

	// Create a logger.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	if servePPROF != nil {

		// Serve PPROF in a new goroutine.
		errs.Go(func() error { return servePPROF(ctx, logger) })
	}

	// Get the configuration.
	urithiruCfg, err := src.GetConfig(*configPath)
	if err != nil {
		logger.Error("Cannot get configuration: "+err.Error(), "config_path", *configPath)
		return
	}

	// Start each proxy in it's own new goroutine.
	for _, proxyCfg := range urithiruCfg.Proxies {
		errs.Go(func() error {
			err := src.StartProxy(ctx, logger, urithiruCfg, &proxyCfg)
			return errors.Wrap(err, "cannot serve proxy")
		})
	}

	if err := errs.Wait(); err != nil {
		l := logger

		switch errors.Cause(err).(type) {
		case *net.AddrError:
			l = logger.With("config_path", *configPath)
		}

		l.Error(err.Error())
	}
}
