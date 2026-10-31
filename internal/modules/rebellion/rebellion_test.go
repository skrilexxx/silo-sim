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

// tiny is a three-floor world: floor 1 Up Top and floor 2 Down Deep, both
// with homes, and floor 3 a Down Deep Mechanical floor; the court is on
// floor 1. Its departments are general, judicial,
// sheriff and mechanical; Judicial's and the Sheriff's Headcount stands
// for the officers on duty. Everyone
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
		Floors: []world.Floor{{Number: 1, Homes: 99, Zone: world.UpTop}, {Number: 2, Homes: 99, Zone: world.DownDeep},
			{Number: 3, Zone: world.DownDeep, Function: world.Mechanical}},
		Depts: []world.Department{{Name: "general"}, {Name: "judicial"}, {Name: "sheriff"}, {Name: "mechanical"}},
		Space: &world.Space{Broken: make([]bool, 2), DeptFloors: [][]int{nil, {1}, nil, nil}},
		Silo:  world.Silo{Intake: [3]float64{1, 1, 1}, AirQuality: 1, WaterSupply: 1}}
	r, _ := rng.New(1, "rebellion")
	cfg := &config.Scenario{}
	cfg.Work.WorkerAge = 18
	cfg.Rebellion.LossHardship, cfg.Rebellion.LossHalfLifeYears, cfg.Rebellion.ScapegoatHardship = 0.5, 2, 0.2
	cfg.Rebellion.ActiveThreshold, cfg.Rebellion.ArrestConstant, cfg.Rebellion.ViolentThreshold = 0.1, 2.3, 0.3
	cfg.Rebellion.StrikeShare, cfg.Rebellion.StairRepairDays = 0.1, 60
	cfg.Rebellion.CollapseLevel, cfg.Rebellion.CollapseDays = 0.2, 30
	cfg.Rebellion.UnrestShare, cfg.Rebellion.UnrestDays = 0.01, 3
	cfg.Rebellion.RebellionShare, cfg.Rebellion.RebellionDays, cfg.Rebellion.QuietDays, cfg.Rebellion.RegimeDays = 0.05, 7, 30, 30
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

