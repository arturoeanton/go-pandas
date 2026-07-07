package column

import (
	"fmt"
	"sort"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
)

// CategoricalColumn stores repeated values as int32 codes into a shared,
// immutable category list (v0.7). Hot paths (groupby, merge, sort,
// comparisons) operate on codes and never box labels.
//
// Invariants: categories are unique and never mutated in place (accessor
// operations build new columns with new category slices, so Take/Slice/
// Copy may share the categories slice safely); codes[i] == -1 iff
// mask[i] == true.
type CategoricalColumn struct {
	codes      []int32
	categories []any
	ordered    bool
	mask       []bool
}

// NewCategorical assembles a categorical column from buffers already in
// code space (engine use: concat, coalesce). The caller owns the
// invariants: codes index categories, codes[i] == -1 iff mask[i], and
// the categories slice is not mutated afterwards.
func NewCategorical(codes []int32, categories []any, ordered bool, mask []bool) *CategoricalColumn {
	return &CategoricalColumn{codes: codes, categories: categories, ordered: ordered, mask: mask}
}

// AsCategorical narrows a column to its categorical implementation.
func AsCategorical(c Column) (*CategoricalColumn, bool) {
	cc, ok := c.(*CategoricalColumn)
	return cc, ok
}

func (c *CategoricalColumn) DType() dtype.DType { return dtype.Category }
func (c *CategoricalColumn) Len() int           { return len(c.codes) }
func (c *CategoricalColumn) IsNA(i int) bool    { return c.mask[i] }

// Categories returns the category labels (copy).
func (c *CategoricalColumn) Categories() []any {
	return append([]any(nil), c.categories...)
}

// Codes returns the category codes (copy; -1 = missing).
func (c *CategoricalColumn) Codes() []int32 {
	return append([]int32(nil), c.codes...)
}

// RawCodes exposes the internal codes for read-only engine use.
func (c *CategoricalColumn) RawCodes() ([]int32, []bool) { return c.codes, c.mask }

// Ordered reports whether the category order is semantically meaningful.
func (c *CategoricalColumn) Ordered() bool { return c.ordered }

// CategoryCount returns the number of categories.
func (c *CategoricalColumn) CategoryCount() int { return len(c.categories) }

// CodeOf resolves a label to its category code (-1 when absent).
func (c *CategoricalColumn) CodeOf(label any) int32 {
	for i, cat := range c.categories {
		if cat == label {
			return int32(i)
		}
	}
	return -1
}

func (c *CategoricalColumn) Value(i int) any {
	if c.mask[i] {
		return nil
	}
	return c.categories[c.codes[i]]
}

func (c *CategoricalColumn) SetValue(i int, v any) error {
	if i < 0 || i >= len(c.codes) {
		return fmt.Errorf("%w: position %d for column of length %d", errs.ErrIndexOutOfBounds, i, len(c.codes))
	}
	if dtype.IsNA(v) {
		c.codes[i] = -1
		c.mask[i] = true
		return nil
	}
	code := c.CodeOf(v)
	if code < 0 {
		return fmt.Errorf("%w: %v is not a category", errs.ErrTypeMismatch, v)
	}
	c.codes[i] = code
	c.mask[i] = false
	return nil
}

func (c *CategoricalColumn) AppendValue(v any) error {
	c.codes = append(c.codes, -1)
	c.mask = append(c.mask, true)
	return c.SetValue(len(c.codes)-1, v)
}

func (c *CategoricalColumn) with(codes []int32, mask []bool) *CategoricalColumn {
	return &CategoricalColumn{
		codes: codes, mask: mask,
		categories: c.categories, // shared: immutable by invariant
		ordered:    c.ordered,
	}
}

func (c *CategoricalColumn) Take(indices []int) (Column, error) {
	codes := make([]int32, len(indices))
	mask := make([]bool, len(indices))
	for out, src := range indices {
		if src < 0 {
			codes[out] = -1
			mask[out] = true
			continue
		}
		if src >= len(c.codes) {
			return nil, fmt.Errorf("%w: take position %d for column of length %d", errs.ErrIndexOutOfBounds, src, len(c.codes))
		}
		codes[out] = c.codes[src]
		mask[out] = c.mask[src]
	}
	return c.with(codes, mask), nil
}

func (c *CategoricalColumn) Slice(start, stop int) (Column, error) {
	if start < 0 || stop < start || stop > len(c.codes) {
		return nil, fmt.Errorf("%w: slice [%d:%d] for column of length %d", errs.ErrIndexOutOfBounds, start, stop, len(c.codes))
	}
	return c.with(
		append([]int32(nil), c.codes[start:stop]...),
		append([]bool(nil), c.mask[start:stop]...),
	), nil
}

func (c *CategoricalColumn) Copy() Column {
	return c.with(
		append([]int32(nil), c.codes...),
		append([]bool(nil), c.mask...),
	)
}

func (c *CategoricalColumn) Values() []any {
	out := make([]any, len(c.codes))
	for i, code := range c.codes {
		if c.mask[i] {
			continue
		}
		out[i] = c.categories[code]
	}
	return out
}

// Float64s reports not-ok: categorical is not a numeric buffer (numeric
// reductions fall back to per-label conversion when labels are numeric).
func (c *CategoricalColumn) Float64s() ([]float64, []bool, bool) { return nil, nil, false }

