package main

import (
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"
)

var addr = flag.String("addr", ":8000", "Address to listen on.")

func main() {
	flag.Parse()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		b, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			slog.Error("Cannot read request body", "err", err.Error())
			return
		}

		// fmt.Println(string(b))

		if _, err := w.Write(b); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error("Cannot write response", "err", err.Error())
		}
	})

	slog.Info("Ready", "addr", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		slog.Error("Cannot listen and serve", "err", err.Error())
	}
}
