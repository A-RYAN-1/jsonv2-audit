// Command jsonv2-audit lists the places where a Go module's encoding/json
// behaviour changes under encoding/json/v2, and the options that keep the
// v1 behaviour, per package.
//
// Usage:
//
//	jsonv2-audit [-json] [-all] [-tests] [-rules] [-skip ids] [-C dir] [packages]
//
// Packages default to ./... . The text output counts case-insensitive
// findings (every decoded struct has one) instead of listing them; -all
// lists them, and -json always includes them. -skip takes a comma-separated
// list of rule IDs not to report. Exit status: 0 no findings, 1 findings,
// 2 the packages did not load.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"

	jsonv2audit "github.com/A-RYAN-1/jsonv2-audit"
)

// Finding is one diagnostic in the -json output.
type Finding struct {
	Package string `json:"package"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Summary is the per-package result in the -json output.
type Summary struct {
	Package  string    `json:"package"`
	Findings []Finding `json:"findings"`
	Options  []string  `json:"options"`
	Fixes    []string  `json:"fixes,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("jsonv2-audit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	tests := fs.Bool("tests", false, "include test files")
	listRules := fs.Bool("rules", false, "list the rules and exit")
	all := fs.Bool("all", false, "list case-insensitive findings one by one")
	dir := fs.String("C", "", "run in this directory")
	skip := fs.String("skip", "", "comma-separated rule IDs not to report")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *listRules {
		for _, r := range jsonv2audit.Rules {
			opt := r.Option
			if opt == "" {
				opt = "(change the code)"
			}
			_, _ = fmt.Fprintf(stdout, "%-18s %s\n%-18s keep v1: %s\n", r.ID, r.Summary, "", opt)
		}
		return 0
	}
	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	an, err := jsonv2audit.NewAnalyzer(jsonv2audit.Config{Skip: strings.Split(*skip, ","), All: true})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "jsonv2-audit:", err)
		return 2
	}
	sums, err := audit(an, *dir, patterns, *tests)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "jsonv2-audit:", err)
		return 2
	}
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(sums); err != nil {
			_, _ = fmt.Fprintln(stderr, "jsonv2-audit:", err)
			return 2
		}
	} else {
		printText(stdout, sums, *all)
	}
	for _, s := range sums {
		if len(s.Findings) > 0 {
			return 1
		}
	}
	return 0
}

func audit(an *analysis.Analyzer, dir string, patterns []string, tests bool) ([]Summary, error) {
	cfg := &packages.Config{Mode: packages.LoadAllSyntax, Tests: tests, Dir: dir}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}
	if n := packages.PrintErrors(pkgs); n > 0 {
		return nil, fmt.Errorf("%d package errors", n)
	}
	g, err := checker.Analyze([]*analysis.Analyzer{an}, pkgs, nil)
	if err != nil {
		return nil, err
	}
	byPkg := map[string]*Summary{}
	seen := map[string]bool{}
	for _, act := range g.Roots {
		if act.Err != nil {
			return nil, act.Err
		}
		path := act.Package.PkgPath
		for _, d := range act.Diagnostics {
			p := act.Package.Fset.Position(d.Pos)
			key := fmt.Sprintf("%s:%d:%d:%s", p.Filename, p.Line, p.Column, d.Message)
			if seen[key] {
				continue // the same file in a package and its test variant
			}
			seen[key] = true
			s := byPkg[path]
			if s == nil {
				s = &Summary{Package: path}
				byPkg[path] = s
			}
			s.Findings = append(s.Findings, Finding{path, p.Filename, p.Line, p.Column, d.Category, d.Message})
		}
	}
	var out []Summary
	for _, s := range byPkg {
		opts, fixes := map[string]bool{}, map[string]bool{}
		for _, f := range s.Findings {
			r, _ := jsonv2audit.RuleByID(f.Rule)
			if r.Advisory {
				continue
			}
			if r.Option == "" {
				fixes[r.ID] = true
			} else {
				opts[r.Option] = true
			}
		}
		s.Options, s.Fixes = keys(opts), keys(fixes)
		sort.Slice(s.Findings, func(i, j int) bool {
			a, b := s.Findings[i], s.Findings[j]
			if a.File != b.File {
				return a.File < b.File
			}
			return a.Line < b.Line
		})
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Package < out[j].Package })
	return out, nil
}

func printText(w io.Writer, sums []Summary, all bool) {
	if len(sums) == 0 {
		_, _ = fmt.Fprintln(w, "no differences found between encoding/json and encoding/json/v2")
		return
	}
	folded := map[string]map[string]int{}
	for _, s := range sums {
		for _, f := range s.Findings {
			if r, _ := jsonv2audit.RuleByID(f.Rule); r.Folded && !all {
				if folded[s.Package] == nil {
					folded[s.Package] = map[string]int{}
				}
				folded[s.Package][f.Rule]++
				continue
			}
			_, _ = fmt.Fprintf(w, "%s:%d:%d: [%s] %s\n", f.File, f.Line, f.Column, f.Rule, f.Message)
		}
	}
	_, _ = fmt.Fprintln(w)
	for _, s := range sums {
		_, _ = fmt.Fprintf(w, "%s: %d finding(s)\n", s.Package, len(s.Findings))
		for _, r := range jsonv2audit.Rules {
			if n := folded[s.Package][r.ID]; n > 0 {
				_, _ = fmt.Fprintf(w, "  %d %s (-all lists them)\n", n, r.FoldedNote)
			}
		}
		if len(s.Options) > 0 {
			_, _ = fmt.Fprintf(w, "  to keep v1 behaviour under v2: %s\n", strings.Join(s.Options, ", "))
		}
		if len(s.Fixes) > 0 {
			_, _ = fmt.Fprintf(w, "  needs a code change: %s\n", strings.Join(s.Fixes, ", "))
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
