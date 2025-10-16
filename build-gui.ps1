# Tatbeeb Link GUI Builder
# This script helps you install dependencies and build the GUI app

Write-Host "🔗 Tatbeeb Link GUI Builder" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Check if GCC is installed
Write-Host "Checking for GCC..." -ForegroundColor Yellow
$gccVersion = & gcc --version 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ GCC is installed: $($gccVersion[0])" -ForegroundColor Green
} else {
    Write-Host "❌ GCC not found!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Would you like to install GCC using Scoop? (Y/N)" -ForegroundColor Yellow
    $install = Read-Host
    
    if ($install -eq 'Y' -or $install -eq 'y') {
        # Check if Scoop is installed
        $scoopPath = Get-Command scoop -ErrorAction SilentlyContinue
        if (-not $scoopPath) {
            Write-Host "Installing Scoop package manager..." -ForegroundColor Yellow
            Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force
            Invoke-RestMethod get.scoop.sh | Invoke-Expression
        }
        
        Write-Host "Installing GCC..." -ForegroundColor Yellow
        scoop install gcc
        
        # Refresh environment
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
        
        Write-Host "✅ GCC installed successfully!" -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Host "Please install GCC manually:" -ForegroundColor Yellow
        Write-Host "  Option 1: Download TDM-GCC from https://jmeubank.github.io/tdm-gcc/download/" -ForegroundColor White
        Write-Host "  Option 2: Install via Scoop: scoop install gcc" -ForegroundColor White
        Write-Host "  Option 3: Install MSYS2 from https://www.msys2.org/" -ForegroundColor White
        Write-Host ""
        Write-Host "Then run this script again." -ForegroundColor Yellow
        exit 1
    }
}

Write-Host ""
Write-Host "Downloading Go dependencies..." -ForegroundColor Yellow
go mod tidy

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to download dependencies" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Dependencies downloaded" -ForegroundColor Green
Write-Host ""
Write-Host "Building GUI application..." -ForegroundColor Yellow

$env:CGO_ENABLED = "1"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

go build -ldflags="-s -w -H windowsgui" -o ..\bin\TatbeebLink.exe .

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✅ Build successful!" -ForegroundColor Green
    Write-Host ""
    Write-Host "📦 Executable created at: ..\bin\TatbeebLink.exe" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Would you like to run it now? (Y/N)" -ForegroundColor Yellow
    $run = Read-Host
    
    if ($run -eq 'Y' -or $run -eq 'y') {
        Start-Process "..\bin\TatbeebLink.exe"
    }
} else {
    Write-Host ""
    Write-Host "❌ Build failed!" -ForegroundColor Red
    Write-Host "Please check the error messages above." -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "Done! 🎉" -ForegroundColor Green

