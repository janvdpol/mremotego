# Build script for MremoteGO (GUI + CLI combined)

Write-Host "Building MremoteGO..." -ForegroundColor Green

# Create bin directory if it doesn't exist
if (!(Test-Path -Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
}

# Embed Windows icon into the executable so it shows when pinned to the taskbar.
# go-winres generates rsrc_windows_*.syso files that go build picks up automatically
# when building a package (not individual files).
if ($IsWindows -or $env:OS -match "Windows") {
    if (Get-Command go-winres -ErrorAction SilentlyContinue) {
        Write-Host "Embedding icon..." -ForegroundColor Cyan
        Push-Location cmd/mremotego-gui
        go-winres simply --icon Icon.png --arch amd64 2>&1 | Out-Null
        Pop-Location
    } else {
        Write-Host "go-winres not found — skipping icon embedding (install with: go install github.com/tc-hib/go-winres@latest)" -ForegroundColor Yellow
    }
}

# Build from the package path (not individual files) so the .syso is included.
# Using -H windowsgui to hide console window on Windows
if ($IsWindows -or $env:OS -match "Windows") {
    go build -ldflags "-H windowsgui" -o mremotego.exe ./cmd/mremotego-gui/
    $outputFile = "mremotego.exe"
} else {
    go build -o mremotego ./cmd/mremotego-gui/
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
