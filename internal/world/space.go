package world

// Space is the state of the space module (ODD §6.10, draft); nil when the
// module is off. Per-person slices are indexed by PersonID and grow with
// People; dead people keep their last entries.
type Space struct {
	Broken     []bool  // Broken[s-1] is stair segment s, between floors s and s+1
	DeptFloors [][]int // per department, in Depts order: its work floors, lowest first; nil for General services and Porters
	Location   []int   // floor where the person is for the day
	Stranded   []bool  // a worker or shadow cut off from every floor of their department
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
