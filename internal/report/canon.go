package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ODD §1.2 canon patterns C1–C4, checked across a batch, not per seed.
// Like the §8 targets they are the spec's numbers, so they live here.
const (
	c1Min, c1Max  = 0.3, 2.0 // rebellions per century, pooled over the batch
	c2Years       = 5        // window after a scripted cleaning failure
	c2Ratio       = 3.0      // a rebellion at least this many times as likely in it
	c3DownDeep    = 0.60     // rebellions starting with most actives Down Deep, at least
	c3Mechanical  = 0.50     // of those, with a Mechanical strike, at least
	c4Years       = 40       // years after a suppressed rebellion's end
	c4KnowHistory = 0.10     // share of adults knowing the true history, below this
)

// Canon is one run's part-2 record, read only when the run wrote
// rebellions.jsonl (the rebellion module was on).
type Canon struct {
	First, Last int // the first and last measured day
	Rebellions  []Rebellion
	Scripted    []int // days of scripted cleaning failures (cleaningFailure)
	Failures    []int // days of every cleaning failure, scripted or sentenced
	History     [][2]float64
	Collapse    string // cause, or "" if none
	Unrest      int
}

// Rebellion is the part of a rebellions.jsonl line the patterns use.
type Rebellion struct {
	Start  int      `json:"start"`
	End    *int     `json:"end"`
	Days   int      `json:"days"`
	Zone   string   `json:"zone"`
	Struck []string `json:"struck"`
	Ending string   `json:"ending"`
}

