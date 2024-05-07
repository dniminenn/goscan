package cliargs

/*
   CLI argument parsing for the web server.
*/

import (
	"flag"
)

type Config struct {
    ListenAddress string
    ListenPort    string
}

func ParseFlags() Config {
    listenAddress := flag.String("listen-address", "0.0.0.0", "IP address for the server to listen on")
    listenPort := flag.String("listen-port", "8080", "Port number for the server to listen on")
    shortAddress := flag.String("l", "", "Short form of IP address for the server to listen on")
    shortPort := flag.String("p", "", "Short form of port number for the server to listen on")

    flag.Parse()

    if *shortAddress != "" {
        *listenAddress = *shortAddress
    }
    if *shortPort != "" {
        *listenPort = *shortPort
    }

    return Config{
        ListenAddress: *listenAddress,
        ListenPort:    *listenPort,
    }
}
