// Package rebellion is grievance and rebellion (ODD §6.14, draft).
package rebellion

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
	loss      float64    // ℓ: hardship of a loss on its day
	lossDays  float64    // T_loss in days
	scapegoat float64    // s_DD
	adultAge  world.Tick // worker age in days (ODD §6.5)
	buf       []world.Tie

	threshold, k float64 // T and k (Epstein 2002)
	violent      float64 // T_v
	strikeShare  float64 // q_strike
	eJ, eR, kill float64 // e_J, e_R, p_kill (decision D6)
	injury       float64 // a work injury's cost to health (ODD §6.2)
	fought       bool    // a fight today
	pool         []world.PersonID
	sab, repair  float64 // p_sab; 1 / D_stair
	mechanical   int     // Mechanical's department index
	runAfter     bool    // output.runAfterCollapse
	critical     []int   // Mechanical, Farms, Supply and Medical (ODD §1.2)
	prev         []world.RebelStage
	def          config.Rebellion // the definitions of unrest and rebellion (ODD §1.2)
	officers     []int            // Judicial's and the Sheriff's department indexes
	courts       []int            // Judicial's office floors
	reach        []bool           // per floor: connected to a court today
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "rebellion" }

// Requires control: grievance reads the losses it reports (cleanings,
// denied wins), and through it beliefs (L) and the network (family).
func (m *Module) Requires() []string { return []string{"control"} }

func (m *Module) Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error {
	if w.Control == nil {
		return fmt.Errorf("rebellion: no control state; list control before rebellion in modules")
	}
	m.loss, m.scapegoat = cfg.Rebellion.LossHardship, cfg.Rebellion.ScapegoatHardship
	m.lossDays = cfg.Rebellion.LossHalfLifeYears * engine.DaysPerYear
	m.adultAge = world.Tick(cfg.Work.WorkerAge * engine.DaysPerYear)
	m.threshold, m.k, m.violent = cfg.Rebellion.ActiveThreshold, cfg.Rebellion.ArrestConstant, cfg.Rebellion.ViolentThreshold
	m.def, m.strikeShare = cfg.Rebellion, cfg.Rebellion.StrikeShare
	m.eJ, m.eR, m.kill, m.injury = cfg.Rebellion.OfficerEffect, cfg.Rebellion.RebelEffect, cfg.Rebellion.KilledShare, cfg.Health.InjuryCost
	m.sab, m.repair, m.runAfter = cfg.Rebellion.SabotageChance, 1/cfg.Rebellion.StairRepairDays, cfg.Output.RunAfterCollapse
	m.mechanical = slices.IndexFunc(w.Depts, func(d world.Department) bool { return d.Name == "mechanical" })
	for d, dep := range w.Depts {
		if dep.Name == "judicial" || dep.Name == "sheriff" {
			m.officers = append(m.officers, d)
		}
		if dep.Name == "judicial" {
			m.courts = w.Space.DeptFloors[d] // control checked that there are some
		}
		if slices.Contains([]string{"mechanical", "farms", "supply", "medical"}, dep.Name) {
			m.critical = append(m.critical, d)
		}
	}
	m.reach = make([]bool, len(w.Floors))
	w.Rebellion = &world.Rebellion{LastAction: world.None, UnrestStart: world.None, RebellionStart: world.None,
		Collapsed: world.None, Striking: make([]int, len(w.Depts))}
	m.newcomers(w)
	m.count(w)
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseRebellion, Run: m.step}}
}

// ODD §6.14 / §3 step 10.9.
func (m *Module) step(w *world.World, r *rng.Stream) error {
	m.newcomers(w)
	m.losses(w)
	m.hardship(w)
	m.strikes(w)
	m.stages(w)
	m.fights(w, r)
	m.sabotage(w, r)
	m.repairs(w, r)
	m.track(w, m.activeShare(w), m.action(w))
	m.collapse(w)
	m.count(w)
	return nil
}

