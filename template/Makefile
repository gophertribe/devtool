# Makefile for building the custom build tool
# This replaces the mage-based build system

BINARY_NAME=dev
SOURCE_DIR=cmd/dev

# Build for multiple platforms
build-all: build build-linux build-darwin build-windows

# Build for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	go build -o $(BINARY_NAME) ./$(SOURCE_DIR)

build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 ./$(SOURCE_DIR)

build-darwin:
	@echo "Building $(BINARY_NAME) for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 ./$(SOURCE_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 ./$(SOURCE_DIR)

build-windows:
	@echo "Building $(BINARY_NAME) for Windows..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe ./$(SOURCE_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning $(BINARY_NAME) artifacts..."
	rm $(BINARY_NAME)*

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-all   - Build for all platforms"
	@echo "  build-linux - Build for Linux"
	@echo "  build-darwin - Build for macOS"
	@echo "  build-windows - Build for Windows"
	@echo "  clean       - Clean build artifacts"
	@echo "  deps        - Install dependencies"
	@echo "  help        - Show this help"

.PHONY: build build-all build-linux build-darwin build-windows clean deps help