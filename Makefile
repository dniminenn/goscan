GOFLAGS=-ldflags="-s -w" -trimpath

.PHONY: all clean

all:
	CGO_ENABLED=0 go build $(GOFLAGS) -o bin/goscan ./cmd

clean:
	rm -f goscan
