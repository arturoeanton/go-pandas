package groupby

import (
	"math"
	"sort"
	"time"
)

// Segment reducers: every function makes one pass over the rows (plus a
// per-group pass for finalization) driven by GroupIDs. No sub-frame, no
// boxing.

// Sizes counts all rows per group (NA values included).
func Sizes(ids []int, count int) []int {
	out := make([]int, count)
	for _, g := range ids {
		if g >= 0 {
			out[g]++
		}
	}
	return out
}

// CountNonNA counts present values per group.
func CountNonNA(mask []bool, ids []int, count int) []int {
	out := make([]int, count)
	for i, g := range ids {
		if g >= 0 && !mask[i] {
			out[g]++
		}
	}
	return out
}

// SumFloat sums present values per group (empty groups sum to 0, like
// Series.Sum).
func SumFloat(vals []float64, mask []bool, ids []int, count int) []float64 {
	out := make([]float64, count)
	for i, g := range ids {
		if g >= 0 && !mask[i] {
			out[g] += vals[i]
		}
	}
	return out
}

// MeanFloat averages present values per group; groups without values are
// masked.
func MeanFloat(vals []float64, mask []bool, ids []int, count int) ([]float64, []bool) {
	sums := SumFloat(vals, mask, ids, count)
	counts := CountNonNA(mask, ids, count)
	na := make([]bool, count)
	for g := range sums {
		if counts[g] == 0 {
			na[g] = true
			continue
		}
		sums[g] /= float64(counts[g])
	}
	return sums, na
}

// VarFloat computes the two-pass sample variance (ddof=1) per group;
// groups with fewer than two values are masked. std toggles the square
// root.
func VarFloat(vals []float64, mask []bool, ids []int, count int, std bool) ([]float64, []bool) {
	means, _ := MeanFloat(vals, mask, ids, count)
	counts := CountNonNA(mask, ids, count)
	acc := make([]float64, count)
	for i, g := range ids {
		if g >= 0 && !mask[i] {
			d := vals[i] - means[g]
			acc[g] += d * d
		}
	}
	na := make([]bool, count)
	for g := range acc {
		if counts[g] < 2 {
			acc[g] = 0
			na[g] = true
			continue
		}
		acc[g] /= float64(counts[g] - 1)
		if std {
			acc[g] = math.Sqrt(acc[g])
		}
	}
	return acc, na
}

// MedianFloat gathers present values into per-group segments of one
// shared buffer, sorts each segment and interpolates the middle.
func MedianFloat(vals []float64, mask []bool, ids []int, count int) ([]float64, []bool) {
	counts := CountNonNA(mask, ids, count)
	offsets := make([]int, count+1)
	for g := 0; g < count; g++ {
		offsets[g+1] = offsets[g] + counts[g]
	}
	buf := make([]float64, offsets[count])
	cursor := append([]int(nil), offsets[:count]...)
	for i, g := range ids {
		if g >= 0 && !mask[i] {
			buf[cursor[g]] = vals[i]
			cursor[g]++
		}
	}
	out := make([]float64, count)
	na := make([]bool, count)
	for g := 0; g < count; g++ {
		seg := buf[offsets[g]:offsets[g+1]]
		if len(seg) == 0 {
			na[g] = true
			continue
		}
		sort.Float64s(seg)
		mid := len(seg) / 2
		if len(seg)%2 == 1 {
			out[g] = seg[mid]
		} else {
			out[g] = (seg[mid-1] + seg[mid]) / 2
		}
	}
	return out, na
}

// FirstIdx returns the first non-NA row per group (-1 when the group has
// none).
func FirstIdx(mask []bool, ids []int, count int) []int {
	out := fillNeg(count)
	for i, g := range ids {
		if g >= 0 && !mask[i] && out[g] == -1 {
			out[g] = i
		}
	}
	return out
}

// LastIdx returns the last non-NA row per group.
func LastIdx(mask []bool, ids []int, count int) []int {
	out := fillNeg(count)
	for i, g := range ids {
		if g >= 0 && !mask[i] {
			out[g] = i
		}
	}
	return out
}

