package dataframe

import (
	"github.com/arturoeanton/go-pandas/series"
)

// IsNA returns a frame of booleans marking missing cells.
func (df *DataFrame) IsNA() *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		cols[i] = c.IsNA()
	}
	out, _ := newFrame(cols, df.index.Clone())
	return out
}

// NotNA returns a frame of booleans marking present cells.
func (df *DataFrame) NotNA() *DataFrame {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		cols[i] = c.NotNA()
	}
	out, _ := newFrame(cols, df.index.Clone())
	return out
}

// HasNA reports whether any cell is missing.
func (df *DataFrame) HasNA() bool {
	for _, c := range df.columns {
		if c.HasNA() {
			return true
		}
	}
	return false
}

// DropNAOptions configures DropNA.
type DropNAOptions struct {
	// How is "any" (default: drop rows with at least one NA) or "all"
	// (drop rows where every value is NA).
	How string
	// Subset restricts the check to these columns.
	Subset []string
	// Thresh keeps rows with at least this many non-NA values (overrides
	// How when positive), like df.dropna(thresh=n).
	Thresh int
	// Axis 0 drops rows (default); axis 1 drops columns containing NA.
	Axis int
}

// DropNAOption mutates DropNAOptions.
type DropNAOption func(*DropNAOptions)

// DropNAHow sets the "any"/"all" behavior.
func DropNAHow(how string) DropNAOption {
	return func(o *DropNAOptions) { o.How = how }
}

// DropNASubset restricts the NA check to a subset of columns.
func DropNASubset(columns ...string) DropNAOption {
	return func(o *DropNAOptions) { o.Subset = columns }
}

// DropNAThresh keeps rows with at least n non-NA values.
func DropNAThresh(n int) DropNAOption {
	return func(o *DropNAOptions) { o.Thresh = n }
}

// DropNAAxis selects the drop axis: 0 for rows, 1 for columns.
func DropNAAxis(axis int) DropNAOption {
	return func(o *DropNAOptions) { o.Axis = axis }
}

// DropNA drops rows (or columns with DropNAAxis(1)) containing missing
// values.
func (df *DataFrame) DropNA(opts ...DropNAOption) *DataFrame {
	o := DropNAOptions{How: "any"}
	for _, f := range opts {
		f(&o)
	}
	if o.Axis == 1 {
		var keep []string
		for _, c := range df.columns {
			naCount := c.Len() - c.Count()
			drop := false
			switch {
			case o.Thresh > 0:
				drop = c.Count() < o.Thresh
			case o.How == "all":
				drop = c.Len() > 0 && naCount == c.Len()
			default:
				drop = naCount > 0
			}
			if !drop {
				keep = append(keep, c.Name())
			}
		}
		out, _ := df.Select(keep...)
		return out
	}
	check := df.columns
	if len(o.Subset) > 0 {
		check = nil
		for _, name := range o.Subset {
			if i, ok := df.byName[name]; ok {
				check = append(check, df.columns[i])
			}
		}
	}
	masks := make([][]bool, len(check))
	for j, c := range check {
		masks[j] = c.IsNA().AsMask()
	}
	var pos []int
	for i := 0; i < df.Len(); i++ {
		naCount := 0
		for j := range check {
			if masks[j][i] {
				naCount++
			}
		}
		drop := false
		switch {
		case o.Thresh > 0:
			drop = len(check)-naCount < o.Thresh
		case o.How == "all":
			drop = len(check) > 0 && naCount == len(check)
		default:
			drop = naCount > 0
		}
		if !drop {
			pos = append(pos, i)
		}
	}
	out, _ := df.Take(pos)
	return out
}

// FillNA replaces missing values per column; columns not present in the
// map are left unchanged.
func (df *DataFrame) FillNA(values map[string]any) (*DataFrame, error) {
	cols := make([]*series.Series, len(df.columns))
	for i, c := range df.columns {
		if v, ok := values[c.Name()]; ok {
			cols[i] = c.FillNA(v)
		} else {
			cols[i] = c.Copy()
		}
	}
	return newFrame(cols, df.index.Clone())
}
