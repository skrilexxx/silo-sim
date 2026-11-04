package report

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ODD §1.2, C2: 20 measured years make four 5-year blocks. A scripted
// failure on day 2000 is a treated window and rules out blocks 2 and 3; a
// sentenced failure on day 6000 rules out block 4, so block 1 is the only
// comparison. A scripted failure on day 7000 runs past the end of the run
// and counts only because a rebellion started in it.
func TestWindows(t *testing.T) {
	c := &Canon{First: 1, Last: 20 * 365, Scripted: []int{2000, 7000}, Failures: []int{2000, 6000, 7000},
		Rebellions: []Rebellion{{Start: 100}, {Start: 2500}, {Start: 7200}}}
	if a, b, x, y := c.windows(); a != 2 || b != 2 || x != 1 || y != 1 {
		t.Errorf("treated %d/%d, comparison %d/%d; want 2/2 and 1/1", b, a, y, x)
	}
	c.Rebellions = c.Rebellions[:2]
	if a, b, _, _ := c.windows(); a != 1 || b != 1 {
		t.Errorf("treated %d/%d; want the window cut short with no rebellion dropped, 1/1", b, a)
	}

	// No failures: a block cut short counts only with a rebellion in it.
	c = &Canon{First: 1, Last: 2000}
	if _, _, x, y := c.windows(); x != 1 || y != 0 {
		t.Errorf("comparison %d/%d; want 0/1", y, x)
	}
	c.Rebellions = []Rebellion{{Start: 1900}}
	if _, _, x, y := c.windows(); x != 2 || y != 1 {
		t.Errorf("comparison %d/%d; want 1/2", y, x)
	}
}

// ODD §1.2, C4: each suppressed rebellion is judged at the first yearly
// snapshot 40 years or more after its end; one with no such snapshot, and
// any other ending, is not judged.
func TestForgotten(t *testing.T) {
	end, late := 365, 400
	c := &Canon{Rebellions: []Rebellion{
		{End: &end, Ending: "suppressed"},   // judged on day 41 × 365: 5% know
		{End: &end, Ending: "regimeChange"}, // not suppressed
		{End: &late, Ending: "suppressed"},  // judged on day 42 × 365: 20% know
		{Ending: "unfinished"},
	}}
	for y := 1; y <= 42; y++ {
		know := 0.5
		switch y {
		case 41:
			know = 0.05
		case 42:
			know = 0.2
		}
		c.History = append(c.History, [2]float64{float64(y * 365), know})
	}
	if e, b := c.forgotten(); e != 2 || b != 1 {
		t.Errorf("%d judged, %d forgotten; want 2 and 1", e, b)
	}
	c.History = c.History[:41]
	if e, b := c.forgotten(); e != 1 || b != 1 {
		t.Errorf("%d judged, %d forgotten; want 1 and 1 (no snapshot 40 years after the second)", e, b)
	}
}

// A run with rebellions.jsonl gets its part-2 record and the canon
// section; one without has neither.
func TestCanonReport(t *testing.T) {
	dir := fake(t)
	if s, err := LoadRun(dir); err != nil || s.Canon != nil {
		t.Fatalf("baseline run: canon %+v, %v", s.Canon, err)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("run.json", `{"seed":7,"finished":true,"outcomes":{"unrestSpells":3,"collapseCause":"air"},"scenario":{"warmupYears":1,"output":{"logWarmup":true}}}`)
	write("rebellions.jsonl", `{"start":500,"end":600,"days":100,"zone":"downDeep","struck":["mechanical"],"ending":"suppressed"}`+"\n")
	var yearly strings.Builder
	yearly.WriteString("day,medianAge,mechanicalInZone,knowHistory\n")
	for y := 1; y <= 11; y++ {
		yearly.WriteString(strings.Join([]string{strconv.Itoa(y * 365), "35", "0.9", "0.3"}, ",") + "\n")
	}
	write("yearly.csv", yearly.String())
	s, err := LoadRun(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c := s.Canon; c == nil || c.First != 366 || c.Last != 11*365 || len(c.Rebellions) != 1 || len(c.History) != 10 || c.Collapse != "air" || c.Unrest != 3 {
		t.Fatalf("canon %+v", s.Canon)
	}
	var out strings.Builder
	if ok, err := Write(&out, []Seed{s}); err != nil || ok {
		t.Fatalf("Write: %v, all holding %v", err, ok)
	}
	// 1 rebellion in 10 years, in the first comparison block; no scripted
	// failure; Down Deep with a
	// Mechanical strike; too early to judge forgetting, so C4 fails; 100
	// of the 3,650 measured days in a rebellion.
	for _, want := range []string{"10.00 (1 in 10 years)", "0/0 after vs 1/2 without", "1/1 (100%), 1/1 (100%)", "0/0 (0%)",
		"suppressed 1,", "unrest spells 3", "collapses 1: air 1", "2.7% of measured days in a rebellion"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report lacks %q:\n%s", want, out.String())
		}
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "C4") && (!strings.HasSuffix(strings.TrimSpace(line), "FAIL") || strings.Contains(line, "not judged")) {
			t.Errorf("C4 not judged: %q", line)
		}
	}
}

// A silo that collapsed in warm-up wrote no measured rows: it is listed
// as failing every target, and the report goes on.
func TestCollapsedInWarmup(t *testing.T) {
	dir := fake(t)
	os.WriteFile(filepath.Join(dir, "run.json"), []byte(`{"seed":7,"finished":true,"outcomes":{"collapseDay":200,"collapseCause":"air"},"scenario":{"warmupYears":1}}`), 0o644)
	os.WriteFile(filepath.Join(dir, "daily.csv"), []byte("day,population\n"), 0o644)
	s, err := LoadRun(dir)
	if err != nil || s.Crashed != "collapsed in warm-up on day 200 (air)" {
		t.Fatalf("got %q, %v", s.Crashed, err)
	}
}

// ODD §1.2, C5: judged only when the runs had Silo 1 on: 1 of 4
// rebellions ending in collapse is 25%, inside 5–50%.
func TestC5(t *testing.T) {
	c := &Canon{First: 1, Last: 100 * 365, Silo1: true, Rebellions: []Rebellion{
		{Ending: "suppressed"}, {Ending: "suppressed"}, {Ending: "split"}, {Ending: "collapse"}}}
	var out strings.Builder
	writeCanon(&out, []Seed{{Canon: c}})
	if !strings.Contains(out.String(), "1/4 (25%)") || !strings.Contains(out.String(), "C5 A rebellion can end a silo") {
		t.Errorf("no C5 line, or a wrong one:\n%s", out.String())
	}
	c.Silo1 = false
	out.Reset()
	writeCanon(&out, []Seed{{Canon: c}})
	if strings.Contains(out.String(), "C5") {
		t.Error("C5 judged without Silo 1")
	}
}
