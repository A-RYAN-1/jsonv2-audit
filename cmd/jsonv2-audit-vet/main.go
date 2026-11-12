// Command jsonv2-audit-vet runs the jsonv2audit analyzer under go vet.
//
// Usage:
//
//	go install github.com/A-RYAN-1/jsonv2-audit/cmd/jsonv2-audit-vet@latest
//	go vet -vettool=$(which jsonv2-audit-vet) ./...
//	go vet -vettool=$(which jsonv2-audit-vet) -jsonv2audit.skip=case-insensitive ./...
//
// go vet analyzes one package at a time, so a finding on a type declared in
// another package is reported at the call, as the jsonv2-audit command does.
// The command adds the per-package option summary; this tool reports the
// findings only.
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	jsonv2audit "github.com/A-RYAN-1/jsonv2-audit"
)

func main() { unitchecker.Main(jsonv2audit.Analyzer) }
