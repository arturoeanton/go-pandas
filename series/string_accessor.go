package series

import (
	"strings"
)

// StringAccessor exposes vectorized string methods, like Series.str.
type StringAccessor struct {
	s *Series
}

// Str returns the string accessor.
func (s *Series) Str() *StringAccessor { return &StringAccessor{s: s} }

// mapString applies f to every string value; non-strings and missing
// values yield missing results.
func (sa *StringAccessor) mapString(f func(s string) any) *Series {
	src := sa.s
	data := make([]any, src.Len())
	mask := make([]bool, src.Len())
	for i := range src.data {
		if src.mask[i] {
			mask[i] = true
			continue
		}
		str, ok := src.data[i].(string)
		if !ok {
			mask[i] = true
			continue
		}
		data[i] = f(str)
	}
	values := dataWithMask(data, mask)
	out := NewSeries(src.name, values, WithIndex(src.index))
	return out
}

// Contains reports whether each string contains substr.
func (sa *StringAccessor) Contains(substr string) *Series {
	return sa.mapString(func(s string) any { return strings.Contains(s, substr) })
}

// HasPrefix reports whether each string starts with prefix (str.startswith).
func (sa *StringAccessor) HasPrefix(prefix string) *Series {
	return sa.mapString(func(s string) any { return strings.HasPrefix(s, prefix) })
}

// HasSuffix reports whether each string ends with suffix (str.endswith).
func (sa *StringAccessor) HasSuffix(suffix string) *Series {
	return sa.mapString(func(s string) any { return strings.HasSuffix(s, suffix) })
}

// Lower lowercases each string.
func (sa *StringAccessor) Lower() *Series {
	return sa.mapString(func(s string) any { return strings.ToLower(s) })
}

// Upper uppercases each string.
func (sa *StringAccessor) Upper() *Series {
	return sa.mapString(func(s string) any { return strings.ToUpper(s) })
}

// Len returns the length of each string.
func (sa *StringAccessor) Len() *Series {
	return sa.mapString(func(s string) any { return len(s) })
}

// Strip trims surrounding whitespace.
func (sa *StringAccessor) Strip() *Series {
	return sa.mapString(func(s string) any { return strings.TrimSpace(s) })
}

// Replace substitutes every occurrence of old with new.
func (sa *StringAccessor) Replace(old, new string) *Series {
	return sa.mapString(func(s string) any { return strings.ReplaceAll(s, old, new) })
}

// Split splits each string by sep; each cell becomes a []string.
func (sa *StringAccessor) Split(sep string) *Series {
	return sa.mapString(func(s string) any { return strings.Split(s, sep) })
}
