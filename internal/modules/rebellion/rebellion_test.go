package rebellion

import (
	"math"
	"slices"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const year = engine.DaysPerYear

// tiny is a two-floor world: floor 1 Up Top, floor 2 Down Deep, both with
// homes; the court is on floor 1. Its departments are general, judicial
// and sheriff, whose Headcount stands for the officers on duty. Everyone
// has L = 0.8 until given a flag, and R = 0.5. The network, beliefs,
// space and control states are sized to the people added; only this
// module runs, and nobody moves.
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
		Depts:  []world.Department{{Name: "general"}, {Name: "judicial"}, {Name: "sheriff"}},
		Space:  &world.Space{Broken: make([]bool, 1), DeptFloors: [][]int{nil, {1}, nil}},
		Silo:   world.Silo{Intake: [3]float64{1, 1, 1}}}
	r, _ := rng.New(1, "rebellion")
	cfg := &config.Scenario{}
	cfg.Work.WorkerAge = 18
	cfg.Rebellion.LossHardship, cfg.Rebellion.LossHalfLifeYears, cfg.Rebellion.ScapegoatHardship = 0.5, 2, 0.2
	cfg.Rebellion.ActiveThreshold, cfg.Rebellion.ArrestConstant = 0.1, 2.3
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
	base, risk := make([]float64, n), make([]float64, n)
	for i := range base {
		base[i], risk[i] = 0.8, 0.5
	}
	s.w.Beliefs = &world.Beliefs{Base: base, Known: make([]world.Flags, n), Risk: risk, Cost: 0.15}
	s.w.Control = &world.Control{Head: world.None, Successor: world.None, Hold: make([]world.Hold, n)}
	sp := s.w.Space
	sp.Location, sp.OffWork = make([]int, n), make([]bool, n)
	for id := range n {
		sp.Location[id] = s.w.Households[s.w.People[id].Household].Floor
	}
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

// doubt gives the person all five flags: L = 0.8 − 5 × 0.15 = 0.05.
func (s *tiny) doubt(id world.PersonID) { s.w.Beliefs.Known[id] = 1<<world.NumFlags - 1 }

// ODD §6.14, Arrest estimate and Stages, by hand. Ten officers on duty,
// all adults on floors connected to the court, so each zone's patrol is
// 10 × its share of them. A Down Deep doubter with health 0.5 has
// H = 1 − 0.5 × 0.8 = 0.6 and G = 0.6 × 0.95 = 0.57. They are active when
// G − R × P > T, so R decides; then the counts move P the next day.
func TestActivation(t *testing.T) {
	s := newTiny(t)
	var deep [4]world.PersonID
	for i := range deep {
		deep[i] = s.add(30, 2, 0.5)
	}
	top := s.add(30, 1, 1)
	s.start()
	w, b, c := s.w, s.w.Rebellion, s.w.Control
	w.Depts[1].Headcount, w.Depts[2].Headcount = 8, 2
	for _, id := range deep {
		s.doubt(id)
	}
	g := 0.6 * (1 - 0.05)
	p := func(actives int) float64 { return 1 - math.Exp(-2.3*10*4.0/5/float64(actives+1)) } // 4 of 5 reachable adults are Down Deep
	edge := (g - 0.1) / p(0)                                                                 // the largest R that is active
	risks := []float64{edge - 1e-9, edge + 1e-9, 0, 0}
	for i, id := range deep {
		w.Beliefs.Risk[id] = risks[i]
	}
	c.Hold[deep[3]] = world.Custody
	s.step()
	if !near(b.G[deep[0]], g) {
		t.Fatalf("G = %v, want %v", b.G[deep[0]], g)
	}
	want := []world.RebelStage{world.Active, world.Calm, world.Active, world.Calm}
	for i, id := range deep {
		if b.Stage[id] != want[i] {
			t.Errorf("Down Deep %d (R = %v): stage %d, want %d", i, risks[i], b.Stage[id], want[i])
		}
	}
	if b.Stage[top] != world.Calm || b.Actives != [3]int{0, 0, 2} || b.Reach != [3]int{1, 0, 4} {
		t.Errorf("Up Top stage %d; counts %v active, %v reachable; want calm, [0 0 2], [1 0 4]", b.Stage[top], b.Actives, b.Reach)
	}
	if len(c.Acts) != 2 || c.Acts[0].Offence != world.Organise || c.Acts[1].Offence != world.Organise {
		t.Errorf("acts %v, want two organise", c.Acts)
	}

	// Two actives yesterday lower P, so the one just above the edge comes out.
	c.Acts = nil
	s.step()
	if b.Stage[deep[1]] != world.Active || p(2) >= p(0) {
		t.Errorf("with 2 actives yesterday (P %v → %v), R just above the old edge: stage %d", p(0), p(2), b.Stage[deep[1]])
	}

	// Fear adds to the felt risk: at F = 1, N = R.
	c.Fear = 1
	w.Beliefs.Risk[deep[0]] = g - 0.1 + 1e-9
	s.step()
	if b.Stage[deep[0]] != world.Calm || b.Stage[deep[2]] != world.Active {
		t.Errorf("at F = 1: R just above G − T is %d, R = 0 is %d; want calm, active", b.Stage[deep[0]], b.Stage[deep[2]])
	}
	c.Fear = 0

	// A broken segment cuts Down Deep off from the court: P = 0 there, and
	// even R = 1 is active.
	w.Space.Broken[0] = true
	w.Beliefs.Risk[deep[1]] = 1
	s.step()
	if b.Stage[deep[1]] != world.Active || b.Reach != [3]int{1, 0, 0} {
		t.Errorf("cut off: R = 1 is %d, reachable %v; want active, [1 0 0]", b.Stage[deep[1]], b.Reach)
	}
	w.Space.Broken[0] = false

	// Officers, the Head of IT and people on the stairs are never active.
	w.People[deep[1]].Stage, w.People[deep[1]].Department = world.Worker, 1
	c.Head = deep[2]
	w.Space.Location[deep[0]] = 0
	s.step()
	for _, id := range deep {
		if b.Stage[id] != world.Calm {
			t.Errorf("person %d: stage %d, want calm", id, b.Stage[id])
		}
	}
	// An officer off work is not on duty, and may be active.
	w.Space.OffWork[deep[1]] = true
	w.Beliefs.Risk[deep[1]] = 0
	s.step()
	if b.Stage[deep[1]] != world.Active {
		t.Errorf("officer off work: stage %d, want active", b.Stage[deep[1]])
	}
}

