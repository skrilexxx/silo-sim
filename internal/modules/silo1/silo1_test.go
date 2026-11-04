package silo1

import (
	"slices"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// ODD §6.16: each escape is logged and orders a response that day; the
// Safeguard is set off, once, when a second escape falls within 5 years of
// the first, 30 days later. Escapes 6 years apart set nothing off.
func TestEscapesAndSafeguard(t *testing.T) {
	run := func(gapYears int) *world.World {
		cfg := &config.Scenario{}
		cfg.Silo1 = config.Silo1{SafeguardEscapes: 2, WindowYears: 5, SafeguardDelayDays: 30}
		w := &world.World{Regime: &world.Regime{}, Control: &world.Control{}}
		m := New()
		if err := m.Init(w, cfg, nil); err != nil {
			t.Fatal(err)
		}
		escapes := []world.Tick{100, 100 + world.Tick(gapYears*engine.DaysPerYear), 5000}
		var orders []world.Tick
		for w.Day = 1; w.Day <= 6000; w.Day++ {
			if slices.Contains(escapes, w.Day) {
				w.Control.Tally.Escapes++
			}
			if err := m.step(w, nil); err != nil {
				t.Fatal(err)
			}
			if w.Silo1.Order {
				orders = append(orders, w.Day)
			}
		}
		if !slices.Equal(orders, escapes) {
			t.Errorf("gap %d years: orders on %v, want on each escape %v", gapYears, orders, escapes)
		}
		return w
	}
	if w := run(4); w.Silo1.Safeguard != 100+4*365+30 {
		t.Errorf("two escapes 4 years apart: Safeguard on day %d, want %d", w.Silo1.Safeguard, 100+4*365+30)
	}
	if w := run(6); w.Silo1.Safeguard != world.None {
		t.Errorf("escapes 6 years apart: Safeguard on day %d, want none", w.Silo1.Safeguard)
	}
}
