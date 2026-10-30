package network

import (
	"slices"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/invariants"
	"github.com/skrilexxx/silo-sim/internal/modules/space"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const year = engine.DaysPerYear

// Departments of the test silo.
const (
	farms = iota
	general
	mechanical
	it
	porters
)

// silo is a 7-floor test world: 1–3 homes Up Top, 4 farms and 5 homes in
// the Mids, 6 homes and 7 Mechanical Down Deep; IT works on floor 1.
// k = 2. People are added with add, then start runs space and network.
type silo struct {
	t     *testing.T
	w     *world.World
	cfg   *config.Scenario
	sp    engine.Module
	m     *Module
	r     *rng.Stream
	spRun func(*world.World, *rng.Stream) error
}

func newSilo(t *testing.T) *silo {
	w := &world.World{
		Day: 100 * year,
		Floors: []world.Floor{
			{Number: 1, Zone: world.UpTop, Homes: 9}, {Number: 2, Zone: world.UpTop, Homes: 9}, {Number: 3, Zone: world.UpTop, Homes: 9},
			{Number: 4, Zone: world.Mids, Function: world.Farm}, {Number: 5, Zone: world.Mids, Homes: 9},
			{Number: 6, Zone: world.DownDeep, Homes: 9}, {Number: 7, Zone: world.DownDeep, Function: world.Mechanical},
		},
		Depts: []world.Department{
			{Name: "farms", Zone: world.Mids}, {Name: "general", Zone: world.AllZones},
			{Name: "mechanical", Zone: world.DownDeep}, {Name: "it", Zone: world.UpTop},
			{Name: "porters", Zone: world.AllZones},
		},
	}
	cfg := &config.Scenario{}
	cfg.Space.OfficeFloors = map[string][]int{"it": {1}}
	cfg.Space.FullDescentHours, cfg.Space.WalkingHoursPerDay = 12, 8
	cfg.Work.ShadowAge, cfg.Work.WorkerAge, cfg.Work.MaxShadowsPerMaster = 14, 18, 2
	cfg.Network.WeakTiesPerKind = 2
	cfg.Output.InvariantEvery = 1
	r, _ := rng.New(1, "network")
	return &silo{t: t, w: w, cfg: cfg, r: r}
}

// add puts a person of the given age in years, stage and department in a
// new household on floor home, and returns their ID.
func (s *silo) add(ageYears int, stage world.Stage, dept, home int) world.PersonID {
	w := s.w
	id, hh := world.PersonID(len(w.People)), world.HouseholdID(len(w.Households))
	w.People = append(w.People, world.Person{ID: id, Alive: true, BirthDay: w.Day - world.Tick(ageYears*year), Household: hh,
		Stage: stage, Department: dept, Partner: world.None, Mother: world.None, Father: world.None, Master: world.None,
		BabyFather: world.None, Health: 1})
	w.Alive = append(w.Alive, id)
	w.Households = append(w.Households, world.Household{ID: hh, Floor: home, Members: []world.PersonID{id}})
	return id
}

func (s *silo) start() {
	s.t.Helper()
	sp := space.New()
	spr, _ := rng.New(1, "space")
	if err := sp.Init(s.w, s.cfg, spr); err != nil {
		s.t.Fatal(err)
	}
	s.spRun = sp.Steps()[0].Run
	s.m = New()
	if err := s.m.Init(s.w, s.cfg, s.r); err != nil {
		s.t.Fatal(err)
	}
	s.check()
}

// day runs the network step, then the location step, as in a tick.
func (s *silo) day() {
	s.t.Helper()
	s.w.Day++
	if err := s.m.step(s.w, s.r); err != nil {
		s.t.Fatal(err)
	}
	spr, _ := rng.New(1, "space")
	if err := s.spRun(s.w, spr); err != nil {
		s.t.Fatal(err)
	}
	s.check()
}

func (s *silo) check() {
	s.t.Helper()
	if err := invariants.Check(s.w, s.cfg); err != nil {
		s.t.Fatal(err)
	}
}

// weak lists a person's weak ties of a kind, sorted.
func (s *silo) weak(id world.PersonID, kind world.TieKind) []world.PersonID {
	var out []world.PersonID
	for _, t := range s.w.Network.Weak[id] {
		if t.Kind == kind {
			out = append(out, t.Other)
		}
	}
	slices.Sort(out)
	return out
}

func (s *silo) want(id world.PersonID, kind world.TieKind, others ...world.PersonID) {
	s.t.Helper()
	if got := s.weak(id, kind); !slices.Equal(got, others) {
		s.t.Errorf("person %d %s ties %v, want %v", id, kind, got, others)
	}
}

// With k = 2 and groups of three, every weak tie is forced: each group
// becomes a triangle. Children under 14 get no weak ties; a tie already
// held (here coworkers on the same home floor) is not drawn twice.
func TestWeakTiesDrawn(t *testing.T) {
	s := newSilo(t)
	a := s.add(70, world.Retired, world.None, 2)
	b := s.add(40, world.Retired, world.None, 2)
	c := s.add(15, world.Child, world.None, 2)
	kid := s.add(10, world.Child, world.None, 2)
	e := s.add(30, world.Worker, mechanical, 6)
	f := s.add(30, world.Worker, mechanical, 6)
	g := s.add(30, world.Shadow, mechanical, 6)
	s.start()

	s.want(a, world.TieNeighbour, b, c)
	s.want(b, world.TieNeighbour, a, c)
	s.want(c, world.TieNeighbour, a, b)
	s.want(kid, world.TieNeighbour)
	s.want(e, world.TieCoworker, f, g)
	s.want(g, world.TieCoworker, e, f)
	s.want(e, world.TieNeighbour) // F and G are already coworkers
}

// Ties end when the reason for them goes, on both sides, and are drawn
// again if someone eligible is left.
func TestWeakTiesEndAndAreRedrawn(t *testing.T) {
	s := newSilo(t)
	a := s.add(70, world.Retired, world.None, 2)
	b := s.add(40, world.Retired, world.None, 2)
	c := s.add(40, world.Retired, world.None, 2)
	e := s.add(30, world.Worker, mechanical, 6)
	f := s.add(30, world.Worker, mechanical, 6)
	g := s.add(30, world.Worker, mechanical, 6)
	s.start()

	// B dies; A and C keep each other, with nobody left to draw.
	w := s.w
	w.People[b].Alive = false
	w.Alive = slices.DeleteFunc(w.Alive, func(id world.PersonID) bool { return id == b })
	w.Households[b].Members, w.Households[b].Floor = nil, world.None
	s.day()
	s.want(a, world.TieNeighbour, c)
	s.want(c, world.TieNeighbour, a)

	// E moves to IT (works on floor 1): her coworker ties end; F and G
	// keep theirs. A new neighbour D on floor 2 fills A's and C's gaps.
	w.People[e].Department = it
	s.day() // the location step sees the change at the end of this day
	s.day()
	s.want(e, world.TieCoworker)
	s.want(f, world.TieCoworker, g)
	d := s.add(40, world.Retired, world.None, 2)
	s.day()
	s.want(a, world.TieNeighbour, c, d)
	s.want(d, world.TieNeighbour, a, c)

	// A moves to floor 3: her neighbour ties end.
	w.Households[a].Floor = 3
	s.day()
	s.want(a, world.TieNeighbour)
	s.want(c, world.TieNeighbour, d)
}

// A porter ties to adults in the other two zones only.
func TestPorterTies(t *testing.T) {
	s := newSilo(t)
	p := s.add(30, world.Worker, porters, 5)
	s.add(40, world.Retired, world.None, 5) // same zone: not eligible
	s.add(16, world.Child, world.None, 2)   // under 18: not eligible
	up := s.add(40, world.Retired, world.None, 2)
	down := s.add(40, world.Retired, world.None, 6)
	s.start()
	s.want(p, world.TiePorter, up, down)
	s.want(up, world.TiePorter, p)

	// Retired, she is no longer a porter: the ties end, and nobody draws.
	s.w.People[p].Stage = world.Retired
	s.day()
	s.want(p, world.TiePorter)
	s.want(up, world.TiePorter)
}

// Ties gives one tie per pair, of the first kind that applies: a partner
// in the household is a partner, a child at home is household, family
// elsewhere is kin, then master and shadow; a weak tie with any of them
// is shadowed.
func TestTiesPrecedence(t *testing.T) {
	s := newSilo(t)
	mum := s.add(60, world.Retired, world.None, 3)
	me := s.add(35, world.Worker, mechanical, 6)
	partner := s.add(35, world.Worker, mechanical, 6)
	child := s.add(5, world.Child, world.None, 6)
	sister := s.add(30, world.Retired, world.None, 1)
	shadow := s.add(15, world.Shadow, mechanical, 6)
	w := s.w
	w.People[me].Mother, w.People[sister].Mother = mum, mum
	w.People[child].Mother = me
	w.People[me].Partner, w.People[partner].Partner = partner, me
	w.People[shadow].Master = me
	for _, id := range []world.PersonID{partner, child} {
		w.Households[w.People[id].Household].Members, w.Households[w.People[id].Household].Floor = nil, world.None
		w.People[id].Household = w.People[me].Household
		w.Households[w.People[me].Household].Members = append(w.Households[w.People[me].Household].Members, id)
	}
	s.start()

	got := map[world.PersonID]world.TieKind{}
	for _, t := range w.Network.Ties(w, me, nil) {
		if _, dup := got[t.Other]; dup {
			s.t.Fatalf("two ties with %d", t.Other)
		}
		got[t.Other] = t.Kind
	}
	want := map[world.PersonID]world.TieKind{
		partner: world.TiePartner, child: world.TieHousehold, mum: world.TieKin,
		sister: world.TieKin, shadow: world.TieMasterShadow,
	}
	for o, k := range want {
		if got[o] != k {
			t.Errorf("tie with %d is %v, want %v", o, got[o], k)
		}
	}
	if len(got) != len(want) {
		t.Errorf("ties %v, want %v", got, want)
	}
}

// ODD §6.11: a worker cut off from her department's floors by the stairs
// keeps her coworkers.
func TestStrandedKeepCoworkers(t *testing.T) {
	s := newSilo(t)
	e := s.add(30, world.Worker, mechanical, 6)
	f := s.add(30, world.Worker, mechanical, 6)
	g := s.add(30, world.Worker, mechanical, 3) // lives Up Top, works on 7
	s.start()
	s.want(g, world.TieCoworker, e, f)

	s.w.Space.Broken[4] = true // floors 5–6: G is cut off from floor 7
	s.day()
	s.day()
	if !s.w.Space.Stranded[g] {
		t.Fatal("G is not stranded")
	}
	s.want(g, world.TieCoworker, e, f)
	s.want(e, world.TieCoworker, f, g)
}
