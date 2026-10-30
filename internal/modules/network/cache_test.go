package network

import (
	"slices"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/modules/governance"
	"github.com/skrilexxx/silo-sim/internal/modules/health"
	"github.com/skrilexxx/silo-sim/internal/modules/resources"
	"github.com/skrilexxx/silo-sim/internal/modules/space"
	"github.com/skrilexxx/silo-sim/internal/modules/work"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// checkedNet runs the network step, then compares every cached tie list
// with what Network.Ties gives at that moment.
type checkedNet struct {
	*Module
	t       *testing.T
	checked *int
}

func (c checkedNet) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseNetwork, Run: func(w *world.World, r *rng.Stream) error {
		if err := c.step(w, r); err != nil {
			return err
		}
		for _, id := range w.Alive {
			cached, fresh := w.Network.TiesOf(w, id), w.Network.Ties(w, id, nil)
			if !slices.Equal(cached, fresh) {
				c.t.Fatalf("day %d: person %d cached ties %v, fresh %v", w.Day, id, cached, fresh)
			}
			*c.checked++
		}
		return nil
	}}}
}

// The cached tie lists are only a speed-up: in a full model run, right
// after every network step, each cached list must equal what Network.Ties
// gives. Births, deaths, couples, moves, shadowing and retraining all
// happen in 10 years of 1,000 people.
func TestCacheMatchesTies(t *testing.T) {
	cfg, err := config.Load("../../../testdata/golden/scenario.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Years = 10
	r, _ := rng.New(cfg.Seed, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	e, err := engine.New(cfg, w, []engine.Module{demography.New(), health.New(), work.New(), resources.New(),
		governance.New(), space.New(), checkedNet{New(), t, &checked}})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Run(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d cached lists checked", checked)
}
