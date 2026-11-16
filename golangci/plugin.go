// Package golangci registers the jsonv2audit analyzer as a golangci-lint
// module plugin named jsonv2audit.
//
// Add it to .custom-gcl.yml and build a custom binary with
// golangci-lint custom:
//
//	version: v2.15.0
//	plugins:
//	  - module: github.com/A-RYAN-1/jsonv2-audit
//	    import: github.com/A-RYAN-1/jsonv2-audit/golangci
//	    version: <version>
//
// Then enable it in .golangci.yml:
//
//	linters:
//	  enable:
//	    - jsonv2audit
//	  settings:
//	    custom:
//	      jsonv2audit:
//	        type: module
//	        settings:
//	          skip: [case-insensitive]
//
//	issues:
//	  uniq-by-line: false
//
// Settings: skip, a list of rule IDs not to report, and all (default false),
// which also reports case-insensitive and null-resets, the two rules that fire
// on nearly every decoded struct. golangci-lint
// keeps one issue per line by default; findings on imported types share the
// call's line, so uniq-by-line: false shows every one.
package golangci

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	jsonv2audit "github.com/A-RYAN-1/jsonv2-audit"
)

func init() { register.Plugin("jsonv2audit", New) }

// Settings is the settings block of the plugin in .golangci.yml.
type Settings struct {
	Skip []string `json:"skip"`
	All  bool     `json:"all"`
}

type plugin struct{ analyzer *analysis.Analyzer }

// New builds the plugin from its settings. Unknown settings and unknown
// rule IDs are errors, so a typo does not silently report everything.
func New(settings any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](settings)
	if err != nil {
		return nil, err
	}
	a, err := jsonv2audit.NewAnalyzer(jsonv2audit.Config{Skip: s.Skip, All: s.All})
	if err != nil {
		return nil, err
	}
	return &plugin{analyzer: a}, nil
}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{p.analyzer}, nil
}

// GetLoadMode asks for type information: the analyzer follows the types of
// the values passed to encoding/json.
func (p *plugin) GetLoadMode() string { return register.LoadModeTypesInfo }
