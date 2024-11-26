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
		svr := &http.Server{
			Addr:    *pprofAddr,
			Handler: http.DefaultServeMux,
		}

		go func() {
			<-ctx.Done()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := svr.Shutdown(shutdownCtx); err != nil && err != context.Canceled {
				logger.Error("Problem shutting down PPROF server: " + err.Error())
			}
		}()

		err := svr.ListenAndServe()
		return errors.Wrap(err, "cannot serve PPROF")
	}
}
