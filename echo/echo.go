package main

import (
	"flag"
	"log/slog"
	"net/http"

	"golang.org/x/net/websocket"
)

var port = flag.String("port", "", "")

func main() {
	flag.Parse()
	err := http.ListenAndServe(":"+*port, websocket.Handler(func(conn *websocket.Conn) {
		for {
			var msg string
			if err := websocket.Message.Receive(conn, &msg); err != nil {
				slog.Error("Cannot receive message", "err", err.Error())
				return
			}

			if err := websocket.Message.Send(conn, []byte(`Hello from `+*port)); err != nil {
				slog.Error("Cannot send message", "err", err.Error())
				return
			}
		}
	}))

	slog.Error(err.Error())
}
