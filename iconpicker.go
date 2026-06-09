package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/image/vector"
)

// IconPickResult holds the user's selection
type IconPickResult struct {
	IconName string `json:"iconName"`
	ColorHex string `json:"colorHex"`
	Canceled bool   `json:"canceled"`
}

// pickIconAndColor shows a native Windows WPF window to pick icon and color
func pickIconAndColor() (*IconPickResult, error) {
	psScript := generatePSScript()

	tmpDir, err := os.MkdirTemp("", "bat2exe-iconpicker-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	psFile := filepath.Join(tmpDir, "iconpicker.ps1")
	if err := os.WriteFile(psFile, []byte(psScript), 0644); err != nil {
		return nil, fmt.Errorf("cannot write script: %v", err)
	}

	// Use a temp file for output to avoid PowerShell stream issues
	outFile := filepath.Join(tmpDir, "result.json")

	cmd := exec.Command("powershell.exe",
		"-ExecutionPolicy", "Bypass",
		"-NoProfile",
		"-NoLogo",
		"-File", psFile,
		outFile,
	)
	cmd.Stderr = nil // discard stderr
	cmd.Stdout = nil // discard stdout - result goes to file
	cmd.Run()

	// Read result from file
	result := &IconPickResult{Canceled: true}
	if data, err := os.ReadFile(outFile); err == nil && len(data) > 0 {
		// Strip UTF-8 BOM if present
		data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
		jsonStr := strings.TrimSpace(string(data))
		if jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
				return nil, fmt.Errorf("cannot parse result: %v (file: %s)", err, jsonStr)
			}
		}
	}

	return result, nil
}

