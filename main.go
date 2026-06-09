package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// MetaTag represents a parsed meta tag from a .bat file
type MetaTag struct {
	Type    string // "ask-admin-permissions", "parameter"
	Name    string
	Default string
}

// ParsedFile represents the parsed contents of a .bat file
type ParsedFile struct {
	MetaTags  []MetaTag
	Commands  []string
	HasParams bool
	Params    []MetaTag
}

const version = "1.2.0"

func main() {
	input := flag.String("input", "", "Path to .bat file")
	output := flag.String("output", "", "Output .exe path (optional)")
	printOnly := flag.Bool("print", false, "Only print generated Go code")
	showVersion := flag.Bool("version", false, "Show version")
	noPickIcon := flag.Bool("no-pick-icon", false, "Skip icon picker and use default icon")
	flag.Parse()

	if *showVersion {
		fmt.Printf("bat2exe v%s\n", version)
		return
	}

	if *input == "" {
		// Check args if no flags provided
		args := flag.Args()
		if len(args) < 1 {
			fmt.Println("Usage: bat2exe -input <file.bat> [-output <file.exe>]")
			fmt.Println("   or:  bat2exe <file.bat>")
			os.Exit(1)
		}
		*input = args[0]
	}

	// Default output name if not specified
	if *output == "" {
		*output = strings.TrimSuffix(*input, ".bat") + ".exe"
	}

	// Convert to absolute paths
	absInput, err := filepath.Abs(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Input path error: %v\n", err)
		os.Exit(1)
	}
	absOutput, err := filepath.Abs(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Output path error: %v\n", err)
		os.Exit(1)
	}

	// Read .bat file
	data, err := os.ReadFile(absInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "File read error: %v\n", err)
		os.Exit(1)
	}

	// Parse content
	parsed := parseBat(string(data))

	// Generate Go code
	goCode := generateGoCode(parsed, absInput)

	// If --print, show code and exit
	if *printOnly {
		fmt.Println(goCode)
		return
	}

	// Save temp .go file
	tmpDir, err := os.MkdirTemp("", "bat2exe-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Temp directory error: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	goFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(goFile, []byte(goCode), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Temp file write error: %v\n", err)
		os.Exit(1)
	}

	// Write go.mod for temp build
	modContent := "module tempbuild\n\ngo 1.21\n"
	modFile := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(modFile, []byte(modContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "go.mod write error: %v\n", err)
		os.Exit(1)
	}

	// By default, show icon picker. Use --no-pick-icon to skip.
	customIconPath := ""
	if !*noPickIcon {
		fmt.Println("🎨 Opening icon picker...")
		result, err := pickIconAndColor()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Icon picker error: %v\n", err)
			os.Exit(1)
		}
		if result.Canceled {
			fmt.Println("Icon selection canceled. Using default icon.")
		} else {
			fmt.Printf("✅ Selected: %s (color: %s)\n", result.IconName, result.ColorHex)
			fmt.Println("   Generating custom icon...")

			// Generate icon PNG
			iconPath, err := generateCustomIcon(result.IconName, result.ColorHex, tmpDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Icon generation error: %v\n", err)
				os.Exit(1)
			}
			customIconPath = iconPath

			// Run go-winres to create .syso files
			fmt.Println("   Embedding icon...")
			if err := runGoWinres(customIconPath, tmpDir); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  go-winres not found or failed. Icon won't be embedded.\n")
				fmt.Fprintf(os.Stderr, "   Install with: go install github.com/tc-hib/go-winres@latest\n")
				customIconPath = ""
			} else {
				fmt.Println("   ✅ Icon embedded successfully!")
			}
		}
	}

	// Compile to .exe (console app, no windowsgui)
	fmt.Printf("Building %s -> %s ...\n", absInput, absOutput)
	cmd := exec.Command("go", "build", "-o", absOutput, "-ldflags", "-s -w", ".")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Build failed. Retrying...")
		cmd2 := exec.Command("go", "build", "-o", absOutput, "-ldflags", "-s -w", ".")
		cmd2.Dir = tmpDir
		cmd2.Stdout = os.Stdout
		cmd2.Stderr = os.Stderr
		if err2 := cmd2.Run(); err2 != nil {
			fmt.Fprintf(os.Stderr, "Build error: %v\n", err2)
			os.Exit(1)
		}
	}

	fmt.Printf("✅ Success! Created: %s\n", absOutput)
}

// parseBat parses .bat content and extracts meta tags and commands
func parseBat(content string) *ParsedFile {
	parsed := &ParsedFile{}
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if line contains a meta tag
		if strings.HasPrefix(line, "<<<") && strings.HasSuffix(line, ">>>") {
			tag := parseMetaTag(line)
			parsed.MetaTags = append(parsed.MetaTags, tag)
			if tag.Type == "parameter" {
				parsed.HasParams = true
				parsed.Params = append(parsed.Params, tag)
			}
		} else {
			parsed.Commands = append(parsed.Commands, line)
		}
	}

	return parsed
}

