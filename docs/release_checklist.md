# Release checklist

## Gate (every release)

```bash
go mod tidy                      # must produce no changes
go test ./...                    # all green, no Python needed
go test ./... -race
go vet ./...
gofmt -l .                       # must print nothing
for d in examples/*/; do go run "./$d" >/dev/null || echo "FAIL $d"; done
go test ./... -bench=. -benchmem -benchtime=1x   # benchmarks compile+run
go run ./cmd/compat-report       # refresh coverage numbers honestly
```

Then:

1. **Goldens** — if compat/python generators changed, regenerate with
   real pandas/NumPy and confirm `git diff` touches only intended
   files; update the golden count in README/coverage_report/prerelease.
2. **Fuzz smoke** — seeds run in `go test ./...`; before a tag, run the
   long loop from docs/fuzzing.md (60s per target). Fix, pin corpus,
   and add a regression test for anything found.
3. **Docs audit** — README status, known_differences, matrices,
   performance tables match reality; no impossible chaining; no stale
   version references.
4. **CHANGELOG** — Fixed/Hardened/Performance/Compatibility/Known
   limitations sections; migration notes for any behavior change.
5. **Tag** — single clean commit, annotated tag `vX.Y.Z`, push with
   tags. Commit and tag messages carry no tooling attribution.

## Cutting v0.10.x patches

Patch releases are stabilization-only: bug fixes, docs, tests, fuzz,
low-risk performance work on existing APIs. No new public API without
bumping the minor version.

## Preparing v1.0

- Resolve the **experimental** entries in docs/api_freeze.md: NDArray
  string-op semantics (keep or convert to errors — decide once),
  MultiIndex level operations (implement or explicitly out of scope),
  resample options.
- Close or explicitly defer the boxed paths (Unstack, N-D
  NDArray.Take) with roadmap entries.
- One full long-fuzz pass (10+ minutes per target).
- Freeze docs/api_freeze.md: everything marked stable becomes the v1
  contract.

## What must not change after the freeze

- Names, signatures and error sentinels of every **stable** API in
  docs/api_freeze.md.
- Documented semantics in compat/known_differences.md (they are part
  of the contract — changing them is a breaking change even when it
  increases pandas parity).
- Golden expectations: goldens may be added, never weakened.
- Zero core dependencies.
