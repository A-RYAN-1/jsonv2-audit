package golangci_test

import (
	"strings"
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/A-RYAN-1/jsonv2-audit/golangci"
)

func build(t *testing.T, settings any) register.LinterPlugin {
	t.Helper()
	newPlugin, err := register.GetPlugin("jsonv2audit")
	if err != nil {
		t.Fatal(err)
	}
	p, err := newPlugin(settings)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRegisteredWithTypesInfo(t *testing.T) {
	p := build(t, nil)
	if got := p.GetLoadMode(); got != register.LoadModeTypesInfo {
		t.Errorf("load mode %q, want %q", got, register.LoadModeTypesInfo)
	}
	as, err := p.BuildAnalyzers()
	if err != nil || len(as) != 1 || as[0].Name != "jsonv2audit" {
		t.Fatalf("analyzers %v, err %v", as, err)
	}
}

// The settings map is what golangci-lint passes after reading YAML.
func TestSkipSetting(t *testing.T) {
	p := build(t, map[string]any{"skip": []any{"nil-slice"}})
	as, _ := p.BuildAnalyzers()
	analysistest.Run(t, analysistest.TestData(), as[0], "skip")
}

func TestBadSettings(t *testing.T) {
	for name, settings := range map[string]any{
		"unknown rule":  map[string]any{"skip": []any{"nil-slcie"}},
		"unknown field": map[string]any{"skpi": []any{"nil-slice"}},
	} {
		if _, err := golangci.New(settings); err == nil {
			t.Errorf("%s: no error", name)
		} else if name == "unknown rule" && !strings.Contains(err.Error(), "nil-slcie") {
			t.Errorf("%s: error %q does not name the rule", name, err)
		}
	}
}
