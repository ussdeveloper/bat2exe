# bat2exe

[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
![Version](https://img.shields.io/badge/version-1.2.0-blue.svg)
[![MDI License](https://img.shields.io/badge/MDI-Apache%202.0-orange.svg)](https://pictogrammers.com/license/)

**bat2exe** converts Windows `.bat` batch files with meta tags into standalone `.exe` executables. No runtime dependencies — the resulting `.exe` is a self-contained Go program that runs on any Windows system.

## Features

- 🔄 **Convert** `.bat` → `.exe` with one command
- 👑 **Auto-elevation** — `<<<ask-admin-permissions>>>` requests admin rights
- ⌨️ **Parameters** — `<<<parameter name default:value>>>` with interactive prompts
- 📋 **Audit** — `--audit` flag shows exactly what's inside any compiled `.exe`
- 🔍 **Verbose** — `--verbose` flag shows step-by-step execution
- 🖼️ **Custom icon** embedded in every build
- 🎨 **Built-in Icon Picker** — choose from **120+ Material Design Icons** with custom color, opens automatically during conversion
- 📦 **Installer** with context menu integration (right-click `.bat` → Convert to EXE)
- ⚡ **No runtime** — standalone executable, zero dependencies

## Quick Start

```bash
# Basic usage
bat2exe script.bat
bat2exe -input script.bat -output custom.exe

# Preview generated Go code
bat2exe -print -input script.bat

# Version
bat2exe --version
```

## Meta Tags

| Tag | Description |
|-----|-------------|
| `<<<ask-admin-permissions>>>` | Request administrator privileges on launch |
| `<<<parameter name default:value>>>` | Define a parameter with optional default |
| `<<<parameter name>>>` | Define a required parameter |

### Meta tag placement

Meta tags must be on their own line:

```bat
@echo off
<<<ask-admin-permissions>>>
<<<parameter server default:localhost>>>
<<<parameter port default:8080>>>
echo "Starting server on %server%:%port%"
```

## Interactive Icon Picker 🎨

The icon picker opens **automatically** during every conversion. A native Windows WPF window lets you:

1. **Browse** 120+ Material Design Icons with live search
2. **Pick** an icon by clicking on it
3. **Choose** a color from presets or type a hex color
4. Click **Apply** — the compiled `.exe` gets your custom icon!

```bash
# Icon picker opens automatically
bat2exe -input script.bat

# Skip icon picker (use default icon)
bat2exe -input script.bat --no-pick-icon
```

> **Note**: Requires `go-winres` for icon embedding.
> Install: `go install github.com/tc-hib/go-winres@latest`

### Icon Preview

| Before | After |
|--------|-------|
| Default bat2exe icon | Your chosen MDI icon + color |

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file.

### Material Design Icons

**bat2exe** includes Material Design Icons by [Pictogrammers](https://pictogrammers.com/).

The icons are licensed under the [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).

```
Material Design Icons by Pictogrammers
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

## Generated `.exe` Flags

Every compiled `.exe` includes these built-in flags:

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Show usage and parameter list |
| `--audit`, `-a` | Show full audit report (source file, commands, meta tags) |
| `--verbose`, `-v` | Show detailed step-by-step execution with pause at end |

### Examples

```bash
# Normal execution — silent, just runs the commands
app.exe

# Verbose — shows each step with pause
app.exe --verbose

# Audit — inspect what the exe contains
app.exe --audit

# Pass parameters
app.exe --verbose myserver 9090
app.exe myserver
```

## Installation

### Option 1 — Installer (recommended)

Download `bat2exe-setup-1.2.0.exe` from the [releases page](../../releases).

The installer:
- Copies `bat2exe.exe` to `Program Files`
- Adds to system `PATH`
- Optionally adds **right-click context menu** for `.bat` and `.cmd` files

### Option 2 — Build from source

```bash
# Prerequisites: Go 1.21+
go install github.com/tc-hib/go-winres@latest
go-winres simply --icon winres/icon.png --product-name "bat2exe" --product-version "1.2.0" --file-version "1.2.0" --manifest cli
go build -o bat2exe.exe -ldflags "-s -w" .
```

## Project Structure

```
bat2exe/
├── main.go                    # Application source
├── go.mod                     # Go module
├── LICENSE                    # MIT License
├── README.md                  # This file
├── bat2exe.exe                # Compiled binary
├── winres/
│   └── icon.png               # Application icon
├── rsrc_windows_amd64.syso    # Icon resources (amd64)
├── rsrc_windows_386.syso      # Icon resources (x86)
└── installer/
    ├── bat2exe.iss            # Inno Setup script
    └── output/
        └── bat2exe-setup-1.1.0.exe
```

## How It Works

1. **Parse** — Read `.bat` file, extract meta tags (`<<<...>>>`) and commands
2. **Generate** — Produce Go source code with all commands embedded as strings
3. **Compile** — Use `go build` to produce a standalone `.exe`

The generated `.exe` contains:
- All original commands
- Parameter handling (CLI args or interactive prompts)
- Admin elevation logic (if `<<<ask-admin-permissions>>>` was used)
- Help, audit, and verbose flags
- Parameter substitution (`%name%` in commands is replaced with actual values)

## License

MIT — see [LICENSE](LICENSE)

