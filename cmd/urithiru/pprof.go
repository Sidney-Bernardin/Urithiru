//go:build pprof
// +build pprof

package main

import (
	"context"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/pkg/errors"
)

func init() {
	servePPROF = func(ctx context.Context, logger *slog.Logger) error {

		// Create PPROF server.
		svr := &http.Server{
			Addr:    *pprofAddr,
			Handler: http.DefaultServeMux,
		}

		// Awaits the context, to gracefully shutdown the sever in a new goroutine.
		go func() {
			<-ctx.Done()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Shutdown the server.
			if err := svr.Shutdown(shutdownCtx); err != nil && err != context.Canceled {
				logger.Warn("Problem shutting down PPROF server: " + err.Error())
			}
		}()

		// Start the server.
		logger.Info("PPROF listening", "address", *pprofAddr)
		err := svr.ListenAndServe()
		return errors.Wrap(err, "cannot serve PPROF")
	}
}
