# Build script for MremoteGO (GUI + CLI combined)

Write-Host "Building MremoteGO..." -ForegroundColor Green

# Create bin directory if it doesn't exist
if (!(Test-Path -Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
}

# Setup go-gl workaround if needed
if (!(Select-String -Path "go.mod" -Pattern "^replace github.com/go-gl/gl" -Quiet)) {
    Write-Host "Setting up go-gl workaround..." -ForegroundColor Yellow
    if (!(Test-Path "go-gl-temp")) {
        git clone --depth 1 https://github.com/go-gl/gl.git go-gl-temp
    }
    Add-Content -Path "go.mod" -Value ""
    Add-Content -Path "go.mod" -Value "replace github.com/go-gl/gl => ./go-gl-temp"
    go mod tidy
    Write-Host "✓ go-gl workaround configured" -ForegroundColor Green
}

# Enable CGO (required for Fyne GUI and 1Password SDK)
$env:CGO_ENABLED = "1"

# Build the application
# Using -H windowsgui to hide console window on Windows
if ($IsWindows -or $env:OS -match "Windows") {
    go build -ldflags "-H windowsgui" -o mremotego.exe cmd/mremotego-gui/main.go cmd/mremotego-gui/theme.go
    $outputFile = "mremotego.exe"
} else {
    go build -o mremotego cmd/mremotego-gui/main.go cmd/mremotego-gui/theme.go
    $outputFile = "mremotego"
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Build successful: $outputFile" -ForegroundColor Green
    Write-Host "  Run GUI: ./$outputFile" -ForegroundColor Cyan
    Write-Host "  Run CLI: ./$outputFile --help" -ForegroundColor Cyan
    
    # Show file size
    $fileSize = (Get-Item $outputFile).Length / 1MB
    Write-Host "File size: $([math]::Round($fileSize, 2)) MB" -ForegroundColor Gray
} else {
    Write-Host "✗ Build failed" -ForegroundColor Red
    exit 1
}
