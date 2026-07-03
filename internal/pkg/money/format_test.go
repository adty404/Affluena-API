package money

import (
	"math"
	"testing"
)

func TestGroupIDR(t *testing.T) {
	cases := []struct {
		name string
		in   int64
		want string
	}{
		{"zero", 0, "0"},
		{"hundreds", 100, "100"},
		{"thousand", 1000, "1.000"},
		{"four digits", 9976, "9.976"},
		{"large", 20000000, "20.000.000"},
		{"spent example", 9976000, "9.976.000"},
		{"negative", -1500, "-1.500"},
		{"negative large", -20000000, "-20.000.000"},
		{"min int64", math.MinInt64, "-9.223.372.036.854.775.808"},
		{"max int64", math.MaxInt64, "9.223.372.036.854.775.807"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GroupIDR(tc.in); got != tc.want {
				t.Fatalf("GroupIDR(%d) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
