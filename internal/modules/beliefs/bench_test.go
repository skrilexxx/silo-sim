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
