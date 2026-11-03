package control

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

// tiny is a four-floor world: homes on floors 1 and 3, the court on
// floor 2, the mines on floor 4. Its departments are general, it, medical,
// judicial and sheriff. The beliefs and space states are sized to the
// people added; nothing else runs, and walk stands in for the location
// step.
type tiny struct {
	t   *testing.T
	cfg *config.Scenario
	w   *world.World
	m   *Module
	r   *rng.Stream
}

func newTiny(t *testing.T) *tiny {
	w := &world.World{Day: 100 * year,
		Floors: []world.Floor{{Number: 1, Homes: 99, Zone: world.UpTop}, {Number: 2, Zone: world.UpTop}, {Number: 3, Homes: 99, Zone: world.Mids},
			{Number: 4, Zone: world.DownDeep, Function: world.Mechanical}},
		Depts: []world.Department{{Name: "general"}, {Name: "it", Coverage: 1}, {Name: "medical"}, {Name: "judicial", Headcount: 100}, {Name: "sheriff"}},
		Space: &world.Space{Broken: make([]bool, 3), DeptFloors: [][]int{nil, nil, nil, {2}, nil}, DescentHours: 3, WalkHours: 8}}
	r, _ := rng.New(1, "control")
	cfg := &config.Scenario{}
	cfg.Work.ShadowAge, cfg.Work.MasterMinSkill = 14, 0.5
	cfg.Control.Surveillance, cfg.Control.ArrestsPerOfficer, cfg.Control.MinesFloor = 1, 1, 4
	cfg.Control.FindChance, cfg.Control.NameChance = 1, 1
	cfg.Control.CleaningLift, cfg.Control.LiftHalfLifeYears = 0.1, 1
	cfg.Control.FearPerTrial, cfg.Control.FearHalfLifeYears = 0.1, 1
	cfg.Pact = []config.Rule{
		{ID: "noRelics", Active: true, Offences: []string{"holdRelic", "tradeRelic"}, Detection: 1,
			First: config.Penalty{Confiscate: true, Prison: 90}, Repeat: &config.Penalty{Confiscate: true, Prison: 365}},
		{ID: "noForbiddenTalk", Active: true, Offences: []string{"speakForbidden"}, Detection: 1,
			First: config.Penalty{FlagFamily: true}, Repeat: &config.Penalty{Prison: 180, FlagFamily: true}},
		{ID: "noOrganising", Active: false, Offences: []string{"organise"}, Detection: 1},
		{ID: "noViolence", Active: true, Offences: []string{"violence"}, Detection: 1, First: config.Penalty{Mines: 730}, Visibility: 0.8},
		{ID: "goingOut", Active: true, Offences: []string{"askToGoOut"}, Detection: 1, First: config.Penalty{Cleaning: true}, Visibility: 0.5},
	}
	return &tiny{t: t, cfg: cfg, w: w, m: New(), r: r}
}

// add puts a person of the given age, stage, department and skill in their
// own household and returns their ID.
func (s *tiny) add(ageYears int, stage world.Stage, dept int, skill float64) world.PersonID {
	w := s.w
	id, hh := world.PersonID(len(w.People)), world.HouseholdID(len(w.Households))
	w.People = append(w.People, world.Person{ID: id, Alive: true, BirthDay: w.Day - world.Tick(ageYears*year), Household: hh,
		Stage: stage, Department: dept, Skill: skill, Partner: world.None, Mother: world.None, Father: world.None, Master: world.None})
	w.Alive = append(w.Alive, id)
	w.Households = append(w.Households, world.Household{ID: hh, Floor: 1, Members: []world.PersonID{id}})
	return id
}

const (
	general  = 0
	it       = 1
	judicial = 3
)

// kill marks a person dead and takes them off the living list, as the
// deaths step does.
func (s *tiny) kill(id world.PersonID) {
	s.w.People[id].Alive = false
	s.w.Alive = slices.DeleteFunc(s.w.Alive, func(x world.PersonID) bool { return x == id })
}

func (s *tiny) start() {
	s.t.Helper()
	n := len(s.w.People)
	s.w.Beliefs = &world.Beliefs{Base: make([]float64, n), Known: make([]world.Flags, n), Lived: make([]world.Flags, n),
		Hidden: make([]world.Flags, n), DosedUntil: make([]world.Tick, n), IsMember: make([]bool, n), Water: world.None}
	sp := s.w.Space
	sp.Location, sp.From, sp.Dest, sp.Away = make([]int, n), make([]int, n), make([]int, n), make([]int, n)
	sp.Arrive, sp.OffWork = make([]world.Tick, n), make([]bool, n)
	for id := range n {
		sp.Location[id] = s.w.Households[s.w.People[id].Household].Floor
	}
	if err := s.m.Init(s.w, s.cfg, s.r); err != nil {
		s.t.Fatal(err)
	}
}

