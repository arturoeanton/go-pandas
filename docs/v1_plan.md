# v1.0 plan

## Stable in v1.0

Everything classified **stable** in [api_freeze.md](api_freeze.md):
DataFrame, Series, the Index family, MultiIndex (storage, constructors,
tuple Loc, and — since rc.1 — DropLevel/SwapLevel/ReorderLevels/XS),
Categorical, GroupBy (including Transform/Filter/AsIndex), Resampler
within its documented scope, the documented Query grammar, CSV/JSON/
NDJSON IO, DTypes, error sentinels and the option styles. The NDArray
core, including the string-array semantics finalized in rc.1.

## Experimental after v1.0

Marked **experimental-post-v1** in api_freeze.md; usable, but their
shape may change in a v1.x with deprecated aliases:

- Resample advanced options (closed/label/origin/offset) when they
  land — the current observed-buckets engine is stable.
- MultiIndex label-range slicing and partial-key forms beyond
  TuplePrefix/XS.
- keepdims/axis-tuple reduction forms.

## Not in v1.0

Timezone engine, Parquet/Excel/SQL/Arrow IO (future separate modules),
linalg beyond MatMul (planned via an optional gonum adapter module),
pandas eval, rolling time windows. The core stays dependency-free.

## Compatibility philosophy

Behavioral parity verified by goldens generated from real pandas/NumPy;
differences are documented in compat/known_differences.md and are
**part of the contract** — changing a documented difference is a
breaking change even when it increases parity. Coverage percentages
are computed from the matrices and never hand-edited.

## Upgrade policy after v1.0

- v1.x minor releases add APIs and may extend documented behavior;
  they never change stable names, signatures, error sentinels or
  documented semantics.
- Anything removed goes through a deprecation cycle: alias + doc note
  for at least one minor release before removal in v2.

## Patch release policy

v1.0.x patches contain bug fixes, docs, tests, fuzz corpus and
low-risk performance work on existing APIs only (the
docs/release_checklist.md gate applies to every tag).

## Deprecation policy

Deprecated APIs keep working for the whole v1 major, carry a
`Deprecated:` doc comment naming the replacement, and appear in the
CHANGELOG and api_freeze.md when introduced.

## Roadmap post-v1

1. v1.1 — resample options, MultiIndex label-range slicing over sorted
   indexes, remaining reduction forms (keepdims).
2. v1.2 — optional adapter modules (gonum linalg, Arrow interchange)
   as separate go.mod modules; core stays zero-dependency.
3. Continuous — golden expansion tracking new pandas releases.
