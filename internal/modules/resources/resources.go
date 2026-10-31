// Package resources holds the silo's resource loop (ODD §6.6): the
// generator and power, parts, water, air and food, and rations (§6.7).
package resources

import (
	"fmt"
	"math"

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
	need := m.zoneNeed(w)
	w.Silo.FoodStock = float64(cfg.Init.FoodStockDays) * (need[world.UpTop] + need[world.Mids] + need[world.DownDeep])
	w.Silo.RationShares = [world.NumZones]float64{m.rations.Weights.UpTop, m.rations.Weights.Mids, m.rations.Weights.DownDeep}
	m.capacity = m.c.CapacityMargin * m.demand(w)
	return nil
}

// ODD §6.7: daily food need per zone, adultKcal per person and
// childNeedShare of that for children under childAgeBelow.
func (m *Module) zoneNeed(w *world.World) [world.NumZones]float64 {
	childDays := world.Tick(m.rations.ChildAgeBelow * engine.DaysPerYear)
	var need [world.NumZones]float64
	for _, id := range w.Alive {
		p := &w.People[id]
		z := w.Floors[w.Households[p.Household].Floor-1].Zone
		if w.Day-p.BirthDay < childDays {
			need[z] += m.rations.ChildNeedShare * m.rations.AdultKcal
		} else {
			need[z] += m.rations.AdultKcal
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
		{Phase: engine.PhaseRations, Run: m.rationsStep},
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
//	failure with chance failChance(c): c −= drop, down for outageDays days,
//	                                   then parts −= min(parts, partsPerFailure)
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
		if s.GeneratorDown == 0 {
			w.Events = append(w.Events, world.GeneratorEvent{EventHeader: world.EventHeader{Day: w.Day, Type: "repair"},
				Condition: s.GeneratorCondition, Parts: s.PartsStock})
		}
	}
	if s.GeneratorDown == 0 {
		c := s.GeneratorCondition - m.c.GeneratorWear
		if s.PartsStock >= m.c.PartsPerRepairDay {
			c += m.c.RepairRate * cov * (1 - c)
			s.PartsStock -= m.c.PartsPerRepairDay
		}
		c = world.Clamp01(c)
		failed := r.Bernoulli(m.failChance(c))
		if b := w.Rebellion; b != nil && b.GeneratorHit { // ODD §6.14: sabotage, as if the draw had come up
			failed, b.GeneratorHit = true, false
		}
		if failed {
			c = world.Clamp01(c - m.c.FailureConditionDrop)
			s.GeneratorDown = m.outageDays(w, mechanical)
			s.PartsStock -= min(s.PartsStock, m.c.PartsPerFailure)
			w.Events = append(w.Events, world.GeneratorEvent{EventHeader: world.EventHeader{Day: w.Day, Type: "failure"},
				OutageDays: s.GeneratorDown, Condition: c, Parts: s.PartsStock})
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

// ODD §6.6: outage length. Repair time is the work divided by the rate
// Mechanical can do it at: ⌈outageMinDays / (skill × partsShare)⌉ days,
// at most outageMaxDays. skill is the average skill of Mechanical workers
// (shadows not counted); partsShare = min(1, parts / partsPerFailure),
// from the stock before the failure takes its parts. No workers or no
// parts gives the cap.
func (m *Module) outageDays(w *world.World, mechanical int) int {
	skill, workers := 0.0, 0
	for _, id := range w.Alive {
		if p := &w.People[id]; p.Stage == world.Worker && p.Department == mechanical {
			skill += p.Skill
			workers++
		}
	}
	if workers > 0 {
		skill /= float64(workers)
	}
	parts := 1.0
	if m.c.PartsPerFailure > 0 {
		parts = min(1, w.Silo.PartsStock/m.c.PartsPerFailure)
	}
	// A zero rate divides to +Inf, which the cap turns into outageMaxDays.
	return int(min(float64(m.c.OutageMaxDays), math.Ceil(float64(m.c.OutageMinDays)/(skill*parts))))
}

// ODD §6.6: daily failure chance f₀ × (1 − c)².
func (m *Module) failChance(c float64) float64 {
	return m.c.FailureBase * (1 - c) * (1 - c)
}

// ODD §6.6: life support, daily. Water = powerFactor × mechanicalCoverage;
// air moves airRecoveryShare of the way toward powerFactor, so a short
// outage is survivable and a long one is not.
//
// Part 2 (ODD §6.14): a water pump wrecked yesterday is down for as long as
// a generator outage takes to repair, using the same parts; while it is
// down, water is 0.
func (m *Module) lifeSupport(w *world.World, _ *rng.Stream) error {
	mechanical, err := dept(w, "mechanical")
	if err != nil {
		return err
	}
	s := &w.Silo
	pf := s.PowerFactor()
	s.WaterSupply = pf * w.Depts[mechanical].Coverage
	if b := w.Rebellion; b != nil {
		if b.PumpHit {
			b.PumpHit, b.PumpUntil = false, w.Day+world.Tick(m.outageDays(w, mechanical))-1
			s.PartsStock -= min(s.PartsStock, m.c.PartsPerFailure)
		}
		if w.Day <= b.PumpUntil {
			s.WaterSupply = 0
		}
	}
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

// ODD §6.7: rations, daily. The silo hands out the day's total need, or
// rationingShare of it while the stock (after today's production) is
// below rationingBelowDays of need, and never more than the stock. Zones
// get share_z = w_z × need_z / Σ w × need, shared within the zone in
// proportion to need, so everyone in zone z has
//
//	intake/need = share_z × distributed / need_z = w_z × distributed / Σ w × need
//
// With equal weights that is the same everywhere. Under unequal weights a
// zone can get more than its need; all of it is handed out (hunger in
// §6.2 gains nothing from it).
func (m *Module) rationsStep(w *world.World, _ *rng.Stream) error {
	s := &w.Silo
	need := m.zoneNeed(w)
	total, weighted := 0.0, 0.0
	for z := range world.NumZones {
		total += need[z]
		weighted += s.RationShares[z] * need[z]
	}
	out := total
	s.FoodNeed = total
	if was := s.Rationing; was != (s.FoodStock < float64(m.rations.RationingBelowDays)*total) {
		s.Rationing = !was
		kind := "rationingOff"
		if s.Rationing {
			kind = "rationingOn"
		}
		w.Events = append(w.Events, world.RationingEvent{EventHeader: world.EventHeader{Day: w.Day, Type: kind},
			FoodStockDays: s.FoodStock / total})
	}
	if s.Rationing {
		out *= m.rations.RationingShare
	}
	out = min(out, s.FoodStock)
	if weighted > 0 {
		for z := range world.NumZones {
			s.Intake[z] = s.RationShares[z] * out / weighted
		}
	}
	s.FoodHandedOut = out
	s.FoodStock -= out
	return nil
}
