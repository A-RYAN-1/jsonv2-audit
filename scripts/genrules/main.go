// Command genrules writes docs/rules.md. Run it from the module root:
//
//	go run ./scripts/genrules
package main

import (
	"fmt"
	"os"

	"github.com/A-RYAN-1/jsonv2-audit/internal/ruledoc"
)

func main() {
	out, err := ruledoc.Render("internal/groundtruth/groundtruth.go")
	if err != nil {
		fmt.Fprintln(os.Stderr, "genrules:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll("docs", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "genrules:", err)
		os.Exit(1)
	}
	if err := os.WriteFile("docs/rules.md", []byte(out), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "genrules:", err)
		os.Exit(1)
	}
}
