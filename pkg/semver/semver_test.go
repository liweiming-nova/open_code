package semver

import "testing"

func TestVersionComparisonOperators(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		gt   bool
		gte  bool
		lt   bool
		lte  bool
		eq   bool
	}{
		{name: "major greater", a: "2.0.0", b: "1.9.9", gt: true, gte: true},
		{name: "minor greater", a: "1.3.0", b: "1.2.9", gt: true, gte: true},
		{name: "patch less", a: "1.2.2", b: "1.2.3", lt: true, lte: true},
		{name: "equal", a: "1.2.3", b: "1.2.3", gte: true, lte: true, eq: true},
		{name: "metadata ignored equality", a: "1.2.3+build.1", b: "1.2.3+build.2", gte: true, lte: true, eq: true},
		{name: "release greater than prerelease", a: "1.0.0", b: "1.0.0-beta", gt: true, gte: true},
		{name: "prerelease numeric compare", a: "1.0.0-beta.2", b: "1.0.0-beta.10", lt: true, lte: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			av := MustNew(tt.a)
			bv := MustNew(tt.b)

			if got := av.GreaterThan(bv); got != tt.gt {
				t.Fatalf("GreaterThan: got %v, want %v", got, tt.gt)
			}
			if got := av.GreaterThanOrEqual(bv); got != tt.gte {
				t.Fatalf("GreaterThanOrEqual: got %v, want %v", got, tt.gte)
			}
			if got := av.LessThan(bv); got != tt.lt {
				t.Fatalf("LessThan: got %v, want %v", got, tt.lt)
			}
			if got := av.LessThanOrEqual(bv); got != tt.lte {
				t.Fatalf("LessThanOrEqual: got %v, want %v", got, tt.lte)
			}
			if got := av.Equal(bv); got != tt.eq {
				t.Fatalf("Equal: got %v, want %v", got, tt.eq)
			}
		})
	}
}

func TestCompareNilSafety(t *testing.T) {
	var v *Version
	if got := v.Compare(nil); got != 0 {
		t.Fatalf("nil compare nil: got %d, want 0", got)
	}
	if got := v.Compare(MustNew("1.0.0")); got != -1 {
		t.Fatalf("nil compare non-nil: got %d, want -1", got)
	}
	if got := MustNew("1.0.0").Compare(nil); got != 1 {
		t.Fatalf("non-nil compare nil: got %d, want 1", got)
	}
}
