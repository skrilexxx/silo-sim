// Package health holds daily health changes and Medical's effect on them
// (ODD §6.2). Mortality reads health (§6.1); implant removals and maternal
// deaths, which act on births, live in demography and read Medical
// coverage from the world.
package health

import (
	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	c config.Health
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "health" }

// Requires work: Medical coverage and the departments behind injury risk.
func (m *Module) Requires() []string { return []string{"work"} }

func (m *Module) Init(_ *world.World, cfg *config.Scenario, _ *rng.Stream) error {
	m.c = cfg.Health
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseHealth, Run: m.health}}
}

// ODD §6.2: daily health change, for every living person in the tick's
// shuffled order:
//
//	health −= hungerDamage × max(0, 1 − intake/need of their zone)
//	health −= airDamage × (1 − airQuality)
//	health −= injuryCost   with chance injuryChance × department factor (workers and shadows)
//	health −= illnessCost  with chance illnessChance
//	health += recoveryRate × medicalCoverage
//
// then clamped to 0–1. Food above need (possible under unequal ration
// weights, §6.7) does not raise health.
func (m *Module) health(w *world.World, r *rng.Stream) error {
	medical, mechanical, farms := world.None, world.None, world.None
	for d, dep := range w.Depts {
		switch dep.Name {
		case "medical":
			medical = d
		case "mechanical":
			mechanical = d
		case "farms":
			farms = d
		}
	}
	// Departments come from the work module's Init, which may run after
	// this module's; looked up here, they are always there.
	recovery := m.c.RecoveryRate * w.Depts[medical].Coverage
	air := m.c.AirDamage * (1 - w.Silo.AirQuality)
	var hunger [world.NumZones]float64
	for z := range world.NumZones {
		hunger[z] = m.c.HungerDamage * max(0, 1-w.Silo.Intake[z])
	}

	for _, id := range w.Alive {
		p := &w.People[id]
		h := p.Health - air - hunger[w.Floors[w.Households[p.Household].Floor-1].Zone]
		if p.Stage == world.Worker || p.Stage == world.Shadow {
			factor := m.c.InjuryFactor.Other
			switch p.Department {
			case mechanical:
				factor = m.c.InjuryFactor.Mechanical
			case farms:
				factor = m.c.InjuryFactor.Farms
			}
			if r.Bernoulli(m.c.InjuryChance * factor) {
				h -= m.c.InjuryCost
			}
		}
		if r.Bernoulli(m.c.IllnessChance) {
			h -= m.c.IllnessCost
		}
		p.Health = world.Clamp01(h + recovery)
	}
	return nil
}
