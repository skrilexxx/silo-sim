package output

import (
	"testing"

	"github.com/skrilexxx/silo-sim/internal/world"
)

// A department is logged when its workers fall below understaffedShare of
// target and again when it is back above, not on every day in between.
func TestDepartmentEvents(t *testing.T) {
	w := &world.World{Depts: []world.Department{{Name: "mechanical", Target: 10}}}
	o := &Writer{w: w, understaffedShare: 0.5, under: make([]bool, 1)}
	for _, workers := range []int{6, 4, 4, 5, 3, 7} {
		w.Day++
		w.Depts[0].Headcount = workers
		o.checkDepartments(w)
	}
	var got []string
	for _, e := range w.Events {
		d := e.(world.DepartmentEvent)
		got = append(got, d.Type)
	}
	want := []string{"understaffed", "restaffed", "understaffed", "restaffed"} // days 2, 4, 5, 6
	if len(got) != len(want) {
		t.Fatalf("events %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events %v, want %v", got, want)
		}
	}
}