// hashableLabel validates that a value can serve as a category label.
func hashableLabel(v any) bool {
	switch v.(type) {
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, string, time.Time:
		return true
	}
	return false
}

// sortLabels orders labels the way pandas builds default categories:
// ascending by value (numbers, strings or times).
func sortLabels(labels []any) {
	sort.SliceStable(labels, func(a, b int) bool {
		return labelLess(labels[a], labels[b])
	})
}

func labelLess(a, b any) bool {
	if fa, ok := dtype.AsFloat(a); ok {
		if fb, ok := dtype.AsFloat(b); ok {
			return fa < fb
		}
	}
	if sa, ok := a.(string); ok {
		if sb, ok := b.(string); ok {
			return sa < sb
		}
	}
	if ta, ok := a.(time.Time); ok {
		if tb, ok := b.(time.Time); ok {
			return ta.Before(tb)
		}
	}
	return false
}

// Factorize builds a categorical column from boxed values.
//
//   - With nil explicit categories, the category list is the SORTED set
//     of distinct labels (pandas' default for astype("category")).
//   - With explicit categories, their order is preserved; values outside
//     the list are an error (strict mode).
//
// NA values become code -1. Unhashable labels are an error.
func Factorize(values []any, explicit []any, ordered bool) (*CategoricalColumn, error) {
	var categories []any
	lookup := make(map[any]int32)
	if explicit != nil {
		for _, cat := range explicit {
			if dtype.IsNA(cat) || !hashableLabel(cat) {
				return nil, fmt.Errorf("%w: invalid category %v", errs.ErrTypeMismatch, cat)
			}
			if _, dup := lookup[cat]; dup {
				return nil, fmt.Errorf("%w: duplicate category %v", errs.ErrInvalidOperation, cat)
			}
			lookup[cat] = int32(len(categories))
			categories = append(categories, cat)
		}
	} else {
		seen := make(map[any]bool)
		for _, v := range values {
			if dtype.IsNA(v) {
				continue
			}
			if !hashableLabel(v) {
				return nil, fmt.Errorf("%w: cannot use %T as a category label", errs.ErrTypeMismatch, v)
			}
			if !seen[v] {
				seen[v] = true
				categories = append(categories, v)
			}
		}
		sortLabels(categories)
		for i, cat := range categories {
			lookup[cat] = int32(i)
		}
	}

	codes := make([]int32, len(values))
	mask := make([]bool, len(values))
	for i, v := range values {
		if dtype.IsNA(v) {
			codes[i] = -1
			mask[i] = true
			continue
		}
		if !hashableLabel(v) {
			return nil, fmt.Errorf("%w: cannot use %T as a category label", errs.ErrTypeMismatch, v)
		}
		code, ok := lookup[v]
		if !ok {
			return nil, fmt.Errorf("%w: %v is not in the explicit categories", errs.ErrTypeMismatch, v)
		}
		codes[i] = code
	}
	return &CategoricalColumn{codes: codes, categories: categories, ordered: ordered, mask: mask}, nil
}

// WithCategories rebuilds the column against a new category list:
// values whose category is absent from the new list become NA. Used by
// the Cat accessor operations.
func (c *CategoricalColumn) WithCategories(categories []any, ordered bool) (*CategoricalColumn, error) {
	lookup := make(map[any]int32, len(categories))
	for i, cat := range categories {
		if dtype.IsNA(cat) || !hashableLabel(cat) {
			return nil, fmt.Errorf("%w: invalid category %v", errs.ErrTypeMismatch, cat)
		}
		if _, dup := lookup[cat]; dup {
			return nil, fmt.Errorf("%w: duplicate category %v", errs.ErrInvalidOperation, cat)
		}
		lookup[cat] = int32(i)
	}
	// old code -> new code remap
	remap := make([]int32, len(c.categories))
	for i, cat := range c.categories {
		if code, ok := lookup[cat]; ok {
			remap[i] = code
		} else {
			remap[i] = -1
		}
	}
	codes := make([]int32, len(c.codes))
	mask := make([]bool, len(c.codes))
	for i, code := range c.codes {
		if c.mask[i] || remap[code] < 0 {
			codes[i] = -1
			mask[i] = true
			continue
		}
		codes[i] = remap[code]
	}
	return &CategoricalColumn{
		codes: codes, mask: mask,
		categories: append([]any(nil), categories...),
		ordered:    ordered,
	}, nil
}

// RenameCategories relabels categories, keeping codes.
func (c *CategoricalColumn) RenameCategories(mapping map[any]any) (*CategoricalColumn, error) {
	categories := make([]any, len(c.categories))
	seen := make(map[any]bool, len(c.categories))
	for i, cat := range c.categories {
		renamed := cat
		if to, ok := mapping[cat]; ok {
			renamed = to
		}
		if dtype.IsNA(renamed) || !hashableLabel(renamed) {
			return nil, fmt.Errorf("%w: invalid category %v", errs.ErrTypeMismatch, renamed)
		}
		if seen[renamed] {
			return nil, fmt.Errorf("%w: rename produces duplicate category %v", errs.ErrInvalidOperation, renamed)
		}
		seen[renamed] = true
		categories[i] = renamed
	}
	return &CategoricalColumn{
		codes:      append([]int32(nil), c.codes...),
		mask:       append([]bool(nil), c.mask...),
		categories: categories,
		ordered:    c.ordered,
	}, nil
}
