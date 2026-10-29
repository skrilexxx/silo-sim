// Package resources holds the silo's resource loop (ODD §6.6): the
// generator and power, parts, water, air and food; rations (§6.7) arrive
// in T3.3.
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
	rations    config.Rations
	farmFloors int
	capacity   float64 // capacityMargin × day-1 power demand
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "resources" }

// Requires work: Mechanical coverage drives repair and power.
func (m *Module) Requires() []string { return []string{"work"} }

// ODD §5 step 8 and §6.6: starting condition, parts and food, and the
// generator's capacity, fixed from the day-1 demand.
func (m *Module) Init(w *world.World, cfg *config.Scenario, _ *rng.Stream) error {
	m.c, m.rations = cfg.Resources, cfg.Rations
	m.farmFloors = 0
	for _, f := range w.Floors {
		if f.Function == world.Farm {
			m.farmFloors++
		}
	}
	w.Silo.GeneratorCondition = cfg.Init.GeneratorCondition
	// Parts are counted in days of normal use: one repair day's worth.
	w.Silo.PartsStock = float64(cfg.Init.PartsStockDays) * m.c.PartsPerRepairDay
	w.Silo.FoodStock = float64(cfg.Init.FoodStockDays) * m.foodNeed(w)
	m.capacity = m.c.CapacityMargin * m.demand(w)
	return nil
}

// ODD §6.7: daily food need, adultKcal per person and childNeedShare of
// that for children under childAgeBelow.
func (m *Module) foodNeed(w *world.World) float64 {
	childDays := world.Tick(m.rations.ChildAgeBelow * engine.DaysPerYear)
	need := 0.0
	for _, id := range w.Alive {
		if w.Day-w.People[id].BirthDay < childDays {
			need += m.rations.ChildNeedShare * m.rations.AdultKcal
		} else {
			need += m.rations.AdultKcal
		}
	}
	return need
}

// dept finds a department by name. Departments come from the work
// module's Init, which may run after this module's, so steps look them up.
func dept(w *world.World, name string) (int, error) {
	for d, dep := range w.Depts {
		if dep.Name == name {
			return d, nil
		}
	}
	return world.None, fmt.Errorf("resources: no department named %q", name)
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{
		{Phase: engine.PhaseGenerator, Run: m.generator},
		{Phase: engine.PhaseLifeSupport, Run: m.lifeSupport},
		{Phase: engine.PhaseFood, Run: m.food},
	}
}

// ODD §6.6: power demand is fixed per person and per farm floor.
func (m *Module) demand(w *world.World) float64 {
	return m.c.PowerDemandPerPerson*float64(len(w.Alive)) + m.c.PowerDemandPerFarmFloor*float64(m.farmFloors)
}

// ODD §6.6: generator, daily. First Supply's parts are added:
// partsPerDayFullCoverage × supplyCoverage, outage days too, so today's
// parts can serve today's repair. While the generator is down only the
// countdown runs:
// condition, parts and the failure draw are frozen. While it runs:
//
//	c −= wear
//	c += ρ × mechanicalCoverage × (1 − c)   if a repair day's parts are in stock (and uses them)
//	failure with chance failChance(c): c −= drop, parts −= min(parts, partsPerFailure),
//	                                   down for a uniform outageMinDays–outageMaxDays days
//
// Then supply = capacity × c × mechanicalCoverage, or backupPower ×
// capacity while down. Coverages are yesterday's: the work step (§3 step
// 5) comes after this one.
func (m *Module) generator(w *world.World, r *rng.Stream) error {
	mechanical, err := dept(w, "mechanical")
	if err != nil {
		return err
	}
	supply, err := dept(w, "supply")
	if err != nil {
		return err
	}
	cov := w.Depts[mechanical].Coverage
	s := &w.Silo
	s.PartsStock += m.c.PartsPerDayFullCoverage * w.Depts[supply].Coverage

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

// ODD §6.6: life support, daily. Water = powerFactor × mechanicalCoverage;
// air moves airRecoveryShare of the way toward powerFactor, so a short
// outage is survivable and a long one is not.
func (m *Module) lifeSupport(w *world.World, _ *rng.Stream) error {
	mechanical, err := dept(w, "mechanical")
	if err != nil {
		return err
	}
	s := &w.Silo
	pf := s.PowerFactor()
	s.WaterSupply = pf * w.Depts[mechanical].Coverage
	s.AirQuality = world.Clamp01(s.AirQuality + m.c.AirRecoveryShare*(pf-s.AirQuality))
	return nil
}

// ODD §6.6: food production, daily: farm output × kcalPerFarmUnit ×
// powerFactor, added to the stock. Farm output is yesterday's (work runs
// at §3 step 5). The opening stock and the amount produced are kept for
// the food-conservation invariant (§8).
func (m *Module) food(w *world.World, _ *rng.Stream) error {
	farms, err := dept(w, "farms")
	if err != nil {
		return err
	}
	s := &w.Silo
	s.FoodOpening = s.FoodStock
	s.FoodProduced = w.Depts[farms].Output * m.c.KcalPerFarmUnit * s.PowerFactor()
	s.FoodStock = s.FoodOpening + s.FoodProduced
	return nil
}
