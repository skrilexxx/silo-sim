package beliefs

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

// tiny is a two-floor world (everyone lives on floor 1); people are added with add, then start runs Init.
type tiny struct {
	t   *testing.T
	cfg *config.Scenario
	w   *world.World
	m   *Module
	r   *rng.Stream
}

func newTiny(t *testing.T) *tiny {
	// The first step is a recompute day (ODD §6.12, Upbringing).
	talks, _ := rng.NewKeyed(1, "talk")
	n := &world.Network{Part: []int{0, 0}, Talks: talks}
	for k := range world.NumTieKinds {
		n.Chance[k] = 1 // every tie is used every day, unless a test says otherwise
	}
	w := &world.World{Day: 1200*engine.DaysPerMonth - 1, Floors: []world.Floor{{Number: 1, Homes: 9}, {Number: 2, Homes: 9}},
		Space: &world.Space{}, Network: n}
	r, _ := rng.New(1, "beliefs")
	cfg := &config.Scenario{}
	cfg.Work.ShadowAge, cfg.Work.WorkerAge = 14, 18
	cfg.Beliefs.TaughtLegitimacy, cfg.Beliefs.EducationWeight, cfg.Beliefs.LegitimacyCostPerFlag = 0.8, 0.5, 0.15
	cfg.Beliefs.HalfLifeHeardYears, cfg.Beliefs.HalfLifeLivedYears = 5, 20
	cfg.Beliefs.RelicHiddenYears, cfg.Beliefs.RelicKeptYears, cfg.Beliefs.CueChance = 50, 10, 0.05
	return &tiny{t: t, cfg: cfg, w: w, m: New(), r: r}
}

// add puts a person of the given age in years and stage in their own
// household and returns their ID.
func (s *tiny) add(ageYears int, stage world.Stage) world.PersonID {
	w := s.w
	id, hh := world.PersonID(len(w.People)), world.HouseholdID(len(w.Households))
	w.People = append(w.People, world.Person{ID: id, Alive: true, BirthDay: w.Day - world.Tick(ageYears*year), Household: hh,
		Stage: stage, Partner: world.None, Mother: world.None, Father: world.None, Master: world.None, BabyFather: world.None})
	w.Alive = append(w.Alive, id)
	w.Households = append(w.Households, world.Household{ID: hh, Floor: 1, Members: []world.PersonID{id}})
	return id
}

// join moves person id into household hh.
func (s *tiny) join(id world.PersonID, hh world.HouseholdID) {
	w := s.w
	old := w.People[id].Household
	w.Households[old].Members = slices.DeleteFunc(w.Households[old].Members, func(o world.PersonID) bool { return o == id })
	w.People[id].Household = hh
	w.Households[hh].Members = append(w.Households[hh].Members, id)
}

func (s *tiny) start() {
	s.startWith(0.5)
}

// startWith starts the module with pass chance π; the network's indexes
// are sized to the people added so far.
func (s *tiny) startWith(pass float64) {
	s.t.Helper()
	n := s.w.Network
	n.Weak, n.Children = make([][]world.Tie, len(s.w.People)), make([][]world.PersonID, len(s.w.People))
	s.cfg.Beliefs.PassChance = pass
	if err := s.m.Init(s.w, s.cfg, s.r); err != nil {
		s.t.Fatal(err)
	}
}

