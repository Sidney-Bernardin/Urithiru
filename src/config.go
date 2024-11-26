package src

import (
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type UrithiruConfig struct {
	PingConfig

	Proxies []ProxyConfig `toml:"proxies"`
}

type ProxyConfig struct {
	PingConfig

	Name     string          `toml:"name"`
	Addr     string          `toml:"addr"`
	TLSCert  string          `toml:"tls_cert"`
	TLSKey   string          `toml:"tls_key"`
	Backends []BackendConfig `toml:"backends"`
}

type BackendConfig struct {
	PingConfig

	Addr string `toml:"addr"`
}

type PingConfig struct {
	PingTimeout           time.Duration `toml:"ping_timeout"`
	PingInterval          time.Duration `toml:"ping_interval"`
	PingReconnectInterval time.Duration `toml:"ping_reconnect_interval"`
}

func GetConfig(path string) (*UrithiruConfig, error) {
	cfg := UrithiruConfig{
		PingConfig: PingConfig{
			PingTimeout:           10 * time.Second,
			PingInterval:          1 * time.Second,
			PingReconnectInterval: 1 * time.Second,
		},
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, errors.Wrap(err, "cannot decode configuration file")
	}

	return &cfg, nil
}
