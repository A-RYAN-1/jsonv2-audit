package jsonv2audit_test

import (
	"os"
	"strings"
	"testing"

	jsonv2audit "github.com/A-RYAN-1/jsonv2-audit"
	"github.com/A-RYAN-1/jsonv2-audit/internal/ruledoc"
)

// TestRulesPageIsCurrent fails when docs/rules.md differs from what
// `go run ./scripts/genrules` would write: the page's examples are real v1
// and v2 output and must be regenerated when a rule or a type changes.
func TestRulesPageIsCurrent(t *testing.T) {
	want, err := ruledoc.Render("internal/groundtruth/groundtruth.go")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("docs/rules.md")
	if err != nil {
		t.Fatalf("docs/rules.md missing: run go run ./scripts/genrules (%v)", err)
	}
	if strings.ReplaceAll(string(got), "\r\n", "\n") != want {
		t.Error("docs/rules.md is out of date: run go run ./scripts/genrules")
	}
}

// TestEveryRuleHasASection checks that each diagnostic link lands on a
// heading of the rules page.
func TestEveryRuleHasASection(t *testing.T) {
	page, err := os.ReadFile("docs/rules.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range jsonv2audit.Rules {
		if !strings.Contains(string(page), "\n## "+r.ID+"\n") {
			t.Errorf("docs/rules.md has no section for %s", r.ID)
		}
	}
}