// ODD §6.14, Hardship and Grievance: the three sources combine like
// independent chances; health is not one; G = H × (1 − L); children have
// neither.
func TestHardshipAndGrievance(t *testing.T) {
	s := newTiny(t)
	deep := s.add(30, 2, 0.5)
	top := s.add(30, 1, 0.5)
	child := s.add(10, 2, 0.5)
	doubter := s.add(30, 2, 1)
	s.start()
	s.w.Silo.Intake[world.DownDeep] = 0.9
	s.w.Beliefs.Known[doubter] = 1<<world.FlagOutside | 1<<world.FlagScreen // L = 0.8 − 2 × 0.15
	s.step()

	b := s.w.Rebellion
	want := 1 - 0.9*0.8 // food 0.1, Down Deep 0.2
	if !near(b.H[deep], want) || !near(b.G[deep], want*(1-0.8)) {
		t.Errorf("Down Deep: H = %v, G = %v, want %v and %v", b.H[deep], b.G[deep], want, want*0.2)
	}
	if b.H[top] != 0 || b.G[top] != 0 {
		t.Errorf("fed, sick Up Top adult: H = %v, G = %v, want 0", b.H[top], b.G[top])
	}
	if b.H[child] != 0 || b.G[child] != 0 {
		t.Errorf("child: H = %v, G = %v, want 0", b.H[child], b.G[child])
	}
	if !near(b.H[doubter], want) || !near(b.G[doubter], want*0.5) {
		t.Errorf("doubter: H = %v, G = %v, want %v and %v", b.H[doubter], b.G[doubter], want, want*0.5)
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
// 10 × its share of them. A Down Deep doubter on half rations has
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
	w.Silo.Intake[world.DownDeep] = 0.5
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
// with all five flags (L = 0.05), losses from 2 to 12 years ago (so H is
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
		for range n {
			s.add(30, 2, 1)
		}
		s.start()
		w, b := s.w, s.w.Rebellion
		for i := range n {
			w.Beliefs.Risk[i] = float64(i) / n
			b.LossDay[i] = w.Day - world.Tick(2*year+10*year*((i*7)%n)/n) // h_loss = 0.25 down to 0.008
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

// tie gives a and b a coworker tie (ODD §6.11), before start.
func (s *tiny) tie(a, b world.PersonID) {
	n := s.w.Network
	n.Weak[a] = append(n.Weak[a], world.Tie{Other: b, Kind: world.TieCoworker})
	n.Weak[b] = append(n.Weak[b], world.Tie{Other: a, Kind: world.TieCoworker})
}

// ODD §6.14, Stages: joining and violence. No officers on duty, so P = 0
// and, with F = 0, N = 0: every Down Deep doubter is active (G > T).
// Hub (and edge, with the same ties) is tied to three of them and one calm
// Up Top adult. Nobody joins on the first day, since nobody was out the
// day before. The next day 3 of hub's 4 ties were out: hub joins with
// R = 0.75, edge with R just above does not; shy, tied to hub and edge
// only, joins even with R = 1; a loner with no ties never joins. Joined
// people with G − N > T_v = 0.3 are violent and commit violence, the rest
// of the active organise.
func TestJoiningAndViolence(t *testing.T) {
	s := newTiny(t)
	// Down Deep on half rations: H = 1 − 0.5 × 0.8 = 0.6, G = 0.6 × 0.95 = 0.57.
	// Up Top on 72% rations: H = 0.28, so G = 0.266 for a doubter, 0.056 for the calm.
	hub := s.add(30, 2, 1)
	hot := s.add(30, 2, 1)   // violent once joined
	cool := s.add(30, 1, 1)  // Up Top: joins, not violent
	shy := s.add(30, 2, 1)   // tied to hub and edge only, R = 1: needs both out
	calm := s.add(30, 1, 1)  // Up Top, no flags: G < T
	loner := s.add(30, 2, 1) // no ties
	edge := s.add(30, 2, 1)  // like hub, R just above 3/4
	s.start()
	w, b, c := s.w, s.w.Rebellion, s.w.Control
	w.Silo.Intake = [3]float64{0.72, 1, 0.5}
	for _, id := range []world.PersonID{hub, hot, cool, shy, loner, edge} {
		s.doubt(id)
	}
	for _, other := range []world.PersonID{hot, cool, shy, calm} {
		s.tie(hub, other)
		s.tie(edge, other)
	}
	risk := map[world.PersonID]float64{hub: 0.75, hot: 0.01, cool: 0.01, shy: 1, loner: 0.01, edge: 0.75 + 1e-9}
	for id, r := range risk {
		w.Beliefs.Risk[id] = r
	}

	s.step()
	for _, id := range []world.PersonID{hub, hot, cool, shy, loner, edge} {
		if b.Stage[id] != world.Active {
			t.Errorf("day 1, person %d: stage %d, want active (nobody was out yesterday)", id, b.Stage[id])
		}
	}
	c.Acts = nil
	s.step()
	want := map[world.PersonID]world.RebelStage{hub: world.Violent, hot: world.Violent, cool: world.Joined,
		shy: world.Violent, calm: world.Calm, loner: world.Active, edge: world.Active}
	for id, st := range want {
		if b.Stage[id] != st {
			t.Errorf("day 2, person %d: stage %d, want %d", id, b.Stage[id], st)
		}
	}
	if len(c.Acts) != 6 {
		t.Errorf("acts %v: want 6", c.Acts)
	}
	for _, a := range c.Acts {
		if v := b.Stage[a.Person] == world.Violent; v != (a.Offence == world.Violence) || !v && a.Offence != world.Organise {
			t.Errorf("person %d at stage %d committed %v", a.Person, b.Stage[a.Person], a.Offence)
		}
	}
}

// ODD §1.2, Definitions, driven day by day with chosen shares of adults
// active: unrest after 3 days at 1%, ending 30 quiet days later and logged
// with the first quiet day as its end; no rebellion without a collective
// action; a rebellion started by one, absorbing the open unrest spell, and
// ending as suppressed after 30 quiet days.
func TestUnrestAndRebellion(t *testing.T) {
	s := newTiny(t)
	s.add(30, 1, 1)
	s.start()
	w, b := s.w, s.w.Rebellion
	w.Events = nil
	day := func(share float64, action bool) {
		w.Day++
		s.m.track(w, share, action)
	}
	day(0.01, false)
	day(0.01, false)
	day(0, false)
	if b.UnrestStart != world.None {
		t.Fatalf("unrest after 2 days at 1%%")
	}
	first := w.Day + 1
	for range 3 {
		day(0.01, false)
	}
	if b.UnrestStart != first {
		t.Fatalf("unrest start %d, want %d", b.UnrestStart, first)
	}
	quiet := w.Day + 1
	for range 29 {
		day(0.009, false)
	}
	if b.UnrestStart == world.None || len(w.Events) != 0 {
		t.Fatalf("unrest ended after 29 quiet days")
	}
	day(0, false)
	if b.UnrestStart != world.None || b.UnrestSpells != 1 || len(w.Events) != 1 {
		t.Fatalf("after 30 quiet days: unrest start %d, %d spells, events %v", b.UnrestStart, b.UnrestSpells, w.Events)
	}
	if ev := w.Events[0].(world.UnrestEvent); ev.Type != "unrest" || ev.Start != first || ev.End != quiet {
		t.Errorf("event %+v, want unrest %d to %d", ev, first, quiet)
	}

	// 10 days at 5% with no action: unrest, not a rebellion. An action on
	// the 11th makes the last 7 days a rebellion; the unrest spell is its
	// run-up and is never logged.
	w.Events = nil
	for range 10 {
		day(0.05, false)
	}
	if b.RebellionStart != world.None || b.UnrestStart == world.None {
		t.Fatalf("rebellion %d, unrest %d after 10 days at 5%% with no action", b.RebellionStart, b.UnrestStart)
	}
	day(0.05, true)
	if b.RebellionStart != w.Day-6 || b.UnrestStart != world.None || b.Rebellions != 1 {
		t.Fatalf("rebellion start %d (want %d), unrest %d, %d rebellions", b.RebellionStart, w.Day-6, b.UnrestStart, b.Rebellions)
	}
	start := b.RebellionStart
	day(0.2, false)
	quiet = w.Day + 1
	for range 30 {
		day(0.001, false)
	}
	if b.RebellionStart != world.None || b.UnrestSpells != 1 || len(w.Events) != 2 {
		t.Fatalf("rebellion start %d, %d unrest spells, events %v", b.RebellionStart, b.UnrestSpells, w.Events)
	}
	if ev := w.Events[1].(world.RebellionEvent); ev.Type != "rebellionEnd" || ev.Start != start || ev.End != quiet || ev.Ending != "suppressed" {
		t.Errorf("event %+v, want rebellionEnd %d to %d, suppressed", ev, start, quiet)
	}

	// The action must fall within the 7 days: one 7 days before the 7th
	// day at 5% is too old.
	day(0, true)
	for range 7 {
		day(0.05, false)
	}
	if b.RebellionStart != world.None {
		t.Errorf("rebellion from an action 8 days back")
	}
	day(0.05, false)
	if b.RebellionStart != world.None {
		t.Errorf("rebellion with no action in the last 7 days")
	}

	// Rebels holding Up Top for 30 days in a row end a running rebellion
	// with a regime change, that day.
	w.Events = nil
	day(0.05, true)
	start = b.RebellionStart
	b.HeldDays[world.UpTop] = 29
	day(0.05, true)
	if b.RebellionStart == world.None {
		t.Fatalf("regime change after 29 days held")
	}
	b.HeldDays[world.UpTop] = 30
	day(0.05, true)
	ev := w.Events[len(w.Events)-1].(world.RebellionEvent)
	if b.RebellionStart != world.None || ev.Type != "rebellionEnd" || ev.Start != start || ev.End != w.Day || ev.Ending != "regimeChange" || b.HeldDays[world.UpTop] != 0 {
		t.Errorf("event %+v, rebellion %d, held days %d; want a regime change today, the count restarted", ev, b.RebellionStart, b.HeldDays[world.UpTop])
	}
}

// ODD §6.14, Strikes, and §1.2: workers who joined yesterday strike today.
// Mechanical, a critical department, is on strike when at least q_strike
// = 10% of its workers are; General is not critical. Shadows strike but
// don't count toward the share.
func TestStrikes(t *testing.T) {
	s := newTiny(t)
	var mech []world.PersonID
	for range 20 {
		mech = append(mech, s.add(30, 2, 1))
	}
	shadow := s.add(15, 2, 1)
	gen := s.add(30, 2, 1)
	s.start()
	w, b := s.w, s.w.Rebellion
	for _, id := range mech {
		w.People[id].Stage, w.People[id].Department = world.Worker, 3
	}
	w.People[shadow].Stage, w.People[shadow].Department = world.Shadow, 3
	w.People[gen].Stage, w.People[gen].Department = world.Worker, 0
	w.Depts[3].Headcount = 20

	b.Stage[mech[0]], b.Stage[shadow], b.Stage[gen] = world.Violent, world.Joined, world.Joined
	b.Stage[mech[1]] = world.Active // active, not on strike
	s.m.strikes(w)
	if b.Striking[3] != 1 || b.Striking[0] != 1 || s.m.action(w) {
		t.Errorf("striking %v, action %v; want 1 in Mechanical and General, no action (1 of 20 < 10%%)", b.Striking, s.m.action(w))
	}
	if !b.OnStrike(w, shadow) || b.OnStrike(w, mech[1]) {
		t.Errorf("shadow on strike %v, active worker on strike %v; want true, false", b.OnStrike(w, shadow), b.OnStrike(w, mech[1]))
	}
	b.Stage[mech[2]] = world.Joined
	s.m.strikes(w)
	if b.Striking[3] != 2 || !s.m.action(w) {
		t.Errorf("striking %v, action %v; want 2 in Mechanical, a strike (2 of 20)", b.Striking, s.m.action(w))
	}
}

// battle is a world for fights: ten Judicial officers living Up Top on
// floor 1 (the court) and ten violent Down Deep rebels on floor 2, tied in
// a ring so they join on the first day (R = 0), the first with a child
// (the last person). Yesterday's counts give each zone half the adults, so
// each zone's patrol is 10 × 10/20 = 5.
func battle(t *testing.T, eJ, eR, kill float64) (*tiny, []world.PersonID, []world.PersonID) {
	s := newTiny(t)
	s.cfg.Rebellion.OfficerEffect, s.cfg.Rebellion.RebelEffect, s.cfg.Rebellion.KilledShare = eJ, eR, kill
	s.cfg.Health.InjuryCost = 0.3
	var officers, rebels []world.PersonID
	for range 10 {
		officers = append(officers, s.add(30, 1, 1))
	}
	for range 10 {
		rebels = append(rebels, s.add(30, 2, 1))
	}
	child := s.add(10, 2, 1)
	s.w.People[child].Mother = rebels[0]
	s.start()
	w := s.w
	w.Network.Children[rebels[0]] = []world.PersonID{child}
	for _, id := range officers {
		w.People[id].Stage, w.People[id].Department = world.Worker, 1
	}
	w.Depts[1].Headcount = 10
	w.Silo.Intake[world.DownDeep] = 0.5 // G = 0.6 × 0.95 = 0.57
	for i, id := range rebels {
		s.doubt(id)
		w.Beliefs.Risk[id] = 0
		s.tie(id, rebels[(i+1)%len(rebels)])
	}
	return s, officers, rebels
}

// ODD §6.14, Fights, decision D6: with a patrol of 5 and 10 violent
// rebels, e_J = 0.4 puts out 0.4 × 5 = 2 rebels and e_R = 0.1 puts out
// 0.1 × 10 = 1 officer. With p_kill = 0 the rebels are arrested (handed to
// control) and the officer wounded; the 8 left outnumber the patrol of 5,
// so the rebels hold the Down Deep. With e_J = 2 every rebel is put out
// and, at p_kill = 1, killed, and their families have a loss; the officers
// killed shrink the patrol.
func TestFights(t *testing.T) {
	s, officers, rebels := battle(t, 0.4, 0.1, 0)
	w, b, c := s.w, s.w.Rebellion, s.w.Control
	w.Events = nil
	s.step()
	out := 0
	for _, id := range rebels {
		if b.Stage[id] == world.Calm {
			out++
		} else if b.Stage[id] != world.Violent {
			t.Errorf("rebel %d: stage %d, want violent or put out", id, b.Stage[id])
		}
	}
	wounded := 0
	for _, id := range officers {
		if w.People[id].Health < 1 {
			wounded++
			if !near(w.People[id].Health, 0.7) {
				t.Errorf("officer %d: health %v, want 0.7", id, w.People[id].Health)
			}
		}
	}
	if out != 2 || len(c.Seized) != 2 || len(c.Doomed) != 0 || wounded != 1 {
		t.Errorf("%d rebels put out, seized %v, doomed %v, %d officers wounded; want 2, 2, none, 1", out, c.Seized, c.Doomed, wounded)
	}
	if b.Tally != (world.FightTally{RebelsArrested: 2, OfficersWounded: 1}) {
		t.Errorf("tally %+v", b.Tally)
	}
	if b.Held != [3]bool{false, false, true} || b.Patrol[world.DownDeep] != 5 || !s.m.action(w) {
		t.Errorf("held %v, patrol %v, action %v; want the Down Deep held against a patrol of 5, an action", b.Held, b.Patrol, s.m.action(w))
	}
	if len(w.Events) != 1 || w.Events[0].(world.ZoneEvent) != (world.ZoneEvent{EventHeader: world.EventHeader{Day: w.Day, Type: "zoneHeld"}, Zone: "downDeep"}) {
		t.Errorf("events %v, want zoneHeld downDeep", w.Events)
	}
	if !b.Holds(w, 2) || b.Holds(w, 1) || b.Holds(w, 0) {
		t.Errorf("Holds: floor 2 %v, floor 1 %v, the stairs %v", b.Holds(w, 2), b.Holds(w, 1), b.Holds(w, 0))
	}

	s, _, rebels = battle(t, 2, 0.5, 1)
	w, b, c = s.w, s.w.Rebellion, s.w.Control
	child := world.PersonID(len(w.People) - 1)
	s.step()
	// 10 rebels and min(5, 0.5 × 10) = 5 officers out, all killed.
	if len(c.Doomed) != 15 || b.Tally != (world.FightTally{RebelsKilled: 10, OfficersKilled: 5}) {
		t.Errorf("doomed %d, tally %+v; want 15 and 10 rebels, 5 officers killed", len(c.Doomed), b.Tally)
	}
	if b.LossDay[child] != w.Day {
		t.Errorf("a killed rebel's child: last loss %d, want today", b.LossDay[child])
	}
	if b.Held[world.DownDeep] || b.Patrol[world.DownDeep] != 2.5 {
		t.Errorf("held %v, patrol %v; want nobody left to hold, patrol (10 − 5) × 10/20 = 2.5", b.Held, b.Patrol)
	}
}

// ODD §6.14, Held zones: rebels on a floor cut off from every court can't
// be fought, but a zone with no patrol is held by any violent rebel in it.
func TestCutOffZoneIsHeld(t *testing.T) {
	s, _, _ := battle(t, 1, 1, 1)
	w, b := s.w, s.w.Rebellion
	w.Space.Broken[0] = true
	s.step() // yesterday's counts still reach the Down Deep: patrol 5, but nobody there can be reached
	if len(w.Control.Doomed) != 0 || !b.Held[world.DownDeep] {
		t.Errorf("doomed %v, held %v; want no fight, the Down Deep held", w.Control.Doomed, b.Held)
	}
	s.step()
	if b.Patrol[world.DownDeep] != 0 || !b.Held[world.DownDeep] || b.HeldDays[world.DownDeep] != 2 {
		t.Errorf("patrol %v, held %v for %d days; want no patrol, held 2 days", b.Patrol, b.Held, b.HeldDays[world.DownDeep])
	}
}

// ODD §6.14, Sabotage, with p_sab = 1 and no officers (so no fights). On
// the Mechanical floor the first violent rebel wrecks the generator, the
// second the pump, the third breaks segment 2 (floors 2–3); on floor 2,
// below the court, the fourth breaks segment 1. Each act is the offence
// sabotage.
func TestSabotage(t *testing.T) {
	s, _, rebels := battle(t, 0, 0, 0)
	s.cfg.Rebellion.SabotageChance = 1
	s.m.sab = 1
	w, b := s.w, s.w.Rebellion
	w.Depts[1].Headcount = 0
	for _, id := range rebels[:3] {
		w.Space.Location[id] = 3
	}
	for _, id := range rebels[4:] {
		b.Stage[id] = world.Calm
		w.Beliefs.Known[id] = 0 // calm for good: G < T
	}
	w.Events, w.Control.Acts = nil, nil
	s.step()
	var got []string
	for _, e := range w.Events {
		if ev, ok := e.(world.SabotageEvent); ok {
			got = append(got, ev.Target)
		}
	}
	if !slices.Equal(got, []string{"generator", "pump", "stairs", "stairs"}) || b.Sabotage != [3]int{1, 1, 2} {
		t.Errorf("sabotage %v, counts %v; want generator, pump, stairs, stairs", got, b.Sabotage)
	}
	if !b.GeneratorHit || !b.PumpHit || !w.Space.Broken[0] || !w.Space.Broken[1] {
		t.Errorf("generator hit %v, pump hit %v, broken %v", b.GeneratorHit, b.PumpHit, w.Space.Broken)
	}
	n := 0
	for _, a := range w.Control.Acts {
		if a.Offence == world.Sabotage {
			n++
		}
	}
	if n != 4 {
		t.Errorf("%d sabotage acts, want 4", n)
	}
	// Nothing left to wreck the next day: the generator and pump wait for
	// the resources module, the segments are broken.
	w.Events = nil
	s.step()
	for _, e := range w.Events {
		if _, ok := e.(world.SabotageEvent); ok {
			t.Errorf("sabotage %+v with nothing left to wreck", e)
		}
	}
}

// ODD §6.14, Stair repairs: at full Mechanical coverage and D_stair = 1 a
// broken segment is repaired the same day, unless a floor at either end
// lies in a held zone. breakStairs breaks a segment and rejects one that
// isn't in the silo.
func TestStairRepairs(t *testing.T) {
	s := newTiny(t)
	s.add(30, 1, 1)
	s.cfg.Rebellion.StairRepairDays = 1
	s.start()
	w, b := s.w, s.w.Rebellion
	w.Depts[3].Coverage = 1
	fire := s.m.EventTypes()[0].Fire
	if err := fire(w, s.r, config.Event{Segment: 3}); err == nil {
		t.Error("breakStairs segment 3 of 2: no error")
	}
	if err := fire(w, s.r, config.Event{Segment: 1}); err != nil || !w.Space.Broken[0] {
		t.Fatalf("breakStairs segment 1: %v, broken %v", err, w.Space.Broken)
	}
	b.Held[world.DownDeep] = true // floor 2, at the segment's lower end
	s.m.repairs(w, s.r)
	if !w.Space.Broken[0] {
		t.Error("repaired next to a held zone")
	}
	b.Held[world.DownDeep] = false
	s.m.repairs(w, s.r)
	if w.Space.Broken[0] || b.Repairs != 1 {
		t.Errorf("broken %v, %d repairs; want repaired", w.Space.Broken, b.Repairs)
	}
}

// ODD §1.2, Collapse: air below 0.2 for 30 days in a row collapses the
// silo on the 30th, ends a running rebellion with that ending and stops
// the run, unless output.runAfterCollapse; a population below P_min
// collapses it too.
func TestCollapse(t *testing.T) {
	for _, after := range []bool{false, true} {
		s := newTiny(t)
		s.add(30, 1, 1)
		s.cfg.Output.RunAfterCollapse = after
		s.start()
		w, b := s.w, s.w.Rebellion
		b.RebellionStart = w.Day - 10
		w.Silo.AirQuality = 0.19
		for range 29 {
			s.m.collapse(w)
		}
		w.Silo.WaterSupply = 0.1 // too late to matter: 1 day
		if b.Collapsed != world.None || w.Over {
			t.Fatal("collapse after 29 days of bad air")
		}
		w.Events = nil
		s.m.collapse(w)
		if b.Collapsed != w.Day || w.Over == after || b.RebellionStart != world.None {
			t.Fatalf("collapsed %d, over %v, rebellion %d", b.Collapsed, w.Over, b.RebellionStart)
		}
		if ev := w.Events[0].(world.CollapseEvent); ev.Cause != "air" || w.Events[1].(world.RebellionEvent).Ending != "collapse" {
			t.Errorf("events %v, want collapse (air) and the rebellion ended by it", w.Events)
		}
	}
	s := newTiny(t)
	s.add(30, 1, 1)
	s.cfg.Rebellion.PointOfNoReturn = 2
	s.start()
	s.m.collapse(s.w)
	if s.w.Rebellion.Collapsed != s.w.Day || s.w.Events[0].(world.CollapseEvent).Cause != "population" {
		t.Errorf("1 person, P_min 2: events %v", s.w.Events)
	}
}
