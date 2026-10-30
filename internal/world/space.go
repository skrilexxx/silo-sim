package world

import (
	"fmt"
	"math"
)

// Space is the state of the space module (ODD §6.10, draft); nil when the
// module is off. Per-person slices are indexed by PersonID and grow with
// People; dead people keep their last entries.
type Space struct {
	Broken     []bool  // Broken[s-1] is stair segment s, between floors s and s+1
	DeptFloors [][]int // per department, in Depts order: its work floors, lowest first; nil for General services and Porters

	// Trip lengths (ODD §6.10): hours of walking for the whole stairwell
	// and per day.
	DescentHours, WalkHours float64

	Location  []int  // floor where the person is today; 0 while on the stairs
	WorkFloor []int  // usual work floor of a worker, shadow or worker on sick leave; 0 for none (porters, the stranded, everyone else)
	Stranded  []bool // a worker or shadow cut off from every floor of their department
	OffWork   []bool // stranded, on the stairs or away: does not work today (only matters for workers and shadows)

	// A trip in progress has Dest > 0: it left From and reaches Dest on
	// day Arrive. Away is the floor a person stays on after a trip, or 0.
	From, Dest []int
	Arrive     []Tick
	Away       []int
}

// Connected reports whether every stair segment between floors a and b is
// intact.
func (s *Space) Connected(a, b int) bool {
	for seg := min(a, b); seg < max(a, b); seg++ {
		if s.Broken[seg-1] {
			return false
		}
	}
	return true
}

// Off reports whether a person does not work today because of where they
// are. False when the space module is off.
func (s *Space) Off(id PersonID) bool {
	return s != nil && int(id) < len(s.OffWork) && s.OffWork[id]
}

// TripDays is how long a walk between floors a and b takes: hours =
// floors crossed × DescentHours / segments, days = max(1, ⌈hours /
// WalkHours⌉). Multiplying before dividing keeps a full descent exact.
func (s *Space) TripDays(a, b int) int {
	hours := float64(max(a-b, b-a)) * s.DescentHours / float64(len(s.Broken))
	return max(1, int(math.Ceil(hours/s.WalkHours)))
}

// StartTrip sends a person from where they are today to floor to (ODD
// §6.10). They are on the stairs, and off work, from now until they
// arrive on day today + TripDays. It refuses someone already on the
// stairs, a floor outside the silo, and a route through a broken segment;
// the module that asked decides what a refusal means.
func (s *Space) StartTrip(id PersonID, today Tick, to int) error {
	if int(id) >= len(s.Location) {
		return fmt.Errorf("person %d has no location yet", id)
	}
	from := s.Location[id]
	switch {
	case from == 0:
		return fmt.Errorf("person %d is already on the stairs", id)
	case to < 1 || to > len(s.Broken)+1:
		return fmt.Errorf("floor %d is not in the silo", to)
	case !s.Connected(from, to):
		return fmt.Errorf("no way from floor %d to %d: the stairs are broken", from, to)
	}
	s.From[id], s.Dest[id], s.Arrive[id] = from, to, today+Tick(s.TripDays(from, to))
	s.Location[id], s.OffWork[id] = 0, true
	return nil
}