// step runs a day: the control step, then walk.
func (s *tiny) step() {
	s.t.Helper()
	s.w.Day++
	if err := s.m.step(s.w, s.r); err != nil {
		s.t.Fatal(err)
	}
	s.walk()
}

// walk does what the location step does with trips (ODD §6.10): a trip
// due tomorrow ends; arriving at home ends being away, except for someone
// held; anyone else away stays where they are.
func (s *tiny) walk() {
	sp, c := s.w.Space, s.w.Control
	for _, id := range s.w.Alive {
		if int(id) >= len(sp.Dest) { // born after the start: never travels here
			continue
		}
		home := s.w.Households[s.w.People[id].Household].Floor
		switch {
		case sp.Dest[id] != 0 && sp.Arrive[id] > s.w.Day+1:
			sp.Location[id] = 0
		case sp.Dest[id] != 0:
			sp.Away[id], sp.Dest[id] = sp.Dest[id], 0
			if sp.Away[id] == home && !c.Held(id) {
				sp.Away[id] = 0
			}
			sp.Location[id] = sp.Away[id]
			if sp.Away[id] == 0 {
				sp.Location[id] = home
			}
		}
	}
}

// days runs n days.
func (s *tiny) days(n int) {
	s.t.Helper()
	for range n {
		s.step()
	}
}

func (s *tiny) heads() []world.HeadOfITEvent {
	var out []world.HeadOfITEvent
	for _, e := range s.w.Events {
		if h, ok := e.(world.HeadOfITEvent); ok {
			out = append(out, h)
		}
	}
	return out
}

// ODD §6.13, Head of IT: on day 1 the most skilled IT worker (ties to the
// lower ID; shadows, the retired and other departments don't count) takes
// the post, learns all five flags as lived and leaves the Flamekeepers.
func TestFirstHeadOfIT(t *testing.T) {
	s := newTiny(t)
	s.add(40, world.Worker, general, 0.99)
	s.add(70, world.Retired, it, 0.99)
	a := s.add(40, world.Worker, it, 0.9)
	b := s.add(45, world.Worker, it, 0.9)
	s.add(30, world.Worker, it, 0.6)
	s.start()
	c := s.w.Control
	if c.Head != a {
		t.Fatalf("Head %d, want %d (most skilled IT worker, lower ID on a tie with %d)", c.Head, a, b)
	}
	all := world.Flags(1<<world.NumFlags - 1)
	if s.w.Beliefs.Known[a] != all || s.w.Beliefs.Lived[a] != all {
		t.Errorf("Head knows %05b, lived %05b; want all five lived", s.w.Beliefs.Known[a], s.w.Beliefs.Lived[a])
	}
	if h := s.heads(); len(h) != 1 || h[0].Head != a || h[0].Successor {
		t.Errorf("events %+v, want one headOfIT for %d, not a successor", h, a)
	}

	// A Flamekeeper taking office leaves the group.
	s2 := newTiny(t)
	member := s2.add(40, world.Worker, it, 0.9)
	other := s2.add(40, world.Worker, general, 0.9)
	n := len(s2.w.People)
	s2.w.Beliefs = &world.Beliefs{Base: make([]float64, n), Known: make([]world.Flags, n), Lived: make([]world.Flags, n),
		Hidden: make([]world.Flags, n), DosedUntil: make([]world.Tick, n), IsMember: []bool{true, true},
		Members: []world.PersonID{other, member}, Water: world.None}
	if err := s2.m.Init(s2.w, s2.cfg, s2.r); err != nil {
		t.Fatal(err)
	}
	if b := s2.w.Beliefs; b.IsMember[member] || !slices.Equal(b.Members, []world.PersonID{other}) {
		t.Errorf("members %v, IsMember %v; want the Head out of the group", b.Members, b.IsMember)
	}
}

// ODD §6.13, Successor: a Head names their own shadow, the older of two;
// a Head with none takes the youngest IT shadow as their own, if they can
// be a master.
func TestSuccessorIsTheHeadsShadow(t *testing.T) {
	s := newTiny(t)
	head := s.add(50, world.Worker, it, 0.9)
	mate := s.add(50, world.Worker, it, 0.8)
	young := s.add(14, world.Shadow, it, 0.1)
	old := s.add(16, world.Shadow, it, 0.1)
	other := s.add(15, world.Shadow, it, 0.1)
	s.add(14, world.Shadow, general, 0.1) // younger, but not IT
	p := s.w.People
	p[young].Master, p[old].Master, p[other].Master = head, head, mate
	s.start()
	if got := s.w.Control.Successor; got != old {
		t.Errorf("successor %d, want the Head's older shadow %d", got, old)
	}

	s = newTiny(t)
	head = s.add(50, world.Worker, it, 0.9)
	mate = s.add(50, world.Worker, it, 0.8)
	young = s.add(14, world.Shadow, it, 0.1)
	old = s.add(16, world.Shadow, it, 0.1)
	s.w.People[young].Master, s.w.People[old].Master = mate, mate
	s.start()
	if got := s.w.Control.Successor; got != young || s.w.People[young].Master != head {
		t.Errorf("successor %d with master %d, want the youngest IT shadow %d taken on by the Head %d", got, s.w.People[young].Master, young, head)
	}

	s = newTiny(t)
	s.add(50, world.Worker, it, 0.5) // not above the master skill
	s.add(14, world.Shadow, it, 0.1)
	s.start()
	if got := s.w.Control.Successor; got != world.None {
		t.Errorf("successor %d, want none: the Head can't be a master", got)
	}
}

