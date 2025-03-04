# goscan

![goscan Screenshot](./goscan.png)

## Overview
Network scanner using ICMP to detect hosts. CLI tool scans interfaces, server provides web UI for host enumeration.

## Install
```bash
go build ./cmd/goscan.go
go build ./web/goscan-server.go
```

## CLI Usage
```bash
# Basic scan
./goscan

# Specific interface
./goscan -i eth0 -t 1000 -m

# Show only alive hosts
./goscan alive -i eth0

# Show only available IPs
./goscan available -i eth0
```

## Server Usage
```bash
./goscan server -l "192.168.1.1" -p "8080" -t 500
```

## Options
### CLI
```
-i, --interface    Network interface
-t, --timeout      Timeout in ms (default: 500)
-m, --measure      Show execution time
-s, --show         Mode: all, alive, available
-q, --scriptable   Raw output
```

### Server
```
-l, --listen-address   Server IP (default: 0.0.0.0)
-p, --listen-port      Port (default: 8080)
-t, --timeout          Timeout in ms (default: 500)
--ssl-cert             SSL certificate file
--ssl-key              SSL key file
--max-subnet-size      Max subnet size (default: 1024)
```

## Requires administrator privileges

## License
MIT License © 2024 Darius Niminenn
