# Binary name
BINARY_NAME=handymkv

# Build directory
BIN_DIR=bin

# Go build command
GO_BUILD=go build

# Main package path
MAIN_PATH=./cmd/handymkv

# Version from git tag (strips leading 'v')
VERSION=$(shell git describe --tags --always 2>/dev/null | sed 's/^v//')
LDFLAGS=-ldflags="-X main.applicationVersion=$(VERSION)"

# Detect current OS and architecture
CURRENT_OS=$(shell go env GOOS)
CURRENT_ARCH=$(shell go env GOARCH)

.PHONY: all clean linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64 current help install

# Default target
.DEFAULT_GOAL := help

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  make current        - Build for current system ($(CURRENT_OS)-$(CURRENT_ARCH))"
	@echo "  make all           - Build for all platforms and architectures"
	@echo "  make linux-amd64   - Build for Linux AMD64"
	@echo "  make linux-arm64   - Build for Linux ARM64"
	@echo "  make darwin-amd64  - Build for macOS AMD64 (Intel)"
	@echo "  make darwin-arm64  - Build for macOS ARM64 (Apple Silicon)"
	@echo "  make windows-amd64 - Build for Windows AMD64"
	@echo "  make windows-arm64 - Build for Windows ARM64"
	@echo "  make install       - Install binary to GOPATH/bin"
	@echo "  make clean         - Remove all built binaries"

# Build for current system
current:
	@echo "Building for current system: $(CURRENT_OS)-$(CURRENT_ARCH)"
	@mkdir -p $(BIN_DIR)
	GOOS=$(CURRENT_OS) GOARCH=$(CURRENT_ARCH) $(GO_BUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME)$(if $(filter windows,$(CURRENT_OS)),.exe,) $(MAIN_PATH)
	@echo "Built: $(BIN_DIR)/$(BINARY_NAME)$(if $(filter windows,$(CURRENT_OS)),.exe,)"

# Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	go install $(MAIN_PATH)
	@echo "Installation complete!"

# Build for all platforms
all: linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64
	@echo "All builds completed successfully!"

# Linux builds
linux-amd64:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BIN_DIR)/linux-amd64
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BIN_DIR)/linux-amd64/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built: $(BIN_DIR)/linux-amd64/$(BINARY_NAME)"

linux-arm64:
	@echo "Building for Linux ARM64..."
	@mkdir -p $(BIN_DIR)/linux-arm64
	GOOS=linux GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o $(BIN_DIR)/linux-arm64/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built: $(BIN_DIR)/linux-arm64/$(BINARY_NAME)"

# macOS builds
darwin-amd64:
	@echo "Building for macOS AMD64 (Intel)..."
	@mkdir -p $(BIN_DIR)/darwin-amd64
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BIN_DIR)/darwin-amd64/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built: $(BIN_DIR)/darwin-amd64/$(BINARY_NAME)"

darwin-arm64:
	@echo "Building for macOS ARM64 (Apple Silicon)..."
	@mkdir -p $(BIN_DIR)/darwin-arm64
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o $(BIN_DIR)/darwin-arm64/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built: $(BIN_DIR)/darwin-arm64/$(BINARY_NAME)"

# Windows builds
windows-amd64:
	@echo "Building for Windows AMD64..."
	@mkdir -p $(BIN_DIR)/windows-amd64
	GOOS=windows GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BIN_DIR)/windows-amd64/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "Built: $(BIN_DIR)/windows-amd64/$(BINARY_NAME).exe"

windows-arm64:
	@echo "Building for Windows ARM64..."
	@mkdir -p $(BIN_DIR)/windows-arm64
	GOOS=windows GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o $(BIN_DIR)/windows-arm64/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "Built: $(BIN_DIR)/windows-arm64/$(BINARY_NAME).exe"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete!"
