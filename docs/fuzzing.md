# Fuzzing go-pandas

All fuzz targets live in the `fuzz/` package (plus a few in
`internal/column` and `index` test files). They are property tests:
each target asserts, after every operation,

- **no panic** on any input,
- structural invariants hold (codes valid or -1, masks aligned,
  names/levels/codes counts match, shape products equal sizes),
- **inputs are not mutated**,
- output lengths/shapes/dtypes follow the documented rules,
- engine equivalences hold where applicable (categorical vs string
  groupby/merge/sort, columnar Where vs Query, lookup vs linear scan,
  searchsorted vs scan).

## Listing targets

```bash
go test ./fuzz/ -list 'Fuzz.*'
```

## Smoke run (seeds only, runs in normal `go test ./...`)

Every target's seed corpus executes as part of the regular test suite —
no extra step needed.

## Short local fuzz (per target)

```bash
go test ./fuzz/ -run '^$' -fuzz '^FuzzQueryParserNoPanic$' -fuzztime 10s
```

## Pre-release long run

Before tagging a release, give every target real time:

```bash
for f in $(go test ./fuzz/ -list 'Fuzz.*' | grep '^Fuzz'); do
  go test ./fuzz/ -run '^$' -fuzz "^$f\$" -fuzztime 60s || break
done
```

Crashing inputs are written to `fuzz/testdata/fuzz/<Target>/` and replay
automatically in `go test ./fuzz/` — commit them as regression cases
once fixed.

## Corpus notes

`fuzz/testdata/fuzz/` holds regression inputs from past runs. Do not
delete entries: they replay in CI-style test runs and pin fixed bugs.