// ODD §6.13, Leaving office: the post ends only with death. The Head
// keeps it on sick leave and in retirement and can't be charged; at
// their death the successor takes it even as a shadow; with no successor
// the most skilled IT worker does; with no IT worker it stays empty.
func TestSuccession(t *testing.T) {
	s := newTiny(t)
	head := s.add(60, world.Worker, it, 0.9)
	best := s.add(50, world.Worker, it, 0.8)
	shadow := s.add(15, world.Shadow, it, 0.3)
	s.w.People[shadow].Master = head
	s.start()
	c := s.w.Control
	for _, stage := range []world.Stage{world.SickLeave, world.Retired} {
		s.w.People[head].Stage = stage
		s.offend(head, world.TradeRelic)
		s.step()
		if c.Head != head || c.Successor != shadow || c.Charges[head] != 0 || c.Held(head) {
			t.Fatalf("stage %d: Head %d, successor %d, charges %b; want %d, %d, never charged", stage, c.Head, c.Successor, c.Charges[head], head, shadow)
		}
	}

	s.kill(head)
	s.step()
	if c.Head != shadow || c.Successor != world.None {
		t.Fatalf("Head %d, successor %d; want the shadow %d, with none of their own yet", c.Head, c.Successor, shadow)
	}
	s.kill(shadow)
	s.step()
	if c.Head != best {
		t.Fatalf("Head %d, want the most skilled IT worker %d", c.Head, best)
	}
	h := s.heads()
	if len(h) != 3 || h[0].Successor || !h[1].Successor || h[2].Successor {
		t.Errorf("events %+v, want three, the second by succession", h)
	}

	s.kill(best)
	s.step()
	if c.Head != world.None {
		t.Errorf("Head %d, want none", c.Head)
	}
}

// ODD §6.13, Detection: an act is seen with chance min(1, σ × IT coverage
// × the rule's weight). Here 0.5 × 0.8 × 0.5 = 0.2, over 10,000 acts.
func TestDetectionRate(t *testing.T) {
	s := newTiny(t)
	s.cfg.Control.Surveillance, s.cfg.Pact[1].Detection = 0.5, 0.5
	var ids []world.PersonID
	for range 100 {
		ids = append(ids, s.add(30, world.Worker, general, 0.5))
	}
	s.start()
	s.w.Depts[it].Coverage = 0.8
	const n = 10000
	for i := range n {
		s.w.Control.Report(world.Act{Person: ids[i%len(ids)], Other: world.None, Offence: world.SpeakForbidden})
	}
	s.step()
	p := 0.2
	sd := math.Sqrt(n * p * (1 - p))
	if got := float64(s.w.Control.Tally.Seen); math.Abs(got-n*p) > 4*sd {
		t.Errorf("%v of %d acts seen, want %v ± %.0f", got, n, n*p, 4*sd)
	}
	t.Logf("%d of %d acts seen (want about %v)", s.w.Control.Tally.Seen, n, n*p)
	if len(s.w.Control.Acts) != 0 {
		t.Errorf("%d acts left after the step", len(s.w.Control.Acts))
	}
}

// ODD §6.13, Offences and Cases: an act no active rule lists, or by a
// child under 14, is not an offence; a seen handover charges both; a
// second charge joins the open case; a case dies with its person; with σ
// = 0 nothing is seen and nothing is drawn.
func TestCases(t *testing.T) {
	s := newTiny(t)
	a := s.add(30, world.Worker, general, 0.5)
	b := s.add(30, world.Worker, general, 0.5)
	kid := s.add(13, world.Child, world.None, 0)
	s.cfg.Control.ArrestsPerOfficer = 0 // nobody arrested
	s.start()
	c := s.w.Control
	for _, act := range []world.Act{
		{Person: a, Other: world.None, Offence: world.Organise},    // the rule is inactive
		{Person: kid, Other: world.None, Offence: world.HoldRelic}, // too young
		{Person: b, Other: world.None, Offence: world.Sabotage},    // no rule lists it
		{Person: a, Other: b, Offence: world.TradeRelic},
		{Person: a, Other: world.None, Offence: world.SpeakForbidden},
	} {
		c.Report(act)
	}
	s.step()
	if !slices.Equal(c.Cases, []world.PersonID{a, b}) || c.Charges[a] != 0b11 || c.Charges[b] != 0b01 || c.Charges[kid] != 0 {
		t.Fatalf("cases %v, charges %v; want a charged under rules 0 and 1, b under 0", c.Cases, c.Charges)
	}
	if c.Tally.Seen != 2 {
		t.Errorf("%d acts seen, want 2", c.Tally.Seen)
	}
	s.kill(a)
	s.step()
	if !slices.Equal(c.Cases, []world.PersonID{b}) || c.Charges[a] != 0 {
		t.Errorf("cases %v, charges %v; want the dead one's case dropped", c.Cases, c.Charges)
	}

	s = newTiny(t)
	s.cfg.Control.Surveillance = 0
	a = s.add(30, world.Worker, general, 0.5)
	s.start()
	before := *s.r
	s.w.Control.Report(world.Act{Person: a, Other: world.None, Offence: world.HoldRelic})
	s.step()
	if s.w.Control.Tally.Seen != 0 || len(s.w.Control.Cases) != 0 || *s.r != before {
		t.Errorf("σ = 0: %d seen, cases %v, stream moved %v", s.w.Control.Tally.Seen, s.w.Control.Cases, *s.r != before)
	}
}

