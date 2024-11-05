package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"urithiru/src"
)

var configPath = flag.String("config", "~/config/urithiru.toml", "Path to configuration file.")

func main() {
	flag.Parse()

	urithiruCfg, err := src.GetConfig(*configPath)
	if err != nil {
		log.Fatalf("Cannot get configuration: %v", err)
	}

	for _, proxyCfg := range urithiruCfg.Proxies {
		for _, backendCfg := range proxyCfg.Backends {
			go func() {
				listener, err := net.Listen("tcp", backendCfg.Addr)
				if err != nil {
					log.Fatalf("Cannot create %s listener: %v", backendCfg.Addr, err)
				}

				log.Printf("%s is listening", backendCfg.Addr)

				for {
					conn, err := listener.Accept()
					if err != nil {
						log.Printf("%s cannot accept connection: %v", backendCfg, err)
						continue
					}

					go func() {
						defer conn.Close()
						if _, err := io.Copy(conn, conn); err != nil {
							log.Printf("%s cannot echo: %v", backendCfg.Addr, err)
						}
					}()
				}
			}()
		}
	}

	<-context.Background().Done()
}
