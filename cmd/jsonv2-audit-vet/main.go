// Command jsonv2-audit-vet runs the jsonv2audit analyzer under go vet.
//
// Usage:
//
//	go install github.com/A-RYAN-1/jsonv2-audit/cmd/jsonv2-audit-vet@latest
//	go vet -vettool=$(which jsonv2-audit-vet) ./...
//	go vet -vettool=$(which jsonv2-audit-vet) -jsonv2audit.skip=case-insensitive ./...
//
// case-insensitive and null-resets are off by default (they fire on nearly
// every decoded struct); -jsonv2audit.all turns them on.
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

func main() {
	// case-insensitive and null-resets fire on nearly every decoded struct,
	// so they are off here; -jsonv2audit.all turns them on.
	a, err := jsonv2audit.NewAnalyzer(jsonv2audit.Config{})
	if err != nil {
		panic(err)
	}
	unitchecker.Main(a)
}
