package report

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fake writes a run folder: 1 warm-up year (logged, to test that it is
// dropped) and 10 measured years. Births: 100 per 5-year block, then 130.
// Population 9,500, one day at 11,001. Rationing on 73 measured days (2%).
// Two failures. Mechanical leaves for 400 days, sheriff stays short from
// day 3,000 to the end. Median age 31–39, Mechanical Down Deep 95% then
// 74% once.
func fake(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "baseline-7")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("run.json", `{"seed":7,"finished":true,"scenario":{"warmupYears":1,"output":{"logWarmup":true}}}`)

	var daily strings.Builder
	daily.WriteString("day,population,births,rationing,foodStockDays,popUpTop,popMids,popDownDeep,healthUpTop,healthMids,healthDownDeep\n")
	for day := 1; day <= 11*365; day++ {
		pop, births, rationing := 9500, 0, 0
		measured := day - 365
		switch {
		case day <= 365:
			pop, births, rationing = 50, 999, 1 // warm-up: would fail everything
		case measured == 500:
			pop = 11001
		}
		if measured > 0 && measured%365 == 1 { // first day of each measured year
			births = 20
			if measured > 5*365 {
				births = 26
			}
		}
		if measured > 0 && measured <= 73 {
			rationing = 1
		}
		// Food stock grows 0.05 days a day over the measured run (a 5%
		// surplus); Down Deep's health dips to 0.5 on day 1000 only.
		food, hDD := 90+0.05*float64(max(measured-1, 0)), 1.0
		if day == 1000 {
			hDD = 0.5
		}
		fmt.Fprintf(&daily, "%d,%d,%d,%d,%.2f,%d,%d,%d,1,1,%.1f\n", day, pop, births, rationing, food, pop/2, pop/4, pop-pop/2-pop/4, hDD)
	}
	write("daily.csv", daily.String())

	var yearly strings.Builder
	yearly.WriteString("day,medianAge,mechanicalInZone\n")
	for y := 1; y <= 11; y++ {
		age, mech := 31.0+float64(y%9), 0.95
		if y == 1 {
			age, mech = 10, 0 // warm-up
		}
		if y == 6 {
			mech = 0.74
		}
		fmt.Fprintf(&yearly, "%d,%.2f,%.4f\n", y*365, age, mech)
	}
	write("yearly.csv", yearly.String())

	write("events.jsonl", strings.Join([]string{
		`{"day":100,"type":"failure"}`, // warm-up
		`{"day":200,"type":"understaffed","department":"it"}`,
		`{"day":1000,"type":"failure"}`,
		`{"day":1200,"type":"understaffed","department":"mechanical"}`,
		`{"day":1600,"type":"restaffed","department":"mechanical"}`,
		`{"day":2000,"type":"failure"}`,
		`{"day":3000,"type":"understaffed","department":"sheriff"}`,
	}, "\n")+"\n")
	return dir
}

func TestLoadRun(t *testing.T) {
	s, err := LoadRun(fake(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.Seed != 7 || s.Days != 10*365 {
		t.Errorf("seed %d, %d measured days; want 7 and %d", s.Seed, s.Days, 10*365)
	}
	if s.PopMin != 9500 || s.PopMax != 11001 {
		t.Errorf("population %d–%d, want 9500–11001", s.PopMin, s.PopMax)
	}
	if math.Abs(s.BirthsShock-0.30) > 1e-12 {
		t.Errorf("births shock %v, want 0.30 (100 → 130 per 5 years)", s.BirthsShock)
	}
	if math.Abs(s.RationingShare-0.02) > 1e-12 {
		t.Errorf("rationing share %v, want 0.02", s.RationingShare)
	}
	if s.FailuresPerCentury != 20 {
		t.Errorf("failures per century %v, want 20 (2 in 10 years)", s.FailuresPerCentury)
	}
	// Sheriff: day 3000 to the day after the last (365 + 3650 + 1 = 4016)
	// is 1016 days; Mechanical's 400 is shorter; IT's warm-up spell is not
	// counted.
	if s.LongestUnderstaffed != 1016 || s.UnderstaffedDept != "sheriff" {
		t.Errorf("longest understaffed %d days (%s), want 1016 (sheriff)", s.LongestUnderstaffed, s.UnderstaffedDept)
	}
	// Population-weighted: a quarter of 9,500 (2,375) at 0.5, the rest at 1.
	if math.Abs(s.FoodSurplus-0.05) > 1e-9 || math.Abs(s.HealthMin-(1-0.5*2375.0/9500)) > 1e-9 {
		t.Errorf("food surplus %v, lowest health %v; want 0.05 and %v", s.FoodSurplus, s.HealthMin, 1-0.5*2375.0/9500)
	}
	if s.AgeMin != 31 || s.AgeMax != 39 || s.MechanicalInZoneMin != 0.74 {
		t.Errorf("median age %v–%v, Mechanical in zone ≥ %v; want 31–39 and 0.74", s.AgeMin, s.AgeMax, s.MechanicalInZoneMin)
	}

	var out strings.Builder
	ok, err := Write(&out, []Seed{s})
	if err != nil {
		t.Fatal(err)
	}
	// Ages and rationing pass; population, births shock (not below 30%),
	// failures, staffing and zones fail.
	for _, want := range []string{"9500–11001 FAIL", "31.0–39.0 PASS", "30.0% FAIL", "2.0% PASS", "20.0 FAIL", "1016 d (sheriff) FAIL", "74.0% FAIL"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("report lacks %q:\n%s", want, out.String())
		}
	}
	if ok {
		t.Error("Write reported every pattern holding")
	}
}

// A run that stopped early fails every target but does not stop the
// report.
func TestCrashedRunFails(t *testing.T) {
	dir := fake(t)
	os.WriteFile(filepath.Join(dir, "run.json"), []byte(`{"seed":7,"finished":false,"error":"day 9: boom"}`), 0o644)
	s, err := LoadRun(dir)
	if err != nil || s.Crashed != "day 9: boom" {
		t.Fatalf("got %+v, %v; want a crashed seed", s, err)
	}
	var out strings.Builder
	if ok, _ := Write(&out, []Seed{s}); ok || !strings.Contains(out.String(), "seed 7 crashed: day 9: boom") || strings.Contains(out.String(), "1/1") {
		t.Errorf("report:\n%s", out.String())
	}
}

func TestOldFilesFail(t *testing.T) {
	dir := fake(t)
	os.WriteFile(filepath.Join(dir, "yearly.csv"), []byte("day,medianAge\n365,30\n800,31\n"), 0o644)
	if _, err := LoadRun(dir); err == nil || !strings.Contains(err.Error(), `no column "mechanicalInZone"`) {
		t.Errorf("got %v, want a missing-column error", err)
	}
}
