# Build Tatbeeb Link Tray App - Simplified Port Tunneling Version
# Default DB Port: 9999
# No Database Credentials Required

Write-Host "Building Tatbeeb Link Tray (Simplified Port Tunneling)..." -ForegroundColor Cyan

# Clean old build
if (Test-Path "tatbeeb-link-tray.exe") {
    Remove-Item "tatbeeb-link-tray.exe" -Force
    Write-Host "Cleaned old build" -ForegroundColor Gray
}

# Tidy dependencies
Write-Host "Tidying dependencies..." -ForegroundColor Yellow
go mod tidy

# Build with Windows subsystem (no console window)
Write-Host "Building executable..." -ForegroundColor Yellow
$env:CGO_ENABLED = "0"
go build -ldflags="-H=windowsgui -s -w" -o tatbeeb-link-tray.exe

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Output: tatbeeb-link-tray.exe" -ForegroundColor Cyan
    Write-Host "Size: $((Get-Item tatbeeb-link-tray.exe).Length / 1MB) MB" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Features:" -ForegroundColor Yellow
    Write-Host "  - Simple port tunneling (no database setup needed)" -ForegroundColor Gray
    Write-Host "  - Default port: 9999" -ForegroundColor Gray
    Write-Host "  - Web interface: http://localhost:8765" -ForegroundColor Gray
    Write-Host "  - System tray icon" -ForegroundColor Gray
    Write-Host ""
    Write-Host "To run: .\tatbeeb-link-tray.exe" -ForegroundColor Green
    Write-Host "To copy to bin: Copy-Item tatbeeb-link-tray.exe ..\bin\TatbeebLink-Tray.exe -Force" -ForegroundColor Yellow
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

