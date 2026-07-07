package series

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/expr"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// SortValues returns the series sorted by value. Missing values go last,
// like pandas. The sort is stable. Categorical series sort by category
// rank via a counting sort over codes — no comparisons at all.
func (s *Series) SortValues(ascending bool) *Series {
	if cc, ok := column.AsCategorical(s.col); ok {
		out, _ := s.Take(catSortOrder(cc, ascending))
		return out
	}
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

// catSortOrder builds the stable row order of a categorical column by
// category rank: an O(n + k) counting sort over codes, NA bucket last.
func catSortOrder(cc *column.CategoricalColumn, ascending bool) []int {
	codes, mask := cc.RawCodes()
	k := cc.CategoryCount()
	counts := make([]int, k+1) // trailing bucket: NA
	for i, code := range codes {
		if mask[i] {
			counts[k]++
			continue
		}
		counts[code]++
	}
	start := make([]int, k+1)
	acc := 0
	if ascending {
		for c := 0; c < k; c++ {
			start[c] = acc
			acc += counts[c]
		}
	} else {
		for c := k - 1; c >= 0; c-- {
			start[c] = acc
			acc += counts[c]
		}
	}
	start[k] = acc // NA last regardless of direction
	pos := make([]int, len(codes))
	for i, code := range codes {
		b := int(code)
		if mask[i] {
			b = k
		}
		pos[start[b]] = i
		start[b]++
	}
	return pos
}

// lessAt orders two positions of a series, NA last regardless of order.
func lessAt(s *Series, i, j int, ascending bool) bool {
	if s.col.IsNA(i) {
		return false
	}
	if s.col.IsNA(j) {
		return true
	}
	c, ok := expr.CompareValues(s.col.Value(i), s.col.Value(j))
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

// hashKey normalizes a value into a map-safe string key: numeric widths
// collapse (int 1 == int64 1 == 1.0, matching pandas), and unhashable
// values (e.g. []string cells from Str().Split) never panic.
func hashKey(v any) string {
	if f, ok := dtype.AsFloat(v); ok {
		if _, isBool := v.(bool); !isBool {
			return "n\x00" + strconv.FormatFloat(f, 'g', -1, 64)
		}
	}
	return fmt.Sprintf("%T\x00%v", v, v)
}

// Unique returns the distinct values in first-seen order (missing values
// contribute a single NA entry, like pandas).
func (s *Series) Unique() *Series {
	seen := make(map[string]bool)
	sawNA := false
	var values []any
	for i := 0; i < s.Len(); i++ {
		if s.col.IsNA(i) {
			if !sawNA {
				sawNA = true
				values = append(values, nil)
			}
			continue
		}
		v := s.col.Value(i)
		k := hashKey(v)
		if !seen[k] {
			seen[k] = true
			values = append(values, v)
		}
	}
	return NewSeries(s.name, values, WithDType(s.DType()))
}

// NUnique counts the distinct values; dropNA excludes the NA entry.
func (s *Series) NUnique(dropNA bool) int {
	seen := make(map[string]bool)
	sawNA := false
	for i := 0; i < s.Len(); i++ {
		if s.col.IsNA(i) {
			sawNA = true
			continue
		}
		seen[hashKey(s.col.Value(i))] = true
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
	if cc, ok := column.AsCategorical(s.col); ok {
		return s.catValueCounts(cc, o)
	}
	counts := make(map[string]int)
	var order []any
	naCount := 0
	total := 0
	for i := 0; i < s.Len(); i++ {
		if s.col.IsNA(i) {
			naCount++
			total++
			continue
		}
		v := s.col.Value(i)
		k := hashKey(v)
		if _, ok := counts[k]; !ok {
			order = append(order, v)
		}
		counts[k]++
		total++
	}
	naLabel := any("<NA>")
	if !o.DropNA && naCount > 0 {
		order = append(order, naLabel)
		counts[hashKey(naLabel)] += naCount
	}
	sort.SliceStable(order, func(a, b int) bool {
		if o.Ascending {
			return counts[hashKey(order[a])] < counts[hashKey(order[b])]
		}
		return counts[hashKey(order[a])] > counts[hashKey(order[b])]
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
			values[i] = float64(counts[hashKey(v)]) / denom
		} else {
			values[i] = counts[hashKey(v)]
		}
	}
	name := "count"
	if o.Normalize {
		name = "proportion"
	}
	return NewSeries(name, values, WithIndex(index.NewStringIndex(labels, s.name)))
}

// catValueCounts counts a categorical series with one array pass over
// codes — no hashing. Like pandas, every category appears in the result,
// including zero-count ones; ties keep category order (stable sort).
func (s *Series) catValueCounts(cc *column.CategoricalColumn, o ValueCountOptions) *Series {
	codes, mask := cc.RawCodes()
	counts := make([]int, cc.CategoryCount())
	naCount := 0
	for i, code := range codes {
		if mask[i] {
			naCount++
			continue
		}
		counts[code]++
	}
	order := make([]int, len(counts))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		if o.Ascending {
			return counts[order[a]] < counts[order[b]]
		}
		return counts[order[a]] > counts[order[b]]
	})
	categories := cc.Categories()
	total := len(codes)
	denom := float64(total)
	if o.DropNA {
		denom = float64(total - naCount)
	}
	n := len(order)
	withNA := !o.DropNA && naCount > 0
	if withNA {
		n++
	}
	labels := make([]string, 0, n)
	values := make([]any, 0, n)
	emit := func(label string, count int) {
		labels = append(labels, label)
		if o.Normalize {
			values = append(values, float64(count)/denom)
		} else {
			values = append(values, count)
		}
	}
	for _, c := range order {
		emit(fmt.Sprint(categories[c]), counts[c])
	}
	if withNA {
		emit("<NA>", naCount)
	}
	name := "count"
	if o.Normalize {
		name = "proportion"
	}
	return NewSeries(name, values, WithIndex(index.NewStringIndex(labels, s.name)))
}
