package resources

import (
	"math"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const year = engine.DaysPerYear

// Department indexes in the baseline table.
const farms, mechanical, supply = 0, 2, 3

func baseline(t *testing.T) *config.Scenario {
	t.Helper()
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// silo builds 100 people in one Up Top household, 2 farm floors and the baseline departments with
// Mechanical coverage 1, then runs Init.
func silo(t *testing.T, cfg *config.Scenario) (*world.World, *Module, *rng.Stream) {
	t.Helper()
	w := &world.World{
		Floors:     []world.Floor{{Number: 1, Function: world.Farm}, {Number: 2, Function: world.Residential}, {Number: 3, Function: world.Farm}},
		Households: []world.Household{{Floor: 2}}, // everyone lives Up Top
	}
	for _, d := range cfg.Work.Departments {
		w.Depts = append(w.Depts, world.Department{Name: d.Name})
	}
	w.Depts[mechanical].Coverage = 1
	for i := range 100 {
		w.People = append(w.People, world.Person{ID: world.PersonID(i), Alive: true})
		w.Alive = append(w.Alive, world.PersonID(i))
	}
	m := New()
	if err := m.Init(w, cfg, nil); err != nil {
		t.Fatal(err)
	}
	r, _ := rng.New(1, "resources")
	return w, m, r
}

func step(t *testing.T, w *world.World, m *Module, r *rng.Stream) {
	t.Helper()
	if err := m.generator(w, r); err != nil {
		t.Fatal(err)
	}
}

// With full Mechanical coverage and parts, condition settles where wear
// and repair balance: c* = 1 − wear / ρ = 0.8, from above and from below.
func TestConditionSettles(t *testing.T) {
	for _, start := range []float64{1, 0.5} {
		cfg := baseline(t)
		cfg.Resources.FailureBase = 0
		w, m, r := silo(t, cfg)
		w.Silo.GeneratorCondition = start
		w.Silo.PartsStock = 1e9
		for range 20 * year {
			step(t, w, m, r)
		}
		if c := w.Silo.GeneratorCondition; math.Abs(c-0.8) > 0.02 {
			t.Errorf("from %v: condition after 20 years = %.4f, want 0.80 ± 0.02", start, c)
		}
	}
}

// f₀ × (1 − c)² at c = 0.8 gives about 2.92 failures per century.
func TestFailureRate(t *testing.T) {
	_, m, r := silo(t, baseline(t))
	const centuries = 100
	failures := 0
	for range centuries * 100 * year {
		if r.Bernoulli(m.failChance(0.8)) {
			failures++
		}
	}
	perCentury := float64(failures) / centuries
	t.Logf("%d failures in %d centuries at c = 0.8: %.2f per century (expected 2.92)", failures, centuries, perCentury)
	if perCentury < 2.5 || perCentury > 3.5 {
		t.Errorf("%.2f failures per century, want 2.5–3.5", perCentury)
	}
}

// With no parts there is no repair: condition falls by exactly the wear
// each day.
func TestNoPartsSteadyFall(t *testing.T) {
	cfg := baseline(t)
	cfg.Resources.FailureBase = 0
	w, m, r := silo(t, cfg)
	w.Silo.PartsStock = 0
	prev := w.Silo.GeneratorCondition
	for day := range year {
		step(t, w, m, r)
		if c := w.Silo.GeneratorCondition; math.Abs(prev-c-cfg.Resources.GeneratorWear) > 1e-12 {
			t.Fatalf("day %d: condition %v → %v, want a fall of exactly %v", day+1, prev, c, cfg.Resources.GeneratorWear)
		}
		prev = w.Silo.GeneratorCondition
	}
	want := cfg.Init.GeneratorCondition - year*cfg.Resources.GeneratorWear
	if c := w.Silo.GeneratorCondition; math.Abs(c-want) > 1e-9 {
		t.Errorf("condition after a year = %v, want %v", c, want)
	}
}

// A failure drops condition, uses parts and gives backup power for its
// whole length; while down, condition and parts stay frozen. Then normal
// power returns.
func TestOutage(t *testing.T) {
	for _, parts := range []float64{100, 10} {
		cfg := baseline(t)
		rc := cfg.Resources
		cfg.Resources.FailureBase = 1
		cfg.Resources.GeneratorWear = 0
		w, m, r := silo(t, cfg)
		w.Silo.GeneratorCondition = 0.5
		w.Silo.PartsStock = parts
		// Wear 0, so the repair before the draw is the only other change.
		cAfterRepair := 0.5 + rc.RepairRate*0.5
		step(t, w, m, r)

		s := &w.Silo
		length := s.GeneratorDown
		if length < rc.OutageMinDays || length > rc.OutageMaxDays {
			t.Fatalf("outage length %d, want %d–%d", length, rc.OutageMinDays, rc.OutageMaxDays)
		}
		if want := cAfterRepair - rc.FailureConditionDrop; math.Abs(s.GeneratorCondition-want) > 1e-12 {
			t.Errorf("condition after failure = %v, want %v", s.GeneratorCondition, want)
		}
		wantParts := max(0, parts-rc.PartsPerRepairDay-rc.PartsPerFailure)
		if s.PartsStock != wantParts {
			t.Errorf("parts %v → %v, want %v", parts, s.PartsStock, wantParts)
		}
		m.c.FailureBase = 0 // the day the generator runs again must not fail at once
		c, p := s.GeneratorCondition, s.PartsStock
		backup := rc.BackupPower * m.capacity
		for day := 1; day <= length; day++ {
			if s.GeneratorDown == 0 || s.PowerSupply != backup {
				t.Fatalf("outage day %d of %d: down %d, supply %v, want backup %v", day, length, s.GeneratorDown, s.PowerSupply, backup)
			}
			if s.GeneratorCondition != c || s.PartsStock != p {
				t.Fatalf("outage day %d: condition or parts changed while down", day)
			}
			step(t, w, m, r)
		}
		if s.GeneratorDown != 0 || s.PowerSupply != m.capacity*s.GeneratorCondition {
			t.Errorf("day after the outage: down %d, supply %v, want normal %v", s.GeneratorDown, s.PowerSupply, m.capacity*s.GeneratorCondition)
		}
	}
}

// Capacity is capacityMargin × day-1 demand, so at condition 0.8 and full
// Mechanical coverage supply is 1.04 × demand and powerFactor is 1.
func TestPowerDayOne(t *testing.T) {
	cfg := baseline(t)
	cfg.Resources.FailureBase = 0
	w, m, r := silo(t, cfg)
	rc := cfg.Resources
	demand := rc.PowerDemandPerPerson*100 + rc.PowerDemandPerFarmFloor*2
	if math.Abs(m.capacity-rc.CapacityMargin*demand) > 1e-9 {
		t.Errorf("capacity = %v, want %v", m.capacity, rc.CapacityMargin*demand)
	}
	w.Silo.PartsStock = 1e9
	step(t, w, m, r)
	s := &w.Silo
	if s.PowerDemand != demand {
		t.Errorf("demand = %v, want %v", s.PowerDemand, demand)
	}
	if ratio := s.PowerSupply / s.PowerDemand; math.Abs(ratio-1.04) > 0.001 {
		t.Errorf("supply / demand = %.4f, want ≈ 1.04", ratio)
	}
	if pf := s.PowerFactor(); pf != 1 {
		t.Errorf("powerFactor = %v, want 1", pf)
	}
}

// A 14-day outage pulls air below 0.3; after the repair it is back above
// 0.95 within 10 days.
func TestOutageAir(t *testing.T) {
	cfg := baseline(t)
	cfg.Resources.FailureBase = 0
	w, m, r := silo(t, cfg)
	s := &w.Silo
	s.AirQuality, s.GeneratorCondition, s.PartsStock = 1, 0.9, 1e9
	s.GeneratorDown = 15 // counts down to 14 on the first step: 14 days on backup
	day := func() {
		t.Helper()
		step(t, w, m, r)
		if err := m.lifeSupport(w, r); err != nil {
			t.Fatal(err)
		}
	}
	for range 14 {
		day()
	}
	if s.GeneratorDown != 1 {
		t.Fatalf("generator down %d after 14 outage days, want 1 (back tomorrow)", s.GeneratorDown)
	}
	t.Logf("air after 14 days on backup: %.3f", s.AirQuality)
	if s.AirQuality >= 0.3 {
		t.Errorf("air after a 14-day outage = %.3f, want below 0.3", s.AirQuality)
	}
	for n := 1; ; n++ {
		day()
		if s.AirQuality > 0.95 {
			t.Logf("air back above 0.95 on day %d after the repair", n)
			break
		}
		if n == 10 {
			t.Fatalf("air still %.3f 10 days after the repair", s.AirQuality)
		}
	}
}

// Water is powerFactor × Mechanical coverage.
func TestWater(t *testing.T) {
	w, m, r := silo(t, baseline(t))
	w.Silo.PowerSupply, w.Silo.PowerDemand = 1, 2
	w.Depts[mechanical].Coverage = 0.5
	if err := m.lifeSupport(w, r); err != nil {
		t.Fatal(err)
	}
	if w.Silo.WaterSupply != 0.25 {
		t.Errorf("water = %v, want 0.5 × 0.5 = 0.25", w.Silo.WaterSupply)
	}
}

// Food production = farm output × kcalPerFarmUnit × powerFactor.
func TestFoodProduction(t *testing.T) {
	cfg := baseline(t)
	w, m, r := silo(t, cfg)
	w.Silo.FoodStock, w.Silo.PowerSupply, w.Silo.PowerDemand = 1000, 1, 2
	w.Depts[farms].Output = 10
	if err := m.food(w, r); err != nil {
		t.Fatal(err)
	}
	want := 10 * cfg.Resources.KcalPerFarmUnit * 0.5
	if w.Silo.FoodProduced != want || w.Silo.FoodStock != 1000+want {
		t.Errorf("produced %v, stock %v; want %v and %v", w.Silo.FoodProduced, w.Silo.FoodStock, want, 1000+want)
	}
}

// Supply adds partsPerDayFullCoverage × supplyCoverage every day, also
// while the generator is down and uses none.
func TestSupplyMakesParts(t *testing.T) {
	cfg := baseline(t)
	w, m, r := silo(t, cfg)
	w.Depts[supply].Coverage = 0.5
	w.Silo.PartsStock, w.Silo.GeneratorDown = 0, 5
	step(t, w, m, r)
	if want := cfg.Resources.PartsPerDayFullCoverage * 0.5; w.Silo.PartsStock != want {
		t.Errorf("parts after an outage day = %v, want %v", w.Silo.PartsStock, want)
	}
}

// ODD §5 step 8: the day-1 food stock is foodStockDays of full need, a
// child under childAgeBelow needing childNeedShare of an adult.
func TestDayOneFoodStock(t *testing.T) {
	cfg := baseline(t)
	w, _, _ := silo(t, cfg) // 100 newborns
	rc := cfg.Rations
	child := rc.ChildNeedShare * rc.AdultKcal
	if want := float64(cfg.Init.FoodStockDays) * 100 * child; math.Abs(w.Silo.FoodStock-want) > 1e-6 {
		t.Errorf("children only: stock %v, want %v", w.Silo.FoodStock, want)
	}
	for i := range 40 {
		w.People[i].BirthDay = world.Tick(-rc.ChildAgeBelow * year) // exactly childAgeBelow: an adult
	}
	m := New()
	m.Init(w, cfg, nil)
	if want := float64(cfg.Init.FoodStockDays) * (40*rc.AdultKcal + 60*child); math.Abs(w.Silo.FoodStock-want) > 1e-6 {
		t.Errorf("40 adults, 60 children: stock %v, want %v", w.Silo.FoodStock, want)
	}
}

// Outage length is ⌈outageMinDays / (average Mechanical worker skill ×
// partsShare)⌉, capped at outageMaxDays. Shadows don't count.
func TestOutageDays(t *testing.T) {
	cfg := baseline(t)
	rc := cfg.Resources
	cases := []struct {
		what   string
		skills []float64 // of Mechanical workers
		parts  float64
		want   int
	}{
		{"perfect skill, full parts", []float64{1, 1}, rc.PartsPerFailure, rc.OutageMinDays},
		{"usual skill", []float64{0.8, 0.9}, 100, 4}, // ⌈3 / 0.85⌉
		{"green crew", []float64{0.2, 0.8}, 100, 6},  // ⌈3 / 0.5⌉
		{"half the parts", []float64{1}, rc.PartsPerFailure / 2, 6},
		{"short on both", []float64{0.5}, rc.PartsPerFailure / 2, rc.OutageMaxDays}, // ⌈12⌉ → cap
		{"no parts", []float64{1}, 0, rc.OutageMaxDays},
		{"no workers", nil, 100, rc.OutageMaxDays},
	}
	for _, tc := range cases {
		w, m, _ := silo(t, cfg)
		for i, sk := range tc.skills {
			w.People[i].Stage, w.People[i].Department, w.People[i].Skill = world.Worker, mechanical, sk
		}
		// A low-skill shadow, ignored.
		w.People[99].Stage, w.People[99].Department, w.People[99].Skill = world.Shadow, mechanical, 0.01
		w.Silo.PartsStock = tc.parts
		if got := m.outageDays(w, mechanical); got != tc.want {
			t.Errorf("%s: %d days, want %d", tc.what, got, tc.want)
		}
	}
}

// zoned builds a silo with one household per zone: Up Top 10 adults,
// Mids 10 adults and 10 children, Down Deep 5 adults and 15 children, so
// the zones differ in size and in their share of children. Food stock is
// set to days × total need.
func zoned(t *testing.T, cfg *config.Scenario, days float64) (*world.World, *Module, [3]float64) {
	t.Helper()
	w := &world.World{
		Floors: []world.Floor{{Number: 1, Zone: world.UpTop}, {Number: 2, Zone: world.Mids}, {Number: 3, Zone: world.DownDeep}},
		Day:    world.Tick(30 * year),
	}
	adult, child := w.Day-world.Tick(30*year), w.Day-world.Tick(5*year)
	for z, mix := range [][2]int{{10, 0}, {10, 10}, {5, 15}} {
		w.Households = append(w.Households, world.Household{ID: world.HouseholdID(z), Floor: z + 1})
		for i := range mix[0] + mix[1] {
			id := world.PersonID(len(w.People))
			born := adult
			if i >= mix[0] {
				born = child
			}
			w.People = append(w.People, world.Person{ID: id, Alive: true, BirthDay: born, Household: world.HouseholdID(z)})
			w.Alive = append(w.Alive, id)
		}
	}
	for _, d := range cfg.Work.Departments {
		w.Depts = append(w.Depts, world.Department{Name: d.Name})
	}
	m := New()
	if err := m.Init(w, cfg, nil); err != nil {
		t.Fatal(err)
	}
	rc := cfg.Rations
	kid := rc.ChildNeedShare * rc.AdultKcal
	need := [3]float64{10 * rc.AdultKcal, 10*rc.AdultKcal + 10*kid, 5*rc.AdultKcal + 15*kid}
	w.Silo.FoodStock = days * (need[0] + need[1] + need[2])
	return w, m, need
}

func ration(t *testing.T, w *world.World, m *Module) {
	t.Helper()
	if err := m.rationsStep(w, nil); err != nil {
		t.Fatal(err)
	}
}

// With equal weights every zone gets the same intake/need, whatever its
// share of children: 1 with enough stock, rationingShare below
// rationingBelowDays of stock, and all that is left when the stock is
// smaller than the day's need.
func TestRationsEqualWeights(t *testing.T) {
	cfg := baseline(t)
	rc := cfg.Rations
	below := float64(rc.RationingBelowDays)
	cases := []struct {
		days      float64 // stock, in days of total need
		rationing bool
		intake    float64
	}{
		{below, false, 1},
		{below - 0.01, true, rc.RationingShare},
		{0.5, true, 0.5},
		{0, true, 0},
	}
	for _, tc := range cases {
		w, m, need := zoned(t, cfg, tc.days)
		total := need[0] + need[1] + need[2]
		stock := w.Silo.FoodStock
		ration(t, w, m)
		s := &w.Silo
		if s.Rationing != tc.rationing {
			t.Errorf("stock %v days: rationing = %v, want %v", tc.days, s.Rationing, tc.rationing)
		}
		for z, got := range s.Intake {
			if math.Abs(got-tc.intake) > 1e-12 {
				t.Errorf("stock %v days: zone %d intake/need = %v, want %v", tc.days, z, got, tc.intake)
			}
		}
		if want := tc.intake * total; math.Abs(s.FoodHandedOut-want) > 1e-6 || s.FoodStock != stock-s.FoodHandedOut {
			t.Errorf("stock %v days: handed out %v (want %v), stock %v → %v", tc.days, s.FoodHandedOut, want, stock, s.FoodStock)
		}
	}
}

// Favouring Up Top (weight 1.5) in a shortage: Up Top gets 1.5× the
// intake/need of the other zones, so Down Deep's hardship is higher.
func TestRationsFavourUpTop(t *testing.T) {
	cfg := baseline(t)
	cfg.Rations.Weights.UpTop = 1.5
	w, m, need := zoned(t, cfg, 10) // below 30 days: rationing
	ration(t, w, m)
	s := &w.Silo
	out := cfg.Rations.RationingShare * (need[0] + need[1] + need[2])
	weighted := 1.5*need[0] + need[1] + need[2]
	want := [3]float64{1.5 * out / weighted, out / weighted, out / weighted}
	for z := range want {
		if math.Abs(s.Intake[z]-want[z]) > 1e-12 {
			t.Errorf("zone %d intake/need = %v, want %v", z, s.Intake[z], want[z])
		}
	}
	upTop, downDeep := 1-s.Intake[world.UpTop], 1-s.Intake[world.DownDeep]
	t.Logf("hardship: Up Top %.3f, Down Deep %.3f", upTop, downDeep)
	if downDeep <= upTop {
		t.Errorf("Down Deep hardship %.3f not above Up Top's %.3f", downDeep, upTop)
	}
	if math.Abs(s.FoodHandedOut-out) > 1e-6 {
		t.Errorf("handed out %v, want all %v distributed", s.FoodHandedOut, out)
	}
}
