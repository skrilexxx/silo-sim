// Package report checks a batch of runs against the ODD §8 calibration
// targets: one row per seed, then the share of seeds passing each target.
// It reads only the output files, never the model.
package report

import (
	"bytes"
	"cmp"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"text/tabwriter"
)

// ODD §8 calibration targets. They are the spec's numbers, so they live
// here with it rather than in the scenario: changing one is a spec change.
const (
	popMin, popMax     = 9000, 11000 // stable population, every measured day
	ageMin, ageMax     = 30.0, 40.0  // median age, every yearly snapshot
	birthsBlockYears   = 5           // births are summed over 5-year blocks
	birthsShockMax     = 0.30        // largest change between consecutive blocks, below this
	rationingMax       = 0.05        // share of days with rationing, below this
	failuresMin        = 1.0         // generator failures per century, on average over the run
	failuresMax        = 6.0
	understaffedMax    = 365  // days a department may stay below 50% of target
	mechanicalInZoneLo = 0.90 // Mechanical workers living Down Deep, above this every year
	seedsNeeded        = 0.90 // a pattern holds if at least this share of seeds pass
	daysPerYear        = 365
)

// Seed holds one run's measured values (warm-up excluded).
type Seed struct {
	Dir     string
	Seed    uint64
	Crashed string // the run's error if it stopped early; it then fails every target
	Days    int    // measured days

	PopMin, PopMax      int
	AgeMin, AgeMax      float64
	BirthsShock         float64
	RationingShare      float64
	FailuresPerCentury  float64
	LongestUnderstaffed int    // days
	UnderstaffedDept    string // the department with that spell
	MechanicalInZoneMin float64
}

// Load reads every run folder in batch (one with a run.json), sorted by
// seed.
func Load(batch string) ([]Seed, error) {
	entries, err := os.ReadDir(batch)
	if err != nil {
		return nil, err
	}
	var seeds []Seed
	for _, e := range entries {
		dir := filepath.Join(batch, e.Name())
		if _, err := os.Stat(filepath.Join(dir, "run.json")); !e.IsDir() || err != nil {
			continue
		}
		s, err := LoadRun(dir)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", dir, err)
		}
		seeds = append(seeds, s)
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("%s holds no runs", batch)
	}
	slices.SortFunc(seeds, func(a, b Seed) int { return cmp.Compare(a.Seed, b.Seed) })
	return seeds, nil
}

// LoadRun measures one run folder.
func LoadRun(dir string) (Seed, error) {
	s := Seed{Dir: filepath.Base(dir)}
	var info struct {
		Seed     uint64 `json:"seed"`
		Finished bool   `json:"finished"`
		Error    string `json:"error"`
		Scenario struct {
			WarmupYears int `json:"warmupYears"`
			Output      struct {
				LogWarmup bool `json:"logWarmup"`
			} `json:"output"`
		} `json:"scenario"`
	}
	data, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return s, err
	}
	s.Seed = info.Seed
	if !info.Finished {
		s.Crashed = info.Error
		if s.Crashed == "" {
			s.Crashed = "did not finish"
		}
		return s, nil
	}
	warmup := 0 // rows on or before this day are warm-up (written only with logWarmup)
	if info.Scenario.Output.LogWarmup {
		warmup = info.Scenario.WarmupYears * daysPerYear
	}

	if err := s.daily(filepath.Join(dir, "daily.csv"), warmup); err != nil {
		return s, err
	}
	if err := s.yearly(filepath.Join(dir, "yearly.csv"), warmup); err != nil {
		return s, err
	}
	return s, s.events(filepath.Join(dir, "events.jsonl"), info.Scenario.WarmupYears*daysPerYear)
}

