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
	if err := countsAgree(w); err != nil {
		return err
	}
	return unitRanges(w)
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