// EventTypes: ODD §6.14, Scripted events. breakStairs breaks a segment,
// just before the rebellion step; it is repaired like any other.
func (m *Module) EventTypes() []engine.EventType {
	return []engine.EventType{{Name: "breakStairs", Phase: engine.PhaseRebellion, Fire: func(w *world.World, _ *rng.Stream, ev config.Event) error {
		if ev.Segment < 1 || ev.Segment > len(w.Space.Broken) {
			return fmt.Errorf("breakStairs: segment %d, must be 1 to %d", ev.Segment, len(w.Space.Broken))
		}
		w.Space.Broken[ev.Segment-1] = true
		return nil
	}}}
}

// ODD §6.14, Sabotage. Each violent rebel, in the tick's shuffled order,
// sabotages with chance p_sab. On a Mechanical floor they wreck the
// generator if it is running, else the water pump if it is running (the
// resources module acts on it at its next step); otherwise, on a floor
// below every court, they break the stair segment between their floor and
// the next one up, if it is intact. Each sabotage is the offence sabotage.
func (m *Module) sabotage(w *world.World, r *rng.Stream) {
	b, s := w.Rebellion, w.Space
	lowest := slices.Max(m.courts)
	for _, id := range w.Alive {
		if b.Stage[id] != world.Violent || r.Float64() >= m.sab {
			continue
		}
		f := s.Where(w, id)
		ev := world.SabotageEvent{EventHeader: world.EventHeader{Day: w.Day, Type: "sabotage"}, Person: id}
		var t world.Target
		switch mech := f != 0 && w.Floors[f-1].Function == world.Mechanical; {
		case mech && w.Silo.GeneratorDown == 0 && !b.GeneratorHit:
			t, b.GeneratorHit = world.TargetGenerator, true
		case mech && w.Day > b.PumpUntil && !b.PumpHit:
			t, b.PumpHit = world.TargetPump, true
		case f > lowest && !s.Broken[f-2]:
			t, s.Broken[f-2], ev.Segment = world.TargetStairs, true, f-1
		default:
			continue
		}
		ev.Target = t.String()
		b.Sabotage[t]++
		w.Events = append(w.Events, ev)
		w.Control.Report(world.Act{Person: id, Other: world.None, Offence: world.Sabotage})
	}
}

// ODD §6.14, Stair repairs: each broken segment, from 1 up, is repaired
// with chance mechanicalCoverage / D_stair, unless a floor at either end
// lies in a held zone.
func (m *Module) repairs(w *world.World, r *rng.Stream) {
	b, s := w.Rebellion, w.Space
	p := w.Depts[m.mechanical].Coverage * m.repair
	for seg := 1; seg <= len(s.Broken); seg++ {
		if !s.Broken[seg-1] || b.Holds(w, seg) || b.Holds(w, seg+1) || r.Float64() >= p {
			continue
		}
		s.Broken[seg-1] = false
		b.Repairs++
		w.Events = append(w.Events, world.SegmentEvent{EventHeader: world.EventHeader{Day: w.Day, Type: "stairRepaired"}, Segment: seg})
	}
}

// ODD §1.2, Collapse: air or water below the collapse level for
// CollapseDays days in a row, or the population below P_min (off while
// 0). The first collapse is recorded; it ends a running rebellion, and the
// run after this tick unless output.runAfterCollapse is set. The
// Safeguard comes with §6.16.
func (m *Module) collapse(w *world.World) {
	b, s, d := w.Rebellion, &w.Silo, m.def
	b.AirLow, b.WaterLow = b.AirLow+1, b.WaterLow+1
	if s.AirQuality >= d.CollapseLevel {
		b.AirLow = 0
	}
	if s.WaterSupply >= d.CollapseLevel {
		b.WaterLow = 0
	}
	if b.Collapsed != world.None {
		return
	}
	cause := ""
	switch {
	case b.AirLow >= d.CollapseDays:
		cause = "air"
	case b.WaterLow >= d.CollapseDays:
		cause = "water"
	case d.PointOfNoReturn > 0 && len(w.Alive) < d.PointOfNoReturn:
		cause = "population"
	default:
		return
	}
	b.Collapsed = w.Day
	w.Events = append(w.Events, world.CollapseEvent{EventHeader: world.EventHeader{Day: w.Day, Type: "collapse"}, Cause: cause})
	if b.RebellionStart != world.None {
		w.Events = append(w.Events, world.RebellionEvent{EventHeader: world.EventHeader{Day: w.Day, Type: "rebellionEnd"},
			Start: b.RebellionStart, End: w.Day, Ending: "collapse"})
		b.RebellionStart = world.None
	}
	w.Over = w.Over || !m.runAfter
}