// The world's offences and the scenario's names agree, in order.
func TestOffenceNames(t *testing.T) {
	for o := range world.NumOffences {
		if config.Offences[o] != o.String() {
			t.Errorf("offence %d: config %q, world %q", o, config.Offences[o], o.String())
		}
	}
	if len(config.Offences) != int(world.NumOffences) {
		t.Errorf("%d names in config, %d offences", len(config.Offences), world.NumOffences)
	}
}

// on adds an adult worker of General services at home on floor f.
func (s *tiny) on(f int) world.PersonID {
	id := s.add(30, world.Worker, general, 0.5)
	s.w.Households[s.w.People[id].Household].Floor = f
	return id
}

// offend reports one act, seen for sure in the tiny world (σ = 1, rule
// weight 1, IT coverage 1).
func (s *tiny) offend(id world.PersonID, o world.Offence) {
	s.w.Control.Report(world.Act{Person: id, Other: world.None, Offence: o})
}

// ODD §6.13, Arrests, Trial, Penalties: an arrest is a walk to the court;
// the trial on arrival gives the first-offence penalty, prison(90) here,
// served on the court floor; the release is a walk home; a second
// offence under the same rule gets the repeat penalty.
func TestArrestTrialPrisonRelease(t *testing.T) {
	s := newTiny(t)
	a := s.on(3)
	s.start()
	c, sp := s.w.Control, s.w.Space
	s.offend(a, world.TradeRelic)
	s.step()
	if c.Hold[a] != world.Custody || c.Tally.Arrests != 1 || len(c.Cases) != 0 || sp.Location[a] != 2 {
		t.Fatalf("after the arrest: hold %d, %d arrests, cases %v, at floor %d", c.Hold[a], c.Tally.Arrests, c.Cases, sp.Location[a])
	}
	s.step() // reached the court: tried
	tried := s.w.Day
	if c.Hold[a] != world.Prison || c.Until[a] != tried+90 || c.Charges[a] != 0 || c.Convicted[a] != 0b1 {
		t.Fatalf("after the trial: hold %d until %d (want %d), charges %b, convicted %b", c.Hold[a], c.Until[a], tried+90, c.Charges[a], c.Convicted[a])
	}
	s.days(90)
	if !c.Held(a) || sp.Location[a] != 2 {
		t.Fatalf("day %d: held %v at floor %d; want in prison on the court floor until day %d", s.w.Day, c.Held(a), sp.Location[a], tried+90)
	}
	s.step() // free, walking home
	s.step()
	if c.Held(a) || sp.Location[a] != 3 || sp.Away[a] != 0 {
		t.Fatalf("after the release: held %v, at floor %d, away %d; want home on floor 3", c.Held(a), sp.Location[a], sp.Away[a])
	}
	s.offend(a, world.TradeRelic)
	s.days(2)
	if c.Hold[a] != world.Prison || c.Until[a] != s.w.Day+365 {
		t.Errorf("repeat: hold %d until %d, want prison until %d", c.Hold[a], c.Until[a], s.w.Day+365)
	}
	if c.Tally.Prisons != 2 {
		t.Errorf("%d prison sentences, want 2", c.Tally.Prisons)
	}
}

