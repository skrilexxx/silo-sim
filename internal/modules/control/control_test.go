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

// tiny is a one-floor world with departments general, it and medical; the
// beliefs state is sized to the people added, and nothing else runs.
type tiny struct {
	t   *testing.T
	cfg *config.Scenario
	w   *world.World
	m   *Module
	r   *rng.Stream
}

func newTiny(t *testing.T) *tiny {
	w := &world.World{Day: 100 * year, Floors: []world.Floor{{Number: 1, Homes: 99}},
		Depts: []world.Department{{Name: "general"}, {Name: "it", Coverage: 1}, {Name: "medical"}}}
	r, _ := rng.New(1, "control")
	cfg := &config.Scenario{}
	cfg.Work.ShadowAge, cfg.Work.MasterMinSkill = 14, 0.5
	cfg.Control.Surveillance = 1
	cfg.Pact = []config.Rule{
		{ID: "noRelics", Active: true, Offences: []string{"holdRelic", "tradeRelic"}, Detection: 1},
		{ID: "noForbiddenTalk", Active: true, Offences: []string{"speakForbidden"}, Detection: 0.5},
		{ID: "noOrganising", Active: false, Offences: []string{"organise"}, Detection: 1},
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
	general = 0
	it      = 1
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

// ODD §6.13, Leaving office: a Head who dies is followed by their
// successor if a worker, otherwise by the most skilled IT worker; a Head
// on sick leave keeps the post; one who retires leaves it.
func TestSuccession(t *testing.T) {
	s := newTiny(t)
	head := s.add(60, world.Worker, it, 0.9)
	best := s.add(50, world.Worker, it, 0.8)
	shadow := s.add(17, world.Shadow, it, 0.3)
	s.w.People[shadow].Master = head
	s.start()

	s.w.People[head].Stage = world.SickLeave
	s.step()
	if c := s.w.Control; c.Head != head || c.Successor != shadow {
		t.Fatalf("on sick leave: Head %d, successor %d; want %d, %d", c.Head, c.Successor, head, shadow)
	}
	s.w.People[head].Stage = world.Worker

	// The successor is still a shadow: the most skilled worker takes over,
	// and the role lapses.
	s.kill(head)
	s.step()
	if c := s.w.Control; c.Head != best || c.Successor == shadow && s.w.People[shadow].Master != best {
		t.Fatalf("Head %d, successor %d; want %d and a successor of their own", c.Head, c.Successor, best)
	}

	// The new Head names the shadow (now theirs), who grows up and follows
	// them on retirement.
	if s.w.Control.Successor != shadow {
		t.Fatalf("successor %d, want %d", s.w.Control.Successor, shadow)
	}
	s.w.People[shadow].Stage = world.Worker
	s.w.People[best].Stage = world.Retired
	s.step()
	if c := s.w.Control; c.Head != shadow || c.Successor != world.None {
		t.Errorf("Head %d, successor %d; want %d, none", c.Head, c.Successor, shadow)
	}
	h := s.heads()
	if len(h) != 3 || h[1].Successor || !h[2].Successor {
		t.Errorf("events %+v, want three, the last by succession", h)
	}

	// With no IT worker left the post stays empty.
	s.kill(shadow)
	s.step()
	if c := s.w.Control; c.Head != world.None {
		t.Errorf("Head %d, want none", c.Head)
	}
}

// ODD §6.13, Detection: an act is seen with chance min(1, σ × IT coverage
// × the rule's weight). Here 0.5 × 0.8 × 0.5 = 0.2, over 10,000 acts.
func TestDetectionRate(t *testing.T) {
	s := newTiny(t)
	s.cfg.Control.Surveillance = 0.5
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
	if got := float64(s.w.Control.Seen); math.Abs(got-n*p) > 4*sd {
		t.Errorf("%v of %d acts seen, want %v ± %.0f", got, n, n*p, 4*sd)
	}
	t.Logf("%d of %d acts seen (want about %v)", s.w.Control.Seen, n, n*p)
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
	s.cfg.Pact[1].Detection = 1 // every act seen
	s.start()
	c := s.w.Control
	for _, act := range []world.Act{
		{Person: a, Other: world.None, Offence: world.Organise},    // the rule is inactive
		{Person: kid, Other: world.None, Offence: world.HoldRelic}, // too young
		{Person: b, Other: world.None, Offence: world.Violence},    // no rule lists it
		{Person: a, Other: b, Offence: world.TradeRelic},
		{Person: a, Other: world.None, Offence: world.SpeakForbidden},
	} {
		c.Report(act)
	}
	s.step()
	if !slices.Equal(c.Cases, []world.PersonID{a, b}) || c.Charges[a] != 0b11 || c.Charges[b] != 0b01 || c.Charges[kid] != 0 {
		t.Fatalf("cases %v, charges %v; want a charged under rules 0 and 1, b under 0", c.Cases, c.Charges)
	}
	if c.Seen != 2 {
		t.Errorf("%d acts seen, want 2", c.Seen)
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
	if s.w.Control.Seen != 0 || len(s.w.Control.Cases) != 0 || *s.r != before {
		t.Errorf("σ = 0: %d seen, cases %v, stream moved %v", s.w.Control.Seen, s.w.Control.Cases, *s.r != before)
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
