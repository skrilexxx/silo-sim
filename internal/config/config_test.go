package config

import (
	"os"
	"path/filepath"
	"slices"
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
		{"kcalPerFarmUnit: 11900", "kcalPerFarmUnit: -1", "resources.kcalPerFarmUnit = -1, must not be negative"},
		{"workerAge: 18", "workerAge: 70", "work.workerAge (70) must be below work.retireAge (65)"},
		{"shadowAge: 14", "shadowAge: 18", "work.shadowAge (18) must be below work.workerAge (18)"},
		{"motherAgeMax: 40", "motherAgeMax: 17", "lottery.motherAgeMin (18) must be below lottery.motherAgeMax (17)"},
		{"outageMaxDays: 10", "outageMaxDays: 2", "resources.outageMinDays (3) must be at most resources.outageMaxDays (2)"},
		{"upTop: 0.32", "upTop: 0.40", "init.zoneSplit sums to"},
		{"share: 0.30", "share: 0.25", "work.departments shares sums to"},
		{"pairedMinAge: 20", "pairedMinAge: 16", "partnering.minAge (18) must be at most init.pairedMinAge (16)"},
		{"farmFloors: 20", "farmFloors: 48", "init.farmFloors (48) must be below init.floorsPerZone (48)"},
		{"mechanicalFloors: 12", "mechanicalFloors: 50", "init.mechanicalFloors (50) must be below init.floorsPerZone (48)"},
		{"homeCapacityMargin: 1.15", "homeCapacityMargin: 0.9", "init.homeCapacityMargin = 0.9, must be at least 1"},
		{"gompertzA: 0.0002", "gompertzA: 0", "mortality.gompertzA = 0, must be greater than 0"},
		{"gompertzB: 0.085", "gompertzB: 0", "mortality.gompertzB = 0, must be greater than 0"},
		{"floorsPerZone: 48", "floorsPerZone: 0", "init.floorsPerZone = 0, must be greater than 0"},
		{"name: medical,", "name: doctors,", `work.departments has no "medical", which the rules need`},
		{"zone: downDeep, share: 0.16", "zone: bottom, share: 0.16", `work.departments[2].zone = "bottom", must be upTop, mids, downDeep or all`},
		{"maternalDeathBase: 0.002", "maternalDeathBase: 0.6", "health.maternalDeathBase = 0.6 gives a maternal death risk of 1.2"},
		{"injuryChance: 0.0002", "injuryChance: 0.6", "health.injuryChance × largest health.injuryFactor = 1.2"},
		{"mode: appointment ", "mode: vote ", `governance.mode = "vote", must be appointment or election`},
		{"term: 5y ", "term: 5 ", `governance.term = "5", must be life or a number of years like 5y`},
		{"term: 5y ", "term: 0y ", `governance.term = "0y", must be life or a number of years like 5y`},
		{"term: 5y ", "term: forever ", `governance.term = "forever", must be life or a number of years like 5y`},
		// Strictly positive keys, each set to 0.
		{"needFactor: 0.84", "needFactor: 0", "coverage.needFactor = 0, must be greater than 0"},
		{"adultKcal: 2200", "adultKcal: 0", "rations.adultKcal = 0, must be greater than 0"},
		{"ageGapScaleYears: 5.0", "ageGapScaleYears: 0", "partnering.ageGapScaleYears = 0, must be greater than 0"},
		{"powerDemandPerPerson: 1.0", "powerDemandPerPerson: 0", "resources.powerDemandPerPerson = 0, must be greater than 0"},
		{"capacityMargin: 1.5", "capacityMargin: 0", "resources.capacityMargin = 0, must be greater than 0"},
		{"kcalPerFarmUnit: 11900", "kcalPerFarmUnit: 0", "resources.kcalPerFarmUnit = 0, must be greater than 0"},
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
		{"outageMaxDays: 10", "outageMaxDays: 3"},              // fixed outage length
		{"maternalDeathBase: 0.002", "maternalDeathBase: 0.5"}, // risk exactly 1 at zero coverage
		{"injuryChance: 0.0002", "injuryChance: 0.5"},          // 0.5 × Mechanical factor 2 = 1
		{"invariantEvery: 365", "invariantEvery: 1"},           // check every tick
		{"term: 5y ", "term: life "},
		{"term: 5y ", "term: 12y "},
		{"mode: appointment ", "mode: election "},
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

// ODD §5: scripted events fall in the measured years (days 18251–91250
// with 50 + 200 years) and name a type.
func TestScriptedEvents(t *testing.T) {
	base := readBaseline(t)
	for _, c := range []struct{ events, want string }{
		{"- { day: 18251, type: shock }\n- { day: 91250, type: shock }", ""},
		{"- { day: 18250, type: shock }", "events[0].day = 18250, must be in the measured years, 18251 to 91250"},
		{"- { day: 91251, type: shock }", "events[0].day = 91251, must be in the measured years"},
		{"- { day: 20000, type: shock }\n- { day: 20000, type: \"\" }", "events[1].type is empty"},
		{"- { day: 20000 }", "missing key events[0].type"},
		{"- { day: 20000, type: shock, days: 365 }", "field days not found"},
	} {
		got := loadErr(t, base+"events:\n"+c.events+"\n")
		if c.want == "" && got != "" || !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.events, got, c.want)
		}
	}
	s, err := Load(baselinePath)
	if err != nil || s.Events != nil {
		t.Errorf("baseline has events %v (err %v), want none", s.Events, err)
	}
}

