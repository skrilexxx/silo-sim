package space

import (
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/work"
	"github.com/skrilexxx/silo-sim/internal/rng"
)

// The location step on a day-1 silo of 10,000 people (T5.4: profile
// before optimising). Run with: go test ./internal/modules/space -bench .
func BenchmarkLocate(b *testing.B) {
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		b.Fatal(err)
	}
	r, _ := rng.New(cfg.Seed, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		b.Fatal(err)
	}
	if err := work.New().Init(w, cfg, r); err != nil {
		b.Fatal(err)
	}
	m := New()
	if err := m.Init(w, cfg, r); err != nil {
		b.Fatal(err)
	}
	r.Shuffle(len(w.Alive), func(i, j int) { w.Alive[i], w.Alive[j] = w.Alive[j], w.Alive[i] })
	for b.Loop() {
		m.locate(w, r)
	}
}