func (m *Module) newcomers(w *world.World) {
	b := w.Rebellion
	for len(b.LossDay) < len(w.People) {
		b.LossDay, b.H, b.G = append(b.LossDay, world.None), append(b.H, 0), append(b.G, 0)
		b.Stage = append(b.Stage, world.Calm)
	}
}

// ODD §6.14, Hardship: losses. The family of each cleaner who went out
// today (partner, household and kin: the strong ties of §6.11 other than
// master–shadow), and both partners of each denied lottery win, have a
// loss today. The dead keep their records (ODD §6.1), so the cleaner's
// family still follows from them.
func (m *Module) losses(w *world.World) {
	b, c := w.Rebellion, w.Control
	for _, id := range c.Cleaned {
		m.grieve(w, id)
	}
	for _, id := range c.DeniedToday {
		if !w.People[id].Alive {
			continue
		}
		b.LossDay[id] = w.Day
		if p := w.People[id].Partner; p != world.None {
			b.LossDay[p] = w.Day
		}
	}
}

// grieve gives the family of someone cleaned or killed in a fight a loss
// today: partner, household and kin (ODD §6.14, Hardship).
func (m *Module) grieve(w *world.World, id world.PersonID) {
	m.buf = w.Network.Ties(w, id, m.buf[:0])
	for _, t := range m.buf {
		if t.Kind <= world.TieKin {
			w.Rebellion.LossDay[t.Other] = w.Day
		}
	}
}

// ODD §6.14, Hardship and Grievance: H combines food, loss and the Down
// Deep like independent chances, so it stays within 0–1 without a clamp;
// G = H × (1 − L). Children have neither. Health is left out: with it, a
// strike that cut air brought out more strikers until the silo died.
func (m *Module) hardship(w *world.World) {
	b := w.Rebellion
	for _, id := range w.Alive {
		p := &w.People[id]
		if w.Day-p.BirthDay < m.adultAge {
			b.H[id], b.G[id] = 0, 0
			continue
		}
		zone := w.Floors[w.Households[p.Household].Floor-1].Zone
		calm := 1 - max(0, 1-w.Silo.Intake[zone])
		if d := b.LossDay[id]; d != world.None {
			calm *= 1 - m.loss*math.Exp2(-float64(w.Day-d)/m.lossDays)
		}
		if zone == world.DownDeep {
			calm *= 1 - m.scapegoat
		}
		b.H[id] = 1 - calm
		b.G[id] = b.H[id] * (1 - w.Beliefs.L(id))
	}
}

// ODD §6.14, Arrest estimate and Stages. An eligible adult is active
// when G − N > T, with net risk N = R × (1 − (1 − P)(1 − F)) and P = 1 −
// exp(−k × O_z / (A_z + 1)) from yesterday's counts in the zone of their
// day location and today's officers. An active adult joins when at least
// a share R of their ties were active or beyond yesterday, and a joined
// one is violent when G − N > T_v. Nothing is drawn and nobody's stage
// depends on another's today, so the visiting order doesn't matter. Each
// violent person commits the offence violence, every other active one
// organise.
func (m *Module) stages(w *world.World) {
	b, c := w.Rebellion, w.Control
	m.reachable(w)
	var p [world.NumZones]float64 // P on floors connected to a court; 0 elsewhere
	for z, patrol := range m.patrols(w, m.onDuty(w)) {
		p[z] = 1 - math.Exp(-m.k*patrol/float64(b.Actives[z]+1))
	}
	m.prev = append(m.prev[:0], b.Stage...)
	for _, id := range w.Alive {
		b.Stage[id] = world.Calm
		floor := w.Space.Where(w, id)
		if b.G[id] <= m.threshold || floor == 0 || !m.eligible(w, id) {
			continue // G − N > T needs G > T, as N ≥ 0; children have G = 0
		}
		pz := 0.0
		if m.reach[floor-1] {
			pz = p[w.Floors[floor-1].Zone]
		}
		r := w.Beliefs.Risk[id]
		n := r * (1 - (1-pz)*(1-c.Fear))
		if b.G[id]-n <= m.threshold {
			continue
		}
		b.Stage[id] = world.Active
		if m.joins(w, id, r) {
			b.Stage[id] = world.Joined
			if b.G[id]-n > m.violent {
				b.Stage[id] = world.Violent
			}
		}
		act := world.Organise
		if b.Stage[id] == world.Violent {
			act = world.Violence
		}
		c.Report(world.Act{Person: id, Other: world.None, Offence: act})
	}
}

