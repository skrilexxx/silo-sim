package demography

import (
	"slices"

	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// ODD §6.3: housing, every 30 days. The queue is handled oldest first.
// Each couple gets a target zone the first time it is handled; it then
// moves into a random free home there, or keeps waiting (living apart)
// if none is free. A couple waiting for one zone never blocks couples
// waiting for another. A couple broken by a death leaves the queue.
func (m *Module) housing(w *world.World, r *rng.Stream) error {
	occupied := make([]int, len(w.Floors))
	for _, h := range w.Households {
		if len(h.Members) > 0 {
			occupied[h.Floor-1]++
		}
	}
	free := make([]float64, len(w.Floors)) // reused per request

	keep := w.HousingQueue[:0]
	for _, q := range w.HousingQueue {
		a, b := &w.People[q.A], &w.People[q.B]
		if !a.Alive || !b.Alive || a.Partner != q.B {
			continue
		}
		if !q.ZoneSet {
			q.Zone, q.ZoneSet = m.targetZone(w, r, q.A, q.B), true
		}
		for i, f := range w.Floors {
			free[i] = 0
			if f.Zone == q.Zone && f.Function == world.Residential {
				free[i] = float64(f.Homes - occupied[i])
			}
		}
		fl := r.WeightedIndex(free) // a random free home: floors weighted by free homes
		if fl < 0 {
			keep = append(keep, q)
			continue
		}
		m.moveIn(w, q.A, q.B, fl+1, occupied)
	}
	w.HousingQueue = keep
	return nil
}

// targetZone: the zone of a partner whose department has a zone; if both
// have one and they differ, one of the two at random; if neither, one of
// the two partners' current home zones at random.
func (m *Module) targetZone(w *world.World, r *rng.Stream, a, b world.PersonID) world.Zone {
	za, tiedA := deptZone(w, a)
	zb, tiedB := deptZone(w, b)
	switch {
	case tiedA && tiedB:
		return pick(r, za, zb)
	case tiedA:
		return za
	case tiedB:
		return zb
	}
	return pick(r, homeZone(w, a), homeZone(w, b))
}

func pick(r *rng.Stream, x, y world.Zone) world.Zone {
	if r.IntRange(0, 1) == 0 {
		return x
	}
	return y
}

func deptZone(w *world.World, id world.PersonID) (world.Zone, bool) {
	d := w.People[id].Department
	if d == world.None || w.Depts[d].Zone == world.AllZones {
		return 0, false
	}
	return w.Depts[d].Zone, true
}

// moveIn starts a household on the given floor for the couple. From each
// household a partner leaves, children under 18 come along if they are a
// partner's own children, or if the partners were the only adults there.
// Either way no child is left without an adult. Emptied households
// dissolve, and their homes are free at once for the rest of the queue
// (ODD §3: changes take effect immediately). occupied counts households
// per floor index and is kept up to date.
func (m *Module) moveIn(w *world.World, a, b world.PersonID, floor int, occupied []int) {
	id := world.HouseholdID(len(w.Households))
	w.Households = append(w.Households, world.Household{ID: id, Floor: floor})
	occupied[floor-1]++
	for _, partner := range []world.PersonID{a, b} {
		from := &w.Households[w.People[partner].Household]
		if from.ID == id {
			continue // both partners shared a household; already moved
		}
		partners := 0
		for _, x := range from.Members {
			if x == a || x == b {
				partners++
			}
		}
		onlyAdults := m.adults(w, from) == partners
		var movers []world.PersonID
		for _, x := range from.Members {
			if x == a || x == b || (!m.isAdult(w, x) && (onlyAdults || m.isChildOf(w, x, a) || m.isChildOf(w, x, b))) {
				movers = append(movers, x)
			}
		}
		from.Members = slices.DeleteFunc(from.Members, func(x world.PersonID) bool { return slices.Contains(movers, x) })
		if len(from.Members) == 0 {
			occupied[from.Floor-1]--
			from.Floor = world.None
		}
		for _, x := range movers {
			w.People[x].Household = id
		}
		w.Households[id].Members = append(w.Households[id].Members, movers...)
	}
}

func (m *Module) isAdult(w *world.World, id world.PersonID) bool {
	return w.Day-w.People[id].BirthDay >= world.Tick(m.cfg.Partnering.MinAge*engine.DaysPerYear)
}

func (m *Module) isChildOf(w *world.World, kid, parent world.PersonID) bool {
	return w.People[kid].Mother == parent || w.People[kid].Father == parent
}

func (m *Module) adults(w *world.World, h *world.Household) int {
	n := 0
	for _, x := range h.Members {
		if m.isAdult(w, x) {
			n++
		}
	}
	return n
}
