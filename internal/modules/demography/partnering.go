package demography

import (
	"math"

	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// ODD §6.3: partnering. Each day each unpartnered adult, in the tick's
// shuffled order, starts a search with chance searchRate and at once
// pairs with someone chosen by partnerWeight from the other unpartnered
// adults. The new couple joins the housing queue.
func (m *Module) partnering(w *world.World, r *rng.Stream) error {
	adult := world.Tick(m.cfg.Partnering.MinAge * engine.DaysPerYear)
	m.pool, m.poolZ = m.pool[:0], m.poolZ[:0]
	for _, id := range w.Alive {
		p := &w.People[id]
		if p.Partner == world.None && w.Day-p.BirthDay >= adult {
			m.pool = append(m.pool, id)
			m.poolZ = append(m.poolZ, homeZone(w, id))
		}
	}
	for i, id := range m.pool {
		if w.People[id].Partner != world.None {
			continue // chosen by an earlier searcher today
		}
		if !r.Bernoulli(m.cfg.Partnering.SearchRate) {
			continue
		}
		j := m.choose(w, r, i)
		if j < 0 {
			continue
		}
		mate := m.pool[j]
		w.People[id].Partner, w.People[mate].Partner = mate, id
		w.HousingQueue = append(w.HousingQueue, world.HousingRequest{A: id, B: mate})
	}
	return nil
}

// choose returns the pool index of the partner picked by pool[i], or -1 if
// there is no candidate.
func (m *Module) choose(w *world.World, r *rng.Stream, i int) int {
	m.weights = m.weights[:0]
	for j := range m.pool {
		m.weights = append(m.weights, m.partnerWeight(w, i, j))
	}
	return r.WeightedIndex(m.weights)
}

// ODD §6.3: the weight of pool[j] as a partner for pool[i]:
// 1 × sameZoneWeight if they live in the same zone × sameDepartmentWeight
// if they work in the same department × sameSexWeight if they are the same
// sex × e^(−|age gap| / ageGapScaleYears). Zero for themselves, anyone
// already partnered today, and close relatives (parent, child, sibling).
func (m *Module) partnerWeight(w *world.World, i, j int) float64 {
	a, b := &w.People[m.pool[i]], &w.People[m.pool[j]]
	if i == j || b.Partner != world.None || related(a, b) {
		return 0
	}
	c := m.cfg.Partnering
	weight := 1.0
	if m.poolZ[i] == m.poolZ[j] {
		weight *= c.SameZoneWeight
	}
	if a.Department != world.None && a.Department == b.Department {
		weight *= c.SameDepartmentWeight
	}
	if a.Sex == b.Sex {
		weight *= c.SameSexWeight
	}
	gapYears := math.Abs(float64(a.BirthDay-b.BirthDay)) / engine.DaysPerYear
	return weight * math.Exp(-gapYears/c.AgeGapScaleYears)
}

// related reports whether a and b are parent and child, or siblings
// (sharing a mother or a father, so half-siblings count).
func related(a, b *world.Person) bool {
	switch {
	case a.Mother == b.ID, a.Father == b.ID, b.Mother == a.ID, b.Father == a.ID:
		return true
	case a.Mother != world.None && a.Mother == b.Mother:
		return true
	case a.Father != world.None && a.Father == b.Father:
		return true
	}
	return false
}

// homeZone is the zone of a living person's household.
func homeZone(w *world.World, id world.PersonID) world.Zone {
	return w.Floors[w.Households[w.People[id].Household].Floor-1].Zone
}
