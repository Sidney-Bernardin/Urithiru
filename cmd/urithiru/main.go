package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"

	"urithiru/internal"
)

var (
	configPath = flag.String("config", "/etc/urithiru/config.toml", "Path to configuration file.")
	pprofAddr  = flag.String("pprof_addr", ":6060", "Address for the PPROF server.")
)

var pprofServer *http.Server

func main() {
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())

	// Create logger.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	if pprofServer != nil {

		// Start PPROF server.
		go func() {
			defer cancel()

			pprofServer.Addr = *pprofAddr
			pprofServer.Handler = http.DefaultServeMux

			logger.Info("PPROF ready", "addr", *pprofAddr)
			err := pprofServer.ListenAndServe()
			logger.Error("Cannot serve PPROF", "err", err.Error())
		}()
	}

	// Create configuration.
	urithiruCfg, err := internal.NewConfig(*configPath)
	if err != nil {
		logger.Error("Cannot get configuration: "+err.Error(), "path", *configPath)
		return
	}

	// Create proxies.
	proxies := []*internal.Proxy{}
	for _, proxyCfg := range urithiruCfg.Proxies {
		if len(proxyCfg.Backends) == 0 {
			continue
		}

		go func() {
			defer cancel()

			proxy, err := internal.NewProxy(ctx, logger, urithiruCfg, &proxyCfg)
			if err != nil {
				logger.Error("Cannot create proxy", "err", err.Error())
				return
			}
			proxies = append(proxies, proxy)

			slog.Info("Proxy ready",
				"name", proxyCfg.Name,
				"addr", proxyCfg.Addr)

			proxy.Run()
		}()
	}

	// Wait for interrupt signals.
	sigCtx, sigCancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	<-sigCtx.Done()
	sigCancel()

	// Close proxies.
	for _, proxy := range proxies {
		if err := proxy.Close(); err != nil {
			logger.Error("Cannot gracefully close proxy", "err", err.Error())
		}
	}

	if pprofServer != nil {
		fmt.Println("hofhwofj")

		// Shutdown PPROF server.
		logger.Info("Gracefully shutting down PPROF server")
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
		defer shutdownCancel()
		if err := pprofServer.Shutdown(shutdownCtx); err != nil {
			logger.Warn("Cannot gracefully shutdown PPROF", "err", err.Error())
		}
	}
}
