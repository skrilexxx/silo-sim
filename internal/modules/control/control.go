// Package control is regime control: the Head of IT, surveillance,
// Judicial and punishment (ODD §6.13, draft).
package control

import (
	"fmt"
	"math"
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

	perOfficer     float64 // a: arrests a day per officer
	officers       []int   // Judicial's and the Sheriff's department indexes
	courts         []int   // Judicial's office floors, lowest first
	mines          int     // the mines floor
	raid           float64 // daily chance of a raid
	residential    []int   // floors with homes, lowest first: where raids go
	find, name     float64 // p_find, p_name
	held, homeward []world.PersonID
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
	m.perOfficer, m.raid = cfg.Control.ArrestsPerOfficer, cfg.Control.RaidsPerYear/engine.DaysPerYear
	m.find, m.name, m.mines = cfg.Control.FindChance, cfg.Control.NameChance, cfg.Control.MinesFloor
	for d, dep := range w.Depts {
		if dep.Name == "judicial" || dep.Name == "sheriff" {
			m.officers = append(m.officers, d)
		}
		if dep.Name == "judicial" {
			m.courts = w.Space.DeptFloors[d]
		}
	}
	if len(m.courts) == 0 {
		return fmt.Errorf("control: Judicial has no office floors (space.officeFloors)")
	}
	if w.Floors[m.mines-1].Zone != world.DownDeep {
		return fmt.Errorf("control: minesFloor %d is not a Down Deep floor", m.mines)
	}
	for _, f := range w.Floors {
		if f.Homes > 0 {
			m.residential = append(m.residential, f.Number)
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
	m.drop(w)
	m.headOfIT(w)
	m.detect(w, r)
	m.releases(w)
	if err := m.trials(w); err != nil {
		return err
	}
	if m.raid > 0 && r.Float64() < m.raid {
		m.raidFloor(w, r)
	}
	m.arrests(w, r)
	return nil
}

// newcomers: people born since the last step have no charges.
func (m *Module) newcomers(w *world.World) {
	c := w.Control
	for len(c.Charges) < len(w.People) {
		c.Charges, c.Convicted = append(c.Charges, 0), append(c.Convicted, 0)
		c.Hold, c.Until, c.Flagged = append(c.Hold, world.Free), append(c.Until, 0), append(c.Flagged, false)
	}
}

// drop forgets the dead: their open cases, custody and detention, and
// walks home.
func (m *Module) drop(w *world.World) {
	c := w.Control
	dead := func(id world.PersonID) bool {
		if w.People[id].Alive {
			return false
		}
		c.Charges[id], c.Hold[id], c.Until[id] = 0, world.Free, 0
		return true
	}
	c.Cases = slices.DeleteFunc(c.Cases, dead)
	m.held = slices.DeleteFunc(m.held, dead)
	m.homeward = slices.DeleteFunc(m.homeward, dead)
}

// headOfIT: ODD §6.13, Head of IT. A Head who died, retired or was
// arrested leaves office; an empty post goes to the successor if they are a worker,
// otherwise to the most skilled IT worker (ties to the lower ID). Then a
// Head without a successor names one.
func (m *Module) headOfIT(w *world.World) {
	c := w.Control
	if h := c.Head; h != world.None && (!w.People[h].Alive || w.People[h].Stage == world.Retired || c.Held(h)) {
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

// nameSuccessor: ODD §6.13, Successor. The role lapses with death,
// retirement or arrest. A Head without a successor names their shadow, the older of
// two; with none, a Head who can be a master takes the youngest IT shadow
// as their own. Ties go to the lower ID.
func (m *Module) nameSuccessor(w *world.World) {
	c := w.Control
	if s := c.Successor; s != world.None && (!w.People[s].Alive || w.People[s].Stage == world.Retired || c.Held(s)) {
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
// handover.
func (m *Module) detect(w *world.World, r *rng.Stream) {
	c := w.Control
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
		c.Tally.Seen++
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

// charge adds a rule to the person's case, opening one if they have none;
// a charge against someone in custody joins the case they will be tried
// for.
func (m *Module) charge(c *world.Control, id world.PersonID, rule int) {
	if c.Charges[id] == 0 && c.Hold[id] != world.Custody {
		c.Cases = append(c.Cases, id)
	}
	c.Charges[id] |= 1 << rule
}

// releases: ODD §6.13, Penalties. Detained people whose last day has
// passed are free and start walking home; so do people freed earlier who
// couldn't, while the stairs were broken.
func (m *Module) releases(w *world.World) {
	c := w.Control
	m.held = slices.DeleteFunc(m.held, func(id world.PersonID) bool {
		if c.Hold[id] == world.Custody || w.Day <= c.Until[id] {
			return false
		}
		c.Hold[id], c.Until[id] = world.Free, 0
		m.homeward = append(m.homeward, id)
		return true
	})
	m.homeward = slices.DeleteFunc(m.homeward, func(id world.PersonID) bool { return m.walkHome(w, id) })
}

// walkHome starts a free person's trip home and reports whether it
// started; with the stairs broken they try again tomorrow. Someone still
// on the stairs (to the mines, say) waits for the trip to end.
func (m *Module) walkHome(w *world.World, id world.PersonID) bool {
	return w.Space.StartTrip(id, w.Day, w.Households[w.People[id].Household].Floor) == nil
}

// trials: ODD §6.13, Trial. People in custody who reached the court are
// tried. One who stopped short, at a break in the stairs, walks on to the
// nearest court they can reach.
func (m *Module) trials(w *world.World) error {
	c, s := w.Control, w.Space
	for _, id := range slices.Clone(m.held) { // a warning removes them
		if c.Hold[id] != world.Custody || s.Location[id] == 0 {
			continue
		}
		if !slices.Contains(m.courts, s.Location[id]) {
			if court := m.court(w, s.Location[id]); court != 0 {
				_ = s.StartTrip(id, w.Day, court) // on a floor, with a court connected: it starts
			}
			continue
		}
		if err := m.try(w, id); err != nil {
			return err
		}
	}
	return nil
}

// try sentences one person: for each rule in their case, the first or
// the repeat penalty; they serve the most severe detention among them,
// and every confiscation and family flag.
func (m *Module) try(w *world.World, id world.PersonID) error {
	c := w.Control
	var worst config.Penalty
	confiscate, flag := false, false
	for i, rule := range m.rules {
		if c.Charges[id]&(1<<i) == 0 {
			continue
		}
		pen := rule.First
		if c.Convicted[id]&(1<<i) != 0 && rule.Repeat != nil {
			pen = *rule.Repeat
		}
		if severity(pen) > severity(worst) {
			worst = pen
		}
		confiscate, flag = confiscate || pen.Confiscate, flag || pen.FlagFamily
	}
	c.Convicted[id] |= c.Charges[id]
	c.Charges[id] = 0
	if confiscate {
		m.confiscate(w, id)
	}
	if flag {
		for _, o := range w.Households[w.People[id].Household].Members {
			c.Flagged[o] = true
		}
	}
	switch {
	case worst.Cleaning:
		return fmt.Errorf("control: person %d sentenced to cleaning, which comes with T11.3", id)
	case worst.Mines > 0:
		c.Tally.Mines++
		c.Hold[id], c.Until[id] = world.Mines, w.Day+world.Tick(worst.Mines)
		_ = w.Space.StartTrip(id, w.Day, m.mines) // with the stairs broken, they serve it in the court's prison
	case worst.Prison > 0:
		c.Tally.Prisons++
		c.Hold[id], c.Until[id] = world.Prison, w.Day+world.Tick(worst.Prison)
	default:
		c.Tally.Warnings++
		c.Hold[id] = world.Free
		m.held = slices.DeleteFunc(m.held, func(x world.PersonID) bool { return x == id })
		if !m.walkHome(w, id) {
			m.homeward = append(m.homeward, id)
		}
	}
	return nil
}

// severity orders detentions: cleaning, then mines, then prison, then
// none; the longer of the same kind.
func severity(p config.Penalty) int {
	switch {
	case p.Cleaning:
		return math.MaxInt
	case p.Mines > 0:
		return math.MaxInt/2 + p.Mines
	}
	return p.Prison
}

// confiscate takes every relic the person holds out of the silo.
func (m *Module) confiscate(w *world.World, id world.PersonID) {
	b := w.Beliefs
	n := len(b.Relics)
	b.Relics = slices.DeleteFunc(b.Relics, func(r world.Relic) bool { return r.Holder == id })
	w.Control.Tally.Taken += n - len(b.Relics)
}

// raidFloor: ODD §6.13, Searches. Judicial raids one residential floor,
// drawn uniformly, and searches every flat on it, in household order.
func (m *Module) raidFloor(w *world.World, r *rng.Stream) {
	w.Control.Tally.Raids++
	floor := m.residential[r.IntRange(0, len(m.residential)-1)]
	for i := range w.Households {
		if hh := &w.Households[i]; hh.Floor == floor && len(hh.Members) > 0 {
			m.search(w, r, hh.ID)
		}
	}
}

// search: ODD §6.13, Searches. Each relic held by someone living in the
// flat is found with chance p_find, in relic order; a found relic leaves
// the silo and its holder is charged with holdRelic. With no active rule
// against holding relics there is nothing to look for.
func (m *Module) search(w *world.World, r *rng.Stream, hh world.HouseholdID) {
	rule := m.ruleOf[world.HoldRelic]
	if rule == world.None {
		return
	}
	b, c := w.Beliefs, w.Control
	kept := b.Relics[:0]
	for _, rel := range b.Relics {
		if rel.Holder == world.None || w.People[rel.Holder].Household != hh || r.Float64() >= m.find {
			kept = append(kept, rel)
			continue
		}
		c.Tally.Taken++
		m.charge(c, rel.Holder, rule)
	}
	b.Relics = kept
}

// arrests: ODD §6.13, Arrests. Judicial arrests up to ⌊a × officers⌋
// people a day, open cases oldest first, among those not already held.
// An arrest is a trip to the nearest court they can reach; if it can't
// start, the case waits and uses no capacity. On arrest their flat is
// searched, and a Flamekeeper loses their seat and names the others.
func (m *Module) arrests(w *world.World, r *rng.Stream) {
	c, s := w.Control, w.Space
	officers := 0
	for _, d := range m.officers {
		officers += w.Depts[d].Headcount // workers not off work (ODD §6.5, §6.10)
	}
	capacity := int(m.perOfficer * float64(officers))
	for i := 0; i < len(c.Cases) && capacity > 0; {
		id := c.Cases[i]
		court := 0
		if s.Location[id] != 0 && !c.Held(id) {
			court = m.court(w, s.Location[id])
		}
		if court == 0 || s.StartTrip(id, w.Day, court) != nil {
			i++
			continue
		}
		capacity--
		c.Tally.Arrests++
		c.Cases = slices.Delete(c.Cases, i, i+1)
		c.Hold[id] = world.Custody
		m.held = append(m.held, id)
		m.homeward = slices.DeleteFunc(m.homeward, func(x world.PersonID) bool { return x == id })
		m.search(w, r, w.People[id].Household)
		m.unseat(w, r, id)
	}
}

// unseat: an arrested Flamekeeper loses their seat and, with chance
// p_name, names one other member, drawn uniformly, who is charged with
// speakForbidden if an active rule lists it. Naming every member with
// chance p_name wiped out the group in the first year of every run
// (T11.2).
func (m *Module) unseat(w *world.World, r *rng.Stream, id world.PersonID) {
	b := w.Beliefs
	if !b.IsMember[id] {
		return
	}
	b.IsMember[id] = false
	b.Members = slices.DeleteFunc(b.Members, func(x world.PersonID) bool { return x == id })
	rule := m.ruleOf[world.SpeakForbidden]
	if rule == world.None || len(b.Members) == 0 || r.Float64() >= m.name {
		return
	}
	m.charge(w.Control, b.Members[r.IntRange(0, len(b.Members)-1)], rule)
}

// court is the Judicial office floor nearest to floor f among those
// connected to it, the lower on a tie; 0 if none is.
func (m *Module) court(w *world.World, f int) int {
	best := 0
	for _, c := range m.courts {
		if w.Space.Connected(f, c) && (best == 0 || dist(c, f) < dist(best, f)) {
			best = c
		}
	}
	return best
}

func dist(a, b int) int { return max(a-b, b-a) }
