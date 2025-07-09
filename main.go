package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Colors for terminal output
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorReset  = "\033[0m"
)

// Version of the application
const Version = "1.1.0"

var (
	showHelp     = flag.Bool("h", false, "Show help")
	showHelpLong = flag.Bool("help", false, "Show help")
	showVersion  = flag.Bool("version", false, "Show version")
	maxInputLen  = 256 // Maximum allowed input length
)

type Dependency struct {
	Path    string
	Version string
}

func init() {
	flag.Usage = func() {
		fmt.Printf("%sUsage: goreplace <partial-package-name>%s\n", ColorBlue, ColorReset)
		fmt.Println("Searches for matching dependencies in go.mod and replaces them with local path if found.")
		fmt.Printf("\n%sOptions:%s\n", ColorYellow, ColorReset)
		fmt.Println("  -h, --help      Show this help message")
		fmt.Println("  -version        Show version information")
		fmt.Printf("\n%sExample:%s\n", ColorYellow, ColorReset)
		fmt.Println("  goreplace proto")
	}
}

func main() {
	flag.Parse()

	if *showHelp || *showHelpLong {
		flag.Usage()
		return
	}

	if *showVersion {
		fmt.Printf("goreplace version %s%s%s\n", ColorGreen, Version, ColorReset)
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		printError("missing required argument <partial-package-name>")
		flag.Usage()
		os.Exit(1)
	}

	partialName := args[0]
	if len(partialName) > maxInputLen {
		printError(fmt.Sprintf("input too long (max %d characters)", maxInputLen))
		os.Exit(1)
	}

	modContent, err := os.ReadFile("go.mod")
	if err != nil {
		printError(fmt.Sprintf("error reading go.mod: %v", err))
		os.Exit(1)
	}

	dependencies, replaces := parseGoMod(string(modContent))
	matched := filterDependencies(dependencies, replaces, partialName)

	if len(matched) == 0 {
		fmt.Println("No matches found.")
		return
	}

	selected, err := selectDependency(matched)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	if !confirmSelection(selected) {
		fmt.Println("Operation canceled.")
		return
	}

	localPath, err := findLocalPath(selected)
	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	if err := replaceInGoMod(selected, localPath); err != nil {
		printError(fmt.Sprintf("failed to update go.mod: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Added replace: %s => %s", selected, localPath))
}

func printError(msg string) {
	fmt.Printf("%sError: %s%s\n", ColorRed, msg, ColorReset)
}

func printSuccess(msg string) {
	fmt.Printf("%s%s%s\n", ColorGreen, msg, ColorReset)
}

func suggestAlternativePath(modulePath string) string {
	basePath := removeVersionFromPath(modulePath)
	versionlessPath := filepath.Join(os.Getenv("GOPATH"), "src", basePath)
	if _, err := os.Stat(versionlessPath); err == nil {
		return versionlessPath
	}
	return ""
}

func confirmAlternativePath() bool {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(input)) != "n"
}

func findLocalPath(modulePath string) (string, error) {
	originalPath := filepath.Join(os.Getenv("GOPATH"), "src", modulePath)
	if _, err := os.Stat(originalPath); err == nil {
		return originalPath, nil
	}

	basePath := removeVersionFromPath(modulePath)
	if basePath != modulePath {
		versionlessPath := filepath.Join(os.Getenv("GOPATH"), "src", basePath)
		if _, err := os.Stat(versionlessPath); err == nil {
			return versionlessPath, nil
		}
	}

	return "", fmt.Errorf("local copy not found: tried %s and %s", originalPath,
		filepath.Join(os.Getenv("GOPATH"), "src", basePath))
}

func removeVersionFromPath(path string) string {
	re := regexp.MustCompile(`(/v\d+)$`)
	return re.ReplaceAllString(path, "")
}

func parseGoMod(content string) ([]Dependency, map[string]bool) {
	var dependencies []Dependency
	replaces := make(map[string]bool)

	lines := strings.Split(content, "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "require ("):
			inRequire = true
		case strings.HasPrefix(line, "require ") && !inRequire:
			line = strings.TrimPrefix(line, "require ")
			if dep := parseRequireLine(line); dep != nil {
				dependencies = append(dependencies, *dep)
			}
		case inRequire && line == ")":
			inRequire = false
		case inRequire:
			if dep := parseRequireLine(line); dep != nil {
				dependencies = append(dependencies, *dep)
			}
		case strings.HasPrefix(line, "replace "):
			if path := extractReplacePath(line); path != "" {
				replaces[path] = true
			}
		}
	}

	return dependencies, replaces
}

func parseRequireLine(line string) *Dependency {
	if strings.Contains(line, "indirect") {
		return nil
	}

	if idx := strings.Index(line, "//"); idx != -1 {
		line = strings.TrimSpace(line[:idx])
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	return &Dependency{
		Path:    parts[0],
		Version: parts[1],
	}
}

func extractReplacePath(line string) string {
	parts := strings.Split(line, "=>")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(parts[0], "replace "))
}

func filterDependencies(deps []Dependency, replaces map[string]bool, partialName string) []string {
	var matched []string
	for _, dep := range deps {
		if replaces[dep.Path] {
			continue
		}
		if strings.Contains(dep.Path, partialName) {
			matched = append(matched, dep.Path)
		}
	}
	return matched
}

func selectDependency(matched []string) (string, error) {
	if len(matched) == 1 {
		return matched[0], nil
	}

	fmt.Printf("\n%sMultiple matches found:%s\n", ColorYellow, ColorReset)
	for i, m := range matched {
		fmt.Printf("%s%d) %s%s\n", ColorBlue, i+1, m, ColorReset)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%sEnter the number of the desired package:%s ", ColorYellow, ColorReset)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input")
	}

	input = strings.TrimSpace(input)
	if len(input) > maxInputLen {
		return "", fmt.Errorf("input too long")
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(matched) {
		return "", fmt.Errorf("invalid selection")
	}

	return matched[idx-1], nil
}

func confirmSelection(selected string) bool {
	fmt.Printf("\n%sYou selected:%s %s%s%s\n", ColorYellow, ColorReset, ColorGreen, selected, ColorReset)
	fmt.Printf("%sConfirm selection (press Enter to continue, any other key to cancel):%s ", ColorYellow, ColorReset)
	confirm, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(confirm) == ""
}

func replaceInGoMod(module, localPath string) error {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	newReplace := fmt.Sprintf("\nreplace %s => %s\n", module, localPath)

	tmpFile := "go.mod.tmp"
	if err := os.WriteFile(tmpFile, append(content, newReplace...), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpFile, "go.mod"); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
