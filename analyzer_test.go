package jsonv2audit_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	jsonv2audit "github.com/A-RYAN-1/jsonv2-audit"
)

func TestPositions(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), jsonv2audit.Analyzer, "a")
}

func TestNewAnalyzer(t *testing.T) {
	if _, err := jsonv2audit.NewAnalyzer(jsonv2audit.Config{Skip: []string{"no-such-rule"}}); err == nil {
		t.Error("an unknown rule ID gave no error")
	}
	a, err := jsonv2audit.NewAnalyzer(jsonv2audit.Config{Skip: []string{" nil-slice ", ""}})
	if err != nil {
		t.Fatal(err)
	}
	if got := a.Flags.Lookup("skip").Value.String(); got != "nil-slice" {
		t.Errorf("skip flag %q, want nil-slice", got)
	}
	if a == jsonv2audit.Analyzer {
		t.Error("NewAnalyzer returned the shared Analyzer")
	}
}

func TestSkipFlag(t *testing.T) {
	f := jsonv2audit.Analyzer.Flags.Lookup("skip")
	if err := f.Value.Set("nil-map,nil-slice"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Value.Set("") }()
	if got := f.Value.String(); got != "nil-slice,nil-map" {
		t.Errorf("flag value %q, want rules order nil-slice,nil-map", got)
	}
	if err := f.Value.Set("nil-map,bogus"); err == nil {
		t.Error("an unknown rule in the flag gave no error")
	}
}

func TestV1Options(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), jsonv2audit.Analyzer, "c")
}

func TestPinnedCases(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), jsonv2audit.Analyzer, "d")
}
