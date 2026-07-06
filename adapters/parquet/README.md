# parquet adapter (planned)

Reads and writes Parquet files as DataFrames.

Planned API (v0.4), guarded by the `parquet` build tag so the core stays
dependency-free:

```go
//go:build parquet

func ReadParquet(path string) (*pd.DataFrame, error)
func WriteParquet(df *pd.DataFrame, path string) error
```

Not yet implemented; this directory only reserves the package path.