// MinIdxFloat / MaxIdxFloat return the row holding the extreme numeric
// value per group; ties keep the earliest row (stable, matching the old
// sequential scan).
func MinIdxFloat(vals []float64, mask []bool, ids []int, count int) []int {
	return extremeIdxFloat(vals, mask, ids, count, func(a, b float64) bool { return a < b })
}

func MaxIdxFloat(vals []float64, mask []bool, ids []int, count int) []int {
	return extremeIdxFloat(vals, mask, ids, count, func(a, b float64) bool { return a > b })
}

func extremeIdxFloat(vals []float64, mask []bool, ids []int, count int, better func(a, b float64) bool) []int {
	out := fillNeg(count)
	for i, g := range ids {
		if g < 0 || mask[i] {
			continue
		}
		if out[g] == -1 || better(vals[i], vals[out[g]]) {
			out[g] = i
		}
	}
	return out
}

// MinIdxString / MaxIdxString order strings lexicographically.
func MinIdxString(vals []string, mask []bool, ids []int, count int) []int {
	return extremeIdxString(vals, mask, ids, count, func(a, b string) bool { return a < b })
}

func MaxIdxString(vals []string, mask []bool, ids []int, count int) []int {
	return extremeIdxString(vals, mask, ids, count, func(a, b string) bool { return a > b })
}

func extremeIdxString(vals []string, mask []bool, ids []int, count int, better func(a, b string) bool) []int {
	out := fillNeg(count)
	for i, g := range ids {
		if g < 0 || mask[i] {
			continue
		}
		if out[g] == -1 || better(vals[i], vals[out[g]]) {
			out[g] = i
		}
	}
	return out
}

// MinIdxTime / MaxIdxTime order timestamps chronologically.
func MinIdxTime(vals []time.Time, mask []bool, ids []int, count int) []int {
	return extremeIdxTime(vals, mask, ids, count, func(a, b time.Time) bool { return a.Before(b) })
}

func MaxIdxTime(vals []time.Time, mask []bool, ids []int, count int) []int {
	return extremeIdxTime(vals, mask, ids, count, func(a, b time.Time) bool { return a.After(b) })
}

func extremeIdxTime(vals []time.Time, mask []bool, ids []int, count int, better func(a, b time.Time) bool) []int {
	out := fillNeg(count)
	for i, g := range ids {
		if g < 0 || mask[i] {
			continue
		}
		if out[g] == -1 || better(vals[i], vals[out[g]]) {
			out[g] = i
		}
	}
	return out
}

// NUniqueFloat counts distinct present numeric values per group through
// one shared (group, value) set.
func NUniqueFloat(vals []float64, mask []bool, ids []int, count int) []int {
	type key struct {
		g int
		v float64
	}
	seen := make(map[key]struct{})
	out := make([]int, count)
	for i, g := range ids {
		if g < 0 || mask[i] {
			continue
		}
		k := key{g, vals[i]}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			out[g]++
		}
	}
	return out
}

// NUniqueString counts distinct present strings per group.
func NUniqueString(vals []string, mask []bool, ids []int, count int) []int {
	type key struct {
		g int
		v string
	}
	seen := make(map[key]struct{})
	out := make([]int, count)
	for i, g := range ids {
		if g < 0 || mask[i] {
			continue
		}
		k := key{g, vals[i]}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			out[g]++
		}
	}
	return out
}

// NUniqueTime counts distinct present timestamps per group.
func NUniqueTime(vals []time.Time, mask []bool, ids []int, count int) []int {
	type key struct {
		g int
		v time.Time
	}
	seen := make(map[key]struct{})
	out := make([]int, count)
	for i, g := range ids {
		if g < 0 || mask[i] {
			continue
		}
		k := key{g, vals[i]}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			out[g]++
		}
	}
	return out
}

func fillNeg(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = -1
	}
	return out
}