// table reads a CSV file into a column-name lookup and its rows after
// the warm-up. Files without one of the needed columns (from older code)
// are an error.
func table(path string, warmup int, needed ...string) (map[string]int, [][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil || len(rows) < 2 {
		return nil, nil, fmt.Errorf("%s: no rows (%v)", filepath.Base(path), err)
	}
	col := map[string]int{}
	for i, name := range rows[0] {
		col[name] = i
	}
	for _, name := range needed {
		if _, ok := col[name]; !ok {
			return nil, nil, fmt.Errorf("%s has no column %q; rerun the batch with current code", filepath.Base(path), name)
		}
	}
	var kept [][]string
	for _, r := range rows[1:] {
		if day, _ := strconv.Atoi(r[0]); day > warmup {
			kept = append(kept, r)
		}
	}
	return col, kept, nil
}

// cell parses a numeric cell of a column table checked for.
func cell(col map[string]int, row []string, name string) float64 {
	x, _ := strconv.ParseFloat(row[col[name]], 64)
	return x
}

func (s *Seed) daily(path string, warmup int) error {
	col, rows, err := table(path, warmup, "population", "rationing", "births")
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("%s: no measured days", filepath.Base(path))
	}
	s.Days = len(rows)
	s.PopMin, s.PopMax = math.MaxInt, 0
	rationing := 0
	var blocks []int
	for i, r := range rows {
		pop := int(cell(col, r, "population"))
		s.PopMin, s.PopMax = min(s.PopMin, pop), max(s.PopMax, pop)
		if cell(col, r, "rationing") == 1 {
			rationing++
		}
		if i%(birthsBlockYears*daysPerYear) == 0 {
			blocks = append(blocks, 0)
		}
		blocks[len(blocks)-1] += int(cell(col, r, "births"))
	}
	s.RationingShare = float64(rationing) / float64(s.Days)
	if s.Days%(birthsBlockYears*daysPerYear) != 0 {
		blocks = blocks[:len(blocks)-1] // an unfinished block is not compared
	}
	for i := 1; i < len(blocks); i++ {
		if blocks[i-1] == 0 {
			continue // after a drop to no births at all, already counted as 100%
		}
		s.BirthsShock = max(s.BirthsShock, math.Abs(float64(blocks[i]-blocks[i-1]))/float64(blocks[i-1]))
	}
	return nil
}

func (s *Seed) yearly(path string, warmup int) error {
	col, rows, err := table(path, warmup, "medianAge", "mechanicalInZone")
	if err != nil {
		return err
	}
	s.AgeMin, s.AgeMax, s.MechanicalInZoneMin = math.Inf(1), 0, 1
	for _, r := range rows {
		age := cell(col, r, "medianAge")
		s.AgeMin, s.AgeMax = min(s.AgeMin, age), max(s.AgeMax, age)
		s.MechanicalInZoneMin = min(s.MechanicalInZoneMin, cell(col, r, "mechanicalInZone"))
	}
	return nil
}

// events counts generator failures and finds the longest spell any
// department spent below the understaffed mark. A spell still open on
// the last day runs to the end of the run. warmupDays is used only to
// place the last day.
func (s *Seed) events(path string, warmupDays int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	end := warmupDays + s.Days + 1 // the day after the last measured day
	since := map[string]int{}      // department → day its spell began
	var depts []string             // in order of first spell, for stable ties
	failures := 0
	for line := range bytes.Lines(data) {
		var e struct {
			Day        int    `json:"day"`
			Type       string `json:"type"`
			Department string `json:"department"`
		}
		if err := json.Unmarshal(line, &e); err != nil {
			return fmt.Errorf("events.jsonl: %w", err)
		}
		if e.Day <= warmupDays && warmupDays > 0 {
			continue // logged only with logWarmup
		}
		switch e.Type {
		case "failure":
			failures++
		case "understaffed":
			since[e.Department] = e.Day
			if !slices.Contains(depts, e.Department) {
				depts = append(depts, e.Department)
			}
		case "restaffed":
			if start, ok := since[e.Department]; ok {
				s.spell(e.Department, e.Day-start)
				delete(since, e.Department)
			}
		}
	}
	for _, d := range depts {
		if start, ok := since[d]; ok {
			s.spell(d, end-start)
		}
	}
	s.FailuresPerCentury = float64(failures) / (float64(s.Days) / (100 * daysPerYear))
	return nil
}

func (s *Seed) spell(dept string, days int) {
	if days > s.LongestUnderstaffed {
		s.LongestUnderstaffed, s.UnderstaffedDept = days, dept
	}
}

