param(
    [switch]$Installer
)

$ErrorActionPreference = "Stop"

Write-Host "=== Building bat2exe ===" -ForegroundColor Cyan

# Refresh PATH
$env:Path = [Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [Environment]::GetEnvironmentVariable("Path","User") + ";$env:USERPROFILE\go\bin"

# Generate resources
Write-Host "[1/2] Generating resources..." -ForegroundColor Yellow
go-winres simply --icon winres/icon.png --product-name "bat2exe" --product-version "1.1.0" --file-version "1.1.0" --manifest cli

# Build
Write-Host "[2/2] Building bat2exe.exe..." -ForegroundColor Yellow
go build -o bat2exe.exe -ldflags "-s -w" .

Write-Host "✅ bat2exe.exe built successfully!" -ForegroundColor Green

if ($Installer) {
    # Find ISCC
    $iscc = Get-ChildItem -Path "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe",
        "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
        "C:\Program Files\Inno Setup 6\ISCC.exe" -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty FullName
    
    if ($iscc) {
        Write-Host "[3/3] Building installer..." -ForegroundColor Yellow
        & $iscc installer\bat2exe.iss
        Write-Host "✅ Installer built successfully!" -ForegroundColor Green
    } else {
        Write-Host "⚠️  Inno Setup not found, skipping installer" -ForegroundColor Red
    }
}