// canon reads the run's rebellions, cleaning failures and yearly share of
// adults knowing the history. warmupDays places the measured days.
func (s *Seed) canon(dir string, warmupDays, warmup int, collapse *string, unrest int) error {
	data, err := os.ReadFile(filepath.Join(dir, "rebellions.jsonl"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	c := &Canon{First: warmupDays + 1, Last: warmupDays + s.Days, Unrest: unrest}
	if collapse != nil {
		c.Collapse = *collapse
	}
	for line := range bytes.Lines(data) {
		var r Rebellion
		if err := json.Unmarshal(line, &r); err != nil {
			return fmt.Errorf("rebellions.jsonl: %w", err)
		}
		c.Rebellions = append(c.Rebellions, r)
	}
	if data, err = os.ReadFile(filepath.Join(dir, "events.jsonl")); err != nil {
		return err
	}
	for line := range bytes.Lines(data) {
		var e struct {
			Day    int    `json:"day"`
			Type   string `json:"type"`
			Event  string `json:"event"`
			Failed bool   `json:"failed"`
		}
		if err := json.Unmarshal(line, &e); err != nil {
			return fmt.Errorf("events.jsonl: %w", err)
		}
		switch {
		case e.Type == "scripted" && e.Event == "cleaningFailure":
			c.Scripted = append(c.Scripted, e.Day)
			c.Failures = append(c.Failures, e.Day)
		case e.Type == "cleaning" && e.Failed:
			c.Failures = append(c.Failures, e.Day)
		}
	}
	col, rows, err := table(filepath.Join(dir, "yearly.csv"), warmup, "knowHistory")
	if err != nil {
		return err
	}
	for _, r := range rows {
		c.History = append(c.History, [2]float64{cell(col, r, "day"), cell(col, r, "knowHistory")})
	}
	s.Canon = c
	return nil
}

// started reports whether a rebellion started in days [from, to].
func (c *Canon) started(from, to int) bool {
	return slices.ContainsFunc(c.Rebellions, func(r Rebellion) bool { return r.Start >= from && r.Start <= to })
}

// windows counts C2's windows in one run (ODD §1.2, Adam 2026-11-01).
// Treated: the c2Years from each scripted cleaning failure on a measured
// day. Comparison: the measured period cut into c2Years blocks from its
// first day, keeping those that overlap no window after any cleaning
// failure, scripted or sentenced. A window cut short by the end of the run
// counts only if a rebellion started in it, so a rebellion that ends the
// silo early is not lost.
func (c *Canon) windows() (treated, treatedHit, control, controlHit int) {
	span := c2Years * daysPerYear
	count := func(from int, n, hit *int) {
		h := c.started(from, from+span-1)
		if from+span-1 <= c.Last || h {
			*n++
			if h {
				*hit++
			}
		}
	}
	for _, d := range c.Scripted {
		if d >= c.First && d <= c.Last {
			count(d, &treated, &treatedHit)
		}
	}
	for from := c.First; from <= c.Last; from += span {
		if !slices.ContainsFunc(c.Failures, func(f int) bool { return f <= from+span-1 && f+span-1 >= from }) {
			count(from, &control, &controlHit)
		}
	}
	return
}

// forgotten counts C4 in one run: suppressed rebellions with a yearly
// snapshot c4Years or more after their end, and those whose first such
// snapshot has under c4KnowHistory of adults knowing the history.
func (c *Canon) forgotten() (eligible, below int) {
	for _, r := range c.Rebellions {
		if r.Ending != "suppressed" || r.End == nil {
			continue
		}
		at := *r.End + c4Years*daysPerYear
		i := slices.IndexFunc(c.History, func(h [2]float64) bool { return int(h[0]) >= at })
		if i < 0 {
			continue
		}
		eligible++
		if c.History[i][1] < c4KnowHistory {
			below++
		}
	}
	return
}

// writeCanon prints C1–C4 over the runs with a part-2 record, and the
// outcomes behind Q1–Q3 as information, the share of measured days spent
// in a rebellion among them (ODD §1.2, §6.15). It reports whether C1–C4
// hold; C4 is judged since T13.2, when the canon regime gained its memory
// wipe. With no such runs it prints nothing and reports true.
func writeCanon(out io.Writer, seeds []Seed) bool {
	var runs []*Canon
	for _, s := range seeds {
		if s.Crashed == "" && s.Canon != nil {
			runs = append(runs, s.Canon)
		}
	}
	if len(runs) == 0 {
		return true
	}
	days, n, dd, ddMech, unrest, inRebellion := 0, 0, 0, 0, 0, 0
	var tT, tH, cT, cH, eligible, below int
	endings, causes := map[string]int{}, map[string]int{} // printed in a fixed order, never ranged over
	for _, c := range runs {
		days += c.Last - c.First + 1
		unrest += c.Unrest
		causes[c.Collapse]++
		for _, r := range c.Rebellions {
			n++
			inRebellion += r.Days
			endings[r.Ending]++
			if r.Zone == "downDeep" {
				dd++
				if slices.Contains(r.Struck, "mechanical") {
					ddMech++
				}
			}
		}
		a, b, x, y := c.windows()
		tT, tH, cT, cH = tT+a, tH+b, cT+x, cH+y
		e, l := c.forgotten()
		eligible, below = eligible+e, below+l
	}
	share := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return float64(a) / float64(b)
	}
	rate := float64(n) / (float64(days) / (100 * daysPerYear))
	pT, pC := share(tH, tT), share(cH, cT)
	c2 := tT > 0 && cT > 0 && tH > 0 && (cH == 0 || pT >= c2Ratio*pC)
	ratio := "no rebellion in either"
	switch {
	case tT == 0:
		ratio = "no scripted failure"
	case cT == 0:
		ratio = "no window without one"
	case cH > 0:
		ratio = fmt.Sprintf("%.1f×", pT/pC)
	case tH > 0:
		ratio = "none without"
	}
	checks := []struct {
		name, rule, value string
		ok                bool
	}{
		{"C1 Long peace", fmt.Sprintf("%.1f–%.0f rebellions per century, pooled", c1Min, c1Max),
			fmt.Sprintf("%.2f (%d in %d years)", rate, n, days/daysPerYear), rate >= c1Min && rate <= c1Max},
		{"C2 Visible failures trigger revolts", fmt.Sprintf("≥ %.0f× as likely within %d years of a scripted failure", c2Ratio, c2Years),
			fmt.Sprintf("%d/%d after vs %d/%d without (%s)", tH, tT, cH, cT, ratio), c2},
		{"C3 Revolts start Down Deep", fmt.Sprintf("≥ %.0f%% Down Deep, ≥ %.0f%% of those with a Mechanical strike", 100*c3DownDeep, 100*c3Mechanical),
			fmt.Sprintf("%d/%d (%.0f%%), %d/%d (%.0f%%)", dd, n, 100*share(dd, n), ddMech, dd, 100*share(ddMech, dd)),
			n > 0 && dd > 0 && share(dd, n) >= c3DownDeep && share(ddMech, dd) >= c3Mechanical},
		{"C4 Crushed rebellions are forgotten", fmt.Sprintf("< %.0f%% know the history %d years on, in ≥ %.0f%%", 100*c4KnowHistory, c4Years, 100*seedsNeeded),
			fmt.Sprintf("%d/%d (%.0f%%)", below, eligible, 100*share(below, eligible)),
			eligible > 0 && share(below, eligible) >= seedsNeeded},
	}
	fmt.Fprintf(out, "\ncanon pattern (ODD §1.2), %d runs\ttarget\tmeasured\t\n", len(runs))
	all := true
	for _, c := range checks {
		all = all && c.ok
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", c.name, c.rule, c.value, mark(c.ok))
	}
	var e, k []string
	for _, end := range []string{"suppressed", "regimeChange", "split", "collapse", "unfinished"} {
		e = append(e, fmt.Sprintf("%s %d", end, endings[end]))
	}
	for _, cause := range []string{"air", "water", "population"} {
		k = append(k, fmt.Sprintf("%s %d", cause, causes[cause]))
	}
	fmt.Fprintf(out, "\n(rebellions %d: %s; unrest spells %d; collapses %d: %s; %.1f%% of measured days in a rebellion)\n",
		n, strings.Join(e, ", "), unrest, len(runs)-causes[""], strings.Join(k, ", "), 100*share(inRebellion, days))
	return all
}
