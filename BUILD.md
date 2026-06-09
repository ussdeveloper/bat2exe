# Building

## Prerequisites

- Go 1.21+
- go-winres (`go install github.com/tc-hib/go-winres@latest`)
- Inno Setup 6 (for installer)

## Build bat2exe.exe

```powershell
# 1. Generate icon resources
go-winres simply --icon winres/icon.png --product-name "bat2exe" --product-version "1.1.0" --file-version "1.1.0" --manifest cli

# 2. Build
go build -o bat2exe.exe -ldflags "-s -w" .
```

## Build installer

```powershell
# Using Inno Setup CLI
& "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" installer\bat2exe.iss
```
