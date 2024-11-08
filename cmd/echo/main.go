package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
)

func main() {

	addr, ok := os.LookupEnv("ADDRESS")
	if !ok {
		addr = ":8080"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("Cannot create listener: "+err.Error(), "address", addr)
		return
	}

	slog.Info("Listening", "address", addr)

	var (
		mu                   = &sync.Mutex{}
		currentConns         int
		mostConsecutiveConns int
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
			currentConns++
			if currentConns > mostConsecutiveConns {
				mostConsecutiveConns = currentConns
			}
			mu.Unlock()

			go func() {
				defer func() {
					conn.Close()

					mu.Lock()
					currentConns--
					mu.Unlock()
				}()

				if _, err := conn.Write([]byte(addr)); err != nil {
					slog.Error("Cannot write outgoing data: " + err.Error())
					return
				}

				if _, err := io.Copy(conn, conn); err != nil {
					slog.Error("Cannot read incoming data: " + err.Error())
					return
				}
			}()
		}
	}()

	<-ctx.Done()
	listener.Close()

	slog.Info("Goodbye", "most_consecutive_connections", mostConsecutiveConns)
}
