// Package resources holds the silo's resource loop (ODD §6.6): the
// generator and power so far; water, air, food and parts arrive in T3.2,
// rations (§6.7) in T3.3.
package resources

import (
	"fmt"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	c          config.Resources
	farmFloors int
	capacity   float64 // capacityMargin × day-1 power demand
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "resources" }

// Requires work: Mechanical coverage drives repair and power.
func (m *Module) Requires() []string { return []string{"work"} }

// ODD §5 step 8 and §6.6: starting condition and parts, and the
// generator's capacity, fixed from the day-1 demand.
func (m *Module) Init(w *world.World, cfg *config.Scenario, _ *rng.Stream) error {
	m.c = cfg.Resources
	m.farmFloors = 0
	for _, f := range w.Floors {
		if f.Function == world.Farm {
			m.farmFloors++
		}
	}
	w.Silo.GeneratorCondition = cfg.Init.GeneratorCondition
	// Parts are counted in days of normal use: one repair day's worth.
	w.Silo.PartsStock = float64(cfg.Init.PartsStockDays) * m.c.PartsPerRepairDay
	m.capacity = m.c.CapacityMargin * m.demand(w)
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseGenerator, Run: m.generator}}
}

// ODD §6.6: power demand is fixed per person and per farm floor.
func (m *Module) demand(w *world.World) float64 {
	return m.c.PowerDemandPerPerson*float64(len(w.Alive)) + m.c.PowerDemandPerFarmFloor*float64(m.farmFloors)
}

// ODD §6.6: generator, daily. While it is down only the countdown runs:
// condition, parts and the failure draw are frozen. While it runs:
//
//	c −= wear
//	c += ρ × mechanicalCoverage × (1 − c)   if a repair day's parts are in stock (and uses them)
//	failure with chance failChance(c): c −= drop, parts −= min(parts, partsPerFailure),
//	                                   down for a uniform outageMinDays–outageMaxDays days
//
// Then supply = capacity × c × mechanicalCoverage, or backupPower ×
// capacity while down. Mechanical coverage is yesterday's: the work step
// (§3 step 5) comes after this one.
func (m *Module) generator(w *world.World, r *rng.Stream) error {
	mechanical := world.None
	for d, dep := range w.Depts {
		if dep.Name == "mechanical" {
			mechanical = d
		}
	}
	if mechanical == world.None {
		return fmt.Errorf("resources: no department named \"mechanical\"")
	}
	cov := w.Depts[mechanical].Coverage
	s := &w.Silo

	if s.GeneratorDown > 0 {
		s.GeneratorDown--
	}
	if s.GeneratorDown == 0 {
		c := s.GeneratorCondition - m.c.GeneratorWear
		if s.PartsStock >= m.c.PartsPerRepairDay {
			c += m.c.RepairRate * cov * (1 - c)
			s.PartsStock -= m.c.PartsPerRepairDay
		}
		c = world.Clamp01(c)
		if r.Bernoulli(m.failChance(c)) {
			c = world.Clamp01(c - m.c.FailureConditionDrop)
			s.PartsStock -= min(s.PartsStock, m.c.PartsPerFailure)
			s.GeneratorDown = r.IntRange(m.c.OutageMinDays, m.c.OutageMaxDays)
		}
		s.GeneratorCondition = c
	}

	s.PowerDemand = m.demand(w)
	if s.GeneratorDown > 0 {
		s.PowerSupply = m.c.BackupPower * m.capacity
	} else {
		s.PowerSupply = m.capacity * s.GeneratorCondition * cov
	}
	return nil
}

// ODD §6.6: daily failure chance f₀ × (1 − c)².
func (m *Module) failChance(c float64) float64 {
	return m.c.FailureBase * (1 - c) * (1 - c)
}
