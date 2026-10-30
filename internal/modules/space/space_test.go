package space

import (
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// tinySilo has 7 floors: 1–3 Up Top (homes), 4 a farm floor and 5 homes
// in the Mids, 6 homes and 7 Mechanical in the Down Deep. IT works on
// floors 1 and 3. Each person lives alone on the given home floor.
func tinySilo(people []world.Person, homes []int) (*world.World, *config.Scenario) {
	w := &world.World{
		Floors: []world.Floor{
			{Number: 1, Zone: world.UpTop}, {Number: 2, Zone: world.UpTop}, {Number: 3, Zone: world.UpTop},
			{Number: 4, Zone: world.Mids, Function: world.Farm}, {Number: 5, Zone: world.Mids},
			{Number: 6, Zone: world.DownDeep}, {Number: 7, Zone: world.DownDeep, Function: world.Mechanical},
		},
		Depts: []world.Department{
			{Name: "farms", Zone: world.Mids}, {Name: "general", Zone: world.AllZones},
			{Name: "mechanical", Zone: world.DownDeep}, {Name: "it", Zone: world.UpTop},
			{Name: "porters", Zone: world.AllZones},
		},
	}
	for i, p := range people {
		p.ID, p.Alive, p.Household = world.PersonID(i), true, world.HouseholdID(i)
		w.People = append(w.People, p)
		w.Alive = append(w.Alive, p.ID)
		w.Households = append(w.Households, world.Household{ID: p.Household, Floor: homes[i], Members: []world.PersonID{p.ID}})
	}
	cfg := &config.Scenario{}
	cfg.Space.OfficeFloors = map[string][]int{"it": {3, 1}} // unsorted on purpose
	return w, cfg
}

const (
	farms = iota
	general
	mechanical
	it
	porters
)

func TestDayLocations(t *testing.T) {
	people := []world.Person{
		{Stage: world.Child, Department: world.None},
		{Stage: world.Worker, Department: it},
		{Stage: world.Worker, Department: it},
		{Stage: world.Worker, Department: mechanical},
		{Stage: world.Shadow, Department: farms},
		{Stage: world.Worker, Department: general},
		{Stage: world.Retired, Department: mechanical},
		{Stage: world.SickLeave, Department: it},
		{Stage: world.Worker, Department: porters},
	}
	homes := []int{5, 5, 2, 2, 6, 5, 1, 6, 6}
	w, cfg := tinySilo(people, homes)
	r, _ := rng.New(1, "space")
	m := New()
	if err := m.Init(w, cfg, r); err != nil {
		t.Fatal(err)
	}
	if got := w.Space.DeptFloors; len(got[farms]) != 1 || got[farms][0] != 4 || got[mechanical][0] != 7 ||
		got[it][0] != 1 || got[it][1] != 3 || got[general] != nil || got[porters] != nil {
		t.Fatalf("department floors %v", got)
	}

	check := func(when string, want []int, stranded []bool) {
		t.Helper()
		for id := range want {
			if w.Space.Location[id] != want[id] || w.Space.Stranded[id] != stranded[id] {
				t.Errorf("%s: person %d on floor %d (stranded %v), want %d (stranded %v)",
					when, id, w.Space.Location[id], w.Space.Stranded[id], want[id], stranded[id])
			}
		}
	}
	// Child, retired and sick at home; IT from 5 goes to 3, from 2 to 1
	// (a tie: the lower floor); Mechanical to 7; the farm shadow to 4;
	// General services at home.
	none := make([]bool, len(people))
	check("intact", []int{5, 3, 1, 7, 4, 5, 1, 6}, none)

	// Break segment 4 (floors 4–5): floors 5–7 are cut off from 1–4.
	w.Space.Broken[3] = true
	if err := m.locate(w, r); err != nil {
		t.Fatal(err)
	}
	check("broken", []int{5, 5, 1, 2, 6, 5, 1, 6},
		[]bool{false, true, false, true, true, false, false, false})

	// Porters draw a floor connected to home: only 5–7 now, each of them.
	seen := map[int]int{}
	for range 300 {
		if err := m.locate(w, r); err != nil {
			t.Fatal(err)
		}
		seen[w.Space.Location[8]]++
	}
	if len(seen) != 3 || seen[5] == 0 || seen[6] == 0 || seen[7] == 0 {
		t.Errorf("porter visited %v, want floors 5, 6 and 7", seen)
	}
}

// Someone born after the last location step gets a location at the next.
func TestNewbornsGetALocation(t *testing.T) {
	w, cfg := tinySilo([]world.Person{{Stage: world.Child, Department: world.None}}, []int{2})
	r, _ := rng.New(1, "space")
	m := New()
	if err := m.Init(w, cfg, r); err != nil {
		t.Fatal(err)
	}
	w.People = append(w.People, world.Person{ID: 1, Alive: true, Household: 0, Department: world.None})
	w.Alive = append(w.Alive, 1)
	if err := m.locate(w, r); err != nil {
		t.Fatal(err)
	}
	if len(w.Space.Location) != 2 || w.Space.Location[1] != 2 {
		t.Errorf("locations %v, want the newborn at home on floor 2", w.Space.Location)
	}
}

func TestOfficeFloorsCheckedAtStart(t *testing.T) {
	for _, c := range []struct {
		floors []int
		want   string
	}{
		{[]int{5}, "floor 5 is not a residential floor in the upTop zone"},
		{[]int{4}, "floor 4 is not a residential floor in the upTop zone"}, // a farm floor
		{[]int{9}, "floor 9 is not"},
		{nil, "no office floors for it"},
	} {
		w, cfg := tinySilo(nil, nil)
		cfg.Space.OfficeFloors["it"] = c.floors
		r, _ := rng.New(1, "space")
		err := New().Init(w, cfg, r)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v: got %v, want %q", c.floors, err, c.want)
		}
	}
}
