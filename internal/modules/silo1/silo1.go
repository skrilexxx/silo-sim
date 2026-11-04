// Package silo1 is Silo 1, lean (ODD §6.16, draft): it counts escapes,
// orders a response through the Head of IT after each one, and sets off
// the Safeguard when they come too often. The orders are carried out by
// the regime module; the Safeguard's collapse is recorded by the
// rebellion module.
package silo1

import (
	"fmt"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	need   int        // escapes within the window that set off the Safeguard
	window world.Tick // in days
	delay  world.Tick
	seen   int // escapes counted so far
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "silo1" }

// Requires regime: Silo 1's orders are entries the regime starts.
func (m *Module) Requires() []string { return []string{"regime"} }

func (m *Module) Init(w *world.World, cfg *config.Scenario, _ *rng.Stream) error {
	if w.Regime == nil {
		return fmt.Errorf("silo1: no regime state; list regime before silo1 in modules")
	}
	m.need, m.delay = cfg.Silo1.SafeguardEscapes, world.Tick(cfg.Silo1.SafeguardDelayDays)
	m.window = world.Tick(cfg.Silo1.WindowYears * engine.DaysPerYear)
	m.seen = w.Control.Tally.Escapes
	w.Silo1 = &world.Silo1{Safeguard: world.None}
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseSilo1, Run: m.step}}
}

// ODD §6.16 / §3 step 10.85: the day's escapes, each logged; an order for
// the regime step if there was one; and the Safeguard set off, once, when
// the escapes within the window reach the threshold.
func (m *Module) step(w *world.World, _ *rng.Stream) error {
	s := w.Silo1
	n := w.Control.Tally.Escapes - m.seen
	m.seen = w.Control.Tally.Escapes
	s.Order = n > 0
	for range n {
		s.Escapes = append(s.Escapes, w.Day)
		w.Events = append(w.Events, world.EventHeader{Day: w.Day, Type: "escape"})
	}
	if n == 0 || s.Safeguard != world.None {
		return nil
	}
	recent := 0
	for _, d := range s.Escapes {
		if d > w.Day-m.window {
			recent++
		}
	}
	if recent >= m.need {
		s.Safeguard = w.Day + m.delay
		w.Events = append(w.Events, world.SafeguardEvent{EventHeader: world.EventHeader{Day: w.Day, Type: "safeguard"}, GoesOff: s.Safeguard})
	}
	return nil
}
