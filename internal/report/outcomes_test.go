package report

import (
	"math"
	"strings"
	"testing"
)

// Kaplan–Meier by hand: events at 1, 3, 3; censored at 2 and 4. S = 4/5
// after 1, then × 1/3 at 3 (3 at risk, the run censored at 2 gone);
// Greenwood: 1/(5·4) + 2/(3·1).
func TestKaplanMeier(t *testing.T) {
	steps := kaplanMeier([]obs{{3, true}, {2, false}, {1, true}, {4, false}, {3, true}})
	want := []step{{0, 1, 0}, {1, 0.8, 0.8 * math.Sqrt(0.05)}, {3, 0.8 / 3, 0.8 / 3 * math.Sqrt(0.05+2.0/3)}}
	if len(steps) != len(want) {
		t.Fatalf("steps %+v, want %+v", steps, want)
	}
	for i := range want {
		if math.Abs(steps[i].t-want[i].t) > 1e-12 || math.Abs(steps[i].s-want[i].s) > 1e-12 || math.Abs(steps[i].se-want[i].se) > 1e-12 {
			t.Errorf("step %d: %+v, want %+v", i, steps[i], want[i])
		}
	}
	if s := at(steps, 2.5).s; s != 0.8 {
		t.Errorf("S(2.5) = %v, want 0.8", s)
	}
	if m := median50(steps); m != "3.0 years" {
		t.Errorf("median %s, want 3.0 years", m)
	}
}

// Wilson's interval for 5 of 10 is 0.2366–0.7634; the Poisson interval
// for 0 is 0–3.69, and for 100 about 81.4–120.6.
func TestIntervals(t *testing.T) {
	if l, u := wilson(5, 10); math.Abs(l-0.2366) > 1e-4 || math.Abs(u-0.7634) > 1e-4 {
		t.Errorf("wilson(5, 10) = %.4f–%.4f", l, u)
	}
	if l, u := poisson(0); l != 0 || u != 3.69 {
		t.Errorf("poisson(0) = %v–%v", l, u)
	}
	if l, u := poisson(100); math.Abs(l-81.36) > 0.1 || math.Abs(u-120.56) > 0.1 {
		t.Errorf("poisson(100) = %.2f–%.2f", l, u)
	}
}

// Two runs of 100 measured years: one with a suppressed rebellion of 10
// years starting in year 20, then a collapse by air in year 60; one with
// nothing. A run that stopped in warm-up is listed and left out.
func TestWriteOutcomes(t *testing.T) {
	y := func(years float64) int { return 1 + int(years*daysPerYear) }
	end := y(30)
	a := Seed{Seed: 1, Canon: &Canon{First: 1, Last: 100 * daysPerYear, Collapse: "air", CollapseDay: y(60),
		Rebellions: []Rebellion{{Start: y(20), End: &end, Days: 10 * daysPerYear, Ending: "suppressed"}}}}
	b := Seed{Seed: 2, Canon: &Canon{First: 1, Last: 100 * daysPerYear}}
	c := Seed{Seed: 3, Crashed: "collapsed in warm-up on day 9 (air)"}
	var out strings.Builder
	if err := WriteOutcomes(&out, []Seed{a, b, c}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"year 20.0", "year 60.0, air", "(left out: collapsed in warm-up", "outcomes across 2 runs",
		"rebellions per century: 0.50", "in a rebellion: 5.0%", "suppressed 1 (100%", "collapses: 1 of 2 runs", "air 1",
		"survival to first rebellion (Kaplan–Meier, Greenwood 95%): median 20.0 years", "survival to collapse (Kaplan–Meier, Greenwood 95%): median 60.0 years"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("outcomes lack %q:\n%s", want, out.String())
		}
	}
}
