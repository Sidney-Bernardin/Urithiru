//go:build pprof
// +build pprof

package main

import (
	"net/http"
)

func init() {
	svr := &http.Server{}
}
