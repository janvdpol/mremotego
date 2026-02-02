# Makefile for MremoteGO

.PHONY: all build clean install test run help

# Binary name
BINARY=mremotego
BINARY_PATH=cmd/mremotego-gui/main.go cmd/mremotego-gui/theme.go

# Build directory
BUILD_DIR=bin

all: clean build

# Build the application (handles both CLI and GUI)
build:
	@echo "Building $(BINARY)..."
	@CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY) $(BINARY_PATH)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY)"
	@echo "  Run GUI: $(BUILD_DIR)/$(BINARY)"
	@echo "  Run CLI: $(BUILD_DIR)/$(BINARY) --help"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(BINARY_PATH)
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 $(BINARY_PATH)
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(BINARY_PATH)
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe $(BINARY_PATH)
	@echo "✓ Multi-platform build complete"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY) $(BINARY).exe
	@echo "Clean complete"

# Install to GOPATH/bin
install:
	@echo "Installing $(CLI_BINARY)..."
	@go install $(CLI_PATH)
	@echo "✓ CLI install complete"
	@echo "Note: GUI must be run from built binary"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run the CLI application
run:
	@go run $(CLI_PATH)

# Run the GUI application
run-gui:
	@go run $(GUI_PATH)

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build both CLI and GUI applications"
	@echo "  build-cli   - Build only the CLI application"
	@echo "  build-gui   - Build only the GUI application"
	@echo "  build-all   - Build for multiple platforms"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install CLI to GOPATH/bin"
	@echo "  test        - Run tests"
	@echo "  run         - Run the CLI application"
	@echo "  run-gui     - Run the GUI application"
	@echo "  help        - Show this help message"
