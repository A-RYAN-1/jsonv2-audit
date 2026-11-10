package jsonv2audit_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	jsonv2audit "github.com/A-RYAN-1/jsonv2-audit"
)

func TestPositions(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), jsonv2audit.Analyzer, "a")
}
