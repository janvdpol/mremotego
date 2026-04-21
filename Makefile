# Makefile for MremoteGO

.PHONY: all build build-cli build-gui clean install test run help

# Binary names
CLI_BINARY=mremotego
GUI_BINARY=mremotego-gui
CLI_PATH=cmd/mremotego/main.go
GUI_PATH=cmd/mremotego-gui/main.go cmd/mremotego-gui/theme.go

# Build directory
BUILD_DIR=bin

all: clean build

# Build both CLI and GUI
build: build-cli build-gui

# Build the CLI application
build-cli:
	@echo "Building $(CLI_BINARY)..."
	@go build -o $(BUILD_DIR)/$(CLI_BINARY) $(CLI_PATH)
	@echo "✓ CLI build complete: $(BUILD_DIR)/$(CLI_BINARY)"

# Build the GUI application
build-gui:
	@echo "Building $(GUI_BINARY)..."
	@go build -ldflags="-H windowsgui" -o $(BUILD_DIR)/$(GUI_BINARY) $(GUI_PATH)
	@echo "✓ GUI build complete: $(BUILD_DIR)/$(GUI_BINARY)"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(CLI_BINARY)-linux-amd64 $(CLI_PATH)
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(CLI_BINARY)-darwin-amd64 $(CLI_PATH)
	@GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(CLI_BINARY)-darwin-arm64 $(CLI_PATH)
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(CLI_BINARY)-windows-amd64.exe $(CLI_PATH)
	@echo "✓ CLI multi-platform build complete"
	@echo "Building GUI for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(GUI_BINARY)-linux-amd64 $(GUI_PATH)
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(GUI_BINARY)-darwin-amd64 $(GUI_PATH)
	@GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(GUI_BINARY)-darwin-arm64 $(GUI_PATH)
	@GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui" -o $(BUILD_DIR)/$(GUI_BINARY)-windows-amd64.exe $(GUI_PATH)
	@echo "✓ GUI multi-platform build complete"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
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
