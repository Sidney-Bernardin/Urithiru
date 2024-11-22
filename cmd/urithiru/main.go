package main

import (
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
	pprofAddr  = flag.String("pprof_addr", ":6060", "")
)

var servePPROF func() error

func main() {
	flag.Parse()
	errs := &errgroup.Group{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	if servePPROF != nil {
		logger.Info("PPROF listening", "address", *pprofAddr)
		errs.Go(servePPROF)
	}

	urithiruCfg, err := src.GetConfig(*configPath)
	if err != nil {
		logger.Error("Cannot get configuration: "+err.Error(), "config_path", *configPath)
		return
	}

	for _, proxyCfg := range urithiruCfg.Proxies {
		errs.Go(func() error {
			err := src.StartProxy(logger, urithiruCfg, &proxyCfg)
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
