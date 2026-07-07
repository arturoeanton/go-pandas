package pandas_test

// v0.3 acceptance tests: typed storage must be real, not logical
// metadata. Every assertion here inspects the physical backing.

import (
	"strings"
	"testing"

	pd "github.com/arturoeanton/go-pandas"
)

func TestNDArrayTypedBackings(t *testing.T) {
	cases := []struct {
		name    string
		arr     *pd.NDArray
		dt      pd.DType
		backing string
	}{
		{"ArrayInt", pd.ArrayInt([]int{1, 2, 3}), pd.Int, "[]int"},
		{"ArrayInt64", pd.ArrayInt64([]int64{1, 2}), pd.Int64, "[]int64"},
		{"ArrayFloat32", pd.ArrayFloat32([]float32{1.5}), pd.Float32, "[]float32"},
		{"ArrayFloat64", pd.ArrayFloat64([]float64{1.5}), pd.Float64, "[]float64"},
		{"ArrayBool", pd.ArrayBool([]bool{true, false}), pd.Bool, "[]bool"},
		{"ArrayString", pd.ArrayString([]string{"a", "b"}), pd.String, "[]string"},
	}
	for _, tc := range cases {
		if tc.arr.DType() != tc.dt {
			t.Errorf("%s dtype = %v, want %v", tc.name, tc.arr.DType(), tc.dt)
		}
		if tc.arr.StorageDType() != tc.dt {
			t.Errorf("%s storage dtype = %v, want %v", tc.name, tc.arr.StorageDType(), tc.dt)
		}
		switch tc.backing {
		case "[]int":
			if _, ok := tc.arr.RawData().([]int); !ok {
				t.Errorf("%s backing = %T", tc.name, tc.arr.RawData())
			}
		case "[]int64":
			if _, ok := tc.arr.RawData().([]int64); !ok {
				t.Errorf("%s backing = %T", tc.name, tc.arr.RawData())
			}
		case "[]float32":
			if _, ok := tc.arr.RawData().([]float32); !ok {
				t.Errorf("%s backing = %T", tc.name, tc.arr.RawData())
			}
		case "[]float64":
			if _, ok := tc.arr.RawData().([]float64); !ok {
				t.Errorf("%s backing = %T", tc.name, tc.arr.RawData())
			}
		case "[]bool":
			if _, ok := tc.arr.RawData().([]bool); !ok {
				t.Errorf("%s backing = %T", tc.name, tc.arr.RawData())
			}
		case "[]string":
			if _, ok := tc.arr.RawData().([]string); !ok {
				t.Errorf("%s backing = %T", tc.name, tc.arr.RawData())
			}
		}
	}
}

func TestNDArrayPromotion(t *testing.T) {
	ints := pd.ArrayInt([]int{1, 2, 3})
	floats := pd.ArrayFloat64([]float64{1.5, 2.5, 3.5})

	// int + float64 -> float64
	c, err := ints.Add(floats)
	if err != nil {
		t.Fatal(err)
	}
	if c.DType() != pd.Float64 {
		t.Errorf("int+float dtype = %v", c.DType())
	}
	if got := c.Data(); got[0] != 2.5 {
		t.Errorf("int+float = %v", got)
	}
	// int + int -> int (typed backing)
	ii, err := ints.Add(pd.ArrayInt([]int{10, 20, 30}))
	if err != nil {
		t.Fatal(err)
	}
	if ii.DType() != pd.Int {
		t.Errorf("int+int dtype = %v", ii.DType())
	}
	if _, ok := ii.RawData().([]int); !ok {
		t.Errorf("int+int backing = %T", ii.RawData())
	}
	// int + int64 -> int64
	i64, err := ints.Add(pd.ArrayInt64([]int64{1, 1, 1}))
	if err != nil {
		t.Fatal(err)
	}
	if i64.DType() != pd.Int64 {
		t.Errorf("int+int64 dtype = %v", i64.DType())
	}
	// float32 + float32 -> float32
	f32, err := pd.ArrayFloat32([]float32{1}).Add(pd.ArrayFloat32([]float32{2}))
	if err != nil {
		t.Fatal(err)
	}
	if f32.DType() != pd.Float32 {
		t.Errorf("f32+f32 dtype = %v", f32.DType())
	}
	// float32 + float64 -> float64
	f64, err := pd.ArrayFloat32([]float32{1}).Add(pd.ArrayFloat64([]float64{2}))
	if err != nil {
		t.Fatal(err)
	}
	if f64.DType() != pd.Float64 {
		t.Errorf("f32+f64 dtype = %v", f64.DType())
	}
	// bool + int -> int
	bi, err := pd.ArrayBool([]bool{true, false}).Add(pd.ArrayInt([]int{1, 1}))
	if err != nil {
		t.Fatal(err)
	}
	if bi.DType() != pd.Int {
		t.Errorf("bool+int dtype = %v", bi.DType())
	}
	// int / int -> float64 (true division)
	div, err := ints.Div(pd.ArrayInt([]int{2, 2, 2}))
	if err != nil {
		t.Fatal(err)
	}
	if div.DType() != pd.Float64 || div.Data()[0] != 0.5 {
		t.Errorf("int/int = %v %v", div.DType(), div.Data())
	}
	// string arithmetic errors
	if _, err := pd.ArrayString([]string{"a"}).Add(pd.ArrayInt([]int{1})); err == nil {
		t.Error("string arithmetic should error")
	}
}

