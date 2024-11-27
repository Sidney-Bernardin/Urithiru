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

// GetConfig returns a new UrithiruConfig resembling the contents of the given file.
func GetConfig(cfgPath string) (*UrithiruConfig, error) {
	cfg := UrithiruConfig{
		PingConfig: PingConfig{
			PingTimeout:           10 * time.Second,
			PingInterval:          1 * time.Second,
			PingReconnectInterval: 1 * time.Second,
		},
	}

	// Decode the config file.
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		return nil, errors.Wrap(err, "cannot decode configuration file")
	}

	return &cfg, nil
}
