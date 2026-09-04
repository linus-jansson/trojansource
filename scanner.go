// Package trojansource detects invisible Unicode characters commonly used in
// Trojan Source attacks.
package trojansource

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const DefaultAllowlistPath = "unicode-security-allowlist.json"

var dangerousCharacters = map[rune]string{
	0x061c: "ARABIC LETTER MARK",
	0x00ad: "SOFT HYPHEN",
	0x200b: "ZERO WIDTH SPACE",
	0x200e: "LEFT-TO-RIGHT MARK",
	0x200f: "RIGHT-TO-LEFT MARK",
	0x202a: "LEFT-TO-RIGHT EMBEDDING",
	0x202b: "RIGHT-TO-LEFT EMBEDDING",
	0x202c: "POP DIRECTIONAL FORMATTING",
	0x202d: "LEFT-TO-RIGHT OVERRIDE",
	0x202e: "RIGHT-TO-LEFT OVERRIDE",
	0x2060: "WORD JOINER",
	0x2066: "LEFT-TO-RIGHT ISOLATE",
	0x2067: "RIGHT-TO-LEFT ISOLATE",
	0x2068: "FIRST STRONG ISOLATE",
	0x2069: "POP DIRECTIONAL ISOLATE",
	0xfeff: "ZERO WIDTH NO-BREAK SPACE",
}

// Finding identifies a dangerous Unicode character in a source file.
type Finding struct {
	FilePath  string
	Line      int
	Column    int
	CodePoint string
	Name      string
}

// ScanContent reports dangerous Unicode characters in content.
func ScanContent(content, filePath string) []Finding {
	findings := []Finding{}
	line, column := 1, 1

	for index, character := range content {
		if character == '\n' {
			line++
			column = 1
			continue
		}

		if name, ok := dangerousCharacters[character]; ok && !(character == 0xfeff && index == 0) {
			findings = append(findings, Finding{
				FilePath:  filePath,
				Line:      line,
				Column:    column,
				CodePoint: fmt.Sprintf("U+%04X", character),
				Name:      name,
			})
		}
		column++
	}

	return findings
}

// FormatFinding formats a finding for compiler-style output.
func FormatFinding(finding Finding) string {
	return fmt.Sprintf(
		"%s:%d:%d: %s %s is an invisible or directional formatting character that can disguise source code.",
		finding.FilePath,
		finding.Line,
		finding.Column,
		finding.CodePoint,
		finding.Name,
	)
}

// Options configures a repository scan.
type Options struct {
	CWD           string
	Mode          string
	AllowlistPath string
}

// ScanRepository scans files in a Git repository. Mode must be "--all" or
// "--staged". An empty option value uses the respective default.
func ScanRepository(options Options) ([]Finding, error) {
	cwd := options.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	mode := options.Mode
	if mode == "" {
		mode = "--all"
	}
	if mode != "--all" && mode != "--staged" {
		return nil, errors.New(`mode must be either "--all" or "--staged"`)
	}

	allowlistedFiles, err := AllowlistedFiles(cwd, options.AllowlistPath)
	if err != nil {
		return nil, err
	}

	files, err := gitFiles(cwd, mode)
	if err != nil {
		return nil, err
	}

	findings := []Finding{}
	for _, filePath := range files {
		if allowlistedFiles[filePath] {
			continue
		}

		content, err := fileContents(cwd, filePath, mode)
		if errors.Is(err, os.ErrNotExist) && mode == "--all" {
			continue
		}
		if err != nil {
			return nil, err
		}
		if strings.IndexByte(string(content), 0) >= 0 {
			continue
		}
		findings = append(findings, ScanContent(string(content), filePath)...)
	}

	return findings, nil
}

// AllowlistedFiles reads repository-relative file paths from the allowlist.
func AllowlistedFiles(cwd, allowlistPath string) (map[string]bool, error) {
	if allowlistPath == "" {
		allowlistPath = DefaultAllowlistPath
	}

	content, err := os.ReadFile(filepath.Join(cwd, allowlistPath))
	if errors.Is(err, os.ErrNotExist) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}

	var config struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}
	if config.Files == nil {
		return nil, fmt.Errorf(`%s must contain a "files" array of repository-relative paths`, allowlistPath)
	}

	files := make(map[string]bool, len(config.Files))
	for _, file := range config.Files {
		files[file] = true
	}
	return files, nil
}

func gitFiles(cwd, mode string) ([]string, error) {
	args := []string{"ls-files", "--cached", "--others", "--exclude-standard", "-z"}
	if mode == "--staged" {
		args = []string{"diff", "--cached", "--name-only", "--diff-filter=ACMR", "-z"}
	}
	output, err := exec.Command("git", append([]string{"-C", cwd}, args...)...).Output()
	if err != nil {
		return nil, fmt.Errorf("list Git files: %w", err)
	}
	return strings.FieldsFunc(string(output), func(character rune) bool {
		return character == 0
	}), nil
}

func fileContents(cwd, filePath, mode string) ([]byte, error) {
	if mode == "--staged" {
		return exec.Command("git", "-C", cwd, "show", ":"+filePath).Output()
	}
	return os.ReadFile(filepath.Join(cwd, filePath))
}
