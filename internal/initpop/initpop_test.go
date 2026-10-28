package initpop

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/invariants"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

func baseline(t *testing.T) *config.Scenario {
	t.Helper()
	cfg, err := config.Load("../../scenarios/baseline.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func build(t *testing.T, cfg *config.Scenario, seed uint64) *world.World {
	t.Helper()
	r, err := rng.New(seed, "initpop")
	if err != nil {
		t.Fatal(err)
	}
	w, err := Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func age(p *world.Person) int { return int(-p.BirthDay) }

func zoneOf(w *world.World, p *world.Person) world.Zone {
	return w.Floors[w.Households[p.Household].Floor-1].Zone
}

func TestInvariantsHoldOnDay1(t *testing.T) {
	cfg := baseline(t)
	w := build(t, cfg, 42)
	if err := invariants.Check(w, cfg); err != nil {
		t.Fatalf("day 0: %v", err)
	}
	cfg.WarmupYears, cfg.Years = 0, 1
	cfg.Output.InvariantEvery = 1
	e, err := engine.New(cfg, w, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Tick(); err != nil {
		t.Fatalf("day 1: %v", err)
	}
	if len(w.People) != 10000 || len(w.Alive) != 10000 {
		t.Errorf("%d people, %d alive, want 10000", len(w.People), len(w.Alive))
	}
}

func TestFloors(t *testing.T) {
	cfg := baseline(t)
	w := build(t, cfg, 42)
	if len(w.Floors) != 144 {
		t.Fatalf("%d floors, want 144", len(w.Floors))
	}
	var farms, mech [world.NumZones]int
	for i, f := range w.Floors {
		if f.Number != i+1 || f.Zone != world.Zone(i/48) {
			t.Fatalf("floor %d: number %d zone %d", i+1, f.Number, f.Zone)
		}
		switch f.Function {
		case world.Farm:
			farms[f.Zone]++
		case world.Mechanical:
			mech[f.Zone]++
			if f.Number <= 144-12 {
				t.Errorf("Mechanical floor %d is not among the lowest 12", f.Number)
			}
		}
		if f.Function != world.Residential && f.Homes != 0 {
			t.Errorf("work floor %d has %d homes", f.Number, f.Homes)
		}
	}
	if farms != [3]int{0, 20, 0} || mech != [3]int{0, 0, 12} {
		t.Errorf("farm floors per zone %v, Mechanical %v", farms, mech)
	}
}

func TestHomes(t *testing.T) {
	w := build(t, baseline(t), 42)
	var households, homes [world.NumZones]int
	used := make([]int, len(w.Floors))
	for _, h := range w.Households {
		f := w.Floors[h.Floor-1]
		if f.Function != world.Residential {
			t.Fatalf("household %d lives on work floor %d", h.ID, f.Number)
		}
		households[f.Zone]++
		used[h.Floor-1]++
	}
	minHomes, maxHomes := [3]int{1 << 30, 1 << 30, 1 << 30}, [3]int{}
	for i, f := range w.Floors {
		if used[i] > f.Homes {
			t.Errorf("floor %d: %d households, %d homes", f.Number, used[i], f.Homes)
		}
		if f.Function == world.Residential {
			homes[f.Zone] += f.Homes
			minHomes[f.Zone] = min(minHomes[f.Zone], f.Homes)
			maxHomes[f.Zone] = max(maxHomes[f.Zone], f.Homes)
		}
	}
	for z := range world.NumZones {
		if want := int(math.Ceil(1.15 * float64(households[z]))); homes[z] != want {
			t.Errorf("zone %d: %d homes for %d households, want %d", z, homes[z], households[z], want)
		}
		if maxHomes[z]-minHomes[z] > 1 {
			t.Errorf("zone %d: homes per floor range %d–%d, want an even spread", z, minHomes[z], maxHomes[z])
		}
	}
}

func TestZoneSplitAndSexes(t *testing.T) {
	cfg := baseline(t)
	w := build(t, cfg, 42)
	var count [world.NumZones]int
	women := 0
	for i := range w.People {
		count[zoneOf(w, &w.People[i])]++
		if w.People[i].Sex == world.Female {
			women++
		}
	}
	want := [3]float64{cfg.Init.ZoneSplit.UpTop, cfg.Init.ZoneSplit.Mids, cfg.Init.ZoneSplit.DownDeep}
	for z := range world.NumZones {
		got := float64(count[z]) / float64(len(w.People))
		t.Logf("zone %d: %.4f (config %.2f)", z, got, want[z])
		if math.Abs(got-want[z]) > 0.01 {
			t.Errorf("zone %d share %.4f, config %.2f", z, got, want[z])
		}
	}
	if women != 5000 {
		t.Errorf("%d women, want 5000", women)
	}
}

// ODD §5 step 2: ages follow the steady-state distribution of the mortality
// rule. Each 5-year band's share must be within 2 percentage points of it.
func TestAgesMatchStationaryDistribution(t *testing.T) {
	cfg := baseline(t)
	w := build(t, cfg, 42)
	cum := stationaryCum(cfg.Mortality)
	total := cum[len(cum)-1]
	band := 5 * engine.DaysPerYear

	expect := map[int]float64{}
	prev := 0.0
	for d, c := range cum {
		expect[d/band] += (c - prev) / total
		prev = c
	}
	got := map[int]float64{}
	for i := range w.People {
		got[age(&w.People[i])/band] += 1 / float64(len(w.People))
	}
	worst := 0.0
	for b := 0; b <= len(cum)/band; b++ {
		diff := math.Abs(got[b] - expect[b])
		worst = max(worst, diff)
		if diff > 0.02 {
			t.Errorf("ages %d–%d: %.3f of people, expected %.3f", 5*b, 5*b+4, got[b], expect[b])
		}
	}
	t.Logf("largest 5-year band difference: %.4f (limit 0.02)", worst)
}

func TestCouplesAndChildren(t *testing.T) {
	cfg := baseline(t)
	w := build(t, cfg, 42)
	lot := cfg.Lottery
	year := engine.DaysPerYear

	var adults20, paired [world.NumZones]int
	for i := range w.People {
		p := &w.People[i]
		z := zoneOf(w, p)
		if age(p) >= cfg.Init.PairedMinAge*year {
			adults20[z]++
		}
		if p.Partner != world.None {
			q := &w.People[p.Partner]
			if q.Partner != p.ID || q.Sex == p.Sex || q.Household != p.Household {
				t.Fatalf("couple %d–%d is not a symmetric woman–man pair in one household", p.ID, q.ID)
			}
			if age(p) < cfg.Init.PairedMinAge*year {
				t.Errorf("person %d paired at %d days", p.ID, age(p))
			}
			paired[z]++
		}
	}
	for z := range world.NumZones {
		share := float64(paired[z]) / float64(adults20[z])
		t.Logf("zone %d: %.3f of adults 20+ paired", z, share)
		if math.Abs(share-cfg.Init.PairedShareOfAdults) > 0.01 {
			t.Errorf("zone %d: %.3f of adults 20+ paired, want %.2f", z, share, cfg.Init.PairedShareOfAdults)
		}
	}

	kidsOf := map[world.PersonID][]int{}
	minAtBirth, maxAtBirth := math.Inf(1), 0.0
	for i := range w.People {
		p := &w.People[i]
		if age(p) >= cfg.Partnering.MinAge*year {
			if p.Mother != world.None {
				t.Errorf("adult %d has a mother on day 1", p.ID)
			}
			continue
		}
		if p.Mother == world.None {
			t.Fatalf("child %d has no mother", p.ID)
		}
		m := &w.People[p.Mother]
		if m.Sex != world.Female || p.Father != m.Partner || p.Household != m.Household {
			t.Errorf("child %d: mother %d, father %d, household %d do not match the couple", p.ID, p.Mother, p.Father, p.Household)
		}
		atBirth := float64(age(m)-age(p)) / float64(year)
		minAtBirth, maxAtBirth = min(minAtBirth, atBirth), max(maxAtBirth, atBirth)
		if atBirth < float64(lot.MotherAgeMin) || atBirth > float64(lot.MotherAgeMax) {
			t.Errorf("child %d: mother was %.2f at the birth", p.ID, atBirth)
		}
		kidsOf[p.Mother] = append(kidsOf[p.Mother], age(p))
	}
	t.Logf("mother's age at birth ranges %.2f–%.2f (rule %d–%d)", minAtBirth, maxAtBirth, lot.MotherAgeMin, lot.MotherAgeMax)
	for m, ages := range kidsOf {
		if len(ages) != w.People[m].Children || len(ages) > lot.MaxChildren {
			t.Errorf("mother %d: %d children placed, count says %d", m, len(ages), w.People[m].Children)
		}
		for i := range ages {
			for j := i + 1; j < len(ages); j++ {
				if d := ages[i] - ages[j]; d < lot.MinYoungestAgeYears*year && -d < lot.MinYoungestAgeYears*year {
					t.Errorf("mother %d: children %d days apart", m, d)
				}
			}
		}
	}

	for _, h := range w.Households {
		if len(h.Members) == 1 && w.People[h.Members[0]].Partner != world.None {
			t.Errorf("household %d: a partnered person lives alone", h.ID)
		}
	}
}

// Fail loudly: with no couples there is no one to raise the children.
func TestNoFittingCoupleFails(t *testing.T) {
	cfg := baseline(t)
	cfg.Init.PairedShareOfAdults = 0
	r, _ := rng.New(42, "initpop")
	_, err := Build(cfg, r)
	if err == nil || !strings.Contains(err.Error(), "can be the parents of person") {
		t.Errorf("got %v, want an error about a child with no fitting couple", err)
	}
}

// At the baseline size every seed must build (none of 300 failed when this
// was measured). Small worlds can genuinely run out of couples that fit a
// child (about 7% of seeds at 1,000 people), so building fails loudly
// there and tests with small worlds use seeds that work; the golden run
// (1,000 people, seed 1) is one.
func TestBuildsForManySeeds(t *testing.T) {
	cfg := baseline(t)
	runs := []struct {
		pop   int
		seeds []uint64
	}{
		{10000, []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}},
		{1000, []uint64{1}},
	}
	for _, run := range runs {
		cfg.Population = run.pop
		for _, seed := range run.seeds {
			r, _ := rng.New(seed, "initpop")
			w, err := Build(cfg, r)
			if err != nil {
				t.Fatalf("population %d, seed %d: %v", run.pop, seed, err)
			}
			if err := invariants.Check(w, cfg); err != nil {
				t.Fatalf("population %d, seed %d: %v", run.pop, seed, err)
			}
		}
	}
}

func TestSameSeedSameWorld(t *testing.T) {
	cfg := baseline(t)
	if !reflect.DeepEqual(build(t, cfg, 7), build(t, cfg, 7)) {
		t.Error("same seed built different worlds")
	}
}
