# Binary name
BINARY_NAME=minepulse
VERSION ?= v0.1.0
COMMIT ?= $(shell git rev-parse --short HEAD 2> /dev/null || echo "dev")
BUILD_FLAGS=-ldflags "-s -w -X github.com/PixelEzz/minepulse/cmd.Version=$(VERSION)-$(COMMIT)"

.PHONY: all build clean test lint cross-compile

all: test build

build:
	go build $(BUILD_FLAGS) -o bin/$(BINARY_NAME) main.go

test:
	go test -v -race -cover ./...

clean:
	rm -rf bin/ dist/

# Cross-compilation targets for production releases
cross-compile: clean
	mkdir -p dist
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-linux-arm64 main.go
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe main.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 main.go
