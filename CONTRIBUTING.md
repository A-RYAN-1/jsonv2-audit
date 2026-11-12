# Contributing

Thanks for looking. jsonv2-audit is a Go analyzer that reports what changes in a module's behaviour when it moves from `encoding/json` to `encoding/json/v2`.

1. **Open an issue first** for anything beyond a typo or an obvious bug, so we agree the change belongs before you
   spend time on it.
2. **Checks must pass locally and in CI:** `go vet ./...`, `go build ./...`, `go test -race -count=1 ./...` and `golangci-lint run`. A bug fix starts with a test that fails without it.
3. **Every finding is backed by the standard library.** A new or changed rule comes with an `analysistest` case under `testdata/` and, where behaviour differs, a check that v1 and v2 really disagree on that input.
4. **No false positives on purpose.** If a rule cannot be sure, it stays silent or says why it is unsure.
5. **Standard library and `golang.org/x/tools` only** unless an issue agrees otherwise.
6. **Code from other projects stays out of tests.** Fixtures are small synthetic packages.
7. **Commit style:** one imperative line in the Conventional Commits form (`fix: ...`, `feat: ...`);
   the pull request title uses the same form, with a body when the reason is not obvious.
8. **Review:** every change goes through a pull request approved by the maintainer. Nobody pushes to
   `main` directly.

**No licence yet.** The repository has no LICENSE file, so the code is visible but not licensed for reuse. Until one is added, open an issue before a pull request; a contribution can only be merged once its terms are settled.

Security problems go through the private route in [SECURITY.md](SECURITY.md), not a public issue.
