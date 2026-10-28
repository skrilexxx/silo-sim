package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const baselinePath = "../../scenarios/baseline.yaml"

func readBaseline(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// edit replaces old with new once in src, failing if old is not there, so
// a test can never pass because its edit silently did nothing.
func edit(t *testing.T, src, old, new string) string {
	t.Helper()
	if !strings.Contains(src, old) {
		t.Fatalf("baseline has no %q", old)
	}
	return strings.Replace(src, old, new, 1)
}

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// loadErr loads content as a scenario and returns the error text ("" if none).
func loadErr(t *testing.T, content string) string {
	t.Helper()
	_, err := Load(write(t, t.TempDir(), "s.yaml", content))
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestLoadBaseline(t *testing.T) {
	s, err := Load(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "baseline" || s.Population != 10000 || len(s.Work.Departments) != 11 {
		t.Errorf("baseline loaded wrong: name=%q population=%d departments=%d",
			s.Name, s.Population, len(s.Work.Departments))
	}
}

func TestUnknownKeyFailsWithKeyName(t *testing.T) {
	base := readBaseline(t)
	for _, c := range []struct{ key, typo string }{
		{"warmupYears:", "warmupYear:"}, // top level
		{"gompertzB:", "gompertzb:"},    // nested
	} {
		got := loadErr(t, edit(t, base, c.key, c.typo))
		want := strings.TrimSuffix(c.typo, ":")
		if !strings.Contains(got, want) {
			t.Errorf("typo %q: got %q, want an error naming it", want, got)
		}
	}
}

func TestMissingKeyFailsWithKeyName(t *testing.T) {
	base := readBaseline(t)
	for _, c := range []struct{ old, new, want string }{
		{"  conceptionChance: 0.0044", "  #", "missing key lottery.conceptionChance"},
		{"warmupYears: 50", "", "missing key warmupYears"},
		{"mids: 0.46, ", "", "missing key init.zoneSplit.mids"},
		{"zone: mids,     share: 0.30", "share: 0.30", "missing key work.departments[0].zone"},
	} {
		got := loadErr(t, edit(t, base, c.old, c.new))
		if !strings.Contains(got, c.want) {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestBadRangeFailsWithKeyName(t *testing.T) {
	base := readBaseline(t)
	for _, c := range []struct{ old, new, want string }{
		{"conceptionChance: 0.0044", "conceptionChance: 1.5", "lottery.conceptionChance = 1.5, must be between 0 and 1"},
		{"population: 10000", "population: -5", "population = -5, must not be negative"},
		{"kcalPerFarmUnit: 16000", "kcalPerFarmUnit: -1", "resources.kcalPerFarmUnit = -1, must not be negative"},
		{"workerAge: 18", "workerAge: 70", "work.workerAge (70) must be below work.retireAge (65)"},
		{"shadowAge: 14", "shadowAge: 18", "work.shadowAge (18) must be below work.workerAge (18)"},
		{"motherAgeMax: 40", "motherAgeMax: 17", "lottery.motherAgeMin (18) must be below lottery.motherAgeMax (17)"},
		{"outageMaxDays: 14", "outageMaxDays: 2", "resources.outageMinDays (3) must be at most resources.outageMaxDays (2)"},
		{"upTop: 0.32", "upTop: 0.40", "init.zoneSplit sums to"},
		{"share: 0.30", "share: 0.25", "work.departments shares sums to"},
		{"pairedMinAge: 20", "pairedMinAge: 16", "partnering.minAge (18) must be at most init.pairedMinAge (16)"},
		{"farmFloors: 20", "farmFloors: 48", "init.farmFloors (48) must be below init.floorsPerZone (48)"},
		{"mechanicalFloors: 12", "mechanicalFloors: 50", "init.mechanicalFloors (50) must be below init.floorsPerZone (48)"},
		{"homeCapacityMargin: 1.15", "homeCapacityMargin: 0.9", "init.homeCapacityMargin = 0.9, must be at least 1"},
		{"gompertzA: 0.0002", "gompertzA: 0", "mortality.gompertzA = 0, must be greater than 0"},
		{"gompertzB: 0.085", "gompertzB: 0", "mortality.gompertzB = 0, must be greater than 0"},
		{"floorsPerZone: 48", "floorsPerZone: 0", "init.floorsPerZone = 0, must be greater than 0"},
		{"maternalDeathBase: 0.002", "maternalDeathBase: 0.6", "health.maternalDeathBase = 0.6 gives a maternal death risk of 1.2"},
		{"injuryChance: 0.0002", "injuryChance: 0.6", "health.injuryChance × largest health.injuryFactor = 1.2"},
		// Strictly positive keys, each set to 0.
		{"needFactor: 0.6", "needFactor: 0", "coverage.needFactor = 0, must be greater than 0"},
		{"adultKcal: 2200", "adultKcal: 0", "rations.adultKcal = 0, must be greater than 0"},
		{"ageGapScaleYears: 5.0", "ageGapScaleYears: 0", "partnering.ageGapScaleYears = 0, must be greater than 0"},
		{"powerDemandPerPerson: 1.0", "powerDemandPerPerson: 0", "resources.powerDemandPerPerson = 0, must be greater than 0"},
		{"capacityMargin: 1.3", "capacityMargin: 0", "resources.capacityMargin = 0, must be greater than 0"},
		{"kcalPerFarmUnit: 16000", "kcalPerFarmUnit: 0", "resources.kcalPerFarmUnit = 0, must be greater than 0"},
		{"targetPopulation: 10000", "targetPopulation: 0", "lottery.targetPopulation = 0, must be greater than 0"},
		{"population: 10000", "population: 0", "population = 0, must be greater than 0"},
		{"windowDays: 365", "windowDays: 0", "lottery.windowDays = 0, must be greater than 0"},
		{"pregnancyDays: 270", "pregnancyDays: 0", "lottery.pregnancyDays = 0, must be greater than 0"},
		{"outageMinDays: 3", "outageMinDays: 0", "resources.outageMinDays = 0, must be greater than 0"},
		{"invariantEvery: 365", "invariantEvery: 0", "output.invariantEvery = 0, must be greater than 0"},
		{"share: 0.01", "share: 0", "work.departments[10].share = 0, must be greater than 0"},
	} {
		got := loadErr(t, edit(t, base, c.old, c.new))
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.new, got, c.want)
		}
	}
}

// Values exactly on a bound are valid.
func TestBoundaryValuesAccepted(t *testing.T) {
	base := readBaseline(t)
	for _, c := range []struct{ old, new string }{
		{"outageMaxDays: 14", "outageMaxDays: 3"},              // fixed outage length
		{"maternalDeathBase: 0.002", "maternalDeathBase: 0.5"}, // risk exactly 1 at zero coverage
		{"injuryChance: 0.0002", "injuryChance: 0.5"},          // 0.5 × Mechanical factor 2 = 1
		{"invariantEvery: 365", "invariantEvery: 1"},           // check every tick
	} {
		if got := loadErr(t, edit(t, base, c.old, c.new)); got != "" {
			t.Errorf("%s rejected: %s", c.new, got)
		}
	}
}

func TestExtendsOverride(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "baseline.yaml", readBaseline(t))
	child := write(t, dir, "child.yaml", `extends: baseline.yaml
name: unequal-rations
seed: 7
rations:
  weights: { upTop: 1.5 }
modules: [demography]
`)
	s, err := Load(child)
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		what string
		ok   bool
	}{
		{"scalar overridden", s.Name == "unequal-rations" && s.Seed == 7},
		{"nested key overridden", s.Rations.Weights.UpTop == 1.5},
		{"sibling keys kept", s.Rations.Weights.Mids == 1 && s.Rations.AdultKcal == 2200},
		{"untouched sections kept", s.Population == 10000 && len(s.Work.Departments) == 11},
		{"list replaced whole", len(s.Modules) == 1 && s.Modules[0] == "demography"},
		{"extends cleared", s.Extends == ""},
	}
	for _, c := range checks {
		if !c.ok {
			t.Errorf("%s: got %+v", c.what, s)
		}
	}
}

func TestExtendsChildIsCheckedToo(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "baseline.yaml", readBaseline(t))
	for _, c := range []struct{ body, want string }{
		{"lottery:\n  conceptionChanse: 0.5\n", "conceptionChanse"},
		{"lottery:\n  conceptionChance: 2\n", "lottery.conceptionChance = 2"},
	} {
		_, err := Load(write(t, dir, "child.yaml", "extends: baseline.yaml\n"+c.body))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want an error containing %q", err, c.want)
		}
	}
}

func TestExtendsLoopFails(t *testing.T) {
	dir := t.TempDir()
	a := write(t, dir, "a.yaml", "extends: b.yaml\n")
	write(t, dir, "b.yaml", "extends: a.yaml\n")
	_, err := Load(a)
	if err == nil || !strings.Contains(err.Error(), "extends loop") {
		t.Errorf("got %v, want an extends loop error", err)
	}
}
