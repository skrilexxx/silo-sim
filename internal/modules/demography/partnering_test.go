package demography

import (
	"math"
	"slices"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/invariants"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// person describes one test person: sex, age in years, zone, department
// (world.None for none).
type person struct {
	sex  world.Sex
	age  int
	zone world.Zone
	dept int
}

// town builds a world with one residential floor per zone (10 homes each)
// and every person living alone, unless households is given.
func town(t *testing.T, people ...person) (*world.World, *Module) {
	t.Helper()
	w := &world.World{Day: 1, Depts: []world.Department{
		{Name: "farms", Zone: world.Mids}, {Name: "general", Zone: world.AllZones},
		{Name: "mechanical", Zone: world.DownDeep}, {Name: "it", Zone: world.UpTop},
	}}
	for z := range world.NumZones {
		w.Floors = append(w.Floors, world.Floor{Number: z + 1, Zone: world.Zone(z), Homes: 10})
	}
	for i, p := range people {
		id := world.PersonID(i)
		w.People = append(w.People, world.Person{ID: id, Sex: p.sex, Alive: true, BirthDay: world.Tick(-p.age * year),
			Household: world.HouseholdID(i), Partner: world.None, Mother: world.None, Father: world.None,
			Department: p.dept, Master: world.None, Health: 1})
		w.Alive = append(w.Alive, id)
		w.Households = append(w.Households, world.Household{ID: world.HouseholdID(i), Floor: int(p.zone) + 1, Members: []world.PersonID{id}})
	}
	m := New()
	if err := m.Init(w, loadBaseline(t), nil); err != nil {
		t.Fatal(err)
	}
	return w, m
}

// picks has person 0 choose a partner n times from everyone else and
// returns how often each person was chosen.
func picks(t *testing.T, w *world.World, m *Module, n int) []int {
	t.Helper()
	m.pool, m.poolZ = m.pool[:0], m.poolZ[:0]
	for _, id := range w.Alive {
		m.pool = append(m.pool, id)
		m.poolZ = append(m.poolZ, homeZone(w, id))
	}
	r := stream(t)
	count := make([]int, len(w.People))
	for range n {
		if j := m.choose(w, r, 0); j >= 0 {
			count[m.pool[j]]++
		}
	}
	return count
}

const draws = 100_000

func ratioNear(t *testing.T, what string, a, b int, want float64) {
	t.Helper()
	got := float64(a) / float64(b)
	t.Logf("%s: chosen %d vs %d times, ratio %.3f (expected %.3f)", what, a, b, got, want)
	if math.Abs(got-want) > 0.05*want {
		t.Errorf("%s: ratio %.3f, want %.3f ± 5%%", what, got, want)
	}
}

// ODD §6.3 done-when: with equal age gaps, a same-zone candidate is chosen
// about sameZoneWeight (4) times as often as a cross-zone one.
func TestSameZoneWeight(t *testing.T) {
	w, m := town(t,
		person{world.Female, 30, world.Mids, world.None},
		person{world.Male, 32, world.Mids, world.None},
		person{world.Male, 28, world.UpTop, world.None})
	c := picks(t, w, m, draws)
	ratioNear(t, "same zone vs other zone", c[1], c[2], 4)
}

func TestSameDepartmentWeight(t *testing.T) {
	w, m := town(t,
		person{world.Female, 30, world.Mids, 0},
		person{world.Male, 30, world.Mids, 0},
		person{world.Male, 30, world.Mids, 1})
	c := picks(t, w, m, draws)
	ratioNear(t, "same department vs other", c[1], c[2], 2)
}

func TestSameSexWeight(t *testing.T) {
	w, m := town(t,
		person{world.Female, 30, world.Mids, world.None},
		person{world.Female, 30, world.Mids, world.None},
		person{world.Male, 30, world.Mids, world.None})
	c := picks(t, w, m, draws)
	ratioNear(t, "same sex vs other sex", c[1], c[2], 0.05)
}

func TestAgeGapWeight(t *testing.T) {
	w, m := town(t,
		person{world.Female, 30, world.Mids, world.None},
		person{world.Male, 30, world.Mids, world.None},
		person{world.Male, 40, world.Mids, world.None})
	c := picks(t, w, m, draws)
	ratioNear(t, "0-year gap vs 10-year gap", c[1], c[2], math.Exp(10.0/5))
}

func TestRelativesAndPartneredNeverChosen(t *testing.T) {
	w, m := town(t,
		person{world.Female, 40, world.Mids, world.None}, // searcher
		person{world.Male, 60, world.Mids, world.None},   // her father
		person{world.Male, 20, world.Mids, world.None},   // her son
		person{world.Male, 38, world.Mids, world.None},   // half-brother: same father
		person{world.Male, 41, world.Mids, world.None},   // partnered
		person{world.Male, 45, world.Mids, world.None})   // the only free candidate
	w.People[0].Father = 1
	w.People[2].Mother = 0
	w.People[3].Father = 1
	w.People[4].Partner = 5 // partner fields are all that matters for the weight
	c := picks(t, w, m, 1000)
	if c[5] != 1000 {
		t.Errorf("chosen counts %v: only person 5 may be chosen", c)
	}
}

// housed runs the housing step once and checks the invariants.
func housed(t *testing.T, w *world.World, m *Module) {
	t.Helper()
	if err := m.housing(w, stream(t)); err != nil {
		t.Fatal(err)
	}
	if err := invariants.Check(w, m.cfg); err != nil {
		t.Fatal(err)
	}
}

func partner(w *world.World, a, b world.PersonID) {
	w.People[a].Partner, w.People[b].Partner = b, a
	w.HousingQueue = append(w.HousingQueue, world.HousingRequest{A: a, B: b})
}

func TestCoupleMovesToTiedZone(t *testing.T) {
	w, m := town(t,
		person{world.Female, 30, world.Mids, world.None},
		person{world.Male, 30, world.UpTop, 2}) // Mechanical: tied to the Down Deep
	partner(w, 0, 1)
	housed(t, w, m)
	h := w.Households[w.People[0].Household]
	if !slices.Equal(h.Members, []world.PersonID{0, 1}) || w.Floors[h.Floor-1].Zone != world.DownDeep {
		t.Errorf("couple lives in %+v, want both in the Down Deep", h)
	}
	if len(w.HousingQueue) != 0 || w.Households[0].Floor != world.None || w.Households[1].Floor != world.None {
		t.Error("queue not emptied or old households not dissolved")
	}
}

func TestTargetZoneRandomChoices(t *testing.T) {
	cases := []struct {
		what   string
		a, b   person
		zx, zy world.Zone
	}{
		{"both tied, different zones", person{world.Female, 30, world.Mids, 3}, person{world.Male, 30, world.Mids, 2}, world.UpTop, world.DownDeep},
		{"neither tied: one of their home zones", person{world.Female, 30, world.Mids, 1}, person{world.Male, 30, world.UpTop, world.None}, world.Mids, world.UpTop},
	}
	for _, c := range cases {
		w, m := town(t, c.a, c.b)
		r := stream(t)
		var n [world.NumZones]int
		for range 10_000 {
			n[m.targetZone(w, r, 0, 1)]++
		}
		t.Logf("%s: zone counts %v", c.what, n)
		if n[c.zx]+n[c.zy] != 10_000 || n[c.zx] < 4800 || n[c.zx] > 5200 {
			t.Errorf("%s: zone counts %v, want about 50/50 between %d and %d", c.what, n, c.zx, c.zy)
		}
	}
}

// One free home Up Top and two couples from the Mids who must live there
// (IT, tied to Up Top): the older couple gets it and the other waits,
// keeping its zone. A Mids couple behind them is not blocked.
func TestQueueOrderAndWaiting(t *testing.T) {
	w, m := town(t,
		person{world.Female, 30, world.Mids, 3}, person{world.Male, 30, world.Mids, 3},
		person{world.Female, 25, world.Mids, 3}, person{world.Male, 25, world.Mids, 3},
		person{world.Female, 35, world.Mids, 0}, person{world.Male, 35, world.Mids, 0})
	w.Floors[world.UpTop].Homes = 1
	partner(w, 0, 1)
	partner(w, 2, 3)
	partner(w, 4, 5)
	housed(t, w, m)
	if w.People[0].Household != w.People[1].Household || homeZone(w, 0) != world.UpTop {
		t.Error("oldest couple was not housed Up Top")
	}
	if w.People[2].Household == w.People[3].Household {
		t.Error("second Up Top couple was housed, but only one home was free")
	}
	if w.People[4].Household != w.People[5].Household {
		t.Error("Mids couple was blocked by the Up Top queue")
	}
	if len(w.HousingQueue) != 1 || w.HousingQueue[0].A != 2 || !w.HousingQueue[0].ZoneSet || w.HousingQueue[0].Zone != world.UpTop {
		t.Errorf("queue = %+v, want only couple 2–3, waiting for Up Top", w.HousingQueue)
	}
	housed(t, w, m)
	if len(w.HousingQueue) != 1 {
		t.Error("waiting couple housed with no free home")
	}
	w.Floors[world.UpTop].Homes = 2
	housed(t, w, m)
	if w.People[2].Household != w.People[3].Household || len(w.HousingQueue) != 0 {
		t.Error("waiting couple not housed once a home was free")
	}
}

// Homes freed by one couple's move are usable by the next couple in the
// same month.
func TestFreedHomeUsedAtOnce(t *testing.T) {
	w, m := town(t,
		person{world.Female, 30, world.UpTop, 3}, person{world.Male, 30, world.Mids, 3},
		person{world.Female, 25, world.Mids, 3}, person{world.Male, 25, world.Mids, 3})
	w.Floors[world.UpTop].Homes = 1 // full: person 0 lives there
	partner(w, 0, 1)                // must live Up Top (IT), but no home is free
	housed(t, w, m)
	if len(w.HousingQueue) != 1 {
		t.Fatalf("queue = %+v; with no free home, couple 0–1 should wait", w.HousingQueue)
	}
	w.Floors[world.UpTop].Homes = 2 // one free home now
	partner(w, 2, 3)
	housed(t, w, m)
	// Couple 0–1 takes the free home and frees person 0's old one, which
	// couple 2–3 then takes in the same month.
	if len(w.HousingQueue) != 0 || homeZone(w, 2) != world.UpTop || homeZone(w, 0) != world.UpTop {
		t.Errorf("queue = %+v; both couples should be Up Top", w.HousingQueue)
	}
}

func TestBrokenCoupleLeavesQueue(t *testing.T) {
	w, m := town(t, person{world.Female, 30, world.Mids, 0}, person{world.Male, 30, world.Mids, 0})
	partner(w, 0, 1)
	if err := m.die(w, stream(t), 1); err != nil {
		t.Fatal(err)
	}
	w.Alive = w.Alive[:1]
	housed(t, w, m)
	if len(w.HousingQueue) != 0 {
		t.Error("couple broken by a death is still queued")
	}
}

// A widow who is the only adult brings every child, step-children too; a
// young adult leaving his parents' home leaves his little sister there.
func TestWhoMovesIn(t *testing.T) {
	w, m := town(t,
		person{world.Female, 35, world.Mids, 0}, // 0 widow
		person{world.Male, 8, world.Mids, 0},    // 1 her son
		person{world.Female, 6, world.Mids, 0},  // 2 her step-daughter
		person{world.Male, 19, world.Mids, 0},   // 3 young adult
		person{world.Female, 10, world.Mids, 0}, // 4 his sister
		person{world.Female, 45, world.Mids, 0}, // 5 their mother
		person{world.Female, 20, world.Mids, 0}, // 6 his new partner
		person{world.Male, 40, world.Mids, 0})   // 7 widow's new partner
	for _, x := range []world.PersonID{0, 1, 2, 3, 4, 5, 6, 7} {
		w.People[x].Department = 0
	}
	w.People[1].Mother = 0
	w.People[3].Mother, w.People[4].Mother = 5, 5
	w.Households = []world.Household{
		{ID: 0, Floor: 2, Members: []world.PersonID{0, 1, 2}},
		{ID: 1, Floor: 2, Members: []world.PersonID{3, 4, 5}},
		{ID: 2, Floor: 2, Members: []world.PersonID{6}},
		{ID: 3, Floor: 2, Members: []world.PersonID{7}},
	}
	for _, h := range w.Households {
		for _, x := range h.Members {
			w.People[x].Household = h.ID
		}
	}
	partner(w, 0, 7)
	partner(w, 3, 6)
	housed(t, w, m)

	widow := w.Households[w.People[0].Household]
	if !slices.Equal(widow.Members, []world.PersonID{0, 1, 2, 7}) {
		t.Errorf("widow's new household %v, want her, both children and her partner", widow.Members)
	}
	if !slices.Equal(w.Households[1].Members, []world.PersonID{4, 5}) {
		t.Errorf("parents' household %v, want the mother and the sister left behind", w.Households[1].Members)
	}
}