// joins: at least a share r of the person's ties were active or beyond
// yesterday (ODD §6.14, Granovetter 1978). Someone with no ties never
// joins.
func (m *Module) joins(w *world.World, id world.PersonID, r float64) bool {
	ties := w.Network.TiesOf(w, id)
	if len(ties) == 0 {
		return false
	}
	out := 0
	for _, t := range ties {
		if int(t.Other) < len(m.prev) && m.prev[t.Other] != world.Calm {
			out++
		}
	}
	return float64(out) >= r*float64(len(ties))
}

// activeShare is today's share of adults active or beyond.
func (m *Module) activeShare(w *world.World) float64 {
	adults, active := 0, 0
	for _, id := range w.Alive {
		if w.Day-w.People[id].BirthDay >= m.adultAge {
			adults++
			if w.Rebellion.Stage[id] != world.Calm {
				active++
			}
		}
	}
	if adults == 0 {
		return 0
	}
	return float64(active) / float64(adults)
}

// strikes counts today's strikers per department, before today's stages
// replace yesterday's (ODD §6.14, Strikes).
func (m *Module) strikes(w *world.World) {
	b := w.Rebellion
	clear(b.Striking)
	for _, id := range w.Alive {
		if p := &w.People[id]; p.Stage == world.Worker && b.OnStrike(w, id) {
			b.Striking[p.Department]++
		}
	}
}

// action reports whether a collective action is under way today (ODD
// §6.14): a fight, a held zone, or a critical department on strike, its
// strikers at least a share q_strike of its workers.
func (m *Module) action(w *world.World) bool {
	if m.fought || slices.Contains(w.Rebellion.Held[:], true) {
		return true
	}
	for _, d := range m.critical {
		if n := w.Rebellion.Striking[d]; n > 0 && float64(n) >= m.strikeShare*float64(w.Depts[d].Headcount) {
			return true
		}
	}
	return false
}

// ODD §1.2, Definitions: unrest and rebellion. A rebellion starts on the
// last of RebellionDays days in a row with at least RebellionShare of
// adults active, if a collective action was under way on one of them; its
// start day is the first of them. Unrest starts on the last of UnrestDays
// days in a row with at least UnrestShare active, its start day the first
// of them. Both end after QuietDays days in a row below UnrestShare, their
// end day the first of those. An unrest spell during which a rebellion
// starts is that rebellion's run-up, not unrest, and no unrest is tracked
// while a rebellion runs (Adam, 2026-10-31). Rebels holding Up Top for
// RegimeDays days in a row end a rebellion with a regime change, that day;
// the count of held days starts again.
func (m *Module) track(w *world.World, share float64, action bool) {
	b, d := w.Rebellion, m.def
	if share >= d.UnrestShare {
		b.Busy, b.Quiet = b.Busy+1, 0
	} else {
		b.Busy, b.Quiet = 0, b.Quiet+1
	}
	if share >= d.RebellionShare {
		b.Rising++
	} else {
		b.Rising = 0
	}
	if action {
		b.LastAction = w.Day
	}
	head := func(kind string) world.EventHeader { return world.EventHeader{Day: w.Day, Type: kind} }
	quietSince := w.Day - world.Tick(d.QuietDays) + 1
	switch {
	case b.RebellionStart != world.None && b.HeldDays[world.UpTop] >= d.RegimeDays:
		w.Events = append(w.Events, world.RebellionEvent{EventHeader: head("rebellionEnd"),
			Start: b.RebellionStart, End: w.Day, Ending: "regimeChange"})
		b.RebellionStart, b.HeldDays[world.UpTop] = world.None, 0
	case b.RebellionStart != world.None:
		if b.Quiet >= d.QuietDays {
			w.Events = append(w.Events, world.RebellionEvent{EventHeader: head("rebellionEnd"),
				Start: b.RebellionStart, End: quietSince, Ending: "suppressed"})
			b.RebellionStart = world.None
		}
	case b.Rising >= d.RebellionDays && b.LastAction != world.None && b.LastAction > w.Day-world.Tick(d.RebellionDays):
		b.RebellionStart, b.UnrestStart = w.Day-world.Tick(d.RebellionDays)+1, world.None
		b.Rebellions++
		w.Events = append(w.Events, world.RebellionEvent{EventHeader: head("rebellionStart"), Start: b.RebellionStart})
	case b.UnrestStart != world.None:
		if b.Quiet >= d.QuietDays {
			w.Events = append(w.Events, world.UnrestEvent{EventHeader: head("unrest"), Start: b.UnrestStart, End: quietSince})
			b.UnrestStart = world.None
			b.UnrestSpells++
		}
	case b.Busy >= d.UnrestDays:
		b.UnrestStart = w.Day - world.Tick(d.UnrestDays) + 1
	}
}

