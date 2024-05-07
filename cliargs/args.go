package cliargs

/*
   CLI argument parsing for the web server.
*/

import (
	"flag"
	"time"
)

type Config struct {
    ListenAddress string
    ListenPort    string
    Timeout       time.Duration
}

const (
    DefaultTimeout = 500
)

func ParseFlags() Config {
    listenAddress := flag.String("listen-address", "0.0.0.0", "IP address for the server to listen on")
    listenPort := flag.String("listen-port", "8080", "Port number for the server to listen on")
    shortAddress := flag.String("l", "", "Short form of IP address for the server to listen on")
    shortPort := flag.String("p", "", "Short form of port number for the server to listen on")
    timeoutValue := flag.Int("timeout", DefaultTimeout, "Timeout value for network operations in milliseconds")
    timeoutValueShort := flag.Int("t", 0, "Short form of timeout value for network operations in milliseconds")

    flag.Parse()

    if *shortAddress != "" {
        *listenAddress = *shortAddress
    }
    if *shortPort != "" {
        *listenPort = *shortPort
    }

    effectiveTimeout := *timeoutValue
    if *timeoutValueShort != 0 {
        effectiveTimeout = *timeoutValueShort
    }

    return Config{
        ListenAddress: *listenAddress,
        ListenPort:    *listenPort,
        Timeout:       time.Duration(effectiveTimeout) * time.Millisecond,
    }
}