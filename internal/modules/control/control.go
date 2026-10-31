// Package control is regime control: the Head of IT, surveillance,
// Judicial and punishment (ODD §6.13, draft).
package control

import (
	"fmt"
	"slices"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	surveillance float64       // σ
	rules        []config.Rule // the Pact (ODD §6.9)
	ruleOf       [world.NumOffences]int
	it           int        // IT's department index
	minSkill     float64    // a master's skill must be above this (ODD §6.5)
	shadowAge    world.Tick // in days: acts by younger children are not offences
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "control" }

// Requires beliefs: what the regime punishes is mostly knowledge passed
// on (ODD §6.12), and the Head of IT knows every flag.
func (m *Module) Requires() []string { return []string{"beliefs"} }

// ODD §6.13: the first Head of IT is chosen on day 1.
func (m *Module) Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error {
	if w.Beliefs == nil {
		return fmt.Errorf("control: no beliefs state; list beliefs before control in modules")
	}
	m.surveillance = cfg.Control.Surveillance
	m.minSkill = cfg.Work.MasterMinSkill
	m.shadowAge = world.Tick(cfg.Work.ShadowAge * engine.DaysPerYear)
	m.it = slices.IndexFunc(w.Depts, func(d world.Department) bool { return d.Name == "it" })
	m.rules = slices.Clone(cfg.Pact)
	for o := range m.ruleOf {
		m.ruleOf[o] = world.None
	}
	for i, rule := range m.rules {
		if !rule.Active {
			continue
		}
		for _, name := range rule.Offences {
			m.ruleOf[slices.Index(config.Offences, name)] = i // config checked the name, and that no other active rule lists it
		}
	}
	w.Control = &world.Control{Head: world.None, Successor: world.None}
	m.newcomers(w)
	m.headOfIT(w)
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseControl, Run: m.step}}
}

// ODD §6.13 / §3 step 10.8.
func (m *Module) step(w *world.World, r *rng.Stream) error {
	m.newcomers(w)
	m.headOfIT(w)
	m.detect(w, r)
	return nil
}

// newcomers: people born since the last step have no charges.
func (m *Module) newcomers(w *world.World) {
	c := w.Control
	for len(c.Charges) < len(w.People) {
		c.Charges = append(c.Charges, 0)
	}
}

// headOfIT: ODD §6.13, Head of IT. A Head who died or retired leaves
// office; an empty post goes to the successor if they are a worker,
// otherwise to the most skilled IT worker (ties to the lower ID). Then a
// Head without a successor names one.
func (m *Module) headOfIT(w *world.World) {
	c := w.Control
	if h := c.Head; h != world.None && (!w.People[h].Alive || w.People[h].Stage == world.Retired) {
		c.Head = world.None
	}
	if c.Head == world.None {
		s := c.Successor
		c.Successor = world.None // a new Head starts without one
		if s != world.None && w.People[s].Alive && w.People[s].Stage == world.Worker {
			m.takeOffice(w, s, true)
		} else if h := m.mostSkilled(w); h != world.None {
			m.takeOffice(w, h, false)
		}
	}
	if c.Head != world.None {
		m.nameSuccessor(w)
	}
}

// mostSkilled is the most skilled IT worker, ties to the lower ID, or None.
func (m *Module) mostSkilled(w *world.World) world.PersonID {
	best := world.PersonID(world.None)
	for _, id := range w.Alive {
		p := &w.People[id]
		if p.Stage != world.Worker || p.Department != m.it {
			continue
		}
		if best == world.None || p.Skill > w.People[best].Skill || p.Skill == w.People[best].Skill && id < best {
			best = id
		}
	}
	return best
}

// takeOffice: the vault tells the new Head every flag, lived; they leave
// the Flamekeepers if they were one.
func (m *Module) takeOffice(w *world.World, id world.PersonID, successor bool) {
	c, b := w.Control, w.Beliefs
	c.Head = id
	for f := range world.NumFlags {
		b.Learn(id, f, true)
	}
	if b.IsMember[id] {
		b.IsMember[id] = false
		b.Members = slices.DeleteFunc(b.Members, func(x world.PersonID) bool { return x == id })
	}
	w.Events = append(w.Events, world.HeadOfITEvent{EventHeader: world.EventHeader{Day: w.Day, Type: "headOfIT"}, Head: id, Successor: successor})
}

// nameSuccessor: ODD §6.13, Successor. The role lapses with death or
// retirement. A Head without a successor names their shadow, the older of
// two; with none, a Head who can be a master takes the youngest IT shadow
// as their own. Ties go to the lower ID.
func (m *Module) nameSuccessor(w *world.World) {
	c := w.Control
	if s := c.Successor; s != world.None && (!w.People[s].Alive || w.People[s].Stage == world.Retired) {
		c.Successor = world.None
	}
	if c.Successor != world.None {
		return
	}
	own, young := world.PersonID(world.None), world.PersonID(world.None)
	for _, id := range w.Alive {
		p := &w.People[id]
		if p.Stage != world.Shadow || p.Department != m.it {
			continue
		}
		if p.Master == c.Head && (own == world.None || earlier(w, id, own)) {
			own = id
		}
		if young == world.None || earlier(w, young, id) {
			young = id
		}
	}
	head := &w.People[c.Head]
	switch {
	case own != world.None:
		c.Successor = own
	case young != world.None && head.Stage == world.Worker && head.Skill > m.minSkill:
		w.People[young].Master = c.Head // the work module recounts masters' shadows every day
		c.Successor = young
	}
}

// earlier reports whether a was born before b, or on the same day with the
// lower ID.
func earlier(w *world.World, a, b world.PersonID) bool {
	da, db := w.People[a].BirthDay, w.People[b].BirthDay
	return da < db || da == db && a < b
}

// detect: ODD §6.13, Detection. Each act reported since the last step, in
// order, is an offence if an active rule lists it and the offender is 14
// or older, and is seen with chance min(1, σ × IT coverage × the rule's
// weight). A seen act charges the offender, and the second offender of a
// handover. Cases of people who died are dropped first.
func (m *Module) detect(w *world.World, r *rng.Stream) {
	c := w.Control
	c.Cases = slices.DeleteFunc(c.Cases, func(id world.PersonID) bool {
		if w.People[id].Alive {
			return false
		}
		c.Charges[id] = 0
		return true
	})
	cov := w.Depts[m.it].Coverage
	for _, a := range c.Acts {
		rule := m.ruleOf[a.Offence]
		if rule == world.None || !m.offender(w, a.Person) {
			continue
		}
		p := min(1, m.surveillance*cov*m.rules[rule].Detection)
		if p == 0 || r.Float64() >= p {
			continue
		}
		c.Seen++
		m.charge(c, a.Person, rule)
		if a.Other != world.None && m.offender(w, a.Other) {
			m.charge(c, a.Other, rule)
		}
	}
	c.Acts = c.Acts[:0]
}

// offender reports whether a person can commit an offence: alive and aged
// 14 or more.
func (m *Module) offender(w *world.World, id world.PersonID) bool {
	return w.People[id].Alive && w.Day-w.People[id].BirthDay >= m.shadowAge
}

// charge adds a rule to the person's case, opening one if they have none.
func (m *Module) charge(c *world.Control, id world.PersonID, rule int) {
	if c.Charges[id] == 0 {
		c.Cases = append(c.Cases, id)
	}
	c.Charges[id] |= 1 << rule
}
