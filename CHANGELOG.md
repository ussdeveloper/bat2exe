# Changelog

## [1.2.0] — 2026-06-09

### Added
- 🎨 **Native WPF Icon Picker** — choose from **120+ Material Design Icons** with custom color
- `mdi_icons.go` — embedded MDI icon data (paths for 120+ icons)
- `iconpicker.go` — PowerShell/WPF native window, icon generation, SVG rasterizer
- `--no-pick-icon` flag — skip the icon picker and use the default icon
- MDI icons attribution and license info in LICENSE and README

### Changed
- Icon picker now opens **automatically by default** during conversion
- Replaced browser-based icon picker with native Windows WPF dark-themed window
- Improved color palette with 12 presets and live preview
- Updated installer version to 1.2.0

### Dependencies
- Added `golang.org/x/image` for SVG path rasterization

## [1.1.0] — 2026-06-09

### Added
- `--audit` / `-a` flag — inspect what's inside any compiled `.exe` (source file, commands, meta tags)
- `--verbose` / `-v` flag — detailed step-by-step execution output with pause at end
- `--help` / `-h` flag — show usage, parameters, and available flags in every compiled exe
- MIT License file (LICENSE)
- Professional installer with Inno Setup (context menu integration for .bat/.cmd files)
- Custom application icon (terminal window design)
- `.gitignore` and `.gitattributes` for repository
- CHANGELOG.md

### Changed
- Generated exes are now console applications (no `-H windowsgui`) — run in the same terminal
- Normal mode: silent execution, only command output is shown (no extra fluff)
- Improved parameter handling — flags (`--verbose`, etc.) are skipped when collecting positional arguments

### Removed
- Removed `-H windowsgui` linker flag from build process

### Fixed
- Terminal no longer closes immediately after double-click — `--verbose` shows "Press Enter to exit..."
- Parameter values are properly substituted in commands using `%name%` syntax

## [1.0.0] — 2026-06-09

### Added
- Initial release
- Convert `.bat` files to standalone `.exe` executables
- Meta tags support: `<<<ask-admin-permissions>>>`, `<<<parameter name default:value>>>`
- Go code generation and compilation
- `--print` flag to preview generated Go code
- `--version` flag
