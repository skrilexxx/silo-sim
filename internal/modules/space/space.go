// Package space places people on floors (ODD §6.10, draft): the stairs,
// work floors, and where everyone is each day.
package space

import (
	"fmt"
	"slices"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

type Module struct {
	general, porters int // department indexes, or None

	// lo[f-1] and hi[f-1] are the lowest and highest floor connected to
	// floor f; near[d][f-1] is department d's work floor for someone living
	// on floor f, or 0 if stranded. All follow from the stairs alone, so
	// they are rebuilt only when a segment changes (seen is the last state).
	lo, hi []int
	near   [][]int
	seen   []bool
}

func New() *Module { return &Module{general: world.None, porters: world.None} }

func (m *Module) Name() string { return "space" }

// Requires work: work floors follow from departments.
func (m *Module) Requires() []string { return []string{"work"} }

// ODD §6.10: all stairs intact on day 1; each department's work floors;
// day-1 locations.
func (m *Module) Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error {
	if len(w.Depts) == 0 {
		return fmt.Errorf("space: no departments; list work before space in modules")
	}
	s := &world.Space{Broken: make([]bool, len(w.Floors)-1), DeptFloors: make([][]int, len(w.Depts))}
	for i, d := range w.Depts {
		switch d.Name {
		case "general":
			m.general = i
		case "porters":
			m.porters = i
		case "farms":
			s.DeptFloors[i] = floorsWith(w, world.Farm)
		case "mechanical":
			s.DeptFloors[i] = floorsWith(w, world.Mechanical)
		default:
			floors := slices.Sorted(slices.Values(cfg.Space.OfficeFloors[d.Name]))
			if len(floors) == 0 {
				return fmt.Errorf("space: no office floors for %s", d.Name)
			}
			for _, f := range floors {
				if f < 1 || f > len(w.Floors) || w.Floors[f-1].Function != world.Residential || w.Floors[f-1].Zone != d.Zone {
					return fmt.Errorf("space.officeFloors.%s: floor %d is not a residential floor in the %s zone", d.Name, f, d.Zone)
				}
			}
			s.DeptFloors[i] = floors
		}
	}
	w.Space = s
	n := len(w.Floors)
	m.lo, m.hi, m.seen = make([]int, n), make([]int, n), nil
	m.near = make([][]int, len(w.Depts))
	for d := range m.near {
		m.near[d] = make([]int, n)
	}
	return m.locate(w, r)
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseLocation, Run: m.locate}}
}

func floorsWith(w *world.World, fn world.FloorFunction) []int {
	var out []int
	for _, f := range w.Floors {
		if f.Function == fn {
			out = append(out, f.Number)
		}
	}
	return out
}

// ODD §6.10 / §3 step 16.5: at the end of each day, where everyone is the
// next day. Workers and shadows go to their work floor; everyone else
// stays home. Porters draw a floor connected to home, in the tick's
// shuffled order.
func (m *Module) locate(w *world.World, r *rng.Stream) error {
	s := w.Space
	for len(s.Location) < len(w.People) {
		s.Location = append(s.Location, 0)
		s.Stranded = append(s.Stranded, false)
	}
	if m.seen == nil || !slices.Equal(m.seen, s.Broken) {
		m.connect(s)
		m.seen = slices.Clone(s.Broken)
	}
	for _, id := range w.Alive {
		p := &w.People[id]
		home := w.Households[p.Household].Floor
		floor, stranded := home, false
		if p.Stage == world.Worker || p.Stage == world.Shadow {
			switch p.Department {
			case m.general: // cafeterias, crafts and cleaning are on every floor
			case m.porters:
				floor = r.IntRange(m.lo[home-1], m.hi[home-1])
			default:
				if f := m.near[p.Department][home-1]; f != 0 {
					floor = f
				} else {
					stranded = true
				}
			}
		}
		s.Location[id], s.Stranded[id] = floor, stranded
	}
	return nil
}

// connect sets lo and hi (a broken segment ends one run of connected
// floors and starts the next), then near.
func (m *Module) connect(s *world.Space) {
	start, n := 1, len(m.lo)
	for f := 1; f <= n; f++ {
		if f == n || s.Broken[f-1] {
			for g := start; g <= f; g++ {
				m.lo[g-1], m.hi[g-1] = start, f
			}
			start = f + 1
		}
	}
	for d, floors := range s.DeptFloors {
		for home := 1; home <= n; home++ {
			m.near[d][home-1] = m.nearest(floors, home)
		}
	}
}

// nearest is the department floor connected to home that is closest to
// it, the lower one on a tie (floors are sorted); 0 if none is connected.
func (m *Module) nearest(floors []int, home int) int {
	best := 0
	for _, f := range floors {
		if f >= m.lo[home-1] && f <= m.hi[home-1] && (best == 0 || dist(f, home) < dist(best, home)) {
			best = f
		}
	}
	return best
}

func dist(a, b int) int { return max(a-b, b-a) }
