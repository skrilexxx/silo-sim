package rebellion

import (
	"math"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const year = engine.DaysPerYear

// tiny is a two-floor world: floor 1 Up Top, floor 2 Down Deep, both with
// homes. Everyone has L = 0.8 until given a flag. The network, beliefs
// and control states are sized to the people added; only this module
// runs.
type tiny struct {
	t   *testing.T
	w   *world.World
	m   *Module
	r   *rng.Stream
	cfg *config.Scenario
}

func newTiny(t *testing.T) *tiny {
	w := &world.World{Day: 100 * year,
		Floors: []world.Floor{{Number: 1, Homes: 99, Zone: world.UpTop}, {Number: 2, Homes: 99, Zone: world.DownDeep}},
		Silo:   world.Silo{Intake: [3]float64{1, 1, 1}}}
	r, _ := rng.New(1, "rebellion")
	cfg := &config.Scenario{}
	cfg.Work.WorkerAge = 18
	cfg.Rebellion.LossHardship, cfg.Rebellion.LossHalfLifeYears, cfg.Rebellion.ScapegoatHardship = 0.5, 2, 0.2
	return &tiny{t: t, w: w, m: New(), r: r, cfg: cfg}
}

// add puts a person of the given age and health in a new household on the
// floor and returns their ID.
func (s *tiny) add(ageYears, floor int, health float64) world.PersonID {
	w := s.w
	id, hh := world.PersonID(len(w.People)), world.HouseholdID(len(w.Households))
	w.People = append(w.People, world.Person{ID: id, Alive: true, BirthDay: w.Day - world.Tick(ageYears*year), Household: hh,
		Health: health, Partner: world.None, Mother: world.None, Father: world.None, Master: world.None})
	w.Alive = append(w.Alive, id)
	w.Households = append(w.Households, world.Household{ID: hh, Floor: floor, Members: []world.PersonID{id}})
	return id
}

// join moves b into a's household.
func (s *tiny) join(a, b world.PersonID) {
	w := s.w
	from := &w.Households[w.People[b].Household]
	from.Members, from.Floor = nil, world.None
	w.People[b].Household = w.People[a].Household
	hh := &w.Households[w.People[a].Household]
	hh.Members = append(hh.Members, b)
}

func (s *tiny) start() {
	s.t.Helper()
	n := len(s.w.People)
	s.w.Network = &world.Network{Weak: make([][]world.Tie, n), Children: make([][]world.PersonID, n)}
	base := make([]float64, n)
	for i := range base {
		base[i] = 0.8
	}
	s.w.Beliefs = &world.Beliefs{Base: base, Known: make([]world.Flags, n), Cost: 0.15}
	s.w.Control = &world.Control{}
	if err := s.m.Init(s.w, s.cfg, s.r); err != nil {
		s.t.Fatal(err)
	}
}

func (s *tiny) step() {
	s.t.Helper()
	if err := s.m.step(s.w, s.r); err != nil {
		s.t.Fatal(err)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

// ODD §6.14, Hardship and Grievance: the four sources combine like
// independent chances; G = H × (1 − L); children have neither.
func TestHardshipAndGrievance(t *testing.T) {
	s := newTiny(t)
	deep := s.add(30, 2, 0.9)
	top := s.add(30, 1, 1)
	child := s.add(10, 2, 0.5)
	doubter := s.add(30, 1, 0.5)
	s.start()
	s.w.Silo.Intake[world.DownDeep] = 0.9
	s.w.Beliefs.Known[doubter] = 1<<world.FlagOutside | 1<<world.FlagScreen // L = 0.8 − 2 × 0.15
	s.step()

	b := s.w.Rebellion
	want := 1 - 0.9*0.9*0.8 // food 0.1, health 0.1, Down Deep 0.2
	if !near(b.H[deep], want) || !near(b.G[deep], want*(1-0.8)) {
		t.Errorf("Down Deep: H = %v, G = %v, want %v and %v", b.H[deep], b.G[deep], want, want*0.2)
	}
	if b.H[top] != 0 || b.G[top] != 0 {
		t.Errorf("fed, healthy Up Top adult: H = %v, G = %v, want 0", b.H[top], b.G[top])
	}
	if b.H[child] != 0 || b.G[child] != 0 {
		t.Errorf("child: H = %v, G = %v, want 0", b.H[child], b.G[child])
	}
	if !near(b.H[doubter], 0.5) || !near(b.G[doubter], 0.5*0.5) {
		t.Errorf("doubter: H = %v, G = %v, want 0.5 and 0.25", b.H[doubter], b.G[doubter])
	}

	// Rationing everywhere: Up Top food hardship 0.1 and nothing else.
	s.w.Silo.Intake = [3]float64{0.9, 0.9, 0.9}
	s.step()
	if !near(b.H[top], 0.1) {
		t.Errorf("Up Top under rationing: H = %v, want 0.1", b.H[top])
	}
	// Overfed: negative zone hardship counts as 0.
	s.w.Silo.Intake[world.UpTop] = 1.2
	s.step()
	if b.H[top] != 0 {
		t.Errorf("Up Top fed above need: H = %v, want 0", b.H[top])
	}
}

// ODD §6.14, Hardship, loss: a cleaner's family (partner, household, kin)
// has a loss on the day; a master, a shadow and a coworker don't. Both
// partners of a denied win have one. h_loss = ℓ on the day, halving every
// T_loss years; a new loss restarts the clock.
func TestLosses(t *testing.T) {
	s := newTiny(t)
	cleaner := s.add(40, 1, 1)
	partner := s.add(40, 1, 1)
	lodger := s.add(30, 1, 1)
	mother := s.add(65, 2, 1)
	son := s.add(20, 1, 1)
	master := s.add(50, 1, 1)
	coworker := s.add(40, 1, 1)
	woman := s.add(25, 1, 1)
	husband := s.add(25, 1, 1)
	s.join(cleaner, lodger)
	w := s.w
	w.People[cleaner].Partner, w.People[partner].Partner = partner, cleaner
	w.People[cleaner].Mother, w.People[son].Mother = mother, cleaner
	w.People[cleaner].Master = master
	w.People[woman].Partner, w.People[husband].Partner = husband, woman
	s.start()
	n := w.Network
	n.Children[mother], n.Children[cleaner] = []world.PersonID{cleaner}, []world.PersonID{son}
	n.Weak[cleaner] = []world.Tie{{Other: coworker, Kind: world.TieCoworker}}
	n.Weak[coworker] = []world.Tie{{Other: cleaner, Kind: world.TieCoworker}}

	// The cleaner died at today's deaths step and went out at the control
	// step; the deaths step leaves their records, as ODD §6.1 does.
	w.People[cleaner].Alive = false
	w.People[partner].Partner = world.None
	w.Alive = w.Alive[1:]
	w.Control.Cleaned = []world.PersonID{cleaner}
	w.Control.DeniedToday = []world.PersonID{woman}
	s.step()

	b := w.Rebellion
	for _, id := range []world.PersonID{partner, lodger, mother, son, woman, husband} {
		if b.LossDay[id] != w.Day {
			t.Errorf("person %d: last loss %d, want today (%d)", id, b.LossDay[id], w.Day)
		}
		if !near(b.H[id], 0.5) && id != mother {
			t.Errorf("person %d: H = %v on the day of a loss, want ℓ = 0.5", id, b.H[id])
		}
	}
	if !near(b.H[mother], 1-0.5*0.8) { // Down Deep too
		t.Errorf("mother: H = %v, want %v", b.H[mother], 1-0.5*0.8)
	}
	for _, id := range []world.PersonID{master, coworker} {
		if b.LossDay[id] != world.None || b.H[id] != 0 {
			t.Errorf("person %d: last loss %d, H = %v; want no loss", id, b.LossDay[id], b.H[id])
		}
	}

	// T_loss years later the loss's hardship has halved.
	w.Control.Cleaned, w.Control.DeniedToday = nil, nil
	w.Day += 2 * year
	s.step()
	if !near(b.H[son], 0.25) {
		t.Errorf("son after T_loss: H = %v, want 0.25", b.H[son])
	}
	// A new loss restarts the clock rather than adding up.
	w.Control.DeniedToday = []world.PersonID{woman}
	s.step()
	if !near(b.H[woman], 0.5) {
		t.Errorf("woman after a second denied win: H = %v, want 0.5", b.H[woman])
	}
}
