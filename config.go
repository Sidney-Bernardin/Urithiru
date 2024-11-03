package main

import "time"

type urithiruConfig struct {
	pingConfig

	Proxies []proxyConfig `toml:"proxies"`
}

type proxyConfig struct {
	pingConfig

	Name     string          `toml:"name"`
	Addr     string          `toml:"addr"`
	Backends []backendConfig `toml:"backends"`
}

type backendConfig struct {
	pingConfig

	Addr string `toml:"addr"`
}

type pingConfig struct {
	PingTimeout           time.Duration `toml:"ping_timeout"`
	PingInterval          time.Duration `toml:"ping_interval"`
	PingReconnectInterval time.Duration `toml:"ping_reconnect_interval"`
}
