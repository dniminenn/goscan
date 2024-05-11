package cliargs

/*
   CLI argument parsing for the web server.
*/

import (
	"time"

	"github.com/spf13/pflag"
)

type Config struct {
	ListenAddress string
	ListenPort    string
	Timeout       time.Duration
}

const (
	DefaultTimeout = 100
)

func ParseFlags() Config {
	var cfg Config

	pflag.StringVarP(&cfg.ListenAddress, "listen-address", "l", "0.0.0.0", "IP address for the server to listen on")
	pflag.StringVarP(&cfg.ListenPort, "listen-port", "p", "8080", "Port number for the server to listen on")
	pflag.DurationVarP(&cfg.Timeout, "timeout", "t", time.Duration(DefaultTimeout)*time.Millisecond, "Timeout value for network operations")

	pflag.Parse()

	return cfg
}