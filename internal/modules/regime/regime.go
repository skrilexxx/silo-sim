// Package regime is regime policy and response (ODD §6.15, draft): the
// playbook's entries start on rebellion triggers, hold the levers for a
// while and lapse, and the regime declares an emergency when air falls.
// The strategies are scenario files; this package holds no strategy and
// no numbers.
package regime

import (
	"fmt"
	"slices"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	cfg     *config.Scenario
	pending []config.Entry // today's policy events, started after the playbook's entries
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "regime" }

// Requires rebellion: its triggers are rebellion events, and through it
// control (the Head of IT and the levers it reads) and beliefs.
func (m *Module) Requires() []string { return []string{"rebellion"} }

func (m *Module) Init(w *world.World, cfg *config.Scenario, _ *rng.Stream) error {
	if w.Rebellion == nil {
		return fmt.Errorf("regime: no rebellion state; list rebellion before regime in modules")
	}
	m.cfg = cfg
	w.Regime = &world.Regime{}
	w.Policy = cfg.InForce(nil)
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseRegime, Run: m.step}}
}

// EventTypes: ODD §6.15, Scripted events. A policy event is an entry
// imposed on its day; it starts at the regime step, after the day's
// playbook entries.
func (m *Module) EventTypes() []engine.EventType {
	return []engine.EventType{{Name: "policy", Phase: engine.PhaseRegime, Fire: func(_ *world.World, _ *rng.Stream, ev config.Event) error {
		m.pending = append(m.pending, ev.Entry())
		return nil
	}}}
}

// ODD §6.15 / §3 step 10.95: entries lapse, the day's triggers start
// theirs and then the policy events start; then the emergency is judged on
// today's air.
func (m *Module) step(w *world.World, _ *rng.Stream) error {
	g := w.Regime
	changed := m.lapse(w)
	for _, t := range w.Rebellion.Fired {
		changed = m.trigger(w, t) || changed
	}
	for i := range m.pending {
		m.start(w, &m.pending[i], "scripted", world.None)
		changed = true
	}
	m.pending = nil // the entries keep pointers into the old slice
	if changed {
		running := make([]config.Entry, len(g.Running))
		for i, r := range g.Running {
			running[i] = *r.Entry
		}
		p := m.cfg.InForce(running)
		if err := p.CheckPact(); err != nil {
			return fmt.Errorf("the running entries put in force a Pact that fails ODD §6.9: %w", err)
		}
		w.Policy = p
	}
	m.emergency(w)
	return nil
}

// lapse ends the entries whose hold has run out: a number of days, or
// quiet once no unrest spell or rebellion is open. It reports whether any
// did.
func (m *Module) lapse(w *world.World) bool {
	b, g := w.Rebellion, w.Regime
	open := b.UnrestStart != world.None || b.RebellionStart != world.None
	n := len(g.Running)
	g.Running = slices.DeleteFunc(g.Running, func(r world.Running) bool {
		if r.Quiet && !open || r.Until != world.None && w.Day >= r.Until {
			m.log(w, "policyEnd", r)
			return true
		}
		return false
	})
	return len(g.Running) < n
}

// trigger starts the playbook's entries for one trigger, in file order.
// The first regime change ends every running entry and the old playbook
// with it: from then on only the regimeChange entries apply, for good. A
// later regime change starts nothing. While the Head of IT's post is
// vacant the old regime starts nothing. It reports whether anything
// changed.
func (m *Module) trigger(w *world.World, t string) bool {
	g := w.Regime
	changed := false
	switch {
	case t == "regimeChange" && g.Changed:
		return false
	case t == "regimeChange":
		g.Changed, changed = true, len(g.Running) > 0
		for _, r := range g.Running {
			m.log(w, "policyEnd", r)
		}
		g.Running = g.Running[:0]
	case g.Changed || w.Control.Head == world.None:
		return false
	}
	for i := range m.cfg.Regime.Playbook {
		if e := &m.cfg.Regime.Playbook[i]; e.On == t {
			m.start(w, e, t, i)
			changed = true
		}
	}
	return changed
}

// start puts an entry in force from today's regime step: its levers and
// rows through the next policy, its lift and water dosing at once. Held a
// number of days, it lapses at the regime step that many days later, so
// every step reads it on exactly that many days.
func (m *Module) start(w *world.World, e *config.Entry, trigger string, index int) {
	days, _ := config.ParseHold(e.Hold) // checked at load
	r := world.Running{Entry: e, Trigger: trigger, Index: index, Start: w.Day, Until: world.None, Quiet: days == config.HoldQuiet}
	switch {
	case trigger == "regimeChange":
		r.Quiet = false // held for good
	case days > 0:
		r.Until = w.Day + world.Tick(days)
	}
	w.Regime.Running = append(w.Regime.Running, r)
	w.Beliefs.Lift += e.Lift
	if e.WaterDosing {
		w.Beliefs.WaterOrdered = true
	}
	m.log(w, "policy", r)
}

// emergency: ODD §6.15, Emergency staffing. Air below the line in force
// today declares an emergency, which control (releases) and work
// (retraining) act on at their next steps. Its start and end are logged.
func (m *Module) emergency(w *world.World) {
	g := w.Regime
	now := w.Silo.AirQuality < w.Policy.Regime.EmergencyAir
	if now != g.Emergency {
		kind := "emergency"
		if !now {
			kind = "emergencyEnd"
		}
		w.Events = append(w.Events, world.EventHeader{Day: w.Day, Type: kind})
	}
	g.Emergency = now
	if now {
		g.EmergencyDays++
	}
}

func (m *Module) log(w *world.World, kind string, r world.Running) {
	ev := world.PolicyEvent{EventHeader: world.EventHeader{Day: w.Day, Type: kind}, Trigger: r.Trigger, Start: r.Start}
	if r.Index != world.None {
		ev.Entry = &r.Index
	}
	w.Events = append(w.Events, ev)
}