// Target is one ODD §8 calibration target.
type Target struct {
	Pattern, Rule string
	Pass          func(Seed) bool
	Value         func(Seed) string
}

var Targets = []Target{
	{"Stable population", "9,000–11,000 every day",
		func(s Seed) bool { return s.PopMin >= popMin && s.PopMax <= popMax },
		func(s Seed) string { return fmt.Sprintf("%d–%d", s.PopMin, s.PopMax) }},
	{"Plausible ages", "median age 30–40 every year",
		func(s Seed) bool { return s.AgeMin >= ageMin && s.AgeMax <= ageMax },
		func(s Seed) string { return fmt.Sprintf("%.1f–%.1f", s.AgeMin, s.AgeMax) }},
	{"No demographic shocks", "5-year births change < 30%",
		func(s Seed) bool { return s.BirthsShock < birthsShockMax },
		func(s Seed) string { return fmt.Sprintf("%.1f%%", 100*s.BirthsShock) }},
	{"Balanced resources", "rationing < 5% of days",
		func(s Seed) bool { return s.RationingShare < rationingMax },
		func(s Seed) string { return fmt.Sprintf("%.1f%%", 100*s.RationingShare) }},
	{"Rare crises", "1–6 failures per century",
		func(s Seed) bool { return s.FailuresPerCentury >= failuresMin && s.FailuresPerCentury <= failuresMax },
		func(s Seed) string { return fmt.Sprintf("%.1f", s.FailuresPerCentury) }},
	{"Staffed departments", "never < 50% of target > 1 year",
		func(s Seed) bool { return s.LongestUnderstaffed <= understaffedMax },
		func(s Seed) string {
			if s.LongestUnderstaffed == 0 {
				return "0 d"
			}
			return fmt.Sprintf("%d d (%s)", s.LongestUnderstaffed, s.UnderstaffedDept)
		}},
	{"Persistent zones", "Mechanical Down Deep > 90% every year",
		func(s Seed) bool { return s.MechanicalInZoneMin > mechanicalInZoneLo },
		func(s Seed) string { return fmt.Sprintf("%.1f%%", 100*s.MechanicalInZoneMin) }},
}

// Write prints the per-seed table and the summary, and reports whether
// every pattern holds in at least seedsNeeded of the seeds.
func Write(out io.Writer, seeds []Seed) (bool, error) {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprint(tw, "seed\tyears")
	for _, t := range Targets {
		fmt.Fprintf(tw, "\t%s", t.Pattern)
	}
	fmt.Fprintln(tw)
	for _, s := range seeds {
		if s.Crashed != "" {
			fmt.Fprintf(tw, "%d\t-", s.Seed)
			for range Targets {
				fmt.Fprint(tw, "\tcrashed")
			}
			fmt.Fprintln(tw)
			continue
		}
		fmt.Fprintf(tw, "%d\t%d", s.Seed, s.Days/daysPerYear)
		for _, t := range Targets {
			fmt.Fprintf(tw, "\t%s %s", t.Value(s), mark(t.Pass(s)))
		}
		fmt.Fprintln(tw)
	}
	fmt.Fprintln(tw)
	for _, s := range seeds {
		if s.Crashed != "" {
			fmt.Fprintf(tw, "seed %d crashed: %s\n", s.Seed, s.Crashed)
		}
	}
	fmt.Fprintln(tw)

	all := true
	fmt.Fprintf(tw, "pattern\ttarget\tseeds passing\t(need %.0f%%)\n", 100*seedsNeeded)
	for _, t := range Targets {
		n := 0
		for _, s := range seeds {
			if s.Crashed == "" && t.Pass(s) {
				n++
			}
		}
		ok := float64(n) >= seedsNeeded*float64(len(seeds))
		all = all && ok
		fmt.Fprintf(tw, "%s\t%s\t%d/%d (%.0f%%)\t%s\n", t.Pattern, t.Rule, n, len(seeds), 100*float64(n)/float64(len(seeds)), mark(ok))
	}
	return all, tw.Flush()
}

func mark(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