// parseMetaTag parses a single meta tag
func parseMetaTag(tag string) MetaTag {
	// Remove <<< and >>>
	inner := strings.TrimPrefix(tag, "<<<")
	inner = strings.TrimSuffix(inner, ">>>")
	inner = strings.TrimSpace(inner)

	parts := strings.Fields(inner)
	if len(parts) == 0 {
		return MetaTag{Type: "unknown"}
	}

	mt := MetaTag{Type: parts[0]}

	if mt.Type == "parameter" && len(parts) >= 2 {
		// Extract parameter name and optional default value
		nameWithDefault := parts[1]
		if idx := strings.Index(nameWithDefault, "default:"); idx >= 0 {
			// format: name default:value
			mt.Name = strings.TrimSpace(nameWithDefault[:idx])
			mt.Default = strings.TrimSpace(nameWithDefault[idx+8:])
		} else if len(parts) >= 3 && parts[1] == "default:" {
			// separate words
			mt.Name = ""
			mt.Default = strings.Join(parts[2:], " ")
		} else {
			// maybe "name default:value" combined
			rest := strings.Join(parts[1:], " ")
			if idx := strings.Index(rest, "default:"); idx >= 0 {
				before := strings.TrimSpace(rest[:idx])
				after := strings.TrimSpace(rest[idx+8:])
				if before != "" && !strings.HasPrefix(before, "default") {
					mt.Name = before
				} else {
					mt.Name = parts[1]
				}
				mt.Default = after
			} else {
				mt.Name = parts[1]
			}
		}
	}

	return mt
}

