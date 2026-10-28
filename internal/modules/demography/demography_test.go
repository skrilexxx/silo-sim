package demography

import (
	"math"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/invariants"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const year = engine.DaysPerYear

func loadBaseline(t *testing.T) *config.Scenario {
	t.Helper()
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Output.InvariantEvery = 1
	return cfg
}

func stream(t *testing.T) *rng.Stream {
	t.Helper()
	r, err := rng.New(1, "demography")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// ODD §6.1, done-when: 100,000 lives at health 1, each simulated day by day
// from birth with one Bernoulli draw per day, have a median age at death of
// 65–69. The daily chances are precomputed from DeathChance, which is what
// the death step calls, only to keep the test to a few seconds.
func TestMedianAgeAtDeath(t *testing.T) {
	m := loadBaseline(t).Mortality
	var p []float64
	for d := 0; ; d++ {
		p = append(p, DeathChance(float64(d)/year, 1, m))
		if p[d] == 1 {
			break
		}
	}
	r := stream(t)
	ages := make([]float64, 100_000)
	for i := range ages {
		d := 0
		for !r.Bernoulli(p[d]) {
			d++
		}
		ages[i] = float64(d) / year
	}
	sort.Float64s(ages)
	median := ages[len(ages)/2]
	t.Logf("median age at death over %d lives: %.2f years (quartiles %.1f, %.1f)", len(ages), median, ages[len(ages)/4], ages[3*len(ages)/4])
	if median < 65 || median > 69 {
		t.Errorf("median age at death %.2f, want 65–69", median)
	}
}

// singles builds n living adults of the given age and health, each alone
// in a household on one Mids floor.
func singles(n int, ageYears, health float64) *world.World {
	w := &world.World{Floors: []world.Floor{{Number: 1, Zone: world.Mids, Homes: n}}}
	for i := range n {
		id := world.PersonID(i)
		w.People = append(w.People, world.Person{ID: id, Alive: true, BirthDay: world.Tick(-ageYears * year),
			Household: world.HouseholdID(i), Partner: world.None, Mother: world.None, Father: world.None, Health: health})
		w.Alive = append(w.Alive, id)
		w.Households = append(w.Households, world.Household{ID: world.HouseholdID(i), Floor: 1, Members: []world.PersonID{id}})
	}
	return w
}

// The death step, run through the engine for a year, kills people at the
// rate DeathChance gives, and poor health raises it as k_h says.
func TestDeathStepRate(t *testing.T) {
	cfg := loadBaseline(t)
	cfg.WarmupYears, cfg.Years = 0, 1
	const n = 20_000
	for _, health := range []float64{1, 0.5} {
		w := singles(n, 70, health)
		e, err := engine.New(cfg, w, []engine.Module{New()})
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Run(); err != nil {
			t.Fatal(err)
		}
		survive := 1.0
		for d := 1; d <= year; d++ {
			survive *= 1 - DeathChance(70+float64(d)/year, health, cfg.Mortality)
		}
		want := n * (1 - survive)
		got := float64(n - len(w.Alive))
		sd := math.Sqrt(n * survive * (1 - survive))
		t.Logf("health %.1f: %.0f deaths in a year among %d people aged 70, expected %.0f ± %.0f", health, got, n, want, sd)
		if math.Abs(got-want) > 4*sd {
			t.Errorf("health %.1f: %.0f deaths, expected %.0f ± %.0f", health, got, want, sd)
		}
	}
}

// family builds, on Mids floor 1: household 0 with a woman (0), a man (1)
// and two children (2, 3); household 1 with one adult (4). Up Top floor 2:
// household 2 with one adult (5).
func family(t *testing.T) (*world.World, *Module) {
	t.Helper()
	w := &world.World{Day: 1, Floors: []world.Floor{{Number: 1, Zone: world.Mids, Homes: 5}, {Number: 2, Zone: world.UpTop, Homes: 5}}}
	add := func(sex world.Sex, ageYears int, hh world.HouseholdID) world.PersonID {
		id := world.PersonID(len(w.People))
		w.People = append(w.People, world.Person{ID: id, Sex: sex, Alive: true, BirthDay: world.Tick(-ageYears * year),
			Household: hh, Partner: world.None, Mother: world.None, Father: world.None, Health: 1})
		w.Alive = append(w.Alive, id)
		return id
	}
	mother, father := add(world.Female, 35, 0), add(world.Male, 37, 0)
	kid1, kid2 := add(world.Female, 8, 0), add(world.Male, 4, 0)
	w.People[mother].Partner, w.People[father].Partner = father, mother
	for _, k := range []world.PersonID{kid1, kid2} {
		w.People[k].Mother, w.People[k].Father = mother, father
	}
	add(world.Male, 50, 1)
	add(world.Female, 60, 2)
	w.Households = []world.Household{
		{ID: 0, Floor: 1, Members: []world.PersonID{0, 1, 2, 3}},
		{ID: 1, Floor: 1, Members: []world.PersonID{4}},
		{ID: 2, Floor: 2, Members: []world.PersonID{5}},
	}
	m := New()
	if err := m.Init(w, loadBaseline(t), nil); err != nil {
		t.Fatal(err)
	}
	return w, m
}

// kill runs the death handling for the given people, then the step's
// cleanup of Alive, and checks the invariants.
func kill(t *testing.T, w *world.World, m *Module, ids ...world.PersonID) {
	t.Helper()
	for _, id := range ids {
		if err := m.die(w, stream(t), id); err != nil {
			t.Fatal(err)
		}
	}
	w.Alive = slices.DeleteFunc(w.Alive, func(id world.PersonID) bool { return !w.People[id].Alive })
	if err := invariants.Check(w); err != nil {
		t.Fatal(err)
	}
}

func TestPartnerFreedChildrenStay(t *testing.T) {
	w, m := family(t)
	kill(t, w, m, 0)
	if w.People[1].Partner != world.None {
		t.Error("widowed partner is still partnered")
	}
	if !slices.Equal(w.Households[0].Members, []world.PersonID{1, 2, 3}) {
		t.Errorf("household 0 is %v, want the father and both children", w.Households[0].Members)
	}
}

func TestStepParentKeepsChildren(t *testing.T) {
	w, m := family(t)
	w.People[2].Father, w.People[3].Father = world.None, world.None // 1 is their step-father
	kill(t, w, m, 0)
	if !slices.Equal(w.Households[0].Members, []world.PersonID{1, 2, 3}) {
		t.Errorf("household 0 is %v, want the step-father and both children", w.Households[0].Members)
	}
}

// Both parents die: the children move together to the only other Mids
// household with an adult (household 1), never to Up Top, and their old
// household dissolves.
func TestOrphansMoveTogetherInZone(t *testing.T) {
	w, m := family(t)
	kill(t, w, m, 0, 1)
	if !slices.Equal(w.Households[1].Members, []world.PersonID{4, 2, 3}) {
		t.Errorf("household 1 is %v, want its adult plus both children", w.Households[1].Members)
	}
	if h := w.Households[0]; len(h.Members) != 0 || h.Floor != world.None {
		t.Errorf("old household not dissolved: %+v", h)
	}
}

func TestOrphansWithNowhereToGoFail(t *testing.T) {
	w, m := family(t)
	kill(t, w, m, 4) // the other Mids adult; household 1 dissolves
	if w.Households[1].Floor != world.None {
		t.Error("single adult's household not dissolved")
	}
	if err := m.die(w, stream(t), 0); err != nil {
		t.Fatal(err)
	}
	err := m.die(w, stream(t), 1)
	if err == nil || !strings.Contains(err.Error(), "no household with an adult in zone 1") {
		t.Errorf("got %v, want an error about orphans with nowhere to go", err)
	}
}
