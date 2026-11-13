# jsonv2-audit

Find what changes before you move a Go module from `encoding/json` to `encoding/json/v2`.

Go 1.27 added `encoding/json/v2`. Its API is easy to adopt; its behaviour is not the same: nil slices become `[]`, `omitempty` keeps `false` and `0`, `time.Duration` has no default format, pointer-receiver marshal methods run in more places, and decoding matches names exactly. `jsonv2-audit` reads your code, follows every value passed to `encoding/json`, and lists each field whose output or parsing changes, with the option that keeps the v1 behaviour.

```
go install github.com/A-RYAN-1/jsonv2-audit/cmd/jsonv2-audit@latest
jsonv2-audit ./...
```

```
pets/pet.go:12:2: [nil-slice] Pet.Nicknames: a nil slice encodes as null in v1 and [] in v2; to keep v1 behaviour pass jsonv2.FormatNilSliceAsNull(true)
pets/pet.go:13:2: [omitempty] Pet.Age: omitempty omits false and 0 in v1 but keeps them in v2 (omitzero omits them in both); to keep v1 behaviour pass json.OmitEmptyWithLegacySemantics(true)

example.com/pets: 2 finding(s)
  to keep v1 behaviour under v2: json.OmitEmptyWithLegacySemantics(true), jsonv2.FormatNilSliceAsNull(true)
```

`jsonv2-audit -rules` lists every rule. `-json` prints machine-readable output, `-all` lists the case-insensitive decoding findings one by one, `-tests` includes test files, and `-skip nil-slice,omitempty` turns rules off. Exit status is 0 with no findings, 1 with findings, 2 when packages fail to load or a skipped rule does not exist.

## Under go vet

```
go install github.com/A-RYAN-1/jsonv2-audit/cmd/jsonv2-audit-vet@latest
go vet -vettool=$(which jsonv2-audit-vet) ./...
go vet -vettool=$(which jsonv2-audit-vet) -jsonv2audit.skip=case-insensitive ./...
```

go vet checks one package at a time, so a finding on a type declared in another package is reported at the `encoding/json` call. The per-package option summary is only in the `jsonv2-audit` command.

## In golangci-lint

The `golangci` package is a [module plugin](https://golangci-lint.run/docs/plugins/module-plugins/). Build a custom binary with `golangci-lint custom` from a `.custom-gcl.yml`:

```yaml
version: v2.15.0
plugins:
  - module: github.com/A-RYAN-1/jsonv2-audit
    import: github.com/A-RYAN-1/jsonv2-audit/golangci
    version: <version>
```

and enable it in `.golangci.yml`:

```yaml
version: "2"
linters:
  enable: [jsonv2audit]
  settings:
    custom:
      jsonv2audit:
        type: module
        settings:
          skip: [case-insensitive]
issues:
  uniq-by-line: false
```

`skip` is the only setting; an unknown rule ID or setting stops the run with an error. golangci-lint keeps one issue per line by default, and every finding on an imported type sits on the call line, so `uniq-by-line: false` shows them all. `scripts/golangci-e2e.sh` builds the custom binary from this checkout and checks it (CI job `golangci-plugin`).

Status: private, in development; analyzer, 13 rules and the command merged 2026-11-10 (#1); go vet tool, golangci-lint plugin and `-skip` added 2026-11-12; proof group added 2026-11-13: tag parser fuzz test, property test over generated struct types against real v1 and v2 output (go1.27.0, 60 seeds of 150 types, all pass), mutation tester `go run ./scripts/mutate` (164 mutants, 137 killed, 83.5%), benchmark 5.9 ms and 47k allocs for 800 call sites. Requires Go 1.27. Last updated: 2026-11-13.