// patrols spreads the officers over the zones in proportion to
// yesterday's adults on floors connected to a court (ODD §6.14, Arrest
// estimate).
func (m *Module) patrols(w *world.World, officers int) [world.NumZones]float64 {
	var p [world.NumZones]float64
	reach := 0
	for _, n := range w.Rebellion.Reach {
		reach += n
	}
	for z := range p {
		if reach > 0 {
			p[z] = float64(officers) * float64(w.Rebellion.Reach[z]) / float64(reach)
		}
	}
	return p
}

// ODD §6.14, Fights (decision D6). In each zone, Up Top first, where
// violent rebels on floors connected to a court (B of them) meet a patrol
// O > 0, Lanchester's square law puts out min(B, e_J × O) rebels, drawn
// uniformly from them, and min(O, e_R × B) officers, drawn uniformly from
// all officers on duty, as patrols are abstract. A fraction is rounded
// down, plus one with a chance equal to it. Each one put out is killed
// with chance p_kill, dying at the next deaths step, and their family has
// a loss; otherwise a rebel is arrested (control takes them into custody
// at its next step) and an officer wounded (a work injury). A rebel put
// out is calm from then on. Then the held zones.
func (m *Module) fights(w *world.World, r *rng.Stream) {
	b, c := w.Rebellion, w.Control
	m.fought = false
	officers := m.onDuty(w)
	patrol := m.patrols(w, officers)
	var duty []world.PersonID // officers on duty not yet put out today, built at the first fight
	killed := 0
	for z := range world.NumZones {
		m.pool = m.pool[:0]
		for _, id := range w.Alive {
			if f := w.Space.Where(w, id); b.Stage[id] == world.Violent && f != 0 && m.reach[f-1] && w.Floors[f-1].Zone == world.Zone(z) {
				m.pool = append(m.pool, id)
			}
		}
		if len(m.pool) == 0 || patrol[z] == 0 {
			continue
		}
		m.fought = true
		if duty == nil {
			for _, id := range w.Alive {
				if m.officer(w, id) {
					duty = append(duty, id)
				}
			}
		}
		for _, id := range draw(r, m.pool, round(r, min(float64(len(m.pool)), m.eJ*patrol[z]))) {
			b.Stage[id] = world.Calm
			if r.Float64() < m.kill {
				c.Doomed = append(c.Doomed, id)
				m.grieve(w, id)
				b.Tally.RebelsKilled++
			} else {
				c.Seized = append(c.Seized, id)
				b.Tally.RebelsArrested++
			}
		}
		n := min(len(duty), round(r, min(patrol[z], m.eR*float64(len(m.pool)))))
		out := draw(r, duty, n)
		for _, id := range out {
			if r.Float64() < m.kill {
				c.Doomed = append(c.Doomed, id)
				m.grieve(w, id)
				b.Tally.OfficersKilled++
				killed++
			} else {
				p := &w.People[id]
				p.Health = max(0, p.Health-m.injury)
				b.Tally.OfficersWounded++
			}
		}
		duty = duty[len(out):] // draw moved the ones put out to the front
	}
	m.hold(w, m.patrols(w, officers-killed))
}

