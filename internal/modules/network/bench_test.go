package network

import (
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/space"
	"github.com/skrilexxx/silo-sim/internal/modules/work"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// The network step on a day-1 silo of 10,000 people, after day 1's draw
// (T5.4: profile before optimising). go test ./internal/modules/network -bench .
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
	for _, m := range []interface {
		Init(*world.World, *config.Scenario, *rng.Stream) error
	}{work.New(), space.New(), New()} {
		if err := m.Init(w, cfg, r); err != nil {
			b.Fatal(err)
		}
	}
	m := New()
	m.Init(w, cfg, r) // a fresh module on the same world: what a day costs
	for b.Loop() {
		m.step(w, r)
	}
}