// escapeForTemplate escapes special characters for Go string literals
func escapeForTemplate(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// generateGoCode generates Go source from parsed .bat content
func generateGoCode(parsed *ParsedFile, inputFile string) string {
	funcMap := template.FuncMap{
		"escapeForTemplate": escapeForTemplate,
		"add":              func(a, b int) int { return a + b },
	}
	tmpl := template.Must(template.New("main").Funcs(funcMap).Parse(goTemplate))

	// Prepare template data
	type templateMetaTag struct {
		Name    string
		Default string
		Type    string
	}
	tmplMetaTags := make([]templateMetaTag, len(parsed.MetaTags))
	for i, mt := range parsed.MetaTags {
		tmplMetaTags[i] = templateMetaTag{Name: mt.Name, Default: mt.Default, Type: mt.Type}
	}
	data := struct {
		Commands    []string
		HasParams   bool
		Params      []MetaTag
		HasAdmin    bool
		SourceFile  string
		MetaTags    []templateMetaTag
		Version     string
	}{
		Commands:   parsed.Commands,
		HasParams:  parsed.HasParams,
		Params:     parsed.Params,
		SourceFile: filepath.Base(inputFile),
		MetaTags:   tmplMetaTags,
		Version:    version,
	}

	for _, mt := range parsed.MetaTags {
		if mt.Type == "ask-admin-permissions" {
			data.HasAdmin = true
		}
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}

	return buf.String()
}

const goTemplate = `package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"bufio"
)

func main() {
	verbose := false
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" || arg == "-?" || arg == "/?" {
			printHelp()
			return
		}
		if arg == "--audit" || arg == "-a" {
			printAudit()
			return
		}
		if arg == "--verbose" || arg == "-v" {
			verbose = true
		}
	}

	{{if .HasAdmin}}
	if !isAdmin() {
		if verbose {
			fmt.Println("⚠️  Administrator privileges required!")
			fmt.Println("Attempting to restart with elevated privileges...")
		}
		runAsAdmin()
		return
	}
	if verbose {
		fmt.Println("✅ Running with administrator privileges")
	}
	{{end}}

	{{if .HasParams}}
	// Get parameters from command line or interactive prompt
	{{range $i, $p := .Params}}param{{$i}} := getParam({{add $i 1}}, "{{$p.Name}}", "{{$p.Default}}", verbose)
	{{end}}
	// Set parameters as environment variables
	{{range $i, $p := .Params}}os.Setenv("{{$p.Name}}", param{{$i}})
	{{end}}
	{{end}}

	{{if not .HasParams}}
	if verbose {
	{{end}}
		fmt.Println("⚡ Executing commands from {{.SourceFile}}...")
		fmt.Println(strings.Repeat("=", 50))
	{{if not .HasParams}}
	}
	{{end}}

	{{range $i, $cmd := .Commands}}
	cmdStr_{{$i}} := "{{$cmd | escapeForTemplate}}"
	{{range $j, $p := $.Params}}cmdStr_{{$i}} = strings.ReplaceAll(cmdStr_{{$i}}, "%{{$p.Name}}%", param{{$j}})
	{{end}}
	if verbose {
		{{if $i}}fmt.Println(strings.Repeat("-", 50)){{end}}
		fmt.Printf("[%d/%d] Executing: %s\n", {{$i}} + 1, {{len $.Commands}}, cmdStr_{{$i}})
	}
	if err := runCommand(cmdStr_{{$i}}); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		}
		os.Exit(1)
	}
	{{end}}

	if verbose {
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("✅ All commands completed successfully!")
		fmt.Println("")
		fmt.Println("Press Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

func printHelp() {
	fmt.Println("=== {{.SourceFile}} ===")
	fmt.Println("Usage: {{.SourceFile | escapeForTemplate}} [options] {{range $i, $p := .Params}}[{{$p.Name}}] {{end}}")
	fmt.Println("")
	{{if .HasParams}}
	fmt.Println("Parameters:")
	{{range $i, $p := .Params}}
	{{if $p.Default}}
	fmt.Printf("  %d. %-15s (default: %s)\n", {{add $i 1}}, "{{$p.Name}}", "{{$p.Default}}")
	{{else}}
	fmt.Printf("  %d. %-15s (required)\n", {{add $i 1}}, "{{$p.Name}}")
	{{end}}
	{{end}}
	fmt.Println("")
	{{end}}
	{{if .HasAdmin}}
	fmt.Println("Note: This program requires administrator privileges.")
	{{end}}
	fmt.Println("Flags:")
	fmt.Println("  --help, -h       Show this help message")
	fmt.Println("  --audit, -a      Show compiled source content (commands, meta tags)")
	fmt.Println("  --verbose, -v    Show detailed execution output")
	os.Exit(0)
}

func printAudit() {
	fmt.Println("+------------------------------------------------+")
	fmt.Println("|           bat2exe - Audit Report               |")
	fmt.Println("+------------------------------------------------+")
	fmt.Println("")
	fmt.Printf("  Source file   : {{.SourceFile | escapeForTemplate}}\n")
	fmt.Printf("  Compiled by   : bat2exe v%s\n", "{{.Version}}")
	fmt.Printf("  Total commands: %d\n", {{len .Commands}})
	fmt.Println("")
	{{if .MetaTags}}
	fmt.Println("  +-- Meta Tags ---------------------------------+")
	{{range $i, $mt := .MetaTags}}
	fmt.Printf("  | %-43s |\n", "{{$mt.Type}}{{if $mt.Name}} {{$mt.Name}}{{end}}{{if $mt.Default}} (default: {{$mt.Default}}){{end}}")
	{{end}}
	fmt.Println("  +-----------------------------------------------+")
	fmt.Println("")
	{{end}}
	{{if .HasParams}}
	fmt.Println("  +-- Parameters ---------------------------------+")
	{{range $i, $p := .Params}}
	{{if $p.Default}}
	fmt.Printf("  | %-43s |\n", "{{$p.Name}} (default: {{$p.Default}})")
	{{else}}
	fmt.Printf("  | %-43s |\n", "{{$p.Name}} (required)")
	{{end}}
	{{end}}
	fmt.Println("  +-----------------------------------------------+")
	fmt.Println("")
	{{end}}
	{{if .HasAdmin}}
	fmt.Println("  * Requires Administrator privileges")
	fmt.Println("")
	{{end}}
	fmt.Println("  +-- Commands -----------------------------------+")
	{{range $i, $cmd := .Commands}}
	fmt.Printf("  | %-43s |\n", "{{add $i 1}}. {{$cmd | escapeForTemplate}}")
	{{end}}
	fmt.Println("  +-----------------------------------------------+")
	fmt.Println("")
	fmt.Println("+------------------------------------------------+")
	fmt.Println("|           End of Audit Report                  |")
	fmt.Println("+------------------------------------------------+")
	os.Exit(0)
}

func runCommand(cmdStr string) error {
	cmd := exec.Command("cmd", "/c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getParam(argIdx int, name, def string, verbose bool) string {
	// Skip flag args when looking for positional params
	pos := 0
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--") || strings.HasPrefix(a, "-") {
			continue
		}
		pos++
		if pos == argIdx {
			return a
		}
	}
	if def != "" {
		if verbose {
			fmt.Printf("Enter %s (default: %s): ", name, def)
		}
	} else {
		if verbose {
			fmt.Printf("Enter %s: ", name)
		}
	}
	if !verbose {
		return def
	}
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" && def != "" {
		return def
	}
	return input
}

{{if .HasAdmin}}
func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	return true
}

func runAsAdmin() {
	exe, _ := os.Executable()
	cmd := exec.Command("powershell", "Start-Process", "-FilePath", exe, "-Verb", "runAs")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
}
{{end}}
`


