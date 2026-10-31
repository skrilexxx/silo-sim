package beliefs

import (
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// The beliefs step on a day-1 silo of 10,000 people, on a recompute day
// (day 0). T5.4: profile before optimising.
// go test ./internal/modules/beliefs -bench .
func BenchmarkStep(b *testing.B) {
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		b.Fatal(err)
	}
	r, _ := rng.New(cfg.Seed, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		b.Fatal(err)
	}
	w.Network = &world.Network{}
	m := New()
	if err := m.Init(w, cfg, r); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		m.step(w, r)
	}
}

// The conversations alone when half the silo knows the history, the
// costliest point of a spread: every knower has ties to people who don't.
func BenchmarkConverseHalfKnow(b *testing.B) {
	s := newStill(b, 1, 0, false) // π = 0: nobody learns, so every day costs the same
	for i, id := range s.w.Alive {
		if i%2 == 0 {
			s.w.Beliefs.Learn(id, world.FlagHistory)
		}
	}
	s.day()
	m := New()
	*m = Module{pass: 0, fresh: make([]world.Flags, len(s.w.People))}
	r, _ := rng.New(1, "beliefs")
	for b.Loop() {
		m.converse(s.w, r)
	}
}
