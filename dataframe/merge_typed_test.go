package dataframe

import (
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/index"
	"github.com/arturoeanton/go-pandas/series"
)

// TestMergeKeyDTypes joins on every typed key dtype.
func TestMergeKeyDTypes(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	left, _ := DataFrameFromMap(map[string][]any{
		"s": {"a", "b"}, "i": {1, 2}, "i64": {int64(1), int64(2)},
		"f": {1.5, 2.5}, "b": {true, false}, "t": {t0, t0.AddDate(0, 1, 0)},
		"l": {"L0", "L1"},
	}, WithColumnOrder("s", "i", "i64", "f", "b", "t", "l"))
	right, _ := DataFrameFromMap(map[string][]any{
		"s": {"b", "c"}, "i": {2, 3}, "i64": {int64(2), int64(3)},
		"f": {2.5, 3.5}, "b": {false, false}, "t": {t0.AddDate(0, 1, 0), t0.AddDate(0, 2, 0)},
		"r": {"R0", "R1"},
	}, WithColumnOrder("s", "i", "i64", "f", "b", "t", "r"))

	for _, key := range []string{"s", "i", "i64", "f", "b", "t"} {
		out, err := left.Merge(right, MergeOptions{On: []string{key}, How: "inner"})
		if err != nil {
			t.Fatalf("merge on %s: %v", key, err)
		}
		wantRows := 1
		if key == "b" {
			wantRows = 2 // left false matches both right falses
		}
		if out.Len() != wantRows {
			t.Errorf("merge on %s rows = %d, want %d", key, out.Len(), wantRows)
		}
		// key column keeps its dtype
		if got := out.MustCol(key).StorageDType(); got != left.MustCol(key).StorageDType() {
			t.Errorf("merge on %s key dtype = %v", key, got)
		}
	}
}

func TestMergeNAKeysNeverMatch(t *testing.T) {
	left, _ := DataFrameFromMap(map[string][]any{
		"id": {1, nil, 3}, "l": {"a", "b", "c"},
	}, WithColumnOrder("id", "l"))
	right, _ := DataFrameFromMap(map[string][]any{
		"id": {nil, 3}, "r": {"x", "y"},
	}, WithColumnOrder("id", "r"))

	inner, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "inner"})
	if err != nil {
		t.Fatal(err)
	}
	// documented difference from pandas: NA keys never match, so only
	// id=3 pairs (pandas would also pair the NaN keys)
	if inner.Len() != 1 {
		t.Fatalf("inner with NA keys = %d rows, want 1", inner.Len())
	}
	lj, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "left"})
	if err != nil {
		t.Fatal(err)
	}
	if lj.Len() != 3 {
		t.Fatalf("left with NA keys = %d rows", lj.Len())
	}
	if v := colValues(t, lj, "r"); v[1] != nil {
		t.Errorf("NA-key left row should have NA right side: %v", v)
	}
	outer, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "outer"})
	if err != nil {
		t.Fatal(err)
	}
	// 3 left rows + the unmatched NA right row
	if outer.Len() != 4 {
		t.Fatalf("outer with NA keys = %d rows", outer.Len())
	}
	if v := colValues(t, outer, "id"); v[3] != nil {
		t.Errorf("right NA key label = %v, want NA", v[3])
	}
	// multi-key with one NA component drops the row from matching
	l2, _ := DataFrameFromMap(map[string][]any{
		"a": {"x", "x"}, "b": {1, nil}, "v": {1, 2},
	}, WithColumnOrder("a", "b", "v"))
	r2, _ := DataFrameFromMap(map[string][]any{
		"a": {"x"}, "b": {1}, "w": {9},
	}, WithColumnOrder("a", "b", "w"))
	mk, err := l2.Merge(r2, MergeOptions{On: []string{"a", "b"}, How: "inner"})
	if err != nil {
		t.Fatal(err)
	}
	if mk.Len() != 1 {
		t.Errorf("multi-key NA component rows = %d", mk.Len())
	}
}

func TestMergeTypedStorageAndMasks(t *testing.T) {
	left, right := mergeFrames(t)
	outer, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "outer"})
	if err != nil {
		t.Fatal(err)
	}
	storage := outer.StorageDTypes()
	if storage["id"] != dtype.Int {
		t.Errorf("key storage = %v", storage["id"])
	}
	if storage["name"] != dtype.String || storage["salary"] != dtype.Float64 {
		t.Errorf("value storage = %v", storage)
	}
	for name := range storage {
		if outer.MustCol(name).IsObjectBacked() {
			t.Errorf("column %s object-backed after merge", name)
		}
	}
	// masks: right-only row has NA name; left-only has NA salary
	if v := colValues(t, outer, "name"); v[3] != nil {
		t.Errorf("right-only name = %v", v[3])
	}
	if v := colValues(t, outer, "salary"); v[2] != nil {
		t.Errorf("left-only salary = %v", v[2])
	}
	// indicator is a typed string column
	ind, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "outer", Indicator: true})
	if err != nil {
		t.Fatal(err)
	}
	if ind.MustCol("_merge").StorageDType() != dtype.String {
		t.Errorf("_merge storage = %v", ind.MustCol("_merge").StorageDType())
	}
}

