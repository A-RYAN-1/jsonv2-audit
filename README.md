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

`jsonv2-audit -rules` lists every rule; [docs/rules.md](docs/rules.md) shows each one with real v1 and v2 output and the option that keeps v1 behaviour (every finding links to its section). `-json` prints machine-readable output, `-all` lists the case-insensitive and null-reset decoding findings one by one (the text output counts them per package), `-tests` includes test files, and `-skip nil-slice,omitempty` turns rules off.

**Code already on v2 with `DefaultOptionsV1`.** A call to `encoding/json/v2` that passes `json.DefaultOptionsV1()` (directly, or through a variable set once from it) gets one `v1-options` finding that lists the options its value really needs and the ones it can drop, for example `Slices: Marshal with json.DefaultOptionsV1() ...; this value needs jsonv2.FormatNilSliceAsNull; pass these instead`. It judges the 12 options the rules model; the 9 that depend on data (`Deterministic`, `EscapeForHTML` and the like) are listed as not judged. Skip it with `-skip v1-options`. Exit status is 0 with no findings, 1 with findings, 2 when packages fail to load or a skipped rule does not exist.

## FAQ

**Why not `go vet`?** The Go team closed a request for a vet check as not feasible because it would not meet vet's accuracy requirement ([golang/go#82895](https://github.com/golang/go/issues/82895), 2026-11-06). This tool reports where behaviour may differ, not a proven bug, and says so with a rule per difference; it also runs under `go vet -vettool` if you want that.

**How is this different from `jsonsplit`?** [`jsonsplit`](https://pkg.go.dev/github.com/go-json-experiment/jsonsplit) is a runtime wrapper from the Go team's authors: you call it at each site instead of `encoding/json`, it runs v1, v2 or both, and it reports differences on real traffic. That finds only what traffic exercises and needs code changes at every call site. This tool reads types and call sites, needs no traffic and no code change, so it can list every place that may differ before you start; it cannot see differences that depend on data (the nine options listed under `v1-options`). They answer different questions and can be used together.

**How is this different from staticcheck?** The staticcheck work on v2 ([dominikh/go-tools#1678](https://github.com/dominikh/go-tools/issues/1678)) checks code that already uses v2 for mistakes. This tool checks v1 code before the move.

**Does a finding mean my output will change?** It means the type reaches a construct whose behaviour differs. The `null-resets` and `case-insensitive` findings fire for almost every decoded struct and matter only when the input has a `null` into a value that already holds data, or a differently cased name; the text output counts them per package and `-all` lists them.

## Known limits

- Values stored in an interface (`any`) are not checked; only the static type passed to the call is.
- A wrapper function that takes `any` hides its callers' types. Only direct calls to the `encoding/json` entry points (and, for `v1-options`, to the `encoding/json/v2` ones) are roots.
- `case-insensitive` is reported for every decoded struct; a better signal (only types decoded from input outside the program) is open.
- Options passed through wrappers or `jsonv2.JoinOptions` are not followed by `v1-options`.
- Message names for types declared inside functions are not unique.

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

Status: private, in development; analyzer, 13 rules and the command merged 2026-11-10 (#1); `v1-options` (calls that pass `DefaultOptionsV1`) added 2026-11-13, proved by running v2 with only the needed options against v2 with all of `DefaultOptionsV1` for every groundtruth case; three more rules 2026-11-13 (`array-length`, `null-resets`, `duplicate-name`, 16 in all), each found by running v1 and v2 side by side on go1.27.0 and proved with its v1-keeping option (`json.UnmarshalArrayFromAnyLength`, `json.MergeWithLegacySemantics`, `json.ReportErrorsWithLegacySemantics`); go vet tool, golangci-lint plugin and `-skip` added 2026-11-12; proof group added 2026-11-13: tag parser fuzz test, property test over generated struct types against real v1 and v2 output, now with null and short-array decode variants (go1.27.0, 60 seeds of 150 types, all pass), mutation tester `go run ./scripts/mutate` (164 mutants, 137 killed, 83.5%), benchmark 5.9 ms and 47k allocs for 800 call sites. Requires Go 1.27. Docs, generated rules page, CHANGELOG and release configuration added 2026-11-13 (no version tagged yet). Last updated: 2026-11-13.
