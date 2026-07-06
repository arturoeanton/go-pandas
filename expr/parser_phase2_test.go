package expr

import "testing"

func TestParseQueryPhase2(t *testing.T) {
	r := map[string]any{"name": "Anaconda", "active": true, "age": 30}
	cases := []struct {
		q    string
		want bool
	}{
		{`name.str.contains("cond")`, true},
		{`name.str.contains("xyz")`, false},
		{`name.str.startswith("Ana")`, true},
		{`name.str.endswith("da")`, true},
		{`active`, true},
		{`not active`, false},
		{`active and age > 20`, true},
		{`age > 40 or active`, true},
	}
	for _, tc := range cases {
		pred, err := ParseQuery(tc.q)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.q, err)
		}
		got, err := pred.EvalBool(r)
		if err != nil {
			t.Fatalf("eval %q: %v", tc.q, err)
		}
		if got != tc.want {
			t.Errorf("%q = %v, want %v", tc.q, got, tc.want)
		}
	}
	if _, err := ParseQuery(`name.str.wat("x")`); err == nil {
		t.Error("unknown str method should fail")
	}
	if _, err := ParseQuery(`name.str.contains(42)`); err == nil {
		t.Error("non-string str argument should fail")
	}
}
