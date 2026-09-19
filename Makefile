BINARY_NAME=why
BUILD_DIR=bin

.PHONY: all build test clean install lint help

all: test build

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-s -w -X main.Version=0.1.0 -X main.BuildDate=$(shell date +'%Y-%m-%d')" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/why

test:
	go test -v -race ./...

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR)
	go clean

help:
	@echo "Available targets:"
	@echo "  build    - Compile the why CLI binary"
	@echo "  test     - Run unit tests with race detection"
	@echo "  install  - Install binary into GOPATH/bin"
	@echo "  clean    - Remove build artifacts"
