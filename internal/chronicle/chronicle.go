// Package chronicle writes the dated facts of one run as Markdown
// (docs/narrator.md, T17.1): a header, an entry per event worth reading,
// and a line per decade. It reads only the run's files, computes no
// statistic and draws no random number, so a run always gives the same
// chronicle, byte for byte.
package chronicle

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	daysPerYear  = 365
	decadeYears  = 10
	daysInDecade = decadeYears * daysPerYear
)

// run is the part of run.json the chronicle reads.
type run struct {
	Seed     uint64 `json:"seed"`
	Outcomes *struct {
		Rebellions    int     `json:"rebellions"`
		UnrestSpells  int     `json:"unrestSpells"`
		CollapseDay   *int    `json:"collapseDay"`
		CollapseCause *string `json:"collapseCause"`
	} `json:"outcomes"`
	Scenario struct {
		Name        string           `json:"name"`
		WarmupYears int              `json:"warmupYears"`
		Years       int              `json:"years"`
		Modules     []string         `json:"modules"`
		Events      []map[string]any `json:"events"`
		Regime      struct {
			Playbook []map[string]any `json:"playbook"`
		} `json:"regime"`
		Silo1 struct {
			Orders map[string]any `json:"orders"`
		} `json:"silo1"`
	} `json:"scenario"`
}

// event holds the fields of every events.jsonl line the chronicle uses.
type event struct {
	Day        int    `json:"day"`
	Type       string `json:"type"`
	Event      string `json:"event"`
	OutageDays int    `json:"outageDays"`
	Start      int    `json:"start"`
	End        int    `json:"end"`
	Trigger    string `json:"trigger"`
	Entry      *int   `json:"entry"`
	Successor  bool   `json:"successor"`
	Zone       string `json:"zone"`
	Reason     string `json:"reason"`
	Mode       string `json:"mode"`
	Rule       string `json:"rule"`
	Failed     bool   `json:"failed"`
	Cause      string `json:"cause"`
	GoesOff    int    `json:"goesOff"`
}

// rebellion is one rebellions.jsonl line (ODD §6.14, Outputs).
type rebellion struct {
	Start             int      `json:"start"`
	Days              int      `json:"days"`
	Trigger           string   `json:"trigger"`
	TriggerDay        int      `json:"triggerDay"`
	Zone              string   `json:"zone"`
	PeakActive        float64  `json:"peakActive"`
	Struck            []string `json:"struck"`
	FightDays         int      `json:"fightDays"`
	Held              []string `json:"held"`
	RebelsKilled      int      `json:"rebelsKilled"`
	OfficersKilled    int      `json:"officersKilled"`
	FightArrests      int      `json:"fightArrests"`
	Arrests           int      `json:"arrests"`
	SegmentsBroken    int      `json:"segmentsBroken"`
	GeneratorSabotage int      `json:"generatorSabotage"`
	PumpSabotage      int      `json:"pumpSabotage"`
	Ending            string   `json:"ending"`
}

// entry is one dated line; a span is placed at its start day.
type entry struct {
	day  int
	text string
}

// chronicle holds a run's files, read once.
type chronicle struct {
	info     run
	warmup   int // last warm-up day
	events   []event
	rebs     []rebellion
	pop      map[int]string // population by day, from daily.csv
	lastDay  int
	yearHead []string
	yearly   [][]string
}

// Write writes the chronicle of the run folder dir.
func Write(out io.Writer, dir string) error {
	c, err := read(dir)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(out)
	c.header(w)
	entries := c.entries()
	decades := c.decades()
	i := 0
	for _, d := range decades {
		fmt.Fprintf(w, "\n## Years %d–%s\n\n", d.from, d.to)
		start := i
		for ; i < len(entries) && entries[i].day <= d.last; i++ {
			fmt.Fprintf(w, "- **%s.** %s\n", c.year(entries[i].day), entries[i].text)
		}
		if d.summary != "" {
			if i > start {
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w, d.summary)
		}
	}
	for ; i < len(entries); i++ { // after the last snapshot (a run cut short)
		fmt.Fprintf(w, "- **%s.** %s\n", c.year(entries[i].day), entries[i].text)
	}
	return w.Flush()
}