// ODD §6.13, Penalties: a warning frees the person at once (and, with
// flagFamily, flags them and their partner, not their child); mines(730) is served on the mines floor; with several rules
// in one case the most severe detention is served, with every
// confiscation and flag.
func TestPenalties(t *testing.T) {
	s := newTiny(t)
	a, partner, kid := s.on(3), s.on(3), s.add(10, world.Child, world.None, 0)
	for _, o := range []world.PersonID{partner, kid} {
		s.w.People[o].Household = s.w.People[a].Household
		s.w.Households[s.w.People[a].Household].Members = append(s.w.Households[s.w.People[a].Household].Members, o)
	}
	s.w.People[a].Partner, s.w.People[partner].Partner = partner, a
	v := s.on(1)
	s.start()
	c, sp := s.w.Control, s.w.Space
	s.offend(a, world.SpeakForbidden)
	s.offend(v, world.Violence)
	s.days(2)
	if c.Held(a) || !c.Flagged[a] || !c.Flagged[partner] || c.Flagged[kid] || c.Flagged[v] || c.Tally.Warnings != 1 {
		t.Errorf("warning: held %v; flagged: offender %v, partner %v, child %v, other %v; %d warnings",
			c.Held(a), c.Flagged[a], c.Flagged[partner], c.Flagged[kid], c.Flagged[v], c.Tally.Warnings)
	}
	s.step()
	if c.Hold[v] != world.Mines || sp.Location[v] != 4 || c.Until[v] != s.w.Day-1+730 {
		t.Errorf("mines: hold %d at floor %d until %d", c.Hold[v], sp.Location[v], c.Until[v])
	}

	s = newTiny(t)
	a = s.on(3)
	s.start()
	c = s.w.Control
	s.w.Beliefs.Relics = []world.Relic{{Floor: 3, Holder: a}, {Floor: 1, Holder: world.None}}
	c.Charges[a] = 0b11 // noRelics (prison 90, confiscate) and noForbiddenTalk (flag)
	c.Cases = []world.PersonID{a}
	s.cfg.Control.FindChance = 0
	s.m.find = 0 // the arrest's search finds nothing; the confiscation still takes it
	s.days(2)
	if c.Hold[a] != world.Prison || c.Until[a] != s.w.Day+90 || !c.Flagged[a] || len(s.w.Beliefs.Relics) != 1 || c.Tally.Taken != 1 {
		t.Errorf("hold %d until %d, flagged %v, relics %v, taken %d; want prison 90, flagged, the held relic taken",
			c.Hold[a], c.Until[a], c.Flagged[a], s.w.Beliefs.Relics, c.Tally.Taken)
	}
}

// ODD §6.13, Arrests: at most ⌊a × officers⌋ a day, oldest case first; a
// case whose arrest can't start (the stairs to the court are broken)
// waits and uses no capacity.
func TestArrestCapacity(t *testing.T) {
	s := newTiny(t)
	s.cfg.Control.ArrestsPerOfficer = 0.02 // × 100 Judicial workers = 2 a day
	cut := s.on(3)
	var rest []world.PersonID
	for range 5 {
		rest = append(rest, s.on(1))
	}
	s.start()
	c := s.w.Control
	s.w.Space.Broken[1] = true // floors 2 and 3 apart
	s.offend(cut, world.SpeakForbidden)
	for _, id := range rest {
		s.offend(id, world.SpeakForbidden)
	}
	for day, want := range []int{2, 4, 5, 5} {
		s.step()
		if c.Tally.Arrests != want {
			t.Fatalf("day %d: %d arrests, want %d", day+1, c.Tally.Arrests, want)
		}
	}
	if !slices.Equal(c.Cases, []world.PersonID{cut}) || c.Held(cut) {
		t.Errorf("cases %v, want only the one cut off from the court", c.Cases)
	}
	for _, id := range rest[:2] {
		if c.Convicted[id] == 0 {
			t.Errorf("%d not tried: the oldest cases go first", id)
		}
	}
}

// ODD §6.13, Searches: an arrest searches the flat; a raid searches every
// flat on one residential floor. Found relics leave the silo and their
// holders are charged with holdRelic.
func TestSearches(t *testing.T) {
	s := newTiny(t)
	a, mate, up, down := s.on(3), s.on(3), s.on(1), s.on(3)
	hh := s.w.People[a].Household
	s.w.People[mate].Household = hh
	s.w.Households[hh].Members = []world.PersonID{a, mate}
	s.start()
	c, b := s.w.Control, s.w.Beliefs
	b.Relics = []world.Relic{{Floor: 3, Holder: mate}, {Floor: 1, Holder: up}, {Floor: 3, Holder: down}, {Floor: 2, Holder: world.None}}
	s.offend(a, world.SpeakForbidden)
	s.step()
	if c.Tally.Taken != 1 || c.Charges[mate] != 0b1 || len(b.Relics) != 3 {
		t.Fatalf("arrest search: %d taken, mate charged %b, relics %v", c.Tally.Taken, c.Charges[mate], b.Relics)
	}

	s = newTiny(t)
	s.cfg.Control.RaidsPerYear, s.cfg.Control.ArrestsPerOfficer = year, 0 // a raid every day, no arrests
	up, down = s.on(1), s.on(3)
	s.start()
	b = s.w.Beliefs
	b.Relics = []world.Relic{{Floor: 1, Holder: up}, {Floor: 3, Holder: down}, {Floor: 2, Holder: world.None}}
	s.step()
	if c := s.w.Control; c.Tally.Raids != 1 || c.Tally.Taken != 1 || len(b.Relics) != 2 || b.Relics[1].Holder != world.None {
		t.Errorf("raid: %d raids, %d taken, relics %v; want one floor's held relic taken, the hidden one kept", c.Tally.Raids, c.Tally.Taken, b.Relics)
	}
}