// ODD §6.10: office floors name departments that work on office floors,
// cover every one of them, and lie inside the silo.
func TestOfficeFloors(t *testing.T) {
	base := readBaseline(t)
	for _, c := range []struct{ old, new, want string }{
		{"    it: [33, 34]\n", "", "space.officeFloors has no floors for it"},
		{"    it: [33, 34]", "    it: []", "space.officeFloors has no floors for it"},
		{"    it: [33, 34]", "    it: [33, 145]", "space.officeFloors.it: floor 145, must be 1 to 144"},
		{"    it: [33, 34]", "    it: [33, 34]\n    farms: [50]", "space.officeFloors.farms: not a department that works on office floors"},
		{"    it: [33, 34]", "    it: [33, 34]\n    porters: [50]", "space.officeFloors.porters: not a department"},
		{"    it: [33, 34]", "    it: [33, 34]\n    cooks: [50]", "space.officeFloors.cooks: not a department"},
		{"walkingHoursPerDay: 8", "walkingHoursPerDay: 0", "space.walkingHoursPerDay = 0, must be greater than 0"},
		{"fullDescentHours: 12", "fullDescentHours: 0", "space.fullDescentHours = 0, must be greater than 0"},
		{"weakTieStrength: 0.3", "weakTieStrength: 1.5", "network.weakTieStrength = 1.5, must be between 0 and 1"},
		{"weakTiesPerKind: 5", "weakTiesPerKind: -1", "network.weakTiesPerKind = -1, must not be negative"},
	} {
		got := loadErr(t, edit(t, base, c.old, c.new))
		if !strings.Contains(got, c.want) {
			t.Errorf("%q: got %q, want %q", c.new, got, c.want)
		}
	}
}

// ODD §6.9, §6.13: the control section and the Pact table.
func TestControlAndPact(t *testing.T) {
	base := readBaseline(t)
	for _, c := range []struct{ old, new, want string }{
		{"surveillance: 0.05", "surveillance: 1.5", "control.surveillance = 1.5, must be between 0 and 1"},
		{"raidsPerYear: 4 ", "raidsPerYear: 400 ", "control.raidsPerYear = 400, must be at most 365"},
		{"minesFloor: 144", "minesFloor: 145", "control.minesFloor = 145, must be 1 to 144"},
		{"liftHalfLifeYears: 1 ", "liftHalfLifeYears: 0 ", "control.liftHalfLifeYears = 0, must be greater than 0"},
		{"deniedTries: 3 ", "deniedTries: 0 ", "control.deniedTries = 0, must be greater than 0"},
		{"implantDoctors: 2 ", "implantDoctors: -1 ", "control.implantDoctors = -1, must not be negative"},
		{"id: noContact ", "id: noRelics ", `pact[4].id = "noRelics", must be unique`},
		{"offences: [organise]", "offences: [organize]", `pact[6].offences: "organize", must be one of`},
		{"offences: [organise]", "offences: [organise, violence]", `pact[6].offences: "violence" is also listed by active rule noViolence`},
		{"offences: [organise]", "offences: []", "pact[6].offences is empty"},
		{"first:  { mines: 730 }", "first:  { mines: 730, cleaning: true }", "pact[5].first: at most one of prison, mines and cleaning"},
		{"repeat: { mines: 365 }", "repeat: { prison: -3 }", "pact[6].repeat: prison and mines days must not be negative"},
		{"visibility: 0.7", "visibility: 1.7", "pact[6].visibility = 1.7, must be between 0 and 1"},
		{"enforcer: sheriff", "enforcer: mayor", `pact[2].enforcer = "mayor", must be judicial, sheriff or silo1`},
		{"    detection: 0.6", "    detection: -1", "pact[6].detection = -1, must not be negative"},
		{"    detection: 0.6\n", "", "missing key pact[6].detection"},
	} {
		got := loadErr(t, edit(t, base, c.old, c.new))
		if !strings.Contains(got, c.want) {
			t.Errorf("%q: got %q, want %q", c.new, got, c.want)
		}
	}
	// An inactive rule may list an offence an active one lists.
	cfgOff := edit(t, base, "offences: [organise]", "offences: [organise, violence]")
	cfgOff = edit(t, cfgOff, "id: noOrganising              # unauthorised gatherings and strikes\n    active: true", "id: noOrganising\n    active: false")
	dir := t.TempDir()
	if _, err := Load(write(t, dir, "s.yaml", cfgOff)); err != nil {
		t.Errorf("inactive rule sharing an offence: %v", err)
	}
}