func (s *tiny) step() {
	s.t.Helper()
	s.w.Day++
	if err := s.m.step(s.w, s.r); err != nil {
		s.t.Fatal(err)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

// ODD §6.12, Upbringing: base = ε L_edu + (1 − ε) × mean L of the living
// parents and, for a shadow, the master.
func TestUpbringing(t *testing.T) {
	s := newTiny(t)
	mother, father := s.add(40, world.Worker), s.add(42, world.Worker)
	child := s.add(5, world.Child)
	shadow := s.add(15, world.Shadow)
	orphan := s.add(10, world.Child)
	deadDad := s.add(50, world.Worker)
	halfOrphan := s.add(8, world.Child)
	p := s.w.People
	p[child].Mother, p[child].Father = mother, father
	p[shadow].Mother, p[shadow].Master = mother, father
	p[halfOrphan].Mother, p[halfOrphan].Father = mother, deadDad
	p[deadDad].Alive = false
	s.w.Alive = []world.PersonID{halfOrphan, mother, child, shadow, orphan, father} // not in ID order
	s.start()

	b := s.w.Beliefs
	b.Known[mother] = 1<<world.FlagHistory | 1<<world.FlagOutside // L = 0.8 − 2 × 0.15 = 0.5
	s.step()
	for _, c := range []struct {
		who  world.PersonID
		want float64
	}{
		{mother, 0.8}, {father, 0.8}, // adults keep the taught base
		{child, 0.4 + 0.5*(0.5+0.8)/2},
		{shadow, 0.4 + 0.5*(0.5+0.8)/2}, // the master counts like a parent
		{orphan, 0.8},
		{halfOrphan, 0.4 + 0.5*0.5}, // the dead father doesn't count
	} {
		if got := b.Base[c.who]; !near(got, c.want) {
			t.Errorf("person %d: base %v, want %v", c.who, got, c.want)
		}
	}
	if got := b.L(mother); !near(got, 0.5) {
		t.Errorf("mother's L %v, want 0.5", got)
	}
}

// ODD §6.12: the base is recomputed every 30 days, not in between, until
// worker age, then fixed for life at its last value.
func TestBaseMonthlyAndFixedAtWorkerAge(t *testing.T) {
	s := newTiny(t)
	mother := s.add(40, world.Worker)
	teen := s.add(18, world.Shadow)
	s.w.People[teen].BirthDay += 10 // reaches worker age 9 days after the first step
	s.w.People[teen].Mother = mother
	s.start()
	b := s.w.Beliefs
	b.Known[mother] = 1 << world.FlagLottery
	s.step() // a recompute day
	want := 0.4 + 0.5*0.65
	if !near(b.Base[teen], want) {
		t.Fatalf("base %v, want %v", b.Base[teen], want)
	}
	b.Known[mother] = 0
	for range engine.DaysPerMonth - 1 {
		s.step()
		if !near(b.Base[teen], want) {
			t.Fatalf("day %d: base changed to %v between recompute days", s.w.Day, b.Base[teen])
		}
	}
	s.step() // the next recompute day, after worker age
	if !near(b.Base[teen], want) {
		t.Errorf("base changed to %v after worker age, want it fixed at %v", b.Base[teen], want)
	}
}

// ODD §6.12, Legitimacy: L = max(0, base − δ × flags known).
func TestLegitimacyNeverBelowZero(t *testing.T) {
	b := &world.Beliefs{Base: []float64{0.2, 0.8}, Known: []world.Flags{0b10101, 0b11111}, Cost: 0.15}
	if got := b.L(0); got != 0 {
		t.Errorf("L %v, want 0", got)
	}
	if got := b.L(1); !near(got, 0.05) {
		t.Errorf("L %v, want 0.05", got)
	}
}

// ODD §6.12, State: risk aversion is drawn once per person, in ID order,
// from the module's stream, first thing in the step: day-1 people get the
// stream's first draws, and a newborn gets one at its first step, kept for
// life.
func TestRiskDrawnOnceInIDOrder(t *testing.T) {
	s := newTiny(t)
	s.add(30, world.Worker)
	s.add(3, world.Child)
	s.start()
	r, _ := rng.New(1, "beliefs")
	b := s.w.Beliefs
	for id := range 2 {
		if want := r.Float64(); b.Risk[id] != want {
			t.Errorf("person %d: R %v, want %v", id, b.Risk[id], want)
		}
	}
	s.w.People = append(s.w.People, world.Person{ID: 2, Alive: true, BirthDay: s.w.Day + 1, Household: 0,
		Mother: world.None, Father: world.None, Master: world.None, Partner: world.None, BabyFather: world.None})
	s.w.Alive = append(s.w.Alive, 2)
	s.step()
	if len(b.Risk) != 3 || b.Risk[2] < 0 || b.Risk[2] >= 1 || b.Risk[2] == b.Risk[1] {
		t.Fatalf("risk values %v", b.Risk)
	}
	newborn := b.Risk[2]
	s.step()
	if b.Risk[2] != newborn {
		t.Errorf("newborn's R changed from %v to %v", newborn, b.Risk[2])
	}
}

func TestNeedsNetwork(t *testing.T) {
	s := newTiny(t)
	s.w.Network = nil
	if err := s.m.Init(s.w, &config.Scenario{}, s.r); err == nil {
		t.Error("started without the network")
	}
}

// ODD §6.12, Telling and believing: a flag passes in a conversation with
// chance π × (1 − L of the listener). With π = 0.5, a listener who knows
// nothing (L 0.8) learns on a day with chance 0.1; one with L 0.65 (as if
// they knew a flag), with chance 0.175. 2,000 listeners in each group.
func TestPassChanceFallsWithListenersL(t *testing.T) {
	s := newTiny(t)
	const each = 2000
	tellA, tellB := s.add(40, world.Worker), s.add(40, world.Worker)
	var groupA, groupB []world.PersonID
	for range each {
		a, b := s.add(30, world.Worker), s.add(30, world.Worker)
		s.join(a, s.w.People[tellA].Household)
		s.join(b, s.w.People[tellB].Household)
		groupA, groupB = append(groupA, a), append(groupB, b)
	}
	s.start()
	bl := s.w.Beliefs
	for _, id := range groupB {
		bl.Base[id] = 0.65
	}
	bl.Learn(tellA, world.FlagHistory, false)
	bl.Learn(tellB, world.FlagHistory, false)
	s.step()
	for _, g := range []struct {
		who  []world.PersonID
		want float64
	}{{groupA, 0.1}, {groupB, 0.175}} {
		got := 0
		for _, id := range g.who {
			if bl.Known[id].Has(world.FlagHistory) {
				got++
			}
		}
		share, sd := float64(got)/each, math.Sqrt(g.want*(1-g.want)/each)
		if math.Abs(share-g.want) > 5*sd {
			t.Errorf("%.4f learned in a day, want %.3f ± %.4f", share, g.want, 5*sd)
		}
		t.Logf("learned in a day: %.4f (want %.3f)", share, g.want)
	}
}

// ODD §6.12: a flag learned today is passed on from tomorrow; nothing
// passes on a tie not used today, nor from the dead.
func TestPassedOnFromTomorrow(t *testing.T) {
	s := newTiny(t)
	a, b, c := s.add(40, world.Worker), s.add(40, world.Worker), s.add(20, world.Worker)
	s.join(b, s.w.People[a].Household)
	s.w.People[c].Mother = b // kin, living apart
	s.startWith(1)
	s.w.Network.Children[b] = []world.PersonID{c}
	bl := s.w.Beliefs
	for _, id := range []world.PersonID{a, b, c} {
		bl.Base[id] = 0 // believes nothing, so always convinced
	}
	bl.Learn(a, world.FlagSilos, false)
	s.step()
	if !bl.Known[b].Has(world.FlagSilos) || bl.Known[c] != 0 {
		t.Fatalf("day 1: b knows %b, c knows %b; want b to know, c not yet", bl.Known[b], bl.Known[c])
	}
	s.step()
	if !bl.Known[c].Has(world.FlagSilos) {
		t.Fatalf("day 2: c knows %b, want silos", bl.Known[c])
	}

	s = newTiny(t)
	a, b, c = s.add(40, world.Worker), s.add(40, world.Worker), s.add(40, world.Worker)
	s.join(b, s.w.People[a].Household)
	s.join(c, s.w.People[a].Household)
	s.startWith(1)
	bl = s.w.Beliefs
	bl.Base[b], bl.Base[c] = 0, 0
	bl.Learn(a, world.FlagOutside, false)
	s.w.Network.Chance[world.TieHousehold] = 0
	s.step()
	s.w.Network.Chance[world.TieHousehold] = 1
	s.w.People[a].Alive = false
	s.step()
	if bl.Known[b] != 0 || bl.Known[c] != 0 {
		t.Errorf("b knows %b, c %b; want nothing, with no talk and then a dead teller", bl.Known[b], bl.Known[c])
	}
	if len(bl.Knowers) != 0 {
		t.Errorf("knowers %v, want the dead one dropped", bl.Knowers)
	}
}

// ODD §6.12, Forgetting: a flag known as heard is lost with a half-life of
// T_heard, one known as lived with T_lived; L comes back with it. Here 1
// and 2 years, 4,000 people each, nobody tied to anybody.
func TestHalfLives(t *testing.T) {
	s := newTiny(t)
	s.cfg.Beliefs.HalfLifeHeardYears, s.cfg.Beliefs.HalfLifeLivedYears = 1, 2
	const each = 4000
	var heard, lived []world.PersonID
	for range each {
		heard, lived = append(heard, s.add(30, world.Worker)), append(lived, s.add(30, world.Worker))
	}
	s.start()
	b := s.w.Beliefs
	for i := range each {
		b.Learn(heard[i], world.FlagScreen, false)
		b.Learn(lived[i], world.FlagScreen, true)
	}
	kept := func(group []world.PersonID) float64 {
		n := 0
		for _, id := range group {
			if b.Known[id].Has(world.FlagScreen) {
				n++
			} else if b.L(id) != b.Base[id] || b.Lived[id] != 0 {
				t.Fatalf("person %d forgot, but L %v (base %v), lived %b", id, b.L(id), b.Base[id], b.Lived[id])
			}
		}
		return float64(n) / each
	}
	for _, c := range []struct {
		years                int
		wantHeard, wantLived float64
	}{{1, 0.5, math.Sqrt(0.5)}, {2, 0.25, 0.5}} {
		for s.w.Day < 1200*engine.DaysPerMonth-1+world.Tick(c.years*year) {
			s.step()
		}
		for _, g := range []struct {
			name  string
			group []world.PersonID
			want  float64
		}{{"heard", heard, c.wantHeard}, {"lived", lived, c.wantLived}} {
			got, sd := kept(g.group), math.Sqrt(g.want*(1-g.want)/each)
			t.Logf("after %d years, %s kept by %.4f (want %.4f)", c.years, g.name, got, g.want)
			if math.Abs(got-g.want) > 5*sd {
				t.Errorf("after %d years, %s kept by %.4f, want %.4f ± %.4f", c.years, g.name, got, g.want, 5*sd)
			}
		}
	}
	if n := len(b.Knowers); n != int(math.Round((kept(heard)+kept(lived))*each)) {
		t.Errorf("%d knowers listed, but %v know", n, (kept(heard)+kept(lived))*each)
	}
}

// ODD §6.12, Scripted events: revealHistory teaches everyone alive aged 14
// or more the history as lived, makes lived what was only heard, and
// passes on from the next day.
func TestRevealHistory(t *testing.T) {
	s := newTiny(t)
	adult, teen, child, dead, heard := s.add(30, world.Worker), s.add(14, world.Shadow), s.add(10, world.Child), s.add(50, world.Worker), s.add(40, world.Worker)
	s.join(child, s.w.People[adult].Household)
	s.w.People[dead].Alive = false
	s.w.Alive = slices.DeleteFunc(s.w.Alive, func(id world.PersonID) bool { return id == dead })
	s.startWith(1)
	b := s.w.Beliefs
	b.Learn(heard, world.FlagHistory, false)

	types := s.m.EventTypes()
	if len(types) != 1 || types[0].Name != "revealHistory" || types[0].Phase != engine.PhaseBeliefs {
		t.Fatalf("event types %+v", types)
	}
	s.w.Day++
	if err := types[0].Fire(s.w, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.m.step(s.w, s.r); err != nil {
		t.Fatal(err)
	}
	for _, id := range []world.PersonID{adult, teen, heard} {
		if !b.Known[id].Has(world.FlagHistory) || !b.Lived[id].Has(world.FlagHistory) {
			t.Errorf("person %d: known %b, lived %b; want the history, lived", id, b.Known[id], b.Lived[id])
		}
	}
	if b.Known[child] != 0 || b.Known[dead] != 0 {
		t.Errorf("child knows %b, the dead %b; want nothing, not even from a conversation today", b.Known[child], b.Known[dead])
	}
	b.Base[child] = 0 // convinced by anything (after today's recompute)
	s.step()
	if !b.Known[child].Has(world.FlagHistory) || b.Lived[child] != 0 {
		t.Errorf("next day the child knows %b (lived %b); want the history, heard", b.Known[child], b.Lived[child])
	}
}

// skip lands each item with chance p, independently: the share landing
// and the share of landings right after one match p.
func TestSkip(t *testing.T) {
	r, _ := rng.New(1, "beliefs")
	const n = 1_000_000
	for _, p := range []float64{0, 0.0005, 0.01, 0.3, 1} {
		sk := newSkip(r, p)
		landed, pairs, prev := 0, 0, false
		for range n {
			l := sk.land(r)
			if l {
				landed++
				if prev {
					pairs++
				}
			}
			prev = l
		}
		share, sd := float64(landed)/n, math.Sqrt(p*(1-p)/n)
		if math.Abs(share-p) > 5*sd+1e-12 {
			t.Errorf("p = %v: %.6f landed, want ± %.6f", p, share, 5*sd)
		}
		if landed > 0 && p < 1 {
			if after, sd2 := float64(pairs)/float64(landed), math.Sqrt(p*(1-p)/float64(landed)); math.Abs(after-p) > 5*sd2+1e-12 {
				t.Errorf("p = %v: %.4f of landings followed one, want %v", p, after, p)
			}
		}
	}
}

// everyDay is the years that make a daily chance 1/(365 × years) equal 1.
const everyDay = 1.0 / year

// ODD §6.12, Relics: a hidden relic is found by someone aged 14 or more
// whose day location is its floor, who holds it and, as the relic works
// on them, learns the history as lived; with nobody there it stays hidden.
func TestRelicFound(t *testing.T) {
	s := newTiny(t)
	s.cfg.Beliefs.RelicHiddenYears, s.cfg.Beliefs.CueChance = everyDay, 1
	child, adult := s.add(10, world.Child), s.add(30, world.Worker)
	s.start()
	b := s.w.Beliefs
	b.Relics = []world.Relic{{Floor: 2, Holder: world.None}, {Floor: 1, Holder: world.None}}
	s.step()
	if got := b.Relics[0]; got.Holder != world.None || got.Floor != 2 {
		t.Errorf("relic on an empty floor: %+v, want it still hidden", got)
	}
	if got := b.Relics[1]; got.Holder != adult || got.Floor != 1 {
		t.Errorf("relic on floor 1: %+v, want held by the adult %d, not the child %d", got, adult, child)
	}
	if !b.Lived[adult].Has(world.FlagHistory) {
		t.Errorf("finder knows %b (lived %b), want the history, lived", b.Known[adult], b.Lived[adult])
	}
}

// ODD §6.12, Relics: a hidden relic is found with daily chance 1/(365 ×
// T_find): with T_find = 1 year, 1 − (1 − 1/365)^365 = 63.3% within a
// year. 2,000 relics.
func TestRelicFindRate(t *testing.T) {
	s := newTiny(t)
	s.cfg.Beliefs.RelicHiddenYears, s.cfg.Beliefs.RelicKeptYears = 1, 1e9
	s.add(30, world.Worker)
	s.start()
	b := s.w.Beliefs
	for range 2000 {
		b.Relics = append(b.Relics, world.Relic{Floor: 1, Holder: world.None})
	}
	for range year {
		s.step()
	}
	found := 0
	for _, r := range b.Relics {
		if r.Holder != world.None {
			found++
		}
	}
	want := 1 - math.Pow(1-1.0/year, year)
	got, sd := float64(found)/2000, math.Sqrt(want*(1-want)/2000)
	t.Logf("found within a year: %.4f (want %.4f)", got, want)
	if math.Abs(got-want) > 5*sd {
		t.Errorf("found within a year: %.4f, want %.4f ± %.4f", got, want, 5*sd)
	}
}

// ODD §6.12, Relics: a holder hands each relic over with daily chance
// 1/(365 × T_keep), to someone aged 14 or more they talk with today; with
// T_keep = 1 year, 2,000 relics between two partners who talk daily change
// hands on 1/365 of the relic-days. Never to a child, nor to someone not
// talked with.
func TestRelicHandOver(t *testing.T) {
	s := newTiny(t)
	s.cfg.Beliefs.RelicKeptYears = 1
	a, b := s.add(30, world.Worker), s.add(30, world.Worker)
	s.join(b, s.w.People[a].Household)
	s.start()
	bl := s.w.Beliefs
	for range 2000 {
		bl.Relics = append(bl.Relics, world.Relic{Floor: 1, Holder: a})
	}
	moves := 0
	for range year {
		before := slices.Clone(bl.Relics)
		s.step()
		for i := range before {
			if before[i].Holder != bl.Relics[i].Holder {
				moves++
			}
		}
	}
	want := 1.0 / year
	got, sd := float64(moves)/(2000*year), math.Sqrt(want*(1-want)/(2000*year))
	t.Logf("handed over on %.5f of relic-days (want %.5f)", got, want)
	if math.Abs(got-want) > 5*sd {
		t.Errorf("handed over on %.5f of relic-days, want %.5f ± %.5f", got, want, 5*sd)
	}

	s = newTiny(t)
	s.cfg.Beliefs.RelicKeptYears = everyDay
	holder, child, partner := s.add(30, world.Worker), s.add(10, world.Child), s.add(30, world.Worker)
	s.join(child, s.w.People[holder].Household)
	s.join(partner, s.w.People[holder].Household)
	s.start()
	bl = s.w.Beliefs
	bl.Relics = []world.Relic{{Floor: 1, Holder: holder}}
	s.w.Network.Chance[world.TieHousehold] = 0 // nobody talks
	s.step()
	if bl.Relics[0].Holder != holder {
		t.Errorf("handed to %d without a conversation", bl.Relics[0].Holder)
	}
	s.w.Network.Chance[world.TieHousehold] = 1
	for range 20 {
		s.step()
		if h := bl.Relics[0].Holder; h == child {
			t.Fatal("handed to a child")
		}
	}
}

// ODD §6.12, Relics: when a holder dies, a relic goes to a household
// member aged 14 or more; with none, it stays hidden in the flat.
func TestRelicInherited(t *testing.T) {
	s := newTiny(t)
	s.cfg.Beliefs.RelicKeptYears = 1e9
	old, heir, lone, child := s.add(70, world.Retired), s.add(40, world.Worker), s.add(70, world.Retired), s.add(10, world.Child)
	s.join(heir, s.w.People[old].Household)
	s.join(child, s.w.People[lone].Household)
	s.start()
	b := s.w.Beliefs
	b.Relics = []world.Relic{{Floor: 1, Holder: old}, {Floor: 1, Holder: lone}}
	for _, id := range []world.PersonID{old, lone} {
		s.w.People[id].Alive = false
		hh := &s.w.Households[s.w.People[id].Household]
		hh.Members = slices.DeleteFunc(hh.Members, func(o world.PersonID) bool { return o == id })
	}
	s.w.Alive = []world.PersonID{heir, child}
	s.step()
	if got := b.Relics[0]; got.Holder != heir {
		t.Errorf("relic of the dead: %+v, want held by the heir %d", got, heir)
	}
	if got := b.Relics[1]; got.Holder != world.None || got.Floor != 1 {
		t.Errorf("relic of the dead with only a child at home: %+v, want hidden on floor 1", got)
	}
}

// ODD §6.12, Relics: holding several relics gives no more draws than one:
// 2,000 holders of 3 relics each learn the history on a day with chance
// ρ = 0.05, not 1 − 0.95³ = 0.14.
func TestRelicsWorkOncePerDay(t *testing.T) {
	s := newTiny(t)
	s.cfg.Beliefs.RelicKeptYears = 1e9
	var holders []world.PersonID
	for range 2000 {
		holders = append(holders, s.add(30, world.Worker))
	}
	s.start()
	b := s.w.Beliefs
	for _, h := range holders {
		for range 3 {
			b.Relics = append(b.Relics, world.Relic{Floor: 1, Holder: h})
		}
	}
	s.step()
	learned := 0
	for _, h := range holders {
		if b.Known[h].Has(world.FlagHistory) {
			learned++
		}
	}
	got, sd := float64(learned)/2000, math.Sqrt(0.05*0.95/2000)
	t.Logf("learned from relics in a day: %.4f (want 0.05)", got)
	if math.Abs(got-0.05) > 5*sd {
		t.Errorf("learned in a day: %.4f, want 0.05 ± %.4f", got, 5*sd)
	}
}
