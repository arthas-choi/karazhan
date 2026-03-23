APP_NAME := karazhan
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BUILD_DIR := build

.PHONY: build build-linux build-mac build-all clean run help

## build: Build for current platform
build:
	go build $(LDFLAGS) -o $(APP_NAME) .

## build-linux: Build for Linux amd64
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .

## build-mac: Build for macOS (arm64 + amd64)
build-mac:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .

## build-all: Build for all platforms
build-all: build-linux build-mac

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR) $(APP_NAME)

## run: Build and run
run: build
	./$(APP_NAME)

## vet: Run go vet
vet:
	go vet ./...

## test: Run tests
test:
	go test ./...

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
