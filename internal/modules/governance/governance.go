// Package governance holds the Mayor (ODD §6.8). In the baseline the
// office changes no rules; it is chosen and logged so the dissent modules
// have it ready.
package governance

import (
	"fmt"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	cfg       *config.Scenario
	termYears int // 0 = life

	votes []int // reused, per zone
	cands []world.PersonID
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "governance" }

// Requires work: candidates are workers, ranked by skill.
func (m *Module) Requires() []string { return []string{"work"} }

// ODD §6.8: the first Mayor is chosen on day 1, so the silo always has
// one and 5-year terms end on yearly governance steps.
func (m *Module) Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error {
	m.cfg = cfg
	var err error
	if m.termYears, err = cfg.Governance.TermYears(); err != nil {
		return err
	}
	if len(w.Depts) == 0 {
		return fmt.Errorf("governance: no departments; list work before governance in modules")
	}
	w.Silo.GovernanceMode = cfg.Governance.Mode
	w.Silo.Mayor = world.None
	m.choose(w, r, "start")
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseGovernance, Run: m.governance}}
}

// ODD §6.8 / §3 step 16: yearly. A new Mayor is chosen if the Mayor died
// since the last step, the term has run out, or nobody could be chosen
// last time. A Mayor who retires keeps the office until then.
func (m *Module) governance(w *world.World, r *rng.Stream) error {
	s := &w.Silo
	switch {
	case s.Mayor == world.None:
		m.choose(w, r, "vacant")
	case !w.People[s.Mayor].Alive:
		m.choose(w, r, "died")
	case m.termYears > 0 && w.Day-s.MayorSince >= world.Tick(m.termYears*engine.DaysPerYear):
		m.choose(w, r, "term")
	}
	return nil
}

// choose picks a new Mayor by the scenario's mode and logs it. The sitting
// Mayor may be chosen again. If nobody can be chosen, the office stays
// vacant and is retried next year; a vacancy that continues is not logged
// again.
func (m *Module) choose(w *world.World, r *rng.Stream, reason string) {
	s := &w.Silo
	var mayor world.PersonID
	var votes []int
	if m.cfg.Governance.Mode == "election" {
		mayor, votes = m.elect(w, r)
	} else {
		mayor = m.appoint(w)
	}
	prev := s.Mayor
	if mayor == world.None && prev == world.None && reason == "vacant" {
		return
	}
	s.Mayor, s.MayorSince = mayor, w.Day
	zone := ""
	if mayor != world.None {
		zone = homeZone(w, mayor).String()
	}
	w.Events = append(w.Events, world.Event{Day: w.Day, Type: "mayor", Mayor: mayor, Previous: prev,
		Zone: zone, Reason: reason, Mode: m.cfg.Governance.Mode, Votes: votes})
}

// ODD §6.8 appointment: the heads of IT, Judicial and Administration (the
// most skilled worker in each) choose the most skilled worker of those
// three departments. Since each head is the most skilled of their own
// department, that is simply the most skilled worker in the three.
func (m *Module) appoint(w *world.World) world.PersonID {
	best := world.PersonID(world.None)
	for _, id := range w.Alive {
		p := &w.People[id]
		if p.Stage != world.Worker {
			continue
		}
		switch w.Depts[p.Department].Name {
		case "it", "judicial", "administration":
			best = better(w, best, id)
		}
	}
	return best
}

// ODD §6.8 election: each zone puts up its most skilled worker living
// there. Every adult (partnering.minAge or older) votes for their own
// zone's candidate with chance ownZoneVote, otherwise for a candidate
// drawn uniformly from all of them (own zone included). Voters whose zone
// has no candidate always draw. Most votes wins; a tie is drawn at random.
// Votes are returned per zone, -1 where a zone had no candidate.
func (m *Module) elect(w *world.World, r *rng.Stream) (world.PersonID, []int) {
	var cand [world.NumZones]world.PersonID
	for z := range cand {
		cand[z] = world.None
	}
	for _, id := range w.Alive {
		if w.People[id].Stage == world.Worker {
			z := homeZone(w, id)
			cand[z] = better(w, cand[z], id)
		}
	}
	m.cands = m.cands[:0]
	for _, c := range cand {
		if c != world.None {
			m.cands = append(m.cands, c)
		}
	}
	if len(m.cands) == 0 {
		return world.None, nil
	}

	m.votes = append(m.votes[:0], make([]int, world.NumZones)...)
	adult := world.Tick(m.cfg.Partnering.MinAge * engine.DaysPerYear)
	for _, id := range w.Alive {
		if w.Day-w.People[id].BirthDay < adult {
			continue
		}
		pick := cand[homeZone(w, id)]
		if pick == world.None || !r.Bernoulli(m.cfg.Governance.OwnZoneVote) {
			pick = m.cands[r.IntRange(0, len(m.cands)-1)]
		}
		m.votes[homeZone(w, pick)]++
	}

	// Winner by votes, ties drawn at random among the tied.
	most := -1
	m.cands = m.cands[:0]
	for z, c := range cand {
		switch {
		case c == world.None:
		case m.votes[z] > most:
			most, m.cands = m.votes[z], append(m.cands[:0], c)
		case m.votes[z] == most:
			m.cands = append(m.cands, c)
		}
	}
	winner := m.cands[0]
	if len(m.cands) > 1 {
		winner = m.cands[r.IntRange(0, len(m.cands)-1)]
	}
	votes := append([]int(nil), m.votes...)
	for z, c := range cand {
		if c == world.None {
			votes[z] = -1
		}
	}
	return winner, votes
}

// better returns whichever of a and b has more skill; a may be None. Ties
// go to the lower ID, so the pick never depends on the shuffled order of
// Alive.
func better(w *world.World, a, b world.PersonID) world.PersonID {
	if a == world.None {
		return b
	}
	sa, sb := w.People[a].Skill, w.People[b].Skill
	if sb > sa || (sb == sa && b < a) {
		return b
	}
	return a
}

// homeZone is the zone of a living person's household.
func homeZone(w *world.World, id world.PersonID) world.Zone {
	return w.Floors[w.Households[w.People[id].Household].Floor-1].Zone
}