// ODD §6.13, Arrests: an arrested Flamekeeper loses their seat and names
// one other member (p_name = 1 here), who is arrested in turn and names
// another: the group is rolled up one by one (T10.5).
func TestFlamekeepersNamed(t *testing.T) {
	s := newTiny(t)
	members := []world.PersonID{s.on(1), s.on(3), s.on(1), s.on(3)}
	s.start()
	b, c := s.w.Beliefs, s.w.Control
	for _, id := range members {
		b.IsMember[id] = true
	}
	b.Members = slices.Clone(members)
	s.offend(members[2], world.SpeakForbidden)
	s.step()
	if len(b.Members) != 0 || c.Tally.Arrests != 4 {
		t.Errorf("members %v, %d arrests; want the whole group arrested", b.Members, c.Tally.Arrests)
	}
	for _, id := range members {
		if b.IsMember[id] || !c.Held(id) {
			t.Errorf("%d: member %v, held %v", id, b.IsMember[id], c.Held(id))
		}
	}
}

// ODD §6.11, §6.13: held people talk with nobody; a death ends custody.
func TestHeldTalkWithNobody(t *testing.T) {
	s := newTiny(t)
	a, b := s.on(1), s.on(1)
	s.w.Network = &world.Network{Part: []int{0, 0, 0, 0}}
	s.start()
	if !s.w.Network.Reachable(s.w, a, b) {
		t.Fatal("not reachable before the arrest")
	}
	s.offend(a, world.TradeRelic)
	s.days(2)
	if s.w.Network.Reachable(s.w, a, b) {
		t.Error("a prisoner can talk")
	}
	s.kill(a)
	s.step()
	if s.w.Control.Held(a) || slices.Contains(s.m.held, a) {
		t.Error("the dead are still held")
	}
}

// deathsStep stands in for the deaths step (ODD §6.1): the condemned die.
func (s *tiny) deathsStep() {
	for _, id := range s.w.Control.Doomed {
		s.kill(id)
	}
	s.w.Control.Doomed = s.w.Control.Doomed[:0]
}

// ODD §6.13, Cleaning: a cleaning sentence holds the condemned overnight;
// they die at the next deaths step, and that day's control step logs the
// cleaning and lifts L by λ × v; the lift halves every T_S. The trial
// adds φ × v to the fear, which halves every T_F.
func TestCleaningLiftsL(t *testing.T) {
	s := newTiny(t)
	a := s.on(1)
	s.on(3)
	s.start()
	c, b := s.w.Control, s.w.Beliefs
	s.offend(a, world.AskToGoOut)
	s.days(2) // arrested, tried
	if c.Hold[a] != world.Condemned || !slices.Equal(c.Doomed, []world.PersonID{a}) {
		t.Fatalf("hold %d, doomed %v", c.Hold[a], c.Doomed)
	}
	if want := 0.1 * 0.5; math.Abs(c.Fear-want*s.m.fearKeep) > 1e-12 {
		t.Errorf("fear %v after the trial and a day's decay, want %v", c.Fear, want*s.m.fearKeep)
	}
	s.deathsStep()
	s.step()
	ev := s.w.Events[len(s.w.Events)-1].(world.CleaningEvent)
	if ev.Person != a || ev.Rule != "goingOut" || ev.Failed || c.Tally.Cleanings != 1 || c.Held(a) {
		t.Errorf("event %+v, %d cleanings, still held %v", ev, c.Tally.Cleanings, c.Held(a))
	}
	if !slices.Equal(c.Cleaned, []world.PersonID{a}) { // ODD §6.14: their family grieves
		t.Errorf("cleaned today %v, want %v", c.Cleaned, []world.PersonID{a})
	}
	if want := 0.1 * 0.5 * s.m.liftKeep; math.Abs(b.Lift-want) > 1e-12 {
		t.Errorf("lift %v, want λ × v = 0.05, a day decayed: %v", b.Lift, want)
	}
	lift, fear := b.Lift, c.Fear
	s.days(year)
	if math.Abs(b.Lift-lift/2) > 1e-9 || math.Abs(c.Fear-fear/2) > 1e-9 {
		t.Errorf("a year on: lift %v (want %v), fear %v (want %v)", b.Lift, lift/2, c.Fear, fear/2)
	}
	b.Base[0] = 0.8
	b.Lift = 0.3
	if b.L(0) != 1 {
		t.Errorf("L %v, want it capped at 1", b.L(0))
	}
}

