package trojansource

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestScanContent(t *testing.T) {
	findings := ScanContent("const value = 1;\n// safe\u202e code\n", "src/example.go")

	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	finding := findings[0]
	if finding.Line != 2 || finding.Column != 8 || finding.CodePoint != "U+202E" || finding.Name != "RIGHT-TO-LEFT OVERRIDE" {
		t.Fatalf("unexpected finding: %#v", finding)
	}
}

func TestScanContentAllowsVisibleUnicodeAndInitialBOM(t *testing.T) {
	findings := ScanContent("\ufeffconst greeting = \"Hej, värld! 👋\";\n", "src/example.go")
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %#v", findings)
	}
}

func TestFormatFinding(t *testing.T) {
	formatted := FormatFinding(Finding{
		FilePath:  "src/example.go",
		Line:      2,
		Column:    8,
		CodePoint: "U+202E",
		Name:      "RIGHT-TO-LEFT OVERRIDE",
	})
	expected := "src/example.go:2:8: U+202E RIGHT-TO-LEFT OVERRIDE is an invisible or directional formatting character that can disguise source code."
	if formatted != expected {
		t.Fatalf("expected %q, got %q", expected, formatted)
	}
}

func TestScanRepositoryUsesCWDAndAllowlist(t *testing.T) {
	repository := t.TempDir()
	runGit(t, repository, "init", "--quiet")

	file := filepath.Join(repository, "source.go")
	if err := os.WriteFile(file, []byte("const value = 1 // \u202e hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := ScanRepository(Options{CWD: repository})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %#v", findings)
	}

	allowlist := []byte("{\"files\":[\"source.go\"]}\n")
	if err := os.WriteFile(filepath.Join(repository, DefaultAllowlistPath), allowlist, 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err = ScanRepository(Options{CWD: repository})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected allowlisted file to be skipped, got %#v", findings)
	}
}

func runGit(t *testing.T, cwd string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", cwd}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
