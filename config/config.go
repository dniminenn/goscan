package config

import (
	"sync"
	"time"
)

type ServerConfig struct {
	ListenAddress string
	ListenPort    string
	Timeout       time.Duration
	SSLCertFile   string
	SSLKeyFile    string
	MaxSubnetSize int
}

var (
	serverConfig ServerConfig
	configMutex  sync.RWMutex
)

func init() {
	serverConfig = ServerConfig{
		ListenAddress: "0.0.0.0",
		ListenPort:    "8080",
		Timeout:       50 * time.Millisecond,
		MaxSubnetSize: 1024,
	}
}

func GetServerConfig() ServerConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return serverConfig
}

func SetServerConfig(config ServerConfig) {
	configMutex.Lock()
	defer configMutex.Unlock()
	serverConfig = config
}
