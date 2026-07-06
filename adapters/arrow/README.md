# arrow adapter (planned)

Converts between `pd.DataFrame` and [Apache Arrow](https://arrow.apache.org)
records for zero-copy interchange with the Arrow ecosystem.

Planned API (v0.4), guarded by the `arrow` build tag so the core stays
dependency-free:

```go
//go:build arrow

func ToArrow(df *pd.DataFrame) (arrow.Record, error)
func FromArrow(record arrow.Record) (*pd.DataFrame, error)
```

Not yet implemented; this directory only reserves the package path.