func TestMergeImmutability(t *testing.T) {
	left, right := mergeFrames(t)
	lb, rb := left.ToRows(), right.ToRows()
	out, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "outer"})
	if err != nil {
		t.Fatal(err)
	}
	_ = out.MustCol("name").Set(0, "MUTATED")
	_ = out.MustCol("id").Set(0, 999)
	la, ra := left.ToRows(), right.ToRows()
	for i := range lb {
		for j := range lb[i] {
			if lb[i][j] != la[i][j] {
				t.Fatal("merge mutated left input")
			}
		}
	}
	for i := range rb {
		for j := range rb[i] {
			if rb[i][j] != ra[i][j] {
				t.Fatal("merge mutated right input")
			}
		}
	}
}

func TestJoinByIndexTypes(t *testing.T) {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		lIdx  index.Index
		rIdx  index.Index
		match int // rows in left join with matches
	}{
		{"range", index.NewRangeIndex(3), index.RangeIndexFrom(1, 4, 1), 3},
		{"int64", index.NewInt64Index([]int64{10, 20, 30}), index.NewInt64Index([]int64{20, 30, 40}), 3},
		{"string", index.NewStringIndex([]string{"a", "b", "c"}), index.NewStringIndex([]string{"b", "c", "d"}), 3},
		{"datetime",
			index.NewDatetimeIndex([]time.Time{t0, t0.AddDate(0, 1, 0), t0.AddDate(0, 2, 0)}),
			index.NewDatetimeIndex([]time.Time{t0.AddDate(0, 1, 0), t0.AddDate(0, 2, 0), t0.AddDate(0, 3, 0)}),
			3},
	}
	for _, tc := range cases {
		l, err := NewDataFrame(series.NewSeries("v", []any{1, 2, 3}, series.WithIndex(tc.lIdx)))
		if err != nil {
			t.Fatal(err)
		}
		r, err := NewDataFrame(series.NewSeries("w", []any{10, 20, 30}, series.WithIndex(tc.rIdx)))
		if err != nil {
			t.Fatal(err)
		}
		out, err := l.Join(r, JoinOptions{})
		if err != nil {
			t.Fatalf("%s join: %v", tc.name, err)
		}
		if out.Len() != tc.match {
			t.Fatalf("%s join rows = %d", tc.name, out.Len())
		}
		// first left label has no right match except when indexes overlap at start
		w := colValues(t, out, "w")
		if tc.name == "range" {
			// left labels 0,1,2; right 1,2,3 -> matches at 1,2
			if w[0] != nil || w[1] != 10 || w[2] != 20 {
				t.Errorf("range join = %v", w)
			}
		}
		// inner join drops unmatched
		inner, err := l.Join(r, JoinOptions{How: "inner"})
		if err != nil {
			t.Fatal(err)
		}
		if inner.Len() != 2 {
			t.Errorf("%s inner join rows = %d", tc.name, inner.Len())
		}
		// value dtype preserved
		if out.MustCol("v").StorageDType() != dtype.Int {
			t.Errorf("%s join dtype = %v", tc.name, out.MustCol("v").StorageDType())
		}
	}
}

func TestMergeObjectKeyFallback(t *testing.T) {
	obj := func(name string, values []any) *series.Series {
		return series.NewSeries(name, values, series.WithDType(dtype.Object))
	}
	left, err := NewDataFrame(obj("id", []any{1, 2}), obj("l", []any{"a", "b"}))
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewDataFrame(obj("id", []any{2, 3}), obj("r", []any{"x", "y"}))
	if err != nil {
		t.Fatal(err)
	}
	out, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "outer"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 3 {
		t.Fatalf("object fallback rows = %d", out.Len())
	}
	if v := colValues(t, out, "id"); v[2] != 3 {
		t.Errorf("object fallback keys = %v", v)
	}
}

func TestMergeDuplicateKeysDeterministic(t *testing.T) {
	left, _ := DataFrameFromMap(map[string][]any{
		"id": {1, 1}, "l": {"a", "b"},
	}, WithColumnOrder("id", "l"))
	right, _ := DataFrameFromMap(map[string][]any{
		"id": {1, 1, 1}, "r": {"x", "y", "z"},
	}, WithColumnOrder("id", "r"))
	out, err := left.Merge(right, MergeOptions{On: []string{"id"}, How: "inner"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 6 {
		t.Fatalf("duplicate cartesian rows = %d, want 6", out.Len())
	}
	// deterministic order: left row order, right matches in right order
	l := colValues(t, out, "l")
	r := colValues(t, out, "r")
	want := []struct{ l, r any }{
		{"a", "x"}, {"a", "y"}, {"a", "z"},
		{"b", "x"}, {"b", "y"}, {"b", "z"},
	}
	for i, w := range want {
		if l[i] != w.l || r[i] != w.r {
			t.Fatalf("pair %d = (%v,%v), want (%v,%v)", i, l[i], r[i], w.l, w.r)
		}
	}
}
