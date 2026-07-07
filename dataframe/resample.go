package dataframe

import (
	"fmt"
	"sort"
	"time"

	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/index"
	gby "github.com/arturoeanton/go-pandas/internal/groupby"
	"github.com/arturoeanton/go-pandas/series"
)

// frequencyKind is a supported resample bucket size (v0.9).
type frequencyKind int

const (
	freqHour frequencyKind = iota
	freqDay
	freqWeek
	freqMonthStart
	freqMonthEnd
)

// parseFrequency resolves the stable Go aliases. Both cases are
// accepted; "M" means month-START in go-pandas (a documented difference:
// pandas' "M"/"ME" are month-end) — use "MS"/"ME" to be explicit.
func parseFrequency(alias string) (frequencyKind, error) {
	switch alias {
	case "h", "H", "hour":
		return freqHour, nil
	case "d", "D", "day":
		return freqDay, nil
	case "w", "W", "week":
		return freqWeek, nil
	case "m", "M", "MS", "ms", "month":
		return freqMonthStart, nil
	case "ME", "me":
		return freqMonthEnd, nil
	}
	return 0, fmt.Errorf("%w: unknown resample frequency %q (use H, D, W, MS or ME)", errs.ErrInvalidOperation, alias)
}

// floorTime truncates a timestamp to its bucket start. Weeks anchor on
// Monday 00:00; months bucket by calendar month (the START is the
// bucket key for both MS and ME — ME only changes the output label).
func floorTime(t time.Time, kind frequencyKind) time.Time {
	switch kind {
	case freqHour:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case freqDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case freqWeek:
		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		back := (int(day.Weekday()) + 6) % 7 // Monday = 0
		return day.AddDate(0, 0, -back)
	default: // freqMonthStart, freqMonthEnd
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	}
}

// bucketLabel converts a bucket key into its output index label: the
// bucket start, except ME which labels the last calendar day 00:00
// (pandas month-end labels).
func bucketLabel(bucket time.Time, kind frequencyKind) time.Time {
	if kind == freqMonthEnd {
		return bucket.AddDate(0, 1, -1)
	}
	return bucket
}

// Resampler is a deferred time-bucket grouping over a DatetimeIndex,
// like df.resample("D"). Only observed buckets are emitted (documented).
type Resampler struct {
	df   *DataFrame
	kind frequencyKind
	err  error
}

// Resample buckets rows by their DatetimeIndex timestamp. Frequencies:
// H (hour), D (day), W (week, Monday anchor), MS (month-start; "M" is
// an alias for MS, a documented difference from pandas), ME (month-end
// labels). The frame's index must be a DatetimeIndex; rows with NA
// timestamps are skipped; input order does not matter.
func (df *DataFrame) Resample(freq string) *Resampler {
	r := &Resampler{df: df}
	if _, isMulti := df.index.(*index.MultiIndex); isMulti {
		r.err = errs.NotImplemented("resample by MultiIndex datetime level")
		return r
	}
	if _, ok := df.index.(*index.DatetimeIndex); !ok {
		r.err = fmt.Errorf("%w: Resample needs a DatetimeIndex (index is %T); SetIndex a datetime column first", errs.ErrInvalidIndex, df.index)
		return r
	}
	kind, err := parseFrequency(freq)
	if err != nil {
		r.err = err
		return r
	}
	r.kind = kind
	return r
}

// buildPlan floors every timestamp to its bucket, maps buckets to dense
// group ids and orders groups by bucket time ascending — the shape the
// typed groupby reducers consume. No per-row boxing.
func (r *Resampler) buildPlan() (*groupPlan, []time.Time, error) {
	if r.err != nil {
		return nil, nil, r.err
	}
	di := r.df.index.(*index.DatetimeIndex)
	times, mask := di.RawTimes()

	ids := make([]int, len(times))
	seen := make(map[int64]int) // bucket unix-nano -> group id
	var buckets []time.Time
	var first []int
	for i, t := range times {
		if mask != nil && mask[i] {
			ids[i] = -1 // NA timestamps are skipped
			continue
		}
		b := floorTime(t, r.kind)
		key := b.UnixNano()
		g, ok := seen[key]
		if !ok {
			g = len(buckets)
			seen[key] = g
			buckets = append(buckets, b)
			first = append(first, i)
		}
		ids[i] = g
	}
	// Output order: buckets ascending.
	order := make([]int, len(buckets))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		return buckets[order[a]].Before(buckets[order[b]])
	})
	sorted := make([]time.Time, len(order))
	for i, g := range order {
		sorted[i] = bucketLabel(buckets[g], r.kind)
	}
	plan := &gby.Plan{GroupIDs: ids, Count: len(buckets), FirstRow: first}
	return &groupPlan{plan: plan, order: order}, sorted, nil
}

// aggregate runs one aggregation over every applicable column.
// numericOnly (sum/mean) skips non-numeric columns like pandas'
// numeric_only=True; the other aggregations use their typed kernels for
// strings/times too.
func (r *Resampler) aggregate(agg string, numericOnly bool) (*DataFrame, error) {
	gp, buckets, err := r.buildPlan()
	if err != nil {
		return nil, err
	}
	di := r.df.index.(*index.DatetimeIndex)
	idx := index.NewDatetimeIndex(buckets, di.Name())

	// Reuse the GroupBy typed reducers through a shell grouping.
	gb := &GroupBy{df: r.df}
	targets, err := gb.valueColumns(nil, numericOnly)
	if err != nil {
		return nil, err
	}
	cols := make([]*series.Series, 0, len(targets))
	for _, s := range targets {
		out, err := gb.aggregateColumn(gp, s, agg)
		if err != nil {
			return nil, err
		}
		cols = append(cols, series.Assemble(s.Name(), out, idx))
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("%w: no columns to aggregate with %s", errs.ErrInvalidOperation, agg)
	}
	return newFrame(cols, idx)
}

// Sum aggregates numeric columns per bucket (NA skipped; all-NA -> 0,
// matching pandas sum semantics).
func (r *Resampler) Sum() (*DataFrame, error) { return r.aggregate("sum", true) }

// Mean averages numeric columns per bucket (all-NA buckets -> NA).
func (r *Resampler) Mean() (*DataFrame, error) { return r.aggregate("mean", true) }

// Count counts non-NA values per bucket for every column.
func (r *Resampler) Count() (*DataFrame, error) { return r.aggregate("count", false) }

// Min takes the per-bucket minimum (numeric, string and time columns).
func (r *Resampler) Min() (*DataFrame, error) { return r.aggregate("min", false) }

// Max takes the per-bucket maximum (numeric, string and time columns).
func (r *Resampler) Max() (*DataFrame, error) { return r.aggregate("max", false) }

// First takes the first non-NA value per bucket in row order.
func (r *Resampler) First() (*DataFrame, error) { return r.aggregate("first", false) }

// Last takes the last non-NA value per bucket in row order.
func (r *Resampler) Last() (*DataFrame, error) { return r.aggregate("last", false) }
