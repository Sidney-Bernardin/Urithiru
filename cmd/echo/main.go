package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
)

var addr = flag.String("addr", ":8080", "")

func main() {
	flag.Parse()

	v, ok := os.LookupEnv("ADDRESS")
	if ok {
		*addr = v
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		slog.Error("Cannot create listener: "+err.Error(), "address", addr)
		return
	}

	slog.Info("Listening", "address", *addr)

	var (
		mu                  = &sync.Mutex{}
		conns               int
		mostConcurrentConns int
	)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return

				default:
					slog.Warn("Cannot accept connection: " + err.Error())
					continue
				}
			}

			mu.Lock()
			conns++
			slog.Info("New connection", "current_connections", conns)
			if conns > mostConcurrentConns {
				mostConcurrentConns = conns
			}
			mu.Unlock()

			go func() {
				defer func() {
					conn.Close()

					mu.Lock()
					conns--
					slog.Info("Connection closed", "current_connections", conns)
					mu.Unlock()
				}()

				if _, err := conn.Write([]byte(*addr)); err != nil {
					slog.Error("Cannot write outgoing data: " + err.Error())
					return
				}

				if _, err := io.Copy(conn, conn); err != nil {
					slog.Error("Cannot echo data: " + err.Error())
					return
				}
			}()
		}
	}()

	<-ctx.Done()
	listener.Close()

	slog.Info("Goodbye", "most_concurrent_connections", mostConcurrentConns)
}
