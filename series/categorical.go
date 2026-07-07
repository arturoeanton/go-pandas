package series

import (
	"fmt"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/internal/column"
)

// CategoricalOptions configures categorical construction.
type CategoricalOptions struct {
	// Categories fixes the category list (and its order). Values outside
	// the list are an error (strict mode).
	Categories []any
	// Ordered marks the category order as semantically meaningful,
	// enabling ordered comparisons.
	Ordered bool
}

// CategoricalOption mutates CategoricalOptions.
type CategoricalOption func(*CategoricalOptions)

// WithCategories fixes the explicit category list and order.
func WithCategories(values ...any) CategoricalOption {
	return func(o *CategoricalOptions) { o.Categories = values }
}

// WithOrdered marks the categorical as ordered.
func WithOrdered(v bool) CategoricalOption {
	return func(o *CategoricalOptions) { o.Ordered = v }
}

// NewCategoricalSeries builds a categorical series from boxed values.
// Without explicit categories, the category list is the sorted distinct
// labels (pandas' astype("category") default).
func NewCategoricalSeries(name string, values []any, opts ...CategoricalOption) (*Series, error) {
	var o CategoricalOptions
	for _, f := range opts {
		f(&o)
	}
	col, err := column.Factorize(values, o.Categories, o.Ordered)
	if err != nil {
		return nil, err
	}
	return fromColumn(name, col, nil), nil
}

// CategoricalSeries builds a categorical series from strings.
func CategoricalSeries(name string, values []string, opts ...CategoricalOption) (*Series, error) {
	boxed := make([]any, len(values))
	for i, v := range values {
		boxed[i] = v
	}
	return NewCategoricalSeries(name, boxed, opts...)
}

// CategoricalAccessor exposes pandas' s.cat surface.
type CategoricalAccessor struct {
	s   *Series
	col *column.CategoricalColumn
}

// Cat returns the categorical accessor; non-categorical series error.
func (s *Series) Cat() (*CategoricalAccessor, error) {
	cc, ok := column.AsCategorical(s.col)
	if !ok {
		return nil, fmt.Errorf("%w: Cat() on %s series", errs.ErrInvalidDType, s.DType())
	}
	return &CategoricalAccessor{s: s, col: cc}, nil
}

// Categories returns the category labels.
func (c *CategoricalAccessor) Categories() []any { return c.col.Categories() }

// Codes returns the category codes (-1 = missing).
func (c *CategoricalAccessor) Codes() []int32 { return c.col.Codes() }

// Ordered reports whether the categorical is ordered.
func (c *CategoricalAccessor) Ordered() bool { return c.col.Ordered() }

func (c *CategoricalAccessor) rebuilt(col *column.CategoricalColumn) *Series {
	return fromColumn(c.s.name, col, c.s.index.Clone())
}

// RenameCategories relabels categories (codes unchanged).
func (c *CategoricalAccessor) RenameCategories(mapping map[any]any) (*Series, error) {
	col, err := c.col.RenameCategories(mapping)
	if err != nil {
		return nil, err
	}
	return c.rebuilt(col), nil
}

// ReorderCategories reorders the existing category set. The new list
// must contain exactly the current categories.
func (c *CategoricalAccessor) ReorderCategories(categories []any, ordered bool) (*Series, error) {
	if len(categories) != c.col.CategoryCount() {
		return nil, fmt.Errorf("%w: reorder must keep the same category set", errs.ErrInvalidOperation)
	}
	current := make(map[any]bool, c.col.CategoryCount())
	for _, cat := range c.col.Categories() {
		current[cat] = true
	}
	for _, cat := range categories {
		if !current[cat] {
			return nil, fmt.Errorf("%w: %v is not an existing category", errs.ErrInvalidOperation, cat)
		}
	}
	col, err := c.col.WithCategories(categories, ordered)
	if err != nil {
		return nil, err
	}
	return c.rebuilt(col), nil
}

// SetCategories replaces the category list; values whose category is
// removed become NA (documented behavior).
func (c *CategoricalAccessor) SetCategories(categories []any, ordered bool) (*Series, error) {
	col, err := c.col.WithCategories(categories, ordered)
	if err != nil {
		return nil, err
	}
	return c.rebuilt(col), nil
}

// AddCategories appends unused categories to the list.
func (c *CategoricalAccessor) AddCategories(categories ...any) (*Series, error) {
	return c.SetCategories(append(c.col.Categories(), categories...), c.col.Ordered())
}

// RemoveCategories drops categories; their values become NA.
func (c *CategoricalAccessor) RemoveCategories(categories ...any) (*Series, error) {
	drop := make(map[any]bool, len(categories))
	for _, cat := range categories {
		drop[cat] = true
	}
	var kept []any
	for _, cat := range c.col.Categories() {
		if !drop[cat] {
			kept = append(kept, cat)
		}
	}
	return c.SetCategories(kept, c.col.Ordered())
}

// asCategorical converts any series into categorical storage (used by
// Astype).
func (s *Series) asCategorical() (*Series, error) {
	if _, ok := column.AsCategorical(s.col); ok {
		return s.Copy(), nil
	}
	col, err := column.Factorize(s.col.Values(), nil, false)
	if err != nil {
		return nil, err
	}
	return fromColumn(s.name, col, s.index.Clone()), nil
}

// compare is the checked ordered-comparison path: unlike Series.Gt and
// friends (which have no error channel and fall back to all-false), it
// reports why a comparison is invalid.
func (c *CategoricalAccessor) compare(v any, satisfied func(cmp int) bool) (*Series, error) {
	if !c.col.Ordered() {
		return nil, fmt.Errorf("%w: ordered comparison on unordered categorical (set ordered via SetCategories or WithOrdered)", errs.ErrInvalidOperation)
	}
	if c.col.CodeOf(v) < 0 {
		return nil, fmt.Errorf("%w: %v is not a category", errs.ErrTypeMismatch, v)
	}
	return c.s.catOrderedCompare(c.col, v, satisfied), nil
}

// Gt, Ge, Lt, Le compare each row's category rank against a label.
// They error on unordered categoricals or unknown labels.
func (c *CategoricalAccessor) Gt(v any) (*Series, error) {
	return c.compare(v, func(x int) bool { return x > 0 })
}
func (c *CategoricalAccessor) Ge(v any) (*Series, error) {
	return c.compare(v, func(x int) bool { return x >= 0 })
}
func (c *CategoricalAccessor) Lt(v any) (*Series, error) {
	return c.compare(v, func(x int) bool { return x < 0 })
}
func (c *CategoricalAccessor) Le(v any) (*Series, error) {
	return c.compare(v, func(x int) bool { return x <= 0 })
}

// catOrderedCompare compares each row's category rank against a label
// (ordered categoricals only).
func (s *Series) catOrderedCompare(cc *column.CategoricalColumn, v any, satisfied func(c int) bool) *Series {
	target := cc.CodeOf(v)
	codes, mask := cc.RawCodes()
	return s.boolSeries(s.name, func(i int) bool {
		if mask[i] || target < 0 {
			return false
		}
		switch {
		case codes[i] < target:
			return satisfied(-1)
		case codes[i] > target:
			return satisfied(1)
		default:
			return satisfied(0)
		}
	})
}