func TestNDArrayStringOps(t *testing.T) {
	s := pd.ArrayString([]string{"b", "a", "b"})
	eq, err := s.Eq(pd.ArrayString([]string{"b", "b", "b"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := eq.Data(); !got[0] || got[1] || !got[2] {
		t.Errorf("string Eq = %v", got)
	}
	sorted := s.Sort()
	if vals := sorted.Values(); vals[0] != "a" || vals[2] != "b" {
		t.Errorf("string sort = %v", vals)
	}
	uniq := pd.Unique(s)
	if uniq.Size() != 2 {
		t.Errorf("string unique size = %d", uniq.Size())
	}
	if v, err := s.ValueAt(0); err != nil || v != "b" {
		t.Errorf("ValueAt = %v, %v", v, err)
	}
	if _, err := s.At(0); err == nil {
		t.Error("At on string array should error")
	}
	if s.Data() != nil {
		t.Error("Data() on string array should be nil")
	}
}

func TestSeriesTypedColumns(t *testing.T) {
	cases := []struct {
		s  *pd.Series
		dt pd.DType
	}{
		{pd.SeriesOf("x", []int{1, 2, 3}), pd.Int},
		{pd.SeriesOf("x", []int64{1}), pd.Int64},
		{pd.SeriesOf("x", []float64{1.5}), pd.Float64},
		{pd.SeriesOf("x", []string{"a"}), pd.String},
		{pd.SeriesOf("x", []bool{true}), pd.Bool},
	}
	for _, tc := range cases {
		if tc.s.DType() != tc.dt || tc.s.StorageDType() != tc.dt {
			t.Errorf("SeriesOf dtype=%v storage=%v, want %v", tc.s.DType(), tc.s.StorageDType(), tc.dt)
		}
		if tc.s.IsObjectBacked() {
			t.Errorf("SeriesOf(%v) is object-backed", tc.dt)
		}
	}
	// homogeneous []any becomes typed
	inferred := pd.NewSeries("x", []any{1, nil, 3})
	if inferred.StorageDType() != pd.Int || inferred.IsObjectBacked() {
		t.Errorf("[]any{1,nil,3} storage = %v", inferred.StorageDType())
	}
	if v, _ := inferred.At(1); v != nil {
		t.Errorf("masked slot = %v", v)
	}
	// mixed int/float promotes to Float64Column
	promoted := pd.NewSeries("x", []any{1, nil, 2.5})
	if promoted.StorageDType() != pd.Float64 || promoted.IsObjectBacked() {
		t.Errorf("mixed numeric storage = %v object=%v", promoted.StorageDType(), promoted.IsObjectBacked())
	}
	// mixed incompatible falls back to object
	object := pd.NewSeries("x", []any{1, "a"})
	if !object.IsObjectBacked() || object.StorageDType() != pd.Object {
		t.Errorf("mixed values should be object-backed, got %v", object.StorageDType())
	}
}

func TestSeriesAstypeChangesStorage(t *testing.T) {
	s := pd.SeriesOf("x", []int{1, 2, 3})
	f, err := s.Astype(pd.Float64)
	if err != nil {
		t.Fatal(err)
	}
	if f.StorageDType() != pd.Float64 || f.IsObjectBacked() {
		t.Errorf("astype float64 storage = %v", f.StorageDType())
	}
	back, err := f.Astype(pd.Int64)
	if err != nil {
		t.Fatal(err)
	}
	if back.StorageDType() != pd.Int64 {
		t.Errorf("astype int64 storage = %v", back.StorageDType())
	}
	str, err := s.Astype(pd.String)
	if err != nil {
		t.Fatal(err)
	}
	if str.StorageDType() != pd.String {
		t.Errorf("astype string storage = %v", str.StorageDType())
	}
	if v, _ := str.At(0); v != "1" {
		t.Errorf("astype string value = %v", v)
	}
	parsed, err := pd.StringSeries("x", []string{"42"}).Astype(pd.Int64)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := parsed.At(0); v != int64(42) {
		t.Errorf("string->int64 = %v (%T)", v, v)
	}
	if _, err := pd.StringSeries("x", []string{"abc"}).Astype(pd.Int64); err == nil {
		t.Error("invalid string->int64 should error")
	}
	// NA survives conversion
	withNA := pd.NewSeries("x", []any{1, nil})
	conv, err := withNA.Astype(pd.Float64)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := conv.At(1); v != nil {
		t.Errorf("NA after astype = %v", v)
	}
}

func TestDataFrameTypedInference(t *testing.T) {
	df, err := pd.DataFrameFromRecords([]map[string]any{
		{"name": "Ana", "age": 30, "salary": 1000.5, "active": true},
		{"name": "Luis", "age": nil, "salary": 2000.0, "active": false},
	}, pd.WithColumnOrder("name", "age", "salary", "active"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]pd.DType{
		"name": pd.String, "age": pd.Int, "salary": pd.Float64, "active": pd.Bool,
	}
	storage := df.StorageDTypes()
	for name, dt := range want {
		if storage[name] != dt {
			t.Errorf("column %s storage = %v, want %v", name, storage[name], dt)
		}
	}
	age := df.MustCol("age")
	if v, _ := age.At(1); v != nil {
		t.Errorf("masked int cell = %v", v)
	}
	// later float promotes the whole column
	mixed, _ := pd.DataFrameFromMap(map[string][]any{"v": {1, nil, 2.5}})
	if mixed.StorageDTypes()["v"] != pd.Float64 {
		t.Errorf("promoted column storage = %v", mixed.StorageDTypes()["v"])
	}
}

func TestReadCSVTypedColumns(t *testing.T) {
	csv := "name,age,salary,active\nAna,30,1000.5,true\nLuis,40,2000.0,false\nNA,NA,NA,NA\n"
	df, err := pd.ReadCSVReader(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	storage := df.StorageDTypes()
	want := map[string]pd.DType{
		"name": pd.String, "age": pd.Int, "salary": pd.Float64, "active": pd.Bool,
	}
	for name, dt := range want {
		if storage[name] != dt {
			t.Errorf("CSV column %s storage = %v, want %v", name, storage[name], dt)
		}
	}
	// the NA row is masked in every typed column
	for _, name := range df.Columns() {
		c := df.MustCol(name)
		if v, _ := c.At(2); v != nil {
			t.Errorf("CSV NA in %s = %v", name, v)
		}
	}
	// parse_dates produces a Time column
	dates, err := pd.ReadCSVReader(strings.NewReader("day\n2024-01-02\nNA\n"),
		pd.WithParseDates("day"))
	if err != nil {
		t.Fatal(err)
	}
	if dates.StorageDTypes()["day"] != pd.Time {
		t.Errorf("date column storage = %v", dates.StorageDTypes()["day"])
	}
	if v, _ := dates.MustCol("day").At(1); v != nil {
		t.Errorf("date NA = %v", v)
	}
}

func TestTypedNAMasksSurviveOps(t *testing.T) {
	s := pd.NewSeries("v", []any{1.0, nil, 3.0})
	sum, err := s.AddScalar(1)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := sum.At(1); v != nil {
		t.Errorf("NA after add = %v", v)
	}
	if sum.IsObjectBacked() {
		t.Error("arithmetic result should be typed")
	}
	filled := s.FillNA(0.0)
	if filled.HasNA() || filled.IsObjectBacked() {
		t.Errorf("FillNA typed: hasNA=%v object=%v", filled.HasNA(), filled.IsObjectBacked())
	}
	// filling with an incompatible value falls back to object storage
	mixed := pd.SeriesOf("v", []int{1}).FillNA("x")
	_ = mixed // no NA to fill; stays typed
	withNA := pd.NewSeries("v", []any{1, nil}).FillNA("x")
	if !withNA.IsObjectBacked() {
		t.Error("cross-type fill should fall back to object storage")
	}
	sorted := s.SortValues(true)
	if v, _ := sorted.At(2); v != nil {
		t.Errorf("NA should sort last, got %v", v)
	}
}
