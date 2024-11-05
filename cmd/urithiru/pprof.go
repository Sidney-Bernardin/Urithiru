//go:build pprof
// +build pprof

package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/pkg/errors"
)

func init() {
	servePPROF = func() error {
		err := http.ListenAndServe(*pprofAddr, nil)
		return errors.Wrap(err, "cannot serve pprof")
	}
}
