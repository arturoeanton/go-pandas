package series_test

import (
	"errors"
	"testing"
	"time"

	"github.com/arturoeanton/go-pandas/dtype"
	"github.com/arturoeanton/go-pandas/errs"
	"github.com/arturoeanton/go-pandas/series"
)

func TestToDatetimeExplicitFormat(t *testing.T) {
	s, err := series.ToDatetime(
		series.StringSeries("d", []string{"01/02/2026", "28/12/2026"}),
		series.WithDatetimeFormat("%d/%m/%Y"))
	if err != nil {
		t.Fatal(err)
	}
	if s.DType() != dtype.Time {
		t.Fatalf("dtype = %v", s.DType())
	}
	want := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if got := s.Values()[0].(time.Time); !got.Equal(want) {
		t.Fatalf("day-first parse = %v, want %v", got, want)
	}
}

func TestToDatetimeInferredFormats(t *testing.T) {
	inputs := []string{
		"2026-01-02",
		"2026-01-02 15:04:05",
		"2026-01-02T15:04:05",
		"2026/01/02",
		"02/01/2026", // day-first wins for the ambiguous slash form
	}
	s, err := series.ToDatetime(series.StringSeries("d", inputs))
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range s.Values() {
		got := v.(time.Time)
		if got.Year() != 2026 || got.Month() != time.January || got.Day() != 2 {
			t.Fatalf("input %q parsed to %v", inputs[i], got)
		}
	}
}

func TestToDatetimeMicroseconds(t *testing.T) {
	s, err := series.ToDatetime(
		series.StringSeries("d", []string{"2026-01-02 10:00:00.123456", "2026-01-02 10:00:00.5"}),
		series.WithDatetimeFormat("%Y-%m-%d %H:%M:%S.%f"))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Values()[0].(time.Time).Nanosecond(); got != 123456000 {
		t.Fatalf("microseconds = %d", got)
	}
	if got := s.Values()[1].(time.Time).Nanosecond(); got != 500000000 {
		t.Fatalf("1-digit fraction = %d", got)
	}
}

func TestToDatetimeErrors(t *testing.T) {
	bad := series.StringSeries("d", []string{"2026-01-01", "nope", ""})

	if _, err := series.ToDatetime(bad); !errors.Is(err, errs.ErrTypeMismatch) {
		t.Fatalf("raise mode must error, got %v", err)
	}

	s, err := series.ToDatetime(bad, series.WithDatetimeErrors("coerce"))
	if err != nil {
		t.Fatal(err)
	}
	v := s.Values()
	if v[0] == nil || v[1] != nil || v[2] != nil {
		t.Fatalf("coerce values = %v", v)
	}

	if _, err := series.ToDatetime(bad, series.WithDatetimeErrors("ignore")); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("ignore mode must be rejected, got %v", err)
	}
	if _, err := series.ToDatetime(bad, series.WithDatetimeErrors("bogus")); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("unknown mode must error, got %v", err)
	}
}

func TestToDatetimeInvalidDirective(t *testing.T) {
	if _, err := series.ToDatetime(series.StringSeries("d", []string{"x"}),
		series.WithDatetimeFormat("%Q")); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("unknown directive must error, got %v", err)
	}
	if _, err := series.ToDatetime(series.StringSeries("d", []string{"x"}),
		series.WithDatetimeFormat("%f")); !errors.Is(err, errs.ErrInvalidOperation) {
		t.Fatalf("%%f without dot must error, got %v", err)
	}
	if _, err := dtype.TranslateTimeFormat("%Y-%"); err == nil {
		t.Fatal("dangling %% must error")
	}
}

func TestToDatetimePassThroughAndNA(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	s, err := series.ToDatetime(series.NewSeries("d", []any{now, nil, "2026-01-01"}))
	if err != nil {
		t.Fatal(err)
	}
	v := s.Values()
	if !v[0].(time.Time).Equal(now) {
		t.Fatalf("time.Time must pass through: %v", v[0])
	}
	if v[1] != nil {
		t.Fatal("nil must stay NA")
	}
	if v[2].(time.Time).Day() != 1 {
		t.Fatalf("mixed string parse = %v", v[2])
	}
}

func TestToDatetimeUnit(t *testing.T) {
	s, err := series.ToDatetime(series.NewSeries("d", []any{int64(1767225600)}),
		series.WithDatetimeUnit("s"))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Values()[0].(time.Time).UTC().Format("2006-01-02"); got != "2026-01-01" {
		t.Fatalf("unix seconds = %v", got)
	}
	// Numeric without a unit is an error under raise.
	if _, err := series.ToDatetime(series.NewSeries("d", []any{1.5})); err == nil {
		t.Fatal("numeric without unit must error")
	}
}

func TestToDatetimePreservesMaskAndInput(t *testing.T) {
	src := series.NewSeries("d", []any{"2026-01-01", nil})
	s, err := series.ToDatetime(src)
	if err != nil {
		t.Fatal(err)
	}
	if s.Values()[1] != nil {
		t.Fatal("mask lost")
	}
	if src.Values()[0] != "2026-01-01" {
		t.Fatal("input mutated")
	}
	if src.DType() == dtype.Time {
		t.Fatal("input dtype changed")
	}
}
