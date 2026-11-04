package report

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
	"text/tabwriter"
)

// z95 is the normal quantile of a two-sided 95% interval.
const z95 = 1.959964

// curveStepYears: the survival curves are printed every this many years.
const curveStepYears = 25

// WriteOutcomes prints the part-2 outcome metrics of ODD §1.2 (T15.1): a
// line per seed, then across seeds with 95% intervals: rebellions per
// century, the share of each ending, the share of measured time in a
// rebellion, causes of collapse, and Kaplan–Meier curves for the time to
// the first rebellion and to collapse. Time counts in measured years from
// the first measured day; a run without the event is censored at its last
// measured day. Runs without a part-2 record, or that stopped in warm-up,
// are listed and left out.
func WriteOutcomes(out io.Writer, seeds []Seed) error {
	var runs []*Canon
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "seed\tyears\tfirst rebellion\trebellions\tper century\tin a rebellion\tcollapse\t")
	for _, s := range seeds {
		c := s.Canon
		if s.Crashed != "" || c == nil {
			why := s.Crashed
			if why == "" {
				why = "no rebellion module"
			}
			fmt.Fprintf(tw, "%d\t—\t(left out: %s)\t\t\t\t\t\n", s.Seed, why)
			continue
		}
		runs = append(runs, c)
		first, collapse := "—", "—"
		if t, ok := c.firstRebellion(); ok {
			first = fmt.Sprintf("year %.1f", t)
		}
		if c.Collapse != "" {
			collapse = fmt.Sprintf("year %.1f, %s", c.years(c.CollapseDay), c.Collapse)
		}
		fmt.Fprintf(tw, "%d\t%.1f\t%s\t%d\t%.2f\t%.1f%%\t%s\t\n", s.Seed, c.span(), first, len(c.Rebellions),
			float64(len(c.Rebellions))/c.span()*100, 100*c.inRebellion(), collapse)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	if len(runs) == 0 {
		return fmt.Errorf("no run has a part-2 record (rebellions.jsonl)")
	}

	n, years := 0, 0.0
	var shares []float64
	endings := map[string]int{} // printed in a fixed order, never ranged over
	causes := map[string]int{}
	var toRebellion, toCollapse []obs
	for _, c := range runs {
		n += len(c.Rebellions)
		years += c.span()
		shares = append(shares, c.inRebellion())
		for _, r := range c.Rebellions {
			endings[r.Ending]++
		}
		causes[c.Collapse]++
		t, ok := c.firstRebellion()
		if !ok {
			t = c.span()
		}
		toRebellion = append(toRebellion, obs{t, ok})
		if c.Collapse != "" {
			toCollapse = append(toCollapse, obs{c.years(c.CollapseDay), true})
		} else {
			toCollapse = append(toCollapse, obs{c.span(), false})
		}
	}

	fmt.Fprintf(out, "\noutcomes across %d runs (ODD §1.2), 95%% intervals\n", len(runs))
	lo, hi := poisson(n)
	fmt.Fprintf(out, "rebellions per century: %.2f (%.2f–%.2f), %d in %.0f measured years\n", float64(n)/years*100, lo/years*100, hi/years*100, n, years)
	m, h := meanCI(shares)
	fmt.Fprintf(out, "share of measured time in a rebellion: %.1f%% (%.1f–%.1f%%), mean over runs\n", 100*m, 100*max(0, m-h), 100*min(1, m+h))
	var e []string
	for _, end := range []string{"suppressed", "regimeChange", "split", "collapse", "unfinished"} {
		l, u := wilson(endings[end], n)
		e = append(e, fmt.Sprintf("%s %d (%.0f%%, %.0f–%.0f%%)", end, endings[end], 100*share(endings[end], n), 100*l, 100*u))
	}
	fmt.Fprintf(out, "endings of %d rebellions: %s\n", n, strings.Join(e, "; "))
	var k []string
	for _, cause := range []string{"air", "water", "population", "safeguard"} {
		k = append(k, fmt.Sprintf("%s %d", cause, causes[cause]))
	}
	l, u := wilson(len(runs)-causes[""], len(runs))
	fmt.Fprintf(out, "collapses: %d of %d runs (%.0f–%.0f%%): %s\n", len(runs)-causes[""], len(runs), 100*l, 100*u, strings.Join(k, ", "))

	last := 0.0
	for _, c := range runs {
		last = max(last, c.span())
	}
	for _, curve := range []struct {
		name string
		obs  []obs
	}{{"first rebellion", toRebellion}, {"collapse", toCollapse}} {
		steps := kaplanMeier(curve.obs)
		fmt.Fprintf(out, "\nsurvival to %s (Kaplan–Meier, Greenwood 95%%): median %s\n", curve.name, median50(steps))
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "year\tat risk\tS(t)\t95%\t")
		for t := 0.0; t <= last; t += curveStepYears {
			st := at(steps, t)
			fmt.Fprintf(tw, "%.0f\t%d\t%.3f\t%.3f–%.3f\t\n", t, atRisk(curve.obs, t), st.s, max(0, st.s-z95*st.se), min(1, st.s+z95*st.se))
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// span is the run's measured length in years.
func (c *Canon) span() float64 { return float64(c.Last-c.First+1) / daysPerYear }

// years is a day's time since the first measured day, in years.
func (c *Canon) years(day int) float64 { return float64(day-c.First+1) / daysPerYear }

// firstRebellion is the time of the earliest rebellion start, if any.
func (c *Canon) firstRebellion() (float64, bool) {
	if len(c.Rebellions) == 0 {
		return 0, false
	}
	first := slices.MinFunc(c.Rebellions, func(a, b Rebellion) int { return a.Start - b.Start })
	return c.years(first.Start), true
}

// inRebellion is the share of measured days in a rebellion.
func (c *Canon) inRebellion() float64 {
	days := 0
	for _, r := range c.Rebellions {
		days += r.Days
	}
	return float64(days) / float64(c.Last-c.First+1)
}

// obs is one run's time to an event, or to censoring.
type obs struct {
	t     float64
	event bool
}

// step is the survival curve from time t on, with its Greenwood standard
// error.
type step struct{ t, s, se float64 }

// kaplanMeier estimates the survival curve: at each time with events, S
// is multiplied by 1 − d/n, n the runs still at risk (neither the event
// nor censored before it); runs censored at that time count as at risk.
func kaplanMeier(o []obs) []step {
	o = slices.Clone(o)
	slices.SortFunc(o, func(a, b obs) int {
		switch {
		case a.t < b.t:
			return -1
		case a.t > b.t:
			return 1
		}
		return 0
	})
	steps := []step{{0, 1, 0}}
	s, green := 1.0, 0.0
	for i := 0; i < len(o); {
		j, d := i, 0
		for ; j < len(o) && o[j].t == o[i].t; j++ {
			if o[j].event {
				d++
			}
		}
		if n := len(o) - i; d > 0 {
			s *= 1 - float64(d)/float64(n)
			if n > d {
				green += float64(d) / float64(n*(n-d))
			}
			steps = append(steps, step{o[i].t, s, s * math.Sqrt(green)})
		}
		i = j
	}
	return steps
}

// at is the curve's value at time t.
func at(steps []step, t float64) step {
	i := 0
	for i+1 < len(steps) && steps[i+1].t <= t {
		i++
	}
	return steps[i]
}

func atRisk(o []obs, t float64) int {
	n := 0
	for _, x := range o {
		if x.t >= t {
			n++
		}
	}
	return n
}

// median50 is the first time the curve is at or below one half.
func median50(steps []step) string {
	for _, st := range steps {
		if st.s <= 0.5 {
			return fmt.Sprintf("%.1f years", st.t)
		}
	}
	return "not reached"
}

// wilson is the Wilson score 95% interval of k successes in n.
func wilson(k, n int) (float64, float64) {
	if n == 0 {
		return 0, 1
	}
	p, nn, z2 := float64(k)/float64(n), float64(n), z95*z95
	mid := (p + z2/(2*nn)) / (1 + z2/nn)
	half := z95 * math.Sqrt(p*(1-p)/nn+z2/(4*nn*nn)) / (1 + z2/nn)
	return max(0, mid-half), min(1, mid+half)
}

// poisson is an approximate 95% interval of a Poisson count n: normal on
// the square root (Bartlett), which holds down to small counts; for 0 the
// upper end is 3.69, the exact one-sided bound.
func poisson(n int) (float64, float64) {
	if n == 0 {
		return 0, 3.69
	}
	r := math.Sqrt(float64(n))
	return math.Max(0, r-z95/2) * math.Max(0, r-z95/2), (r + z95/2) * (r + z95/2)
}

// meanCI is the mean and the half-width of its normal 95% interval.
func meanCI(x []float64) (float64, float64) {
	m := 0.0
	for _, v := range x {
		m += v
	}
	m /= float64(len(x))
	if len(x) < 2 {
		return m, 0
	}
	ss := 0.0
	for _, v := range x {
		ss += (v - m) * (v - m)
	}
	return m, z95 * math.Sqrt(ss/float64(len(x)-1)/float64(len(x)))
}

func share(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}
