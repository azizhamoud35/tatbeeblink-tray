# Tatbeeb Link GUI - Installation Guide

## Prerequisites

To build the GUI version of Tatbeeb Link, you need a C compiler (GCC) on Windows.

### Option 1: Install TDM-GCC (Recommended - Easiest)

1. Download TDM-GCC from: https://jmeubank.github.io/tdm-gcc/download/
2. Choose **tdm64-gcc-10.3.0-2.exe** (64-bit)
3. Run the installer
4. Select **"Create"** (not Update)
5. Choose installation directory (default is fine: `C:\TDM-GCC-64`)
6. Check **"Add to PATH"**
7. Complete installation
8. Restart PowerShell/Terminal

### Option 2: Install MSYS2 with MinGW

1. Download MSYS2 from: https://www.msys2.org/
2. Run the installer
3. Open MSYS2 terminal and run:
   ```bash
   pacman -S mingw-w64-x86_64-gcc
   ```
4. Add to PATH: `C:\msys64\mingw64\bin`
5. Restart PowerShell/Terminal

### Option 3: Use Scoop Package Manager (Fastest)

```powershell
# Install Scoop if you don't have it
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
irm get.scoop.sh | iex

# Install GCC
scoop install gcc

# Verify installation
gcc --version
```

## Verify GCC Installation

Open a new PowerShell window and run:

```powershell
gcc --version
```

You should see output like:
```
gcc (tdm64-1) 10.3.0
...
```

## Build the GUI App

Once GCC is installed:

```powershell
cd C:\Users\drabd\OneDrive\DevDrive\JS\Tatbeeb\Link\tray-gui

# Download dependencies
go mod tidy

# Build the GUI app
$env:CGO_ENABLED="1"
go build -ldflags="-s -w -H windowsgui" -o ../bin/TatbeebLink.exe .
```

## Alternative: Pre-built Binary

If you don't want to build from source, you can download a pre-built binary from GitHub Releases (once available).

## Troubleshooting

### Error: "gcc not found"
- Restart your terminal after installing GCC
- Verify GCC is in PATH: `echo $env:PATH`
- Add manually if needed: 
  ```powershell
  $env:PATH += ";C:\TDM-GCC-64\bin"
  ```

### Error: "undefined reference to WinMain"
- Make sure you're using `-H windowsgui` flag
- This tells Go to build a GUI app, not a console app

### Build is slow
- First build will download all dependencies (~100MB)
- Subsequent builds will be much faster

## Next Steps

Once built, you can:
1. Run `TatbeebLink.exe` directly
2. Create a desktop shortcut
3. Add to Windows startup (Settings → Apps → Startup)

