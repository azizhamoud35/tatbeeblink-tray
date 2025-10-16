# Build script for Tatbeeb Link Web GUI
# This creates a Windows executable that opens a web browser interface

Write-Host "🔨 Building Tatbeeb Link (Web Version)..." -ForegroundColor Cyan

# Create bin directory
$binDir = "..\bin"
if (-not (Test-Path $binDir)) {
    New-Item -ItemType Directory -Path $binDir | Out-Null
}

# Download dependencies
Write-Host "📦 Downloading dependencies..." -ForegroundColor Yellow
go mod download

# Build the executable
Write-Host "🏗️  Compiling..." -ForegroundColor Yellow
$env:CGO_ENABLED = "0"
go build -ldflags "-s -w -H windowsgui" -o "$binDir\TatbeebLink-Web.exe" .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Output: $binDir\TatbeebLink-Web.exe" -ForegroundColor Green
    Write-Host "   Size: $([math]::Round((Get-Item "$binDir\TatbeebLink-Web.exe").Length / 1MB, 2)) MB"
    Write-Host ""
    Write-Host "To run: .\bin\TatbeebLink-Web.exe" -ForegroundColor Cyan
    Write-Host "   The app will open in your default browser" -ForegroundColor Gray
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