// round rounds x down, plus one with a chance equal to the fraction.
func round(r *rng.Stream, x float64) int {
	n := int(x)
	if f := x - float64(n); f > 0 && r.Float64() < f {
		n++
	}
	return n
}

// draw moves n people drawn uniformly from ids to its front (a partial
// Fisher–Yates shuffle) and returns them.
func draw(r *rng.Stream, ids []world.PersonID, n int) []world.PersonID {
	for i := range n {
		j := r.IntRange(i, len(ids)-1)
		ids[i], ids[j] = ids[j], ids[i]
	}
	return ids[:n]
}

// ODD §6.14, Held zones: rebels hold a zone when its violent rebels,
// wherever they are in it, outnumber its patrol after today's fights.
func (m *Module) hold(w *world.World, patrol [world.NumZones]float64) {
	b := w.Rebellion
	var violent [world.NumZones]int
	for _, id := range w.Alive {
		if f := w.Space.Where(w, id); b.Stage[id] == world.Violent && f != 0 {
			violent[w.Floors[f-1].Zone]++
		}
	}
	b.Patrol = patrol
	for z := range world.NumZones {
		held := float64(violent[z]) > patrol[z]
		if held {
			b.HeldDays[z]++
		} else {
			b.HeldDays[z] = 0
		}
		if held != b.Held[z] {
			kind := "zoneFreed"
			if held {
				kind = "zoneHeld"
			}
			w.Events = append(w.Events, world.ZoneEvent{EventHeader: world.EventHeader{Day: w.Day, Type: kind}, Zone: world.Zone(z).String()})
		}
		b.Held[z] = held
	}
}

// eligible: not an officer on duty, the Head of IT or held by Judicial
// (ODD §6.14, Stages).
func (m *Module) eligible(w *world.World, id world.PersonID) bool {
	if w.Control.IsHead(id) || w.Control.Held(id) {
		return false
	}
	return !m.officer(w, id)
}

// officer: a Judicial or Sheriff worker not off work (ODD §6.13).
func (m *Module) officer(w *world.World, id world.PersonID) bool {
	p := &w.People[id]
	return p.Stage == world.Worker && slices.Contains(m.officers, p.Department) && !w.Space.Off(id)
}

// onDuty counts the officers: the workers of Judicial and the Sheriff not
// off work (ODD §6.13, Arrests).
func (m *Module) onDuty(w *world.World) int {
	n := 0
	for _, d := range m.officers {
		n += w.Depts[d].Headcount
	}
	return n
}

// reachable marks the floors connected to a court today, walking out
// from each court until a broken segment (segment s joins floors s and
// s + 1). Asking Space.Connected for every floor and court costs a
// floor-length walk each.
func (m *Module) reachable(w *world.World) {
	clear(m.reach)
	broken := w.Space.Broken
	for _, c := range m.courts {
		m.reach[c-1] = true
		for f := c; f > 1 && !broken[f-2]; f-- {
			m.reach[f-2] = true
		}
		for f := c; f < len(m.reach) && !broken[f-1]; f++ {
			m.reach[f] = true
		}
	}
}

// count keeps tomorrow's zone counts (ODD §6.14, Step 9): adults active
// or beyond, and adults on floors connected to a court, by the zone of
// their day location.
func (m *Module) count(w *world.World) {
	b := w.Rebellion
	m.reachable(w)
	b.Actives, b.Reach = [world.NumZones]int{}, [world.NumZones]int{}
	for _, id := range w.Alive {
		floor := w.Space.Where(w, id)
		if floor == 0 || w.Day-w.People[id].BirthDay < m.adultAge {
			continue
		}
		z := w.Floors[floor-1].Zone
		if b.Stage[id] != world.Calm {
			b.Actives[z]++
		}
		if m.reach[floor-1] {
			b.Reach[z]++
		}
	}
}