func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// generatePSScript creates a PowerShell script with WPF icon picker
func generatePSScript() string {
	names := MDIIconNames()
	sort.Strings(names)

	var iconEntries strings.Builder
	for _, name := range names {
		path := MDIIcons[name]
		iconEntries.WriteString(fmt.Sprintf("    @{Name='%s'; Path='%s'}\n", escapePS(name), escapePS(path)))
	}

	return fmt.Sprintf(`# bat2exe Icon Picker - Native WPF
$ProgressPreference = 'SilentlyContinue'
$InformationPreference = 'SilentlyContinue'
$VerbosePreference = 'SilentlyContinue'
$DebugPreference = 'SilentlyContinue'
$outFile = $args[0]
Add-Type -AssemblyName PresentationFramework, PresentationCore, WindowsBase

$icons = @(
%s
)

$presets = @(
    @{Name='Purple'; Hex='#667eea'}, @{Name='Red';    Hex='#e74c3c'},
    @{Name='Green';  Hex='#2ecc71'}, @{Name='Blue';   Hex='#3498db'},
    @{Name='Orange'; Hex='#f39c12'}, @{Name='Pink';   Hex='#e91e63'},
    @{Name='Teal';   Hex='#1abc9c'}, @{Name='Cyan';   Hex='#00bcd4'},
    @{Name='Lime';   Hex='#8bc34a'}, @{Name='Amber';  Hex='#ffc107'},
    @{Name='DeepPurple'; Hex='#7c4dff'}, @{Name='White'; Hex='#ffffff'}
)

$script:selectedIcon = $null
$script:selectedColor = '#667eea'

# === COLORS ===
$bgDark   = '#1a1b2e'
$bgCard   = '#25264a'
$bgHover  = '#2f3060'
$bgSelect = '#3a3b7e'
$textMain = '#e8e8f0'
$textDim  = '#9899b0'
$textMuted= '#686980'
$accent   = '#667eea'
$borderClr= '#3a3b5e'

# === WINDOW ===
$window = New-Object Windows.Window
$window.Title = 'bat2exe - Icon & Color Picker'
$window.Width = 760; $window.Height = 650
$window.MinWidth = 600; $window.MinHeight = 500
$window.WindowStartupLocation = 'CenterScreen'
$window.Background = $bgDark
$window.FontFamily = 'Segoe UI'
$window.Foreground = $textMain
$window.ResizeMode = 'CanResize'
$window.UseLayoutRounding = $true

# Main Grid rows
$grid = New-Object Windows.Controls.Grid; $grid.Margin = '0'
$grid.RowDefinitions.Add((New-Object Windows.Controls.RowDefinition -Property @{Height='Auto'}))  # 0: title
$grid.RowDefinitions.Add((New-Object Windows.Controls.RowDefinition -Property @{Height='Auto'}))  # 1: search+color
$grid.RowDefinitions.Add((New-Object Windows.Controls.RowDefinition -Property @{Height='*'}))     # 2: icons
$grid.RowDefinitions.Add((New-Object Windows.Controls.RowDefinition -Property @{Height='Auto'}))  # 3: bottom

# === ROW 0: TITLE BAR ===
$titleBar = New-Object Windows.Controls.Border
$titleBar.Background = '#12132a'
$titleBar.Padding = '16,10,16,10'

$titleGrid = New-Object Windows.Controls.Grid
$titleGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='Auto'}))
$titleGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='*'}))
$titleGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='Auto'}))

$iconCount = $icons.Count
$titleText = New-Object Windows.Controls.TextBlock
$titleText.Text = "Select an icon and color for your compiled .exe"
$titleText.FontSize = 15; $titleText.FontWeight = 'SemiBold'
$titleText.Foreground = $accent; $titleText.VerticalAlignment = 'Center'
[Windows.Controls.Grid]::SetColumn($titleText, 0); $titleGrid.Children.Add($titleText)

$countText = New-Object Windows.Controls.TextBlock
$countText.Text = "$iconCount icons"
$countText.FontSize = 12; $countText.Foreground = $textMuted
$countText.VerticalAlignment = 'Center'; $countText.HorizontalAlignment = 'Right'
[Windows.Controls.Grid]::SetColumn($countText, 2); $titleGrid.Children.Add($countText)

$titleBar.Child = $titleGrid
[Windows.Controls.Grid]::SetRow($titleBar, 0); $grid.Children.Add($titleBar)

# === ROW 1: SEARCH + COLOR ===
$toolbar = New-Object Windows.Controls.Border
$toolbar.Background = $bgDark; $toolbar.Padding = '12,8,12,8'

$toolGrid = New-Object Windows.Controls.Grid
$toolGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='200'}))  # search
$toolGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='Auto'}))  # spacer
$toolGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='*'}))     # colors
$toolGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='Auto'}))  # hex

# --- Search ---
$searchBorder = New-Object Windows.Controls.Border
$searchBorder.Background = $bgCard; $searchBorder.CornerRadius = 6
$searchBorder.BorderThickness = '1'; $searchBorder.BorderBrush = $borderClr
$searchBorder.Height = 30

$searchStack = New-Object Windows.Controls.StackPanel; $searchStack.Orientation = 'Horizontal'
$searchStack.Margin = '8,0,0,0'

$searchIcon = New-Object Windows.Controls.TextBlock
$searchIcon.Text = "\U+1F50D"; $searchIcon.FontSize = 13; $searchIcon.VerticalAlignment = 'Center'
$searchIcon.Margin = '0,0,6,0'
$searchStack.Children.Add($searchIcon)

$searchBox = New-Object Windows.Controls.TextBox
$searchBox.Background = 'Transparent'; $searchBox.BorderThickness = '0'
$searchBox.Foreground = $textMain; $searchBox.FontSize = 13
$searchBox.Width = 150; $searchBox.Height = 26
$searchBox.VerticalAlignment = 'Center'; $searchBox.VerticalContentAlignment = 'Center'
$searchBox.CaretBrush = $accent
$searchStack.Children.Add($searchBox)
$searchBorder.Child = $searchStack
[Windows.Controls.Grid]::SetColumn($searchBorder, 0); $toolGrid.Children.Add($searchBorder)

# --- Color presets ---
$colorsPanel = New-Object Windows.Controls.StackPanel; $colorsPanel.Orientation = 'Horizontal'
$colorsPanel.VerticalAlignment = 'Center'; $colorsPanel.Margin = '0,0,0,0'

$colorLabel = New-Object Windows.Controls.TextBlock
$colorLabel.Text = 'Color: '; $colorLabel.FontSize = 12; $colorLabel.Foreground = $textDim
$colorLabel.VerticalAlignment = 'Center'; $colorLabel.Margin = '0,0,6,0'
$colorsPanel.Children.Add($colorLabel)

foreach ($p in $presets) {
    $btn = New-Object Windows.Controls.Button
    $btn.Width = 22; $btn.Height = 22; $btn.Margin = '2,0'; $btn.Cursor = 'Hand'
    $btn.ToolTip = $p.Name; $btn.Tag = $p.Hex; $btn.Background = $p.Hex
    $btn.BorderThickness = '0'; $btn.Padding = '0'
    $btn.Add_Click({
        $h=$this.Tag; $script:selectedColor=$h; $hexInput.Text=$h; UpdatePreview
        # Update search border accent
        $searchBorder.BorderBrush = $h
    })
    $colorsPanel.Children.Add($btn)
}
[Windows.Controls.Grid]::SetColumn($colorsPanel, 2); $toolGrid.Children.Add($colorsPanel)

# --- Hex input ---
$hexPanel = New-Object Windows.Controls.StackPanel; $hexPanel.Orientation = 'Horizontal'
$hexPanel.VerticalAlignment = 'Center'; $hexPanel.Margin = '8,0,0,0'

$hexInput = New-Object Windows.Controls.TextBox
$hexInput.Text = '#667eea'; $hexInput.Width = 80; $hexInput.Height = 26
$hexInput.FontFamily = 'Consolas'; $hexInput.FontSize = 12
$hexInput.Background = $bgCard; $hexInput.Foreground = $accent
$hexInput.BorderThickness = '1'; $hexInput.BorderBrush = $borderClr
$hexInput.CaretBrush = $accent; $hexInput.VerticalContentAlignment = 'Center'
$hexInput.TextAlignment = 'Center'
$hexInput.Add_TextChanged({
    $c = $this.Text.Trim()
    if ($c -match '^#[0-9a-fA-F]{6}$') {
        $script:selectedColor = $c; $searchBorder.BorderBrush = $c; UpdatePreview
    }
})
$hexPanel.Children.Add($hexInput)
[Windows.Controls.Grid]::SetColumn($hexPanel, 3); $toolGrid.Children.Add($hexPanel)

$toolbar.Child = $toolGrid
[Windows.Controls.Grid]::SetRow($toolbar, 1); $grid.Children.Add($toolbar)

# === ROW 2: ICON GRID ===
$iconBorder = New-Object Windows.Controls.Border
$iconBorder.Background = $bgDark; $iconBorder.Padding = '8,0,8,0'
$iconBorder.BorderThickness = '1,0'; $iconBorder.BorderBrush = $borderClr

$scrollViewer = New-Object Windows.Controls.ScrollViewer
$scrollViewer.VerticalScrollBarVisibility = 'Auto'
$scrollViewer.HorizontalScrollBarVisibility = 'Disabled'
$scrollViewer.Background = 'Transparent'

$wrapPanel = New-Object Windows.Controls.WrapPanel
$wrapPanel.Margin = '4,4,4,4'

$script:iconButtons = @{}; $script:allButtons = @()

foreach ($icon in $icons) {
    $btn = New-Object Windows.Controls.Button
    $btn.Width = 66; $btn.Height = 66; $btn.Margin = '3'; $btn.Cursor = 'Hand'
    $btn.ToolTip = $icon.Name; $btn.Tag = $icon.Name; $btn.Background = 'Transparent'
    $btn.BorderThickness = '0'; $btn.Padding = '0'
    $btn.FocusVisualStyle = $null

    $border = New-Object Windows.Controls.Border
    $border.BorderThickness = '2'; $border.BorderBrush = $borderClr
    $border.CornerRadius = 8; $border.Background = $bgCard
    $border.Width = 64; $border.Height = 64

    $sp = New-Object Windows.Controls.StackPanel; $sp.Orientation = 'Vertical'

    $viewbox = New-Object Windows.Controls.Viewbox
    $viewbox.Width = 28; $viewbox.Height = 28; $viewbox.Margin = '6,5,6,1'

    $pathEl = New-Object Windows.Shapes.Path
    $pathEl.Data = [Windows.Media.Geometry]::Parse($icon.Path)
    $pathEl.Fill = $accent; $pathEl.Stretch = 'Uniform'
    $viewbox.Child = $pathEl

    $nameBlock = New-Object Windows.Controls.TextBlock
    $nameBlock.Text = $icon.Name; $nameBlock.FontSize = 7.5; $nameBlock.Foreground = $textDim
    $nameBlock.TextAlignment = 'Center'; $nameBlock.Margin = '2,0,2,3'
    $nameBlock.TextTrimming = 'CharacterEllipsis'; $nameBlock.MaxHeight = 14

    $sp.Children.Add($viewbox); $sp.Children.Add($nameBlock)
    $border.Child = $sp; $btn.Content = $border

    $script:iconButtons[$icon.Name] = @{Button=$btn; Border=$border; Path=$pathEl}
    $script:allButtons += $btn

    $btn.Add_Click({
        $name = $this.Tag; $data = $script:iconButtons[$name]
        foreach ($b in $script:allButtons) { $b.Content.BorderBrush = $borderClr; $b.Content.Background = $bgCard }
        $data.Border.BorderBrush = $script:selectedColor; $data.Border.Background = $bgSelect
        $data.Path.Fill = $script:selectedColor
        $script:selectedIcon = $name; UpdatePreview
    })

    $btn.Add_MouseEnter({
        if ($this.Content.BorderBrush -ne $script:selectedColor) {
            $this.Content.Background = $bgHover
        }
    })
    $btn.Add_MouseLeave({
        if ($this.Content.BorderBrush -ne $script:selectedColor) {
            $this.Content.Background = $bgCard
        }
    })

    $wrapPanel.Children.Add($btn)
}

$scrollViewer.Content = $wrapPanel
$iconBorder.Child = $scrollViewer
[Windows.Controls.Grid]::SetRow($iconBorder, 2); $grid.Children.Add($iconBorder)

# === ROW 3: BOTTOM BAR ===
$bottomBar = New-Object Windows.Controls.Border
$bottomBar.Background = '#12132a'; $bottomBar.Padding = '12,8,12,8'

$bottomGrid = New-Object Windows.Controls.Grid
$bottomGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='Auto'}))  # preview
$bottomGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='*'}))     # spacer
$bottomGrid.ColumnDefinitions.Add((New-Object Windows.Controls.ColumnDefinition -Property @{Width='Auto'}))  # buttons

# --- Preview ---
$previewBorder = New-Object Windows.Controls.Border
$previewBorder.BorderThickness = '1'; $previewBorder.BorderBrush = $borderClr
$previewBorder.CornerRadius = 6; $previewBorder.Padding = '6,4,10,4'
$previewBorder.Background = $bgCard; $previewBorder.Visibility = 'Collapsed'
$previewBorder.Name = 'previewBorder'

$previewStack = New-Object Windows.Controls.StackPanel; $previewStack.Orientation = 'Horizontal'

$previewViewbox = New-Object Windows.Controls.Viewbox
$previewViewbox.Width = 36; $previewViewbox.Height = 36; $previewViewbox.Margin = '0,0,10,0'

$previewPath = New-Object Windows.Shapes.Path
$previewPath.Data = [Windows.Media.Geometry]::Parse('M12,2A10,10 0,0,0 2,12A10,10 0,0,0 12,22A10,10 0,0,0 22,12A10,10 0,0,0 12,2Z')
$previewPath.Fill = $accent; $previewPath.Stretch = 'Uniform'
$previewViewbox.Child = $previewPath

$previewInfo = New-Object Windows.Controls.StackPanel; $previewInfo.VerticalAlignment = 'Center'
$previewName = New-Object Windows.Controls.TextBlock
$previewName.Text = 'No icon selected'; $previewName.FontSize = 13; $previewName.FontWeight = 'SemiBold'; $previewName.Foreground = $textMain
$previewColorText = New-Object Windows.Controls.TextBlock
$previewColorText.Text = 'Color: #667eea'; $previewColorText.FontSize = 10; $previewColorText.Foreground = $textDim
$previewInfo.Children.Add($previewName); $previewInfo.Children.Add($previewColorText)
$previewStack.Children.Add($previewViewbox); $previewStack.Children.Add($previewInfo)
$previewBorder.Child = $previewStack
[Windows.Controls.Grid]::SetColumn($previewBorder, 0); $bottomGrid.Children.Add($previewBorder)

# --- Buttons ---
$btnPanel = New-Object Windows.Controls.StackPanel; $btnPanel.Orientation = 'Horizontal'
$btnPanel.HorizontalAlignment = 'Right'; $btnPanel.VerticalAlignment = 'Center'

$okBtn = New-Object Windows.Controls.Button
$okBtn.Content = 'Apply'; $okBtn.Width = 95; $okBtn.Height = 32; $okBtn.FontSize = 13; $okBtn.FontWeight = 'SemiBold'
$okBtn.Margin = '0,0,8,0'; $okBtn.Cursor = 'Hand'; $okBtn.IsEnabled = $false
$okBtn.Background = $accent; $okBtn.Foreground = 'White'
$okBtn.BorderThickness = '0'
$okBtn.Add_Loaded({
    # Make button corner rounded via template (simplified)
})
$okBtn.Add_Click({
    if ($script:selectedIcon) {
        $r = @{iconName=$script:selectedIcon; colorHex=$script:selectedColor; canceled=$false}
        ($r | ConvertTo-Json -Compress) | Out-File -FilePath $outFile -Encoding ascii -Force
        $window.Close()
    }
})

$cancelBtn = New-Object Windows.Controls.Button
$cancelBtn.Content = 'Cancel'; $cancelBtn.Width = 90; $cancelBtn.Height = 32; $cancelBtn.FontSize = 13
$cancelBtn.Cursor = 'Hand'; $cancelBtn.Background = $bgCard; $cancelBtn.Foreground = $textDim
$cancelBtn.BorderThickness = '1'; $cancelBtn.BorderBrush = $borderClr
$cancelBtn.Add_Click({ '{"canceled":true}' | Out-File -FilePath $outFile -Encoding ascii -Force; $window.Close() })

$btnPanel.Children.Add($okBtn); $btnPanel.Children.Add($cancelBtn)
[Windows.Controls.Grid]::SetColumn($btnPanel, 2); $bottomGrid.Children.Add($btnPanel)

$bottomBar.Child = $bottomGrid
[Windows.Controls.Grid]::SetRow($bottomBar, 3); $grid.Children.Add($bottomBar)

# === FUNCTIONS ===
function UpdatePreview {
    if ($script:selectedIcon -and $script:iconButtons.ContainsKey($script:selectedIcon)) {
        $d = $script:iconButtons[$script:selectedIcon]
        $previewPath.Data = $d.Path.Data; $previewPath.Fill = $script:selectedColor
        $previewName.Text = $script:selectedIcon
        $previewColorText.Text = "Color: $($script:selectedColor)"
        $previewBorder.Visibility = 'Visible'
        $okBtn.IsEnabled = $true; $okBtn.Background = $script:selectedColor
    }
}

# Search with debounce-like behavior (real-time)
$searchBox.Add_TextChanged({
    $q = $this.Text.ToLower(); $visibleCount = 0
    foreach ($icon in $icons) {
        $b = $script:iconButtons[$icon.Name].Button
        if ($icon.Name.ToLower().Contains($q)) {
            $b.Visibility = 'Visible'; $visibleCount++
        } else {
            $b.Visibility = 'Collapsed'
        }
    }
    $countText.Text = "$visibleCount / $iconCount icons"
})

$window.Add_KeyDown({
    if ($_.Key -eq 'Escape') { '{"canceled":true}' | Out-File -FilePath $outFile -Encoding ascii -Force; $window.Close() }
    elseif ($_.Key -eq 'Enter' -and $script:selectedIcon) {
        $r = @{iconName=$script:selectedIcon; colorHex=$script:selectedColor; canceled=$false}
        ($r | ConvertTo-Json -Compress) | Out-File -FilePath $outFile -Encoding ascii -Force; $window.Close()
    }
})

$window.Content = $grid
$window.ShowDialog() | Out-Null
`, iconEntries.String())
}