// ODD §6.13, Cleaning: a failed cleaning teaches every watcher aged 14 or
// more and not dosed the outside and the screen as lived, with no lift;
// with visibility v each watches with chance v. The scripted
// cleaningFailure is seen by everyone.
func TestCleaningFailure(t *testing.T) {
	s := newTiny(t)
	s.cfg.Control.CleaningFailChance = 1
	a := s.on(1)
	kid := s.add(10, world.Child, world.None, 0)
	dosed := s.on(3)
	var crowd []world.PersonID
	for range 2000 {
		crowd = append(crowd, s.on(3))
	}
	s.start()
	b := s.w.Beliefs
	b.DosedUntil[dosed] = s.w.Day + 100
	s.offend(a, world.AskToGoOut)
	s.days(2)
	s.deathsStep()
	s.step()
	both := world.Flags(1<<world.FlagOutside | 1<<world.FlagScreen)
	seen := 0
	for _, id := range crowd {
		if b.Known[id] == both && b.Lived[id] == both {
			seen++
		}
	}
	share, sd := float64(seen)/2000, math.Sqrt(0.25/2000)
	t.Logf("%.3f of the crowd watched (v = 0.5)", share)
	if math.Abs(share-0.5) > 5*sd || b.Known[kid] != 0 || b.Known[dosed] != 0 || b.Lift != 0 || s.w.Control.Tally.Failed != 1 {
		t.Errorf("watched %.3f, kid knows %05b, dosed %05b, lift %v, failed %d", share, b.Known[kid], b.Known[dosed], b.Lift, s.w.Control.Tally.Failed)
	}

	s = newTiny(t)
	a, kid = s.on(1), s.add(10, world.Child, world.None, 0)
	s.start()
	s.m.EventTypes()[0].Fire(s.w, s.r, config.Event{})
	if b := s.w.Beliefs; b.Known[a] != both || b.Known[kid] != 0 {
		t.Errorf("scripted: adult knows %05b, child %05b", b.Known[a], b.Known[kid])
	}
}

// ODD §6.13, Covert lottery denial: the N_doc most skilled Medical
// workers are the implant doctors; a post left empty goes to the next
// most skilled. Each denied win teaches every doctor not dosed the
// lottery flag, as lived. A child born to a flagged parent is counted.
func TestImplantDoctors(t *testing.T) {
	s := newTiny(t)
	s.cfg.Control.ImplantDoctors = 2
	const medical = 2
	a, b, d := s.add(50, world.Worker, medical, 0.9), s.add(50, world.Worker, medical, 0.8), s.add(50, world.Worker, medical, 0.7)
	s.add(70, world.Retired, medical, 0.99)
	mother := s.on(1)
	s.start()
	c, bl := s.w.Control, s.w.Beliefs
	if !slices.Equal(c.Doctors, []world.PersonID{a, b}) {
		t.Fatalf("doctors %v, want %v", c.Doctors, []world.PersonID{a, b})
	}
	s.kill(a)
	bl.DosedUntil[d] = s.w.Day + 10
	c.Denied = []world.PersonID{mother}
	s.step()
	if !slices.Equal(c.Doctors, []world.PersonID{b, d}) {
		t.Fatalf("doctors %v, want %v", c.Doctors, []world.PersonID{b, d})
	}
	if bl.Known[b] != 1<<world.FlagLottery || bl.Lived[b] != 1<<world.FlagLottery || bl.Known[d] != 0 || c.Tally.Denied != 1 || len(c.Denied) != 0 {
		t.Errorf("doctor knows %05b (lived %05b), dosed doctor %05b, %d denied", bl.Known[b], bl.Lived[b], bl.Known[d], c.Tally.Denied)
	}
	if !slices.Equal(c.DeniedToday, []world.PersonID{mother}) { // ODD §6.14: a loss for the couple
		t.Errorf("denied today %v, want %v", c.DeniedToday, []world.PersonID{mother})
	}

	c.Flagged[mother] = true
	for _, mum := range []world.PersonID{mother, b} {
		kid := s.add(0, world.Child, world.None, 0)
		s.w.People[kid].Mother = mum
	}
	s.step()
	if len(c.DeniedToday) != 0 {
		t.Errorf("denied today %v the next day, want none", c.DeniedToday)
	}
	if c.Tally.FlaggedBirths != 1 {
		t.Errorf("%d births to flagged parents, want 1", c.Tally.FlaggedBirths)
	}
}

