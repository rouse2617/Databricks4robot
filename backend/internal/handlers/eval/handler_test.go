package eval

import "testing"

func TestNormalizeMetricOp(t *testing.T) {
	cases := map[string]string{
		"eq":  "=",
		"=":   "=",
		"gt":  ">",
		">":   ">",
		"gte": ">=",
		">=":  ">=",
		"lt":  "<",
		"<":   "<",
		"lte": "<=",
		"<=":  "<=",
		"bad": "",
	}

	for in, want := range cases {
		got := normalizeMetricOp(in)
		if got != want {
			t.Fatalf("normalizeMetricOp(%q)=%q want %q", in, got, want)
		}
	}
}