// T12.2 done-when: Epstein's qualitative result. 1,000 Down Deep adults
// with all five flags (L = 0.05), health spread over 0.75–1 (so H is
// 0.2–0.4) and R over 0–1. Officers are cut by one a day from 200 to 0.
// Actives don't follow the officers' steady fall; they burst: some
// 10-day window, 5% of the fall, holds more than a quarter of the whole
// rise, five times its share, as fewer officers per active lowers P,
// which brings out more actives. The zone rule has one balance point for
// each number of officers, so the burst is a steep tip, not a jump; bursts
// that repeat need arrests and releases (control). With the regime's L (no
// flags), G ≤ 0.4 × 0.2 < T and nobody is ever active, even with no
// officers at all.
func TestEpsteinBursts(t *testing.T) {
	for _, doubting := range []bool{true, false} {
		s := newTiny(t)
		const n = 1000
		for i := range n {
			s.add(30, 2, 0.75+0.25*float64((i*7)%n)/n)
		}
		s.start()
		w, b := s.w, s.w.Rebellion
		for i := range n {
			w.Beliefs.Risk[i] = float64(i) / n
			if doubting {
				s.doubt(world.PersonID(i))
			}
		}
		var actives []int
		for officers := 200; officers >= 0; officers-- {
			w.Depts[1].Headcount = officers
			s.step()
			w.Control.Acts = nil
			actives = append(actives, b.Actives[world.DownDeep])
		}
		if !doubting {
			if slices.Max(actives) != 0 {
				t.Errorf("with L = 0.8, up to %d actives; want none", slices.Max(actives))
			}
			continue
		}
		rise, burst, at := actives[len(actives)-1]-actives[0], 0, 0
		for d := 10; d < len(actives); d++ {
			if r := actives[d] - actives[d-10]; r > burst {
				burst, at = r, d
			}
		}
		t.Logf("actives every 20 days as officers fall 200 → 0: %v", everyNth(actives, 20))
		t.Logf("rise %d; largest 10-day rise %d, ending with %d officers", rise, burst, 200-at)
		if rise < n/2 || 4*burst <= rise {
			t.Errorf("rise %d, largest 10-day rise %d: want a rise of at least %d, more than a quarter of it within 10 days", rise, burst, n/2)
		}
	}
}

func everyNth(xs []int, n int) []int {
	var out []int
	for i := 0; i < len(xs); i += n {
		out = append(out, xs[i])
	}
	return out
}
