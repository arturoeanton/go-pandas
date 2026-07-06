package series

import (
	"fmt"
	"sort"

	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
)

// SortValues returns the series sorted by value. Missing values go last,
// like pandas. The sort is stable.
func (s *Series) SortValues(ascending bool) *Series {
	pos := make([]int, s.Len())
	for i := range pos {
		pos[i] = i
	}
	sort.SliceStable(pos, func(a, b int) bool {
		return lessAt(s, pos[a], pos[b], ascending)
	})
	out, _ := s.Take(pos)
	return out
}

// lessAt orders two positions of a series, NA last regardless of order.
func lessAt(s *Series, i, j int, ascending bool) bool {
	if s.mask[i] {
		return false
	}
	if s.mask[j] {
		return true
	}
	c, ok := expr.CompareValues(s.data[i], s.data[j])
	if !ok {
		return false
	}
	if ascending {
		return c < 0
	}
	return c > 0
}

// SortIndex returns the series sorted by its index labels.
func (s *Series) SortIndex(ascending bool) *Series {
	pos := make([]int, s.Len())
	for i := range pos {
		pos[i] = i
	}
	sort.SliceStable(pos, func(a, b int) bool {
		c, ok := expr.CompareValues(s.index.At(pos[a]), s.index.At(pos[b]))
		if !ok {
			return false
		}
		if ascending {
			return c < 0
		}
		return c > 0
	})
	out, _ := s.Take(pos)
	return out
}

// Unique returns the distinct values in first-seen order (missing values
// contribute a single NA entry, like pandas).
func (s *Series) Unique() *Series {
	seen := make(map[any]bool)
	sawNA := false
	var values []any
	for i, v := range s.data {
		if s.mask[i] {
			if !sawNA {
				sawNA = true
				values = append(values, nil)
			}
			continue
		}
		if !seen[v] {
			seen[v] = true
			values = append(values, v)
		}
	}
	return NewSeries(s.name, values, WithDType(s.dtype))
}

// NUnique counts the distinct values; dropNA excludes the NA entry.
func (s *Series) NUnique(dropNA bool) int {
	seen := make(map[any]bool)
	sawNA := false
	for i, v := range s.data {
		if s.mask[i] {
			sawNA = true
			continue
		}
		seen[v] = true
	}
	n := len(seen)
	if sawNA && !dropNA {
		n++
	}
	return n
}

// ValueCountOptions controls ValueCounts.
type ValueCountOptions struct {
	// Ascending sorts counts smallest-first when true.
	Ascending bool
	// DropNA excludes missing values (default true).
	DropNA bool
	// Normalize divides counts by the total when true.
	Normalize bool
}

// ValueCountOption mutates ValueCountOptions.
type ValueCountOption func(*ValueCountOptions)

// ValueCountsAscending sorts counts ascending.
func ValueCountsAscending(v bool) ValueCountOption {
	return func(o *ValueCountOptions) { o.Ascending = v }
}

// ValueCountsDropNA includes/excludes missing values.
func ValueCountsDropNA(v bool) ValueCountOption {
	return func(o *ValueCountOptions) { o.DropNA = v }
}

// ValueCountsNormalize returns relative frequencies instead of counts.
func ValueCountsNormalize(v bool) ValueCountOption {
	return func(o *ValueCountOptions) { o.Normalize = v }
}

// ValueCounts returns a series of counts indexed by value (rendered as
// labels), sorted by count descending — like Series.value_counts(). Note:
// pandas returns a Series here too; go-pandas keeps that shape.
func (s *Series) ValueCounts(opts ...ValueCountOption) *Series {
	o := ValueCountOptions{DropNA: true}
	for _, f := range opts {
		f(&o)
	}
	counts := make(map[any]int)
	var order []any
	naCount := 0
	total := 0
	for i, v := range s.data {
		if s.mask[i] {
			naCount++
			total++
			continue
		}
		if _, ok := counts[v]; !ok {
			order = append(order, v)
		}
		counts[v]++
		total++
	}
	if !o.DropNA && naCount > 0 {
		order = append(order, "<NA>")
		counts["<NA>"] = naCount
	}
	sort.SliceStable(order, func(a, b int) bool {
		if o.Ascending {
			return counts[order[a]] < counts[order[b]]
		}
		return counts[order[a]] > counts[order[b]]
	})
	labels := make([]string, len(order))
	values := make([]any, len(order))
	denom := float64(total)
	if o.DropNA {
		denom = float64(total - naCount)
	}
	for i, v := range order {
		labels[i] = fmt.Sprint(v)
		if o.Normalize {
			values[i] = float64(counts[v]) / denom
		} else {
			values[i] = counts[v]
		}
	}
	name := "count"
	if o.Normalize {
		name = "proportion"
	}
	return NewSeries(name, values, WithIndex(index.NewStringIndex(labels, s.name)))
}
