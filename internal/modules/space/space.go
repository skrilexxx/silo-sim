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
	s := &world.Space{Broken: make([]bool, len(w.Floors)-1), DeptFloors: make([][]int, len(w.Depts)),
		DescentHours: cfg.Space.FullDescentHours, WalkHours: cfg.Space.WalkingHoursPerDay}
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
// next day. Travellers stay on the stairs until the day they arrive; a
// person away stays on the floor they reached. Otherwise workers and
// shadows go to their work floor and everyone else stays home. Porters
// draw a floor connected to home, in the tick's shuffled order.
func (m *Module) locate(w *world.World, r *rng.Stream) error {
	s := w.Space
	for len(s.Location) < len(w.People) {
		s.Location, s.Stranded, s.OffWork = append(s.Location, 0), append(s.Stranded, false), append(s.OffWork, false)
		s.WorkFloor = append(s.WorkFloor, 0)
		s.From, s.Dest, s.Arrive, s.Away = append(s.From, 0), append(s.Dest, 0), append(s.Arrive, 0), append(s.Away, 0)
	}
	s.Travelling = 0
	if m.seen == nil || !slices.Equal(m.seen, s.Broken) {
		m.connect(s)
		m.seen = slices.Clone(s.Broken)
	}
	for _, id := range w.Alive {
		p := &w.People[id]
		home := w.Households[p.Household].Floor
		s.WorkFloor[id] = m.workFloor(p, home)
		if s.Dest[id] != 0 && !m.travel(w, id, home) {
			continue // still on the stairs tomorrow
		}
		if s.Away[id] != 0 {
			s.Location[id], s.Stranded[id], s.OffWork[id] = s.Away[id], false, true
			continue
		}
		floor, stranded := home, false
		if p.Stage == world.Worker || p.Stage == world.Shadow {
			switch p.Department {
			case m.general: // cafeterias, crafts and cleaning are on every floor
			case m.porters:
				floor = r.IntRange(m.lo[home-1], m.hi[home-1])
			default:
				if f := s.WorkFloor[id]; f != 0 {
					floor = f
				} else {
					stranded = true
				}
			}
		}
		s.Location[id], s.Stranded[id], s.OffWork[id] = floor, stranded, stranded
	}
	return nil
}

// workFloor is where a worker, shadow or worker on sick leave usually
// works (ODD §6.10): home for General services, the nearest connected
// department floor otherwise; 0 for porters (no fixed floor), for the
// stranded and for everyone else.
func (m *Module) workFloor(p *world.Person, home int) int {
	switch {
	case p.Stage != world.Worker && p.Stage != world.Shadow && p.Stage != world.SickLeave:
		return 0
	case p.Department == m.general:
		return home
	case p.Department == m.porters:
		return 0
	}
	return m.near[p.Department][home-1]
}

// ODD §6.10: trips. travel moves a trip on by a day and reports whether
// it ends tomorrow.
// A segment that broke on the route stops the traveller on the floor next
// to the break, on the side they left from; they still arrive on the
// planned day. Arriving at home ends being away, except for someone held
// by Judicial (ODD §6.13); anywhere else starts it.
func (m *Module) travel(w *world.World, id world.PersonID, home int) bool {
	s := w.Space
	from, dest := s.From[id], s.Dest[id]
	if dest > from {
		dest = min(dest, m.hi[from-1])
	} else {
		dest = max(dest, m.lo[from-1])
	}
	s.Dest[id] = dest
	if s.Arrive[id] > w.Day+1 {
		s.Location[id], s.Stranded[id], s.OffWork[id] = 0, false, true
		s.Travelling++
		return false
	}
	s.Dest[id], s.Away[id] = 0, dest
	if dest == home && !w.Control.Held(id) {
		s.Away[id] = 0
	}
	return true
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
