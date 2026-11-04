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
	o, err := outcomes(runs)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "\noutcomes across %d runs (ODD §1.2), 95%% intervals\n", o.Runs)
	r := o.PerCentury
	fmt.Fprintf(out, "rebellions per century: %.2f (%.2f–%.2f), %d in %.0f measured years\n", r.Value, r.Lo, r.Hi, o.Rebellions, o.Years)
	r = o.InRebellion
	fmt.Fprintf(out, "share of measured time in a rebellion: %.1f%% (%.1f–%.1f%%), mean over runs\n", 100*r.Value, 100*r.Lo, 100*r.Hi)
	var e []string
	for _, end := range o.Endings {
		e = append(e, fmt.Sprintf("%s %d (%.0f%%, %.0f–%.0f%%)", end.Name, end.Count, 100*end.Share, 100*end.Lo, 100*end.Hi))
	}
	fmt.Fprintf(out, "endings of %d rebellions: %s\n", o.Rebellions, strings.Join(e, "; "))
	var k []string
	for _, c := range o.Causes {
		k = append(k, fmt.Sprintf("%s %d", c.Name, c.Count))
	}
	c := o.Collapses
	fmt.Fprintf(out, "collapses: %d of %d runs (%.0f–%.0f%%): %s\n", c.Count, o.Runs, 100*c.Lo, 100*c.Hi, strings.Join(k, ", "))

	for _, curve := range o.Curves {
		median := "not reached"
		if curve.Median != nil {
			median = fmt.Sprintf("%.1f years", *curve.Median)
		}
		fmt.Fprintf(out, "\nsurvival to %s (Kaplan–Meier, Greenwood 95%%): median %s\n", curve.Name, median)
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "year\tat risk\tS(t)\t95%\t")
		for t := 0.0; t <= o.Last; t += curveStepYears {
			st := at(curve.steps, t)
			fmt.Fprintf(tw, "%.0f\t%d\t%.3f\t%.3f–%.3f\t\n", t, atRisk(curve.obs, t), st.s, max(0, st.s-z95*st.se), min(1, st.s+z95*st.se))
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// Outcomes are the across-seed numbers of WriteOutcomes, which prints
// them; the viewer gets them as JSON (docs/viewer.md, T16.4).
type Outcomes struct {
	Runs        int       `json:"runs"`       // runs counted, left-out ones not
	LeftOut     []LeftOut `json:"leftOut"`    // runs without a part-2 record, or stopped in warm-up
	Rebellions  int       `json:"rebellions"` // in all counted runs
	Years       float64   `json:"years"`      // measured years of all counted runs
	Last        float64   `json:"last"`       // the longest run's measured years
	PerCentury  Interval  `json:"perCentury"`
	InRebellion Interval  `json:"inRebellion"` // share of measured time, mean over runs
	Endings     []Ending  `json:"endings"`     // share of rebellions
	Collapses   Ending    `json:"collapses"`   // share of runs
	Causes      []Ending  `json:"causes"`      // collapses by cause; counts only
	Curves      []Curve   `json:"curves"`      // to the first rebellion, to collapse
}

// LeftOut is a run not counted, and why.
type LeftOut struct {
	Seed uint64 `json:"seed"`
	Why  string `json:"why"`
}

// Interval is a value with its 95% interval.
type Interval struct {
	Value float64 `json:"value"`
	Lo    float64 `json:"lo"`
	Hi    float64 `json:"hi"`
}

// Ending is a count, its share and the share's Wilson 95% interval.
type Ending struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Share float64 `json:"share"`
	Lo    float64 `json:"lo"`
	Hi    float64 `json:"hi"`
}

// Curve is a Kaplan–Meier curve: S from each step's time on, with the
// Greenwood 95% band clipped to 0–1.
type Curve struct {
	Name   string   `json:"name"`
	Median *float64 `json:"median"` // first time S ≤ ½, nil if not reached
	Steps  []KMStep `json:"steps"`
	steps  []step
	obs    []obs
}

// KMStep is one step of a Curve.
type KMStep struct {
	T  float64 `json:"t"`
	S  float64 `json:"s"`
	Lo float64 `json:"lo"`
	Hi float64 `json:"hi"`
}

// OutcomesOf is the across-seed outcomes of a batch, the runs
// WriteOutcomes leaves out listed in LeftOut.
func OutcomesOf(seeds []Seed) (Outcomes, error) {
	var runs []*Canon
	var left []LeftOut
	for _, s := range seeds {
		switch {
		case s.Crashed != "":
			left = append(left, LeftOut{s.Seed, s.Crashed})
		case s.Canon == nil:
			left = append(left, LeftOut{s.Seed, "no rebellion module"})
		default:
			runs = append(runs, s.Canon)
		}
	}
	o, err := outcomes(runs)
	o.LeftOut = left
	return o, err
}

// outcomes computes the across-seed numbers of the counted runs.
func outcomes(runs []*Canon) (Outcomes, error) {
	if len(runs) == 0 {
		return Outcomes{}, fmt.Errorf("no run has a part-2 record (rebellions.jsonl)")
	}
	o := Outcomes{Runs: len(runs)}
	var shares []float64
	endings := map[string]int{} // read in a fixed order, never ranged over
	causes := map[string]int{}
	var toRebellion, toCollapse []obs
	for _, c := range runs {
		o.Rebellions += len(c.Rebellions)
		o.Years += c.span()
		o.Last = max(o.Last, c.span())
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

	n := o.Rebellions
	lo, hi := poisson(n)
	o.PerCentury = Interval{float64(n) / o.Years * 100, lo / o.Years * 100, hi / o.Years * 100}
	m, h := meanCI(shares)
	o.InRebellion = Interval{m, max(0, m-h), min(1, m+h)}
	for _, end := range []string{"suppressed", "regimeChange", "split", "collapse", "unfinished"} {
		l, u := wilson(endings[end], n)
		o.Endings = append(o.Endings, Ending{end, endings[end], share(endings[end], n), l, u})
	}
	for _, cause := range []string{"air", "water", "population", "safeguard"} {
		o.Causes = append(o.Causes, Ending{Name: cause, Count: causes[cause]})
	}
	k := len(runs) - causes[""]
	l, u := wilson(k, len(runs))
	o.Collapses = Ending{"collapse", k, share(k, len(runs)), l, u}

	for _, curve := range []struct {
		name string
		obs  []obs
	}{{"first rebellion", toRebellion}, {"collapse", toCollapse}} {
		c := Curve{Name: curve.name, steps: kaplanMeier(curve.obs), obs: curve.obs}
		c.Median = median(c.steps)
		for _, st := range c.steps {
			c.Steps = append(c.Steps, KMStep{st.t, st.s, max(0, st.s-z95*st.se), min(1, st.s+z95*st.se)})
		}
		o.Curves = append(o.Curves, c)
	}
	return o, nil
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

// median is the first time the curve is at or below one half, nil if
// never.
func median(steps []step) *float64 {
	for _, st := range steps {
		if st.s <= 0.5 {
			return &st.t
		}
	}
	return nil
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