// ODD §6.15: the playbook and policy events. A hold is quiet, forever or
// days (a YAML number decodes as days); each entry is checked against the
// scenario it would put in force, so a lever out of its setting's range,
// or rows that break the Pact, fail at load with the entry's key.
func TestRegimeEntries(t *testing.T) {
	base := readBaseline(t)
	playbook := func(entries string) string {
		return edit(t, base, "  playbook: []", "  playbook:\n"+entries)
	}
	s, err := Load(write(t, t.TempDir(), "s.yaml", playbook(
		"    - { on: unrestStart, hold: quiet, set: { scapegoatHardship: 0.5, rationWeights: { upTop: 1, mids: 1, downDeep: 2 } } }\n"+
			"    - { on: suppressed, hold: 30, lift: -0.2, waterDosing: true }\n")))
	if err != nil {
		t.Fatal(err)
	}
	if days, _ := ParseHold(s.Regime.Playbook[1].Hold); days != 30 {
		t.Errorf("hold: 30 read as %d days", days)
	}
	if *s.Regime.Playbook[0].Set.ScapegoatHardship != 0.5 || s.Regime.Playbook[0].Set.Surveillance != nil {
		t.Error("levers set and unset read wrong")
	}

	for _, c := range []struct{ entries, want string }{
		{"    - { on: lunch, hold: 3 }\n", "regime.playbook[0].on"},
		{"    - { on: suppressed, hold: quiet }\n", "regime.playbook[0].hold = quiet"},
		{"    - { on: unrestStart, hold: 0 }\n", "regime.playbook[0].hold"},
		{"    - { on: unrestStart, hold: 3, lift: 1.5 }\n", "regime.playbook[0].lift"},
		{"    - { on: unrestStart, hold: 3, set: { surveillance: 2 } }\n", "control.surveillance"},
		{"    - { on: unrestStart, hold: 3, set: { governance: vote } }\n", "governance.mode"},
		{"    - { on: unrestStart, hold: 3, set: { rationWeights: { downDeep: 2 } } }\n", "missing key regime.playbook[0].set.rationWeights.upTop"},
		{"    - { on: unrestStart, hold: 3, pact: [ { id: twice, active: true, offences: [organise], detection: 1, first: {}, enforcer: judicial, visibility: 1 } ] }\n",
			`"organise" is also listed by active rule`},
		{"    - { on: unrestStart }\n", "missing key regime.playbook[0].hold"},
	} {
		if got := loadErr(t, playbook(c.entries)); !strings.Contains(got, c.want) {
			t.Errorf("%q: error %q, want it to mention %q", c.entries, got, c.want)
		}
	}

	event := func(e string) string { return base + "events:\n  - " + e + "\n" }
	if got := loadErr(t, event("{ day: 20000, type: policy, hold: quiet }")); !strings.Contains(got, "events[0].hold = quiet") {
		t.Errorf("a quiet policy event: %q", got)
	}
	if got := loadErr(t, event("{ day: 20000, type: policy, hold: forever, set: { minesPlaces: 0 } }")); got != "" {
		t.Errorf("a valid policy event failed: %s", got)
	}
}

// ODD §6.15: the scenario in force keeps every rule at a fixed index (the
// scenario's rows, then new ids), a rule only an entry adds is inactive
// until the entry runs, and the later of two entries wins a lever.
func TestInForce(t *testing.T) {
	s, err := Load(baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	strict := Rule{ID: "strict", Active: true, Offences: []string{"organise"}, Enforcer: "judicial"}
	off := s.Pact[slices.IndexFunc(s.Pact, func(r Rule) bool { return r.ID == "noOrganising" })]
	off.Active = false
	low, high := 0.1, 0.3
	a := Entry{On: "unrestStart", Hold: "quiet", Set: Levers{Surveillance: &low}, Pact: []Rule{off, strict}}
	b := Entry{On: "rebellionStart", Hold: "quiet", Set: Levers{Surveillance: &high}}
	s.Regime.Playbook = []Entry{a, b}

	peace := s.InForce(nil)
	n := len(s.Pact)
	if len(peace.Pact) != n+1 || peace.Pact[n].ID != "strict" || peace.Pact[n].Active {
		t.Fatalf("peacetime Pact %+v: want the scenario's rows, then strict, inactive", peace.Pact[n:])
	}
	if peace.Control.Surveillance != s.Control.Surveillance || peace.Regime.Playbook != nil {
		t.Error("peacetime changed a lever or kept the playbook")
	}
	p := s.InForce([]Entry{a, b})
	if p.Control.Surveillance != high || !p.Pact[n].Active || len(p.Pact) != n+1 {
		t.Errorf("σ %v, strict active %v: want the later entry's σ and strict in force at its index", p.Control.Surveillance, p.Pact[n].Active)
	}
	if err := p.CheckPact(); err != nil {
		t.Error(err)
	}
	if s.Pact[0].Active != true || len(s.Pact) != n {
		t.Error("InForce changed the scenario")
	}
}