func read(dir string) (*chronicle, error) {
	c := &chronicle{pop: map[int]string{}}
	data, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &c.info); err != nil {
		return nil, fmt.Errorf("run.json: %w", err)
	}
	c.warmup = c.info.Scenario.WarmupYears * daysPerYear
	if err := lines(filepath.Join(dir, "events.jsonl"), &c.events); err != nil {
		return nil, err
	}
	if err := lines(filepath.Join(dir, "rebellions.jsonl"), &c.rebs); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	head, rows, err := table(filepath.Join(dir, "daily.csv"))
	if err != nil {
		return nil, err
	}
	if p := slices.Index(head, "population"); p >= 0 {
		for _, r := range rows {
			day, _ := strconv.Atoi(r[0])
			c.pop[day] = r[p]
			c.lastDay = max(c.lastDay, day)
		}
	}
	c.yearHead, c.yearly, err = table(filepath.Join(dir, "yearly.csv"))
	return c, err
}

// lines decodes a JSON-lines file into the slice v points to.
func lines[T any](path string, v *[]T) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for line := range bytes.Lines(data) {
		var x T
		if err := json.Unmarshal(line, &x); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		*v = append(*v, x)
	}
	return nil
}

func table(path string) ([]string, [][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil || len(rows) == 0 {
		return nil, nil, fmt.Errorf("%s: no rows (%v)", filepath.Base(path), err)
	}
	return rows[0], rows[1:], nil
}

// year is a day as a measured year; days of the warm-up say so.
func (c *chronicle) year(day int) string {
	if day <= c.warmup {
		return "Warm-up"
	}
	return fmt.Sprintf("Year %.1f", float64(day-c.warmup)/daysPerYear)
}

func (c *chronicle) header(w io.Writer) {
	s := c.info.Scenario
	fmt.Fprintf(w, "# Silo 18: %s, seed %d\n\n", s.Name, c.info.Seed)
	measured := float64(c.lastDay-c.warmup) / daysPerYear
	fmt.Fprintf(w, "%.1f measured years of %d, after %d years of warm-up. Modules: %s.\n\n",
		measured, s.Years, s.WarmupYears, strings.Join(s.Modules, ", "))
	o := c.info.Outcomes
	if o == nil {
		fmt.Fprintf(w, "The silo was still standing at the end.\n")
		return
	}
	if o.CollapseCause != nil && o.CollapseDay != nil {
		fmt.Fprintf(w, "The silo collapsed in %s (%s).", strings.ToLower(c.year(*o.CollapseDay)), *o.CollapseCause)
	} else {
		fmt.Fprintf(w, "The silo was still standing at the end.")
	}
	fmt.Fprintf(w, " Rebellions: %d. Spells of unrest: %d.\n", o.Rebellions, o.UnrestSpells)
}

// entries builds every dated line, in day order; lines on the same day
// keep the order of the files (events, then rebellions).
func (c *chronicle) entries() []entry {
	var out []entry
	used := make([]bool, len(c.events)) // emergencies folded into a failure, policy ends matched
	for i, e := range c.events {
		if used[i] {
			continue
		}
		switch e.Type {
		case "scripted":
			if t := c.scripted(e); t != "" {
				out = append(out, entry{e.Day, t})
			}
		case "failure":
			t := fmt.Sprintf("The generator failed: %s without power.", days(e.OutageDays))
			if j, end := c.pair(used, i, "emergency", "emergencyEnd", e.Day+e.OutageDays); j >= 0 {
				t += " An emergency was declared" + c.until(end) + "."
			}
			out = append(out, entry{e.Day, t})
		case "emergency":
			_, end := c.pair(used, i-1, "emergency", "emergencyEnd", e.Day)
			out = append(out, entry{e.Day, "An emergency was declared, with air below the line" + c.until(end) + "."})
		case "unrest":
			out = append(out, entry{e.Start, fmt.Sprintf("A spell of unrest began, lasting until %s.", strings.ToLower(c.year(e.End)))})
		case "zoneHeld":
			out = append(out, entry{e.Day, fmt.Sprintf("Rebels took %s.", zone(e.Zone))})
		case "zoneFreed":
			out = append(out, entry{e.Day, fmt.Sprintf("Rebels lost their hold on %s.", zone(e.Zone))})
		case "policy":
			out = append(out, entry{e.Day, c.policy(used, i)})
		case "headOfIT":
			how := "chosen as the most skilled IT worker, with no shadow to succeed"
			if e.Successor {
				how = "the last Head's shadow"
			}
			out = append(out, entry{e.Day, "A new Head of IT took office: " + how + "."})
		case "mayor":
			why := "the last one's term ended"
			if e.Reason == "died" {
				why = "the last one died"
			} else if e.Reason != "term" {
				why = e.Reason
			}
			out = append(out, entry{e.Day, fmt.Sprintf("A new Mayor from %s, by %s, after %s.", zone(e.Zone), e.Mode, why)})
		case "cleaning":
			if e.Failed {
				out = append(out, entry{e.Day, fmt.Sprintf("A cleaner, sent out under the Pact rule %s, walked out of sight.", e.Rule)})
			}
		case "safeguard":
			out = append(out, entry{e.Day, fmt.Sprintf("Silo 1 set off the Safeguard; it goes off in %s.", strings.ToLower(c.year(e.GoesOff)))})
		case "collapse":
			out = append(out, entry{e.Day, fmt.Sprintf("The silo collapsed: %s.", e.Cause)})
		case "extinction":
			out = append(out, entry{e.Day, "The last person died."})
		}
	}
	for _, r := range c.rebs {
		out = append(out, entry{r.Start, c.rebellion(r)})
	}
	slices.SortStableFunc(out, func(a, b entry) int { return a.day - b.day })
	return out
}

// pair finds the first unused start event after index i on or before day
// by, marks it and its end used, and returns its index and the end's day
// (0 if it never ended).
func (c *chronicle) pair(used []bool, i int, start, end string, by int) (int, int) {
	for j := i + 1; j < len(c.events) && c.events[j].Day <= by; j++ {
		if used[j] || c.events[j].Type != start {
			continue
		}
		used[j] = true
		for k := j + 1; k < len(c.events); k++ {
			if !used[k] && c.events[k].Type == end {
				used[k] = true
				return j, c.events[k].Day
			}
		}
		return j, 0
	}
	return -1, 0
}

func (c *chronicle) until(end int) string {
	if end == 0 {
		return ", still in force at the end"
	}
	return ", lifted in " + strings.ToLower(c.year(end))
}

func (c *chronicle) scripted(e event) string {
	switch e.Event {
	case "cleaningFailure":
		return "A cleaning failure (scripted): a cleaner walked out of sight."
	case "breakStairs":
		if ev := c.scenarioEvent(e.Day, "breakStairs"); ev != nil {
			return fmt.Sprintf("The stairs were broken at segment %v (scripted).", ev["segment"])
		}
		return "The stairs were broken (scripted)."
	case "cull":
		if ev := c.scenarioEvent(e.Day, "cull"); ev != nil {
			return fmt.Sprintf("A cull (scripted) of %v of the people.", ev["share"])
		}
		return "A cull (scripted)."
	case "policy":
		return "" // its policy entry says what it did
	}
	return fmt.Sprintf("A scripted event: %s.", e.Event)
}

func (c *chronicle) scenarioEvent(day int, kind string) map[string]any {
	for _, ev := range c.info.Scenario.Events {
		if d, _ := ev["day"].(float64); int(d) == day && ev["type"] == kind {
			return ev
		}
	}
	return nil
}

var triggers = map[string]string{
	"unrestStart":    "unrest",
	"rebellionStart": "a rebellion",
	"suppressed":     "a suppressed rebellion",
	"split":          "a split silo",
	"regimeChange":   "the new regime",
	"silo1":          "Silo 1's order",
	"scripted":       "a scripted policy",
}

// policy is a policy event with the entry it started and the day its
// matching policyEnd says it lapsed.
func (c *chronicle) policy(used []bool, i int) string {
	e := c.events[i]
	end := 0
	for k := i + 1; k < len(c.events); k++ {
		f := c.events[k]
		if !used[k] && f.Type == "policyEnd" && f.Trigger == e.Trigger && f.Start == e.Start && same(f.Entry, e.Entry) {
			used[k], end = true, f.Day
			break
		}
	}
	var spec map[string]any
	switch {
	case e.Entry != nil && *e.Entry < len(c.info.Scenario.Regime.Playbook):
		spec = c.info.Scenario.Regime.Playbook[*e.Entry]
	case e.Trigger == "silo1":
		spec = c.info.Scenario.Silo1.Orders
	case e.Trigger == "scripted":
		spec = c.scenarioEvent(e.Day, "policy")
	}
	why := triggers[e.Trigger]
	if why == "" {
		why = e.Trigger
	}
	t := fmt.Sprintf("The regime answered %s: %s", why, levers(spec))
	if end == 0 {
		return t + ", for the rest of the run."
	}
	return t + ", until " + strings.ToLower(c.year(end)) + "."
}

func same(a, b *int) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

// levers describes a playbook entry's parts, keys in sorted order.
func levers(spec map[string]any) string {
	var parts []string
	if set, ok := spec["set"].(map[string]any); ok {
		keys := make([]string, 0, len(set))
		for k := range set {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			v, _ := json.Marshal(set[k])
			parts = append(parts, fmt.Sprintf("%s %s", k, v))
		}
	}
	if pact, ok := spec["pact"].([]any); ok && len(pact) > 0 {
		parts = append(parts, fmt.Sprintf("%d Pact rules rewritten", len(pact)))
	}
	if lift, ok := spec["lift"].(float64); ok && lift != 0 {
		parts = append(parts, fmt.Sprintf("a legitimacy lift of %v", lift))
	}
	if dose, ok := spec["waterDosing"].(bool); ok && dose {
		parts = append(parts, "Vitamin D+ in the water")
	}
	if len(parts) == 0 {
		return "no change of levers"
	}
	return strings.Join(parts, ", ")
}

var endings = map[string]string{
	"suppressed":   "suppressed",
	"regimeChange": "the regime fell",
	"split":        "the silo split",
	"collapse":     "the silo collapsed",
	"unfinished":   "still running at the end",
}

func (c *chronicle) rebellion(r rebellion) string {
	var b strings.Builder
	fmt.Fprintf(&b, "A rebellion began in %s", zone(r.Zone))
	if r.Trigger != "" && r.Trigger != "none" {
		what := r.Trigger
		if what == "cleaningFailure" {
			what = "cleaning failure"
		}
		fmt.Fprintf(&b, ", after the %s of %s", what, strings.ToLower(c.year(r.TriggerDay)))
	}
	fmt.Fprintf(&b, ". It lasted %s, with at most %.1f%% of adults active.", days(r.Days), 100*r.PeakActive)
	if len(r.Struck) > 0 {
		fmt.Fprintf(&b, " Struck: %s.", strings.Join(r.Struck, ", "))
	}
	if len(r.Held) > 0 {
		names := make([]string, len(r.Held))
		for i, z := range r.Held {
			names[i] = zone(z)
		}
		fmt.Fprintf(&b, " Rebels held %s.", strings.Join(names, ", "))
	}
	if r.FightDays > 0 {
		fmt.Fprintf(&b, " %s of fighting.", capital(days(r.FightDays)))
	}
	fmt.Fprintf(&b, " %s and %s killed; %s arrested", count(r.RebelsKilled, "rebel"), count(r.OfficersKilled, "officer"), num(r.Arrests))
	if r.FightArrests > 0 {
		fmt.Fprintf(&b, " (%s in fights)", num(r.FightArrests))
	}
	b.WriteString(".")
	if r.SegmentsBroken > 0 {
		fmt.Fprintf(&b, " Stair segments broken: %d.", r.SegmentsBroken)
	}
	if r.GeneratorSabotage > 0 || r.PumpSabotage > 0 {
		fmt.Fprintf(&b, " Sabotage: the generator %d times, the water pump %d times.", r.GeneratorSabotage, r.PumpSabotage)
	}
	end := endings[r.Ending]
	if end == "" {
		end = r.Ending
	}
	fmt.Fprintf(&b, " Ending: %s.", end)
	return b.String()
}

// decade is one section: its years, last day, and summary line.
type decade struct {
	from    int
	to      string
	last    int
	summary string
}

// decades cuts the measured years into tens, each closed by a line from
// the yearly snapshot on its last day; a run that ends mid-decade gets a
// last, shorter section closed by its final snapshot.
func (c *chronicle) decades() []decade {
	cleanings := map[int]map[string]int{} // decade → rule → successful cleanings
	for _, e := range c.events {
		if e.Type == "cleaning" && !e.Failed && e.Day > c.warmup {
			k := (e.Day - c.warmup - 1) / daysInDecade
			if cleanings[k] == nil {
				cleanings[k] = map[string]int{}
			}
			cleanings[k][e.Rule]++
		}
	}
	var out []decade
	for k := 0; c.warmup+k*daysInDecade < c.lastDay; k++ {
		last := min(c.warmup+(k+1)*daysInDecade, c.lastDay)
		to := strconv.Itoa((k + 1) * decadeYears)
		if last < c.warmup+(k+1)*daysInDecade {
			to = strings.TrimSuffix(fmt.Sprintf("%.1f", float64(last-c.warmup)/daysPerYear), ".0")
		}
		out = append(out, decade{from: k * decadeYears, to: to, last: last, summary: c.summary(last, cleanings[k])})
	}
	return out
}

// summary is the decade line: the last yearly snapshot on or before day,
// the population on the snapshot's day, so the line is of one moment, and
// the decade's cleanings by rule. Columns of a module that was off are
// left out.
func (c *chronicle) summary(day int, cleanings map[string]int) string {
	var row []string
	snap := day
	for _, r := range c.yearly {
		if d, _ := strconv.Atoi(r[0]); d <= day {
			row, snap = r, d
		}
	}
	day = snap
	col := func(name string) (string, bool) {
		i := slices.Index(c.yearHead, name)
		if i < 0 || row == nil {
			return "", false
		}
		return row[i], true
	}
	var parts []string
	if p, ok := c.pop[day]; ok {
		n, _ := strconv.Atoi(p)
		parts = append(parts, "population "+num(n))
	}
	if v, ok := col("meanL"); ok {
		x, _ := strconv.ParseFloat(v, 64)
		parts = append(parts, fmt.Sprintf("mean legitimacy %.2f", x))
	}
	if v, ok := col("knowHistory"); ok {
		x, _ := strconv.ParseFloat(v, 64)
		parts = append(parts, fmt.Sprintf("%.1f%% of adults know the true history", 100*x))
	}
	if v, ok := col("flamekeepers"); ok {
		n, _ := strconv.Atoi(v)
		parts = append(parts, num(n)+" Flamekeepers")
	}
	if v, ok := col("held"); ok {
		n, _ := strconv.Atoi(v)
		parts = append(parts, num(n)+" people held by Judicial")
	}
	if len(cleanings) > 0 {
		rules := make([]string, 0, len(cleanings))
		for r := range cleanings {
			rules = append(rules, r)
		}
		slices.Sort(rules)
		for i, r := range rules {
			rules[i] = fmt.Sprintf("%s %d", r, cleanings[r])
		}
		parts = append(parts, "cleanings this decade: "+strings.Join(rules, ", "))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("*%s:* %s.", c.year(day), strings.Join(parts, "; "))
}

func zone(z string) string {
	switch z {
	case "upTop":
		return "Up Top"
	case "mids":
		return "the Mids"
	case "downDeep":
		return "the Down Deep"
	}
	return z
}

func days(n int) string {
	switch {
	case n == 1:
		return "1 day"
	case n < 2*daysPerYear:
		return fmt.Sprintf("%d days", n)
	}
	return fmt.Sprintf("%.1f years", float64(n)/daysPerYear)
}

// count is n and a noun, plural unless n is 1.
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return num(n) + " " + noun + "s"
}

func capital(s string) string { return strings.ToUpper(s[:1]) + s[1:] }

// num writes an integer with thousands separators.
func num(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		return "-" + num(-n)
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
