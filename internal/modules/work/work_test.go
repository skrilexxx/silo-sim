package work

import (
	"math"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/invariants"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const year = engine.DaysPerYear

// Department indexes in the baseline table.
const (
	farms, general, mechanical, supply, judicial, porters, education, medical, it, administration, sheriff = 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10
)

func baseline(t *testing.T) *config.Scenario {
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
	r, err := rng.New(1, "work")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// tiny is a hand-built world: one floor per zone, the baseline department
// table, and people added with add. The module is set up without the
// day-1 job assignment, which tests of Init cover separately.
type tiny struct {
	t *testing.T
	w *world.World
	m *Module
}

func newTiny(t *testing.T) *tiny {
	cfg := baseline(t)
	w := &world.World{Day: 10 * year}
	for z := range world.NumZones {
		w.Floors = append(w.Floors, world.Floor{Number: z + 1, Zone: world.Zone(z), Homes: 100})
	}
	m := New()
	m.cfg, m.c = cfg, cfg.Work
	for _, d := range m.c.Departments {
		w.Depts = append(w.Depts, world.Department{Name: d.Name, Zone: parseZone(d.Zone), TargetShare: d.Share})
		m.need = append(m.need, 1) // tests of coverage set their own
	}
	m.general = general
	return &tiny{t: t, w: w, m: m}
}

// add puts a person aged ageDays in their own household in zone z.
func (tw *tiny) add(stage world.Stage, ageDays world.Tick, z world.Zone, dept int, skill float64) world.PersonID {
	w := tw.w
	id := world.PersonID(len(w.People))
	hh := world.HouseholdID(len(w.Households))
	w.People = append(w.People, world.Person{ID: id, Alive: true, BirthDay: w.Day - ageDays, Household: hh,
		Partner: world.None, Mother: world.None, Father: world.None, BabyFather: world.None,
		Stage: stage, Department: dept, Master: world.None, Skill: skill, Health: 1})
	w.Alive = append(w.Alive, id)
	w.Households = append(w.Households, world.Household{ID: hh, Floor: int(z) + 1, Members: []world.PersonID{id}})
	return id
}

// master adds a skilled worker of department d in zone z.
func (tw *tiny) master(d int, z world.Zone) world.PersonID {
	return tw.add(world.Worker, 30*year, z, d, 0.9)
}

func (tw *tiny) check() {
	tw.t.Helper()
	if err := invariants.Check(tw.w, tw.m.cfg); err != nil {
		tw.t.Fatal(err)
	}
}

func (tw *tiny) day() {
	tw.t.Helper()
	tw.w.Day++
	if err := tw.m.lifeStages(tw.w, stream(tw.t)); err != nil {
		tw.t.Fatal(err)
	}
	tw.check()
}

// staffed adds a master to every department so each has room, then counts.
func (tw *tiny) staffed() {
	for d, dep := range tw.w.Depts {
		z := dep.Zone
		if z == world.AllZones {
			z = world.Mids
		}
		tw.master(d, z)
	}
	if err := tw.m.staffing(tw.w, stream(tw.t)); err != nil {
		tw.t.Fatal(err)
	}
}

func TestSkillGrowth(t *testing.T) {
	tw := newTiny(t)
	id := tw.add(world.Worker, 30*year, world.Mids, farms, 0.5)
	kid := tw.add(world.Child, 5*year, world.Mids, world.None, 0)
	tw.m.output(tw.w, true)
	alpha := tw.m.c.SkillGrowth
	if got, want := tw.w.People[id].Skill, 0.5+alpha*0.5; math.Abs(got-want) > 1e-15 {
		t.Errorf("skill after a day %v, want %v", got, want)
	}
	if tw.w.People[kid].Skill != 0 {
		t.Error("a child's skill grew")
	}
	// ODD §6.5: from 0, about 0.8 after 10 years of work.
	tw.w.People[id].Skill = 0
	for range 10 * year {
		tw.m.output(tw.w, true)
	}
	t.Logf("skill after 10 years from 0: %.3f", tw.w.People[id].Skill)
	if s := tw.w.People[id].Skill; s < 0.79 || s > 0.81 {
		t.Errorf("skill after 10 years %.3f, want about 0.8", s)
	}
}

func TestStageTransitions(t *testing.T) {
	tw := newTiny(t)
	tw.staffed()
	c := tw.m.c
	kid := tw.add(world.Child, world.Tick(c.ShadowAge*year)-1, world.Mids, world.None, 0)
	sh := tw.add(world.Shadow, world.Tick(c.WorkerAge*year)-1, world.Mids, farms, 0.3)
	tw.w.People[sh].Master = 0 // the farms master
	old := tw.add(world.Worker, world.Tick(c.RetireAge*year)-1, world.Mids, supply, 0.9)
	tw.m.recount(tw.w)
	tw.check()

	tw.day()
	if p := tw.w.People[kid]; p.Stage != world.Shadow || p.Department == world.None {
		t.Errorf("child on their 14th birthday: stage %d, department %d", p.Stage, p.Department)
	}
	if p := tw.w.People[sh]; p.Stage != world.Worker || p.Department != farms || p.Master != world.None {
		t.Errorf("shadow on their 18th birthday: %+v", p)
	}
	if p := tw.w.People[old]; p.Stage != world.Retired || p.Department != supply {
		t.Errorf("worker on their 65th birthday: stage %d, department %d (kept as a record)", p.Stage, p.Department)
	}
	if tw.w.Depts[supply].Headcount != 1 || tw.w.Depts[farms].Headcount != 2 {
		t.Errorf("headcounts supply %d, farms %d after the changes", tw.w.Depts[supply].Headcount, tw.w.Depts[farms].Headcount)
	}
}

// ODD §6.5: sickLeaveDays in a row below sickLeaveHealth puts a worker on
// sick leave (off the headcount, department kept); health back at the mark
// returns them to work.
func TestSickLeave(t *testing.T) {
	tw := newTiny(t)
	tw.staffed()
	sick := tw.add(world.Worker, 40*year, world.Mids, farms, 0.9)
	tw.m.recount(tw.w)
	c := tw.m.c
	tw.w.People[sick].Health = c.SickLeaveHealth / 2
	for range c.SickLeaveDays - 1 {
		tw.day()
	}
	if tw.w.People[sick].Stage != world.Worker {
		t.Fatalf("on leave after %d days of low health, want %d", c.SickLeaveDays-1, c.SickLeaveDays)
	}
	tw.w.People[sick].Health = 1 // one good day resets the count
	tw.day()
	tw.w.People[sick].Health = c.SickLeaveHealth / 2
	for range c.SickLeaveDays - 1 {
		tw.day()
	}
	if tw.w.People[sick].Stage != world.Worker {
		t.Fatal("the count did not restart after a day of good health")
	}
	before := tw.w.Depts[farms].Headcount
	tw.day()
	if p := tw.w.People[sick]; p.Stage != world.SickLeave || p.Department != farms || tw.w.Depts[farms].Headcount != before-1 {
		t.Fatalf("after %d days in a row of low health: stage %d, department %d, headcount %d → %d; want sick leave from farms, one fewer",
			c.SickLeaveDays, p.Stage, p.Department, before, tw.w.Depts[farms].Headcount)
	}
	tw.day() // still sick: stays on leave
	if tw.w.People[sick].Stage != world.SickLeave {
		t.Fatal("left sick leave while still below the mark")
	}
	tw.w.People[sick].Health = c.SickLeaveHealth
	tw.day()
	if p := tw.w.People[sick]; p.Stage != world.Worker || p.Department != farms || tw.w.Depts[farms].Headcount != before {
		t.Errorf("health back at %v: stage %d, headcount %d; want back at work in farms", c.SickLeaveHealth, p.Stage, tw.w.Depts[farms].Headcount)
	}
}

// Someone on sick leave who reaches retireAge retires instead of
// returning.
func TestSickLeaveRetiresAtAge(t *testing.T) {
	tw := newTiny(t)
	tw.staffed()
	sick := tw.add(world.Worker, 64*year, world.Mids, farms, 0.9)
	tw.w.People[sick].Stage, tw.w.People[sick].Health = world.SickLeave, tw.m.c.SickLeaveHealth/2
	tw.m.recount(tw.w)
	for range year {
		tw.day()
		if tw.w.People[sick].Stage == world.Worker {
			t.Fatal("returned to work while still sick")
		}
	}
	if p := tw.w.People[sick]; p.Stage != world.Retired {
		t.Errorf("stage %d at 65, want retired", p.Stage)
	}
}

// placeMany places the same child n times, resetting in between, and
// returns how often each department was picked.
func placeMany(tw *tiny, kid world.PersonID, n int) []int {
	r := stream(tw.t)
	counts := make([]int, len(tw.w.Depts))
	for range n {
		p := &tw.w.People[kid]
		p.Stage, p.Department, p.Master = world.Child, world.None, world.None
		tw.m.recount(tw.w)
		tw.m.place(tw.w, r, kid)
		counts[tw.w.People[kid].Department]++
	}
	return counts
}

// With every shortfall 0 the weights fall back to target shares, so a Mids
// child whose mother is in Mechanical (Down Deep) shadows Mechanical with
// chance 0.6 (follow) + 0.4 × 0.05 (cross-zone) × 0.16/0.39 (Mechanical's
// share of the other-zone departments) = 0.6082.
func TestFollowParent(t *testing.T) {
	tw := newTiny(t)
	tw.staffed()
	tw.m.target = make([]float64, len(tw.w.Depts))
	mother := tw.add(world.Worker, 40*year, world.DownDeep, mechanical, 0.9)
	kid := tw.add(world.Child, 14*year, world.Mids, world.None, 0)
	tw.w.People[kid].Mother = mother
	const n = 20_000
	got := float64(placeMany(tw, kid, n)[mechanical]) / n
	want := 0.6 + 0.4*0.05*0.16/0.39
	t.Logf("followed the mother's department %.4f of %d placements (expected %.4f)", got, n, want)
	if math.Abs(got-want) > 0.015 {
		t.Errorf("followed %.4f, want %.4f", got, want)
	}
}

// A Mids child with no parents leaves their zone's departments with chance
// crossZoneChance (5%).
func TestCrossZoneShare(t *testing.T) {
	tw := newTiny(t)
	tw.staffed()
	tw.m.target = make([]float64, len(tw.w.Depts))
	kid := tw.add(world.Child, 14*year, world.Mids, world.None, 0)
	const n = 20_000
	counts := placeMany(tw, kid, n)
	out := 0
	for d, c := range counts {
		if z := tw.w.Depts[d].Zone; z != world.Mids && z != world.AllZones {
			out += c
		}
	}
	share := float64(out) / n
	t.Logf("placed in another zone's department: %.4f (crossZoneChance %.2f)", share, tw.m.c.CrossZoneChance)
	if math.Abs(share-tw.m.c.CrossZoneChance) > 0.006 {
		t.Errorf("cross-zone share %.4f, want %.2f", share, tw.m.c.CrossZoneChance)
	}
}

// Own-zone picks follow the shortfall: farms short by 30 and supply by 10
// are picked 3:1, and departments with no shortfall never.
func TestShortfallWeights(t *testing.T) {
	tw := newTiny(t)
	tw.staffed()
	tw.m.c.CrossZoneChance = 0
	tw.m.target = make([]float64, len(tw.w.Depts))
	// Live counts: farms and supply each have 1 worker (their master).
	tw.m.target[farms], tw.m.target[supply] = 31, 11 // short by 30 and 10
	tw.m.target[mechanical] = 51                     // another zone: out of reach
	kid := tw.add(world.Child, 14*year, world.Mids, world.None, 0)
	counts := placeMany(tw, kid, 20_000)
	ratio := float64(counts[farms]) / float64(counts[supply])
	t.Logf("farms vs supply picked %d vs %d, ratio %.3f (expected 3)", counts[farms], counts[supply], ratio)
	if math.Abs(ratio-3) > 0.15 || counts[farms]+counts[supply] != 20_000 {
		t.Errorf("counts %v: want only farms and supply, 3:1", counts)
	}
}

func TestStaffingShortfall(t *testing.T) {
	tw := newTiny(t)
	for range 7 {
		tw.add(world.Worker, 30*year, world.Mids, farms, 0.9)
	}
	for range 3 {
		tw.add(world.Worker, 30*year, world.Mids, general, 0.9)
	}
	tw.add(world.Shadow, 15*year, world.Mids, general, 0.1)
	if err := tw.m.staffing(tw.w, stream(tw.t)); err != nil {
		t.Fatal(err)
	}
	// 10 workers: farms target 3.0 (has 7), general 2.0 (has 3 + 1 shadow),
	// mechanical 1.6 (has 0).
	got := []float64{tw.m.shortfall(tw.w, farms), tw.m.shortfall(tw.w, general), tw.m.shortfall(tw.w, mechanical)}
	if got[0] != 0 || got[1] != 0 || math.Abs(got[2]-1.6) > 1e-12 {
		t.Errorf("shortfalls farms, general, mechanical = %v, want 0, 0, 1.6", got)
	}
	// A placement shrinks the gap at once, before the next monthly count.
	w := tw.w
	w.Depts[mechanical].Shadows++
	if s := tw.m.shortfall(w, mechanical); math.Abs(s-0.6) > 1e-12 {
		t.Errorf("mechanical shortfall %v after one more shadow, want 0.6", s)
	}
}

// A parent's department with no free master is not followed; departments
// without room are skipped; with no room anywhere in reach the child is
// placed without a master.
func TestRoom(t *testing.T) {
	tw := newTiny(t)
	tw.m.target = make([]float64, len(tw.w.Depts))
	tw.m.c.CrossZoneChance = 0
	tw.m.c.FollowParentChance = 1
	mother := tw.add(world.Worker, 40*year, world.Mids, farms, 0.4) // not skilled enough to be a master
	tw.master(supply, world.Mids)                                   // the only free master in reach
	kid := tw.add(world.Child, 14*year, world.Mids, world.None, 0)
	tw.w.People[kid].Mother = mother
	counts := placeMany(tw, kid, 1000)
	if counts[supply] != 1000 {
		t.Errorf("counts %v: only supply has room", counts)
	}
	if tw.w.People[kid].Master != 1 {
		t.Errorf("master %d, want the supply worker", tw.w.People[kid].Master)
	}

	tw.w.People[1].Skill = 0.4 // now nobody can be a master
	counts = placeMany(tw, kid, 1000)
	if counts[farms]+counts[supply]+counts[general]+counts[porters] != 1000 || tw.w.People[kid].Master != world.None {
		t.Errorf("with no room anywhere: counts %v, master %d; want an own-zone department and no master", counts, tw.w.People[kid].Master)
	}
	tw.check()
}

func TestMasterHasFewestShadows(t *testing.T) {
	tw := newTiny(t)
	busy, free := tw.master(farms, world.Mids), tw.master(farms, world.Mids)
	sh := tw.add(world.Shadow, 15*year, world.Mids, farms, 0.1)
	tw.w.People[sh].Master = busy
	tw.m.recount(tw.w)
	r := stream(t)
	for range 100 {
		if got := tw.m.freeMaster(tw.w, r, farms); got != free {
			t.Fatalf("picked master %d, want %d (fewest shadows)", got, free)
		}
	}
	// Full masters are never picked.
	tw.w.People[sh].Master = free
	sh2 := tw.add(world.Shadow, 15*year, world.Mids, farms, 0.1)
	tw.w.People[sh2].Master = free
	tw.w.People[busy].Skill = 0.1
	tw.m.recount(tw.w)
	if got := tw.m.freeMaster(tw.w, r, farms); got != world.None {
		t.Errorf("picked master %d, but %d is full and %d unskilled", got, free, busy)
	}
}

// A shadow whose master dies gets the free master with the fewest shadows
// the next day; with none free, they go on without one.
func TestRemaster(t *testing.T) {
	tw := newTiny(t)
	a, b := tw.master(farms, world.Mids), tw.master(farms, world.Mids)
	sh := tw.add(world.Shadow, 15*year, world.Mids, farms, 0.1)
	tw.w.People[sh].Master = a
	tw.m.recount(tw.w)
	tw.check()

	tw.w.People[a].Alive = false
	tw.w.Alive = []world.PersonID{b, sh}
	tw.w.Households[a].Members = nil
	tw.day()
	if tw.w.People[sh].Master != b {
		t.Errorf("master %d after the old one died, want %d", tw.w.People[sh].Master, b)
	}
	tw.w.People[b].Alive = false
	tw.w.Alive = []world.PersonID{sh}
	tw.w.Households[b].Members = nil
	tw.day()
	if tw.w.People[sh].Master != world.None {
		t.Errorf("master %d with no worker left, want none", tw.w.People[sh].Master)
	}
}
