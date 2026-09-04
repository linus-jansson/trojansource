package trojansource

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestScanContentDetectsEveryDangerousCharacter(t *testing.T) {
	for character, name := range dangerousCharacters {
		t.Run(name, func(t *testing.T) {
			findings := ScanContent("a"+string(character), "source.go")
			expected := []Finding{{
				FilePath:  "source.go",
				Line:      1,
				Column:    2,
				CodePoint: fmt.Sprintf("U+%04X", character),
				Name:      name,
			}}
			assertFindings(t, findings, expected)
		})
	}
}

func TestScanContentHandlesUnicodePositionsAndLineEndings(t *testing.T) {
	content := "\ufeffemoji 👋\u202e\r\nnext\u200b\n\ufeff"
	findings := ScanContent(content, "source.go")

	assertFindings(t, findings, []Finding{
		{
			FilePath: "source.go", Line: 1, Column: 9,
			CodePoint: "U+202E", Name: "RIGHT-TO-LEFT OVERRIDE",
		},
		{
			FilePath: "source.go", Line: 2, Column: 5,
			CodePoint: "U+200B", Name: "ZERO WIDTH SPACE",
		},
		{
			FilePath: "source.go", Line: 3, Column: 1,
			CodePoint: "U+FEFF", Name: "ZERO WIDTH NO-BREAK SPACE",
		},
	})
}

func TestScanContentPermitsVisibleUnicodeAndInitialBOM(t *testing.T) {
	findings := ScanContent("\ufeffconst greeting = \"Hej, värld! 👋\";\n", "source.go")
	assertFindings(t, findings, []Finding{})
}

func TestFormatFinding(t *testing.T) {
	formatted := FormatFinding(Finding{
		FilePath: "source.go", Line: 2, Column: 8,
		CodePoint: "U+202E", Name: "RIGHT-TO-LEFT OVERRIDE",
	})
	expected := "source.go:2:8: U+202E RIGHT-TO-LEFT OVERRIDE is an invisible or directional formatting character that can disguise source code."
	if formatted != expected {
		t.Fatalf("expected %q, got %q", expected, formatted)
	}
}

func TestAllowlistedFiles(t *testing.T) {
	directory := t.TempDir()

	files, err := AllowlistedFiles(directory, DefaultAllowlistPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("expected missing allowlist to be empty, got %#v", files)
	}

	writeFile(t, filepath.Join(directory, "custom.json"), `{"files":["a.go","nested/b.go","a.go"]}`)
	files, err = AllowlistedFiles(directory, "custom.json")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{"a.go": true, "nested/b.go": true}
	if !reflect.DeepEqual(files, expected) {
		t.Fatalf("expected %#v, got %#v", expected, files)
	}
}

func TestAllowlistedFilesRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"invalid JSON", `{"files":`},
		{"missing files", `{}`},
		{"null files", `{"files":null}`},
		{"non-array files", `{"files":"source.go"}`},
		{"non-string entry", `{"files":[1]}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			writeFile(t, filepath.Join(directory, DefaultAllowlistPath), test.content)
			_, err := AllowlistedFiles(directory, DefaultAllowlistPath)
			if err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestScanRepositoryAllAndStagedModes(t *testing.T) {
	repository := newRepository(t)
	writeFile(t, filepath.Join(repository, ".gitignore"), "ignored.go\n")
	writeFile(t, filepath.Join(repository, "tracked.go"), "const safe = true\n")
	runGit(t, repository, "add", ".gitignore", "tracked.go")

	writeFile(t, filepath.Join(repository, "untracked.go"), "const hidden = '\u202e'\n")
	writeFile(t, filepath.Join(repository, "ignored.go"), "const hidden = '\u202e'\n")
	if err := os.WriteFile(
		filepath.Join(repository, "binary.dat"),
		[]byte("prefix\x00\u202esuffix"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	findings, err := ScanRepository(Options{CWD: repository})
	if err != nil {
		t.Fatal(err)
	}
	assertFindings(t, findings, []Finding{{
		FilePath: "untracked.go", Line: 1, Column: 17,
		CodePoint: "U+202E", Name: "RIGHT-TO-LEFT OVERRIDE",
	}})

	findings, err = ScanRepository(Options{CWD: repository, Mode: "--staged"})
	if err != nil {
		t.Fatal(err)
	}
	assertFindings(t, findings, []Finding{})

	writeFile(t, filepath.Join(repository, "tracked.go"), "const hidden = '\u202e'\n")
	findings, err = ScanRepository(Options{CWD: repository, Mode: "--staged"})
	if err != nil {
		t.Fatal(err)
	}
	assertFindings(t, findings, []Finding{})

	runGit(t, repository, "add", "tracked.go")
	findings, err = ScanRepository(Options{CWD: repository, Mode: "--staged"})
	if err != nil {
		t.Fatal(err)
	}
	assertFindings(t, findings, []Finding{{
		FilePath: "tracked.go", Line: 1, Column: 17,
		CodePoint: "U+202E", Name: "RIGHT-TO-LEFT OVERRIDE",
	}})
}

func TestScanRepositorySkipsAllowlistedAndDeletedFiles(t *testing.T) {
	repository := newRepository(t)
	writeFile(t, filepath.Join(repository, "allowlisted.go"), "const hidden = '\u202e'\n")
	writeFile(t, filepath.Join(repository, "deleted.go"), "const safe = true\n")
	runGit(t, repository, "add", "allowlisted.go", "deleted.go")
	if err := os.Remove(filepath.Join(repository, "deleted.go")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repository, DefaultAllowlistPath), `{"files":["allowlisted.go"]}`)

	findings, err := ScanRepository(Options{CWD: repository})
	if err != nil {
		t.Fatal(err)
	}
	assertFindings(t, findings, []Finding{})
}

func TestScanRepositoryRejectsInvalidModeAndNonRepository(t *testing.T) {
	_, err := ScanRepository(Options{Mode: "--changed"})
	if err == nil || err.Error() != `mode must be either "--all" or "--staged"` {
		t.Fatalf("unexpected invalid-mode error: %v", err)
	}

	_, err = ScanRepository(Options{CWD: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "list Git files") {
		t.Fatalf("expected Git repository error, got %v", err)
	}
}

func assertFindings(t *testing.T, actual, expected []Finding) {
	t.Helper()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected %#v, got %#v", expected, actual)
	}
}

func newRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	runGit(t, repository, "init", "--quiet")
	return repository
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, cwd string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", cwd}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
