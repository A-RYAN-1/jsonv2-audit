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

`jsonv2-audit -rules` lists every rule. `-json` prints machine-readable output, `-all` lists the case-insensitive decoding findings one by one, and `-tests` includes test files. Exit status is 0 with no findings, 1 with findings, 2 when packages fail to load.

Status: private, in development. Requires Go 1.27.
