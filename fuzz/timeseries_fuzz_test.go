package fuzz_test

import (
	"fmt"
	"testing"
	"time"

	pd "github.com/arturoeanton/go-pandas"
	"github.com/arturoeanton/go-pandas/index"
)

// fuzzTimes derives deterministic timestamps (with NAs) from fuzz input.
func fuzzTimes(seed int8, n int) []any {
	mod := func(x, m int) int { return ((x % m) + m) % m }
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]any, n)
	for i := 0; i < n; i++ {
		k := mod(i*13+int(seed), 17)
		if k == 16 {
			continue // NA
		}
		// spread over ~10 days, minute granularity, unsorted
		out[i] = base.Add(time.Duration(mod(k*i*37+int(seed), 14400)) * time.Minute)
	}
	return out
}

// FuzzToDatetimeFormats round-trips times through strings under both
// explicit formats and inference; invalid strings must error under
// raise and become NA under coerce — never panic.
func FuzzToDatetimeFormats(f *testing.F) {
	f.Add(int8(1), uint8(12), true)
	f.Add(int8(-7), uint8(40), false)
	f.Fuzz(func(t *testing.T, seed int8, size uint8, useFormat bool) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		n := int(size)%48 + 1
		times := fuzzTimes(seed, n)
		strs := make([]any, n)
		for i, v := range times {
			if v == nil {
				continue
			}
			if mod(i+int(seed), 9) == 8 {
				strs[i] = fmt.Sprintf("garbage-%d", i)
				continue
			}
			strs[i] = v.(time.Time).Format("2006-01-02 15:04:05")
		}
		hasGarbage := false
		for _, v := range strs {
			if s, ok := v.(string); ok && len(s) > 7 && s[:7] == "garbage" {
				hasGarbage = true
			}
		}
		opts := []pd.DatetimeOption{pd.WithDatetimeErrors("coerce")}
		if useFormat {
			opts = append(opts, pd.WithDatetimeFormat("%Y-%m-%d %H:%M:%S"))
		}
		s, err := pd.ToDatetime(pd.NewSeries("d", strs), opts...)
		if err != nil {
			t.Fatalf("coerce must not error: %v", err)
		}
		for i, v := range s.Values() {
			orig := times[i]
			if orig == nil {
				if v != nil {
					t.Fatalf("row %d: NA lost", i)
				}
				continue
			}
			if str, ok := strs[i].(string); ok && len(str) > 7 && str[:7] == "garbage" {
				if v != nil {
					t.Fatalf("row %d: garbage must coerce to NA, got %v", i, v)
				}
				continue
			}
			want := orig.(time.Time).Truncate(time.Second)
			if !v.(time.Time).Equal(want) {
				t.Fatalf("row %d: roundtrip %v != %v", i, v, want)
			}
		}
		// Raise mode must error iff garbage present.
		_, err = pd.ToDatetime(pd.NewSeries("d", strs))
		if hasGarbage && err == nil {
			t.Fatal("raise mode must error on garbage")
		}
		if !hasGarbage && err != nil {
			t.Fatalf("raise mode errored on clean input: %v", err)
		}
	})
}

// FuzzDatetimeIndexTake checks mask-aware typed gather invariants.
func FuzzDatetimeIndexTake(f *testing.F) {
	f.Add(int8(3), uint8(20), uint8(30))
	f.Add(int8(-2), uint8(6), uint8(3))
	f.Fuzz(func(t *testing.T, seed int8, size, nTake uint8) {
		mod := func(x, m int) int { return ((x % m) + m) % m }
		n := int(size)%48 + 1
		boxed := fuzzTimes(seed, n)
		values := make([]time.Time, n)
		mask := make([]bool, n)
		for i, v := range boxed {
			if v == nil {
				mask[i] = true
				continue
			}
			values[i] = v.(time.Time)
		}
		ix := index.NewDatetimeIndexWithMask(values, mask, "d").(*index.DatetimeIndex)
		before := fmt.Sprint(ix.Values())

		positions := make([]int, int(nTake)%64)
		for i := range positions {
			p := mod(i*11+int(seed), n+1)
			if p == n {
				p = -1
			}
			positions[i] = p
		}
		taken := ix.Take(positions).(*index.DatetimeIndex)
		if taken.Len() != len(positions) {
			t.Fatalf("take len = %d", taken.Len())
		}
		for i, p := range positions {
			got := taken.At(i)
			if p < 0 {
				if got != nil {
					t.Fatalf("negative position must be NA, got %v", got)
				}
				continue
			}
			want := ix.At(p)
			if want == nil {
				if got != nil {
					t.Fatalf("NA label lost at %d", i)
				}
				continue
			}
			if !got.(time.Time).Equal(want.(time.Time)) {
				t.Fatalf("take %d: %v != %v", i, got, want)
			}
		}
		if fmt.Sprint(ix.Values()) != before {
			t.Fatal("Take mutated the source index")
		}
	})
}

// fuzzResample runs one frequency and checks the shared properties.
func fuzzResample(t *testing.T, seed int8, size uint8, freq string) {
	n := int(size)%64 + 1
	times := fuzzTimes(seed, n)
	vals := make([]any, n)
	nonNATimes := 0
	for i := range vals {
		vals[i] = float64(i)
		if times[i] != nil {
			nonNATimes++
		}
	}
	dates, err := pd.ToDatetime(pd.NewSeries("date", times))
	if err != nil {
		t.Fatal(err)
	}
	df, err := pd.NewDataFrame(dates, pd.NewSeries("v", vals))
	if err != nil {
		t.Fatal(err)
	}
	indexed, err := df.SetIndex("date")
	if err != nil {
		t.Fatal(err)
	}
	before := fmt.Sprint(indexed.ToRows())
	if nonNATimes == 0 {
		return // no observable buckets; nothing to assert
	}
	out, err := indexed.Resample(freq).Sum()
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 || out.Len() > nonNATimes {
		t.Fatalf("buckets = %d for %d timed rows", out.Len(), nonNATimes)
	}
	di := out.Index().(*index.DatetimeIndex)
	if !di.IsMonotonicIncreasing() {
		t.Fatal("bucket labels must be sorted ascending")
	}
	// Total sum is preserved (sum over buckets == sum over timed rows).
	var want float64
	for i := range vals {
		if times[i] != nil {
			want += float64(i)
		}
	}
	var got float64
	for _, v := range out.MustCol("v").Values() {
		got += v.(float64)
	}
	if got != want {
		t.Fatalf("sum not preserved: %v != %v", got, want)
	}
	if fmt.Sprint(indexed.ToRows()) != before {
		t.Fatal("Resample mutated the input")
	}
}

func FuzzResampleDaily(f *testing.F) {
	f.Add(int8(5), uint8(24))
	f.Add(int8(-1), uint8(60))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		fuzzResample(t, seed, size, "D")
	})
}

func FuzzResampleHourly(f *testing.F) {
	f.Add(int8(2), uint8(30))
	f.Add(int8(-9), uint8(7))
	f.Fuzz(func(t *testing.T, seed int8, size uint8) {
		fuzzResample(t, seed, size, "H")
	})
}