// ODD §6.14, Fights: a rebel arrested in a fight is taken into custody at
// the next control step, charged with violence without a detection draw
// and using no arrest capacity, then tried at the court as anyone else.
// With no active rule listing violence there is nothing to charge, and
// they stay free.
func TestSeizedInFight(t *testing.T) {
	for _, ruled := range []bool{true, false} {
		s := newTiny(t)
		s.cfg.Control.ArrestsPerOfficer = 0 // no capacity at all
		s.cfg.Pact[3].Active = ruled        // noViolence: mines(730)
		a := s.on(3)
		s.start()
		c := s.w.Control
		c.Seized = []world.PersonID{a}
		s.step()
		if !ruled {
			if c.Held(a) || c.Tally.Arrests != 0 {
				t.Errorf("no rule against violence: held %v, %d arrests; want free", c.Held(a), c.Tally.Arrests)
			}
			continue
		}
		if c.Hold[a] != world.Custody || c.Charges[a] != 1<<3 || c.Tally.Arrests != 1 || len(c.Cases) != 0 || len(c.Seized) != 0 {
			t.Fatalf("hold %d, charges %b, %d arrests, cases %v, seized %v", c.Hold[a], c.Charges[a], c.Tally.Arrests, c.Cases, c.Seized)
		}
		s.days(2) // to the court, tried
		if c.Hold[a] != world.Mines {
			t.Errorf("after the trial: hold %d, want the mines", c.Hold[a])
		}
	}
}

// ODD §6.14, Held zones: Judicial arrests nobody whose day location is in
// a zone the rebels hold; the case waits until the zone is freed.
func TestNoArrestInHeldZone(t *testing.T) {
	s := newTiny(t)
	a := s.on(3) // the Mids
	s.start()
	c := s.w.Control
	s.w.Rebellion = &world.Rebellion{}
	s.w.Rebellion.Held[world.Mids] = true
	s.offend(a, world.TradeRelic)
	s.step()
	if c.Held(a) || !slices.Equal(c.Cases, []world.PersonID{a}) {
		t.Fatalf("in a held zone: held %v, cases %v; want free with the case waiting", c.Held(a), c.Cases)
	}
	s.w.Rebellion.Held[world.Mids] = false
	s.step()
	if c.Hold[a] != world.Custody {
		t.Errorf("zone freed: hold %d, want custody", c.Hold[a])
	}
}

// ODD §6.13, Places: with one prison place, the second convict tried waits
// at home, free, and their new case waits too; when the first is released
// the second takes the place, with a walk to the court, for their full 90
// days, losing a Flamekeeper seat taken while waiting. The mines (0 = no
// limit) take everyone. A waiting convict put out
// in a fight leaves the queue for custody.
func TestPlaces(t *testing.T) {
	s := newTiny(t)
	s.cfg.Control.PrisonPlaces = 1
	a, b, v := s.on(3), s.on(3), s.on(3)
	s.start()
	c, sp := s.w.Control, s.w.Space
	s.offend(a, world.TradeRelic)
	s.offend(b, world.TradeRelic)
	s.offend(v, world.Violence)
	s.days(2) // arrested, then tried
	if c.Hold[a] != world.Prison || c.Hold[b] != world.Free || !s.m.waiting[b] || len(s.m.queue[0]) != 1 || c.Hold[v] != world.Mines {
		t.Fatalf("hold a %d, b %d (waiting %v, queue %v), v %d; want a in prison, b waiting free, v in the mines",
			c.Hold[a], c.Hold[b], s.m.waiting[b], s.m.queue[0], c.Hold[v])
	}
	s.days(2)
	s.offend(b, world.TradeRelic)
	s.step()
	if sp.Location[b] != 3 || c.Held(b) || len(c.Cases) != 1 {
		t.Fatalf("waiting: b at floor %d, held %v, cases %v; want home, free, the new case waiting", sp.Location[b], c.Held(b), c.Cases)
	}
	bl := s.w.Beliefs
	bl.IsMember[b], bl.Members = true, []world.PersonID{b} // took a Flamekeeper seat while waiting
	for c.Hold[a] == world.Prison {
		s.step()
	}
	admitted := s.w.Day
	if bl.IsMember[b] || len(bl.Members) != 0 {
		t.Errorf("admitted: member %v, members %v; want the seat lost", bl.IsMember[b], bl.Members)
	}
	if c.Hold[b] != world.Prison || c.Until[b] != admitted+90 || s.m.waiting[b] || len(s.m.queue[0]) != 0 {
		t.Fatalf("day a was released: b hold %d until %d (want %d), waiting %v", c.Hold[b], c.Until[b], admitted+90, s.m.waiting[b])
	}
	s.step()
	if sp.Location[b] != 2 || len(c.Cases) != 1 {
		t.Errorf("b at floor %d, cases %v; want in prison on the court floor, the new case still waiting", sp.Location[b], c.Cases)
	}

	s = newTiny(t)
	s.cfg.Control.PrisonPlaces = 1
	a, b = s.on(3), s.on(3)
	s.start()
	c = s.w.Control
	s.offend(a, world.TradeRelic)
	s.offend(b, world.TradeRelic)
	s.days(3)
	c.Seized = []world.PersonID{b}
	s.step()
	if c.Hold[b] != world.Custody || s.m.waiting[b] || len(s.m.queue[0]) != 0 {
		t.Errorf("seized while waiting: hold %d, waiting %v, queue %v; want custody, out of the queue", c.Hold[b], s.m.waiting[b], s.m.queue[0])
	}
}