// ===== Icon Generation (SVG → PNG → .exe embedding) =====

func parseHexColor(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b uint8
	if len(hex) == 6 {
		fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	} else if len(hex) == 3 {
		fmt.Sscanf(hex, "%1x%1x%1x", &r, &g, &b)
		r, g, b = r*17, g*17, b*17
	} else {
		r, g, b = 102, 126, 234
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// renderSVGPathToPNG renders an SVG path to a PNG file with the given color
func renderSVGPathToPNG(pathData string, clr color.RGBA, outputPath string, size int) error {
	scale := float64(size) / 24.0
	ras := vector.NewRasterizer(size, size)

	var cx, cy float64
	var sx, sy float64
	var lastCmd byte
	var prevCPX, prevCPY float64

	data := []byte(pathData)
	i := 0

	for i < len(data) {
		for i < len(data) && (data[i] == ' ' || data[i] == ',' || data[i] == '\n' || data[i] == '\r' || data[i] == '\t') {
			i++
		}
		if i >= len(data) {
			break
		}

		var cmd byte
		if (data[i] >= 'A' && data[i] <= 'Z') || (data[i] >= 'a' && data[i] <= 'z') {
			cmd = data[i]
			i++
			lastCmd = cmd
		} else if lastCmd != 0 {
			cmd = lastCmd
		} else {
			break
		}

		relative := cmd >= 'a' && cmd <= 'z'

		switch cmd {
		case 'M', 'm':
			for {
				x, y, ok := readFloatPair(data, &i)
				if !ok {
					break
				}
				if relative {
					x += cx
					y += cy
				}
				ras.MoveTo(float32(x*scale), float32(y*scale))
				cx, cy = x, y
				sx, sy = cx, cy

				saveI := i
				for i < len(data) && (data[i] == ' ' || data[i] == ',' || data[i] == '\n' || data[i] == '\r' || data[i] == '\t') {
					i++
				}
				if i >= len(data) || (data[i] >= 'A' && data[i] <= 'Z') || (data[i] >= 'a' && data[i] <= 'z') {
					i = saveI
					break
				}
				i = saveI

				if cmd == 'm' {
					cmd = 'l'
				} else {
					cmd = 'L'
				}
			}

		case 'L', 'l':
			for {
				x, y, ok := readFloatPair(data, &i)
				if !ok {
					break
				}
				if relative {
					x += cx
					y += cy
				}
				ras.LineTo(float32(x*scale), float32(y*scale))
				cx, cy = x, y

				saveI := i
				for i < len(data) && (data[i] == ' ' || data[i] == ',' || data[i] == '\n' || data[i] == '\r' || data[i] == '\t') {
					i++
				}
				if i >= len(data) || (data[i] >= 'A' && data[i] <= 'Z') || (data[i] >= 'a' && data[i] <= 'z') {
					i = saveI
					break
				}
				i = saveI
			}

		case 'H', 'h':
			val, ok := readFloatValue(data, &i)
			if !ok {
				break
			}
			if relative {
				cx += val
			} else {
				cx = val
			}
			ras.LineTo(float32(cx*scale), float32(cy*scale))

		case 'V', 'v':
			val, ok := readFloatValue(data, &i)
			if !ok {
				break
			}
			if relative {
				cy += val
			} else {
				cy = val
			}
			ras.LineTo(float32(cx*scale), float32(cy*scale))

		case 'C', 'c':
			p := make([]float64, 6)
			ok := true
			for j := 0; j < 6; j++ {
				val, err := readFloatValue(data, &i)
				if !err {
					ok = false
					break
				}
				if relative {
					if j%2 == 0 {
						p[j] = val + cx
					} else {
						p[j] = val + cy
					}
				} else {
					p[j] = val
				}
			}
			if !ok {
				break
			}
			ras.CubeTo(
				float32(p[0]*scale), float32(p[1]*scale),
				float32(p[2]*scale), float32(p[3]*scale),
				float32(p[4]*scale), float32(p[5]*scale),
			)
			prevCPX, prevCPY = p[2], p[3]
			cx, cy = p[4], p[5]

		case 'S', 's':
			p := make([]float64, 4)
			ok := true
			for j := 0; j < 4; j++ {
				val, err := readFloatValue(data, &i)
				if !err {
					ok = false
					break
				}
				if relative {
					if j%2 == 0 {
						p[j] = val + cx
					} else {
						p[j] = val + cy
					}
				} else {
					p[j] = val
				}
			}
			if !ok {
				break
			}
			rx := cx + (cx - prevCPX)
			ry := cy + (cy - prevCPY)
			ras.CubeTo(
				float32(rx*scale), float32(ry*scale),
				float32(p[0]*scale), float32(p[1]*scale),
				float32(p[2]*scale), float32(p[3]*scale),
			)
			prevCPX, prevCPY = p[0], p[1]
			cx, cy = p[2], p[3]

		case 'Q', 'q':
			p := make([]float64, 4)
			ok := true
			for j := 0; j < 4; j++ {
				val, err := readFloatValue(data, &i)
				if !err {
					ok = false
					break
				}
				if relative {
					if j%2 == 0 {
						p[j] = val + cx
					} else {
						p[j] = val + cy
					}
				} else {
					p[j] = val
				}
			}
			if !ok {
				break
			}
			ras.QuadTo(
				float32(p[0]*scale), float32(p[1]*scale),
				float32(p[2]*scale), float32(p[3]*scale),
			)
			prevCPX, prevCPY = p[0], p[1]
			cx, cy = p[2], p[3]

		case 'T', 't':
			x, y, ok := readFloatPair(data, &i)
			if !ok {
				break
			}
			if relative {
				x += cx
				y += cy
			}
			rx := cx + (cx - prevCPX)
			ry := cy + (cy - prevCPY)
			ras.QuadTo(float32(rx*scale), float32(ry*scale), float32(x*scale), float32(y*scale))
			prevCPX, prevCPY = rx, ry
			cx, cy = x, y

		case 'A', 'a':
			for j := 0; j < 7; j++ {
				readFloatValue(data, &i)
			}
			var ex, ey float64
			if relative {
				ex += cx
				ey += cy
			}
			ras.LineTo(float32(ex*scale), float32(ey*scale))
			cx, cy = ex, ey

		case 'Z', 'z':
			ras.LineTo(float32(sx*scale), float32(sy*scale))
			cx, cy = sx, sy
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), image.Transparent, image.Point{}, draw.Src)
	ras.Draw(img, img.Bounds(), image.NewUniform(clr), image.Point{})

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func readFloatPair(data []byte, i *int) (float64, float64, bool) {
	x, ok := readFloatValue(data, i)
	if !ok {
		return 0, 0, false
	}
	y, ok := readFloatValue(data, i)
	if !ok {
		return 0, 0, false
	}
	return x, y, true
}

func readFloatValue(data []byte, i *int) (float64, bool) {
	for *i < len(data) && (data[*i] == ' ' || data[*i] == ',' || data[*i] == '\n' || data[*i] == '\r' || data[*i] == '\t') {
		*i++
	}
	if *i >= len(data) {
		return 0, false
	}

	start := *i
	if data[*i] == '-' || data[*i] == '+' {
		*i++
	}

	hasDigits := false
	for *i < len(data) && data[*i] >= '0' && data[*i] <= '9' {
		*i++
		hasDigits = true
	}
	if *i < len(data) && data[*i] == '.' {
		*i++
		for *i < len(data) && data[*i] >= '0' && data[*i] <= '9' {
			*i++
			hasDigits = true
		}
	}
	if *i < len(data) && (data[*i] == 'e' || data[*i] == 'E') {
		*i++
		if *i < len(data) && (data[*i] == '-' || data[*i] == '+') {
			*i++
		}
		for *i < len(data) && data[*i] >= '0' && data[*i] <= '9' {
			*i++
		}
	}
	if !hasDigits {
		return 0, false
	}

	var val float64
	fmt.Sscanf(string(data[start:*i]), "%f", &val)
	return val, true
}

// generateCustomIcon creates a custom icon PNG from SVG path + color
func generateCustomIcon(iconName, colorHex, outputDir string) (string, error) {
	pathData, ok := MDIIcons[iconName]
	if !ok {
		for _, v := range MDIIcons {
			pathData = v
			break
		}
	}

	clr := parseHexColor(colorHex)
	pngPath := filepath.Join(outputDir, "icon.png")

	if err := renderSVGPathToPNG(pathData, clr, pngPath, 256); err != nil {
		return "", fmt.Errorf("failed to render icon: %v", err)
	}

	return pngPath, nil
}

// runGoWinres generates .syso files using go-winres
func runGoWinres(iconPngPath, workDir string) error {
	cmd := exec.Command("go-winres", "simply",
		"--icon", iconPngPath,
		"--product-name", "bat2exe",
		"--product-version", version,
		"--file-version", version,
		"--manifest", "cli")
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
