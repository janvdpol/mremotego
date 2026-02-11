#!/bin/bash
# Build script for MremoteGO GUI (Linux/Mac/WSL)

echo "Building MremoteGO GUI..."

# Setup go-gl workaround if needed
if ! grep -q "^replace github.com/go-gl/gl" go.mod; then
    echo "Setting up go-gl workaround..."
    if [ ! -d "go-gl-temp" ]; then
        git clone --depth 1 https://github.com/go-gl/gl.git go-gl-temp
    fi
    echo "" >> go.mod
    echo "replace github.com/go-gl/gl => ./go-gl-temp" >> go.mod
    go mod tidy
    echo "✓ go-gl workaround configured"
fi

# Build the GUI application
go build -o mremotego cmd/mremotego-gui/main.go cmd/mremotego-gui/theme.go

if [ $? -eq 0 ]; then
    echo "✓ Build successful: mremotego"
    
    # Make executable
    chmod +x mremotego
    
    # Show file size
    ls -lh mremotego | awk '{print "File size: " $5}'
else
    echo "✗ Build failed"
    exit 1
fi
