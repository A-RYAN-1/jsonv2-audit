# Changelog

All notable changes to this project are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions will follow [Semantic Versioning](https://semver.org/) once a first version is tagged.

## [Unreleased]

### Changed

- The vet tool and the golangci-lint plugin no longer report `case-insensitive` and `null-resets` by default; `-jsonv2audit.all` and the plugin setting `all` turn them on, and `Config.All` does the same in code (2026-11-16).

### Added

- Analyzer for `encoding/json` call sites with 13 rules, each with the v2 option that keeps v1 behaviour, and the `jsonv2-audit` command (2026-11-10).
- `go vet -vettool` entry point, golangci-lint module plugin and the `-skip` setting (2026-11-12).
- Proof group: tag parser fuzz test, property test over generated struct types against real v1 and v2 output, mutation tester, benchmark (2026-11-13).
- Rules `array-length`, `null-resets` and `duplicate-name` (2026-11-13).
- Rule `v1-options`: for v2 calls that pass `json.DefaultOptionsV1()`, the options the value needs and the ones it can drop (2026-11-13).
- `docs/rules.md`, generated from the rule table with real v1 and v2 output; every finding links to its section.
- GoReleaser configuration and a release workflow that runs on a `v*` tag and creates a draft release.
- Mutation tester `-shard i/n` and `-json`, and a `heavy-run` workflow that runs one command split into shards on Actions (2026-11-14).

### Fixed

- Anonymous structs were all named "struct" in messages; they are now named by path.
