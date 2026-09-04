package main

import (
	"fmt"
	"os"

	"github.com/linus-jansson/trojansource"
)

func main() {
	if len(os.Args) > 2 || (len(os.Args) == 2 && os.Args[1] != "--all" && os.Args[1] != "--staged") {
		fmt.Fprintln(os.Stderr, "Usage: trojansource [--all|--staged]")
		os.Exit(2)
	}

	mode := "--all"
	if len(os.Args) == 2 {
		mode = os.Args[1]
	}
	findings, err := trojansource.ScanRepository(trojansource.Options{Mode: mode})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(findings) == 0 {
		fmt.Println("Unicode security check passed.")
		return
	}

	fmt.Fprintln(os.Stderr, "Unicode security check failed:")
	for _, finding := range findings {
		fmt.Fprintln(os.Stderr, trojansource.FormatFinding(finding))
	}
	fmt.Fprintf(
		os.Stderr,
		"Remove the character or explicitly allowlist its repository-relative path in %s.\n",
		trojansource.DefaultAllowlistPath,
	)
	os.Exit(1)
}
