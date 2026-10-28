// Package invariants holds the ODD §8 checks that must hold on every tick.
// A broken simulation still produces plausible numbers, so these are the
// main guard against silent bugs. More checks join as the state they
// cover gets built.
package invariants

import (
	"fmt"

	"github.com/skrilexxx/silo-sim/internal/world"
)

// Check runs every invariant and returns the first violation.
func Check(w *world.World) error {
	for _, check := range []func(*world.World) error{countsAgree, oneHousehold, partnerships, unitRanges} {
		if err := check(w); err != nil {
			return err
		}
	}
	return nil
}

// ODD §8: people alive = sum over zones = sum over households.
func countsAgree(w *world.World) error {
	alive := 0
	for i := range w.People {
		if w.People[i].Alive {
			alive++
		}
	}

	// The Alive list must hold each living person exactly once.
	seen := make([]bool, len(w.People))
	for _, id := range w.Alive {
		if int(id) < 0 || int(id) >= len(w.People) {
			return fmt.Errorf("Alive holds unknown person %d", id)
		}
		if !w.People[id].Alive {
			return fmt.Errorf("Alive holds dead person %d", id)
		}
		if seen[id] {
			return fmt.Errorf("Alive holds person %d twice", id)
		}
		seen[id] = true
	}
	if len(w.Alive) != alive {
		return fmt.Errorf("%d people alive, but Alive lists %d", alive, len(w.Alive))
	}

	var inHouseholds int
	var inZone [3]int // UpTop, Mids, DownDeep
	for _, h := range w.Households {
		if len(h.Members) == 0 {
			continue // dissolved
		}
		if h.Floor < 1 || h.Floor > len(w.Floors) {
			return fmt.Errorf("household %d is on floor %d, which does not exist", h.ID, h.Floor)
		}
		z := w.Floors[h.Floor-1].Zone
		if z < world.UpTop || z > world.DownDeep {
			return fmt.Errorf("floor %d has no residential zone (%d)", h.Floor, z)
		}
		inHouseholds += len(h.Members)
		inZone[z] += len(h.Members)
	}
	if inHouseholds != alive {
		return fmt.Errorf("%d people alive, but households hold %d", alive, inHouseholds)
	}
	if sum := inZone[world.UpTop] + inZone[world.Mids] + inZone[world.DownDeep]; sum != alive {
		return fmt.Errorf("%d people alive, but zones hold %d", alive, sum)
	}
	return nil
}

// ODD §8: every living person is in exactly one household, the one their
// Household field names, and households hold only the living.
func oneHousehold(w *world.World) error {
	count := make([]int, len(w.People))
	for _, h := range w.Households {
		for _, id := range h.Members {
			if int(id) < 0 || int(id) >= len(w.People) || !w.People[id].Alive {
				return fmt.Errorf("household %d holds person %d, who is not alive", h.ID, id)
			}
			if w.People[id].Household != h.ID {
				return fmt.Errorf("household %d holds person %d, whose household is %d", h.ID, id, w.People[id].Household)
			}
			count[id]++
		}
	}
	for _, id := range w.Alive {
		if count[id] != 1 {
			return fmt.Errorf("person %d is in %d households", id, count[id])
		}
	}
	return nil
}

// ODD §8: partnerships are symmetric and both partners are alive.
func partnerships(w *world.World) error {
	for _, id := range w.Alive {
		q := w.People[id].Partner
		if q == world.None {
			continue
		}
		if q == id {
			return fmt.Errorf("person %d is their own partner", id)
		}
		if int(q) < 0 || int(q) >= len(w.People) || !w.People[q].Alive {
			return fmt.Errorf("person %d's partner %d is not alive", id, q)
		}
		if w.People[q].Partner != id {
			return fmt.Errorf("person %d's partner %d has partner %d", id, q, w.People[q].Partner)
		}
	}
	return nil
}

// ODD §8: every 0–1 variable stays within 0–1. Written as !(0 ≤ x ≤ 1) so
// NaN fails too.
func unitRanges(w *world.World) error {
	for _, id := range w.Alive {
		p := &w.People[id]
		if !in01(p.Health) {
			return fmt.Errorf("person %d health = %v, outside 0–1", id, p.Health)
		}
		if !in01(p.Skill) {
			return fmt.Errorf("person %d skill = %v, outside 0–1", id, p.Skill)
		}
	}
	s := &w.Silo
	for _, v := range []struct {
		name string
		x    float64
	}{
		{"generator condition", s.GeneratorCondition},
		{"water supply", s.WaterSupply},
		{"air quality", s.AirQuality},
	} {
		if !in01(v.x) {
			return fmt.Errorf("silo %s = %v, outside 0–1", v.name, v.x)
		}
	}
	return nil
}

func in01(x float64) bool { return x >= 0 && x <= 1 }
