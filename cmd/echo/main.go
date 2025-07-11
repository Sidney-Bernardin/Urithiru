package main

import (
	"bytes"
	"flag"
	"log/slog"
	"os"

	"github.com/valyala/fasthttp"
)

var (
	addr         = flag.String("addr", ":8000", "Address to listen on.")
	responseSize = flag.Int("response_size", 1024*1024, "Address to listen on.")
)

func main() {
	flag.Parse()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})))
	response := bytes.Repeat([]byte("a"), *responseSize)

	slog.Info("Ready", "addr", *addr)
	err := fasthttp.ListenAndServe(*addr, func(ctx *fasthttp.RequestCtx) {
		if _, err := ctx.Write(response); err != nil {
			slog.Error("Cannot write response", "err", err.Error())
		}
	})
	slog.Error("Cannot listen and serve", "err", err.Error())
}
