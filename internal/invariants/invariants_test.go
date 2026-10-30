package invariants

import (
	"math"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// lot holds the lottery rules the reproduction invariant needs.
var lot = &config.Scenario{Lottery: config.Lottery{
	WindowDays: 365, PregnancyDays: 270, MotherAgeMin: 18, MotherAgeMax: 40, MaxChildren: 3, MinYoungestAgeYears: 2,
}}

// tiny builds a valid world: 3 people (one dead) in 2 households, Up Top
// and Down Deep.
func tiny() *world.World {
	w := &world.World{
		Floors: []world.Floor{{Number: 1, Zone: world.UpTop, Homes: 1}, {Number: 2, Zone: world.DownDeep, Homes: 2}},
		Silo:   world.Silo{GeneratorCondition: 0.8, WaterSupply: 1, AirQuality: 1},
	}
	for i := range 3 {
		w.People = append(w.People, world.Person{ID: world.PersonID(i), Alive: i != 1, Health: 1, Skill: 0.5,
			Partner: world.None, Department: world.None, Master: world.None})
	}
	w.People[0].Household, w.People[2].Household = 0, 1
	w.People[1].Household = 1 // the dead keep their last household as a record
	w.Alive = []world.PersonID{2, 0}
	w.Households = []world.Household{
		{ID: 0, Floor: 1, Members: []world.PersonID{0}},
		{ID: 1, Floor: 2, Members: []world.PersonID{2}},
	}
	return w
}

func TestValidWorldPasses(t *testing.T) {
	for _, w := range []*world.World{{}, tiny()} {
		if err := Check(w, lot); err != nil {
			t.Errorf("valid world failed: %v", err)
		}
	}
}

func TestViolationsAreCaught(t *testing.T) {
	cases := []struct {
		want   string
		break_ func(w *world.World)
	}{
		{"Alive holds dead person 1", func(w *world.World) { w.Alive = append(w.Alive, 1) }},
		{"Alive holds person 0 twice", func(w *world.World) { w.Alive = []world.PersonID{0, 0} }},
		{"Alive holds unknown person 9", func(w *world.World) { w.Alive = append(w.Alive, 9) }},
		{"2 people alive, but Alive lists 1", func(w *world.World) { w.Alive = w.Alive[:1] }},
		{"2 people alive, but households hold 1", func(w *world.World) { w.Households[1].Members = nil }},
		{"household 1 is on floor 3, which does not exist", func(w *world.World) { w.Households[1].Floor = 3 }},
		{"floor 2 has no residential zone", func(w *world.World) { w.Floors[1].Zone = world.AllZones }},
		{"person 0 health = 1.5, outside 0–1", func(w *world.World) { w.People[0].Health = 1.5 }},
		{"person 2 skill = -0.1, outside 0–1", func(w *world.World) { w.People[2].Skill = -0.1 }},
		{"person 0 health = NaN", func(w *world.World) { w.People[0].Health = math.NaN() }},
		{"food stock 5, but opening 0 + produced 0 − handed out 0 = 0", func(w *world.World) { w.Silo.FoodStock = 5 }},
		{"silo parts stock = -1, below 0", func(w *world.World) { w.Silo.PartsStock = -1 }},
		{"silo generator down days = -2, below 0", func(w *world.World) { w.Silo.GeneratorDown = -2 }},
		{"silo power factor = -0.5, outside 0–1", func(w *world.World) { w.Silo.PowerSupply, w.Silo.PowerDemand = -1, 2 }},
		{"zone 1 intake/need = NaN, below 0", func(w *world.World) { w.Silo.Intake[1] = math.NaN() }},
		{"silo air quality = 1.2, outside 0–1", func(w *world.World) { w.Silo.AirQuality = 1.2 }},
		{"silo generator condition = -1", func(w *world.World) { w.Silo.GeneratorCondition = -1 }},
		{"household 0 holds person 2, whose household is 1", func(w *world.World) {
			w.Households[0].Members = []world.PersonID{0, 2}
			w.Households[1].Members = nil
		}},
		{"floor 1 holds 1 households but has 0 homes", func(w *world.World) { w.Floors[0].Homes = 0 }},
		{"person 0's partner 1 is not alive", func(w *world.World) { w.People[0].Partner = 1 }},
		{"person 0 is their own partner", func(w *world.World) { w.People[0].Partner = 0 }},
		{"person 2 is male but has implant state 1", func(w *world.World) {
			w.People[2].Sex, w.People[2].Implant = world.Male, world.WindowOpen
		}},
		{"person 0 has a permit without a male partner", func(w *world.World) {
			w.People[0].Implant, w.People[0].WindowEnds = world.WindowOpen, 365
			w.People[0].BirthDay = -25 * 365
		}},
		{"person 0 won a permit on day 0 while not eligible (aged 0, 0 children)", func(w *world.World) {
			w.People[0].Implant, w.People[0].WindowEnds = world.WindowOpen, 365
			w.People[2].Sex = world.Male
			w.People[0].Partner, w.People[2].Partner = 2, 0
		}},
		{"person 0: permit day 5 is after today 0", func(w *world.World) {
			w.People[0].Implant, w.People[0].PermitDay = world.Queued, 5
		}},
		{"person 0: window 0–365 (permit day 0) is open on day 365", func(w *world.World) {
			w.Day = 365
			w.People[0].Implant, w.People[0].WindowEnds = world.WindowOpen, 365
		}},
		{"person 0: pregnancy conceived day 0, due 270, outside window 0–365", func(w *world.World) {
			w.People[0].Implant, w.People[0].WindowEnds, w.People[0].DueDay = world.Pregnant, 365, 270
		}},
		{"person 0's partner 2 has partner -1", func(w *world.World) { w.People[0].Partner = 2 }},
	}
	for _, c := range cases {
		w := tiny()
		c.break_(w)
		err := Check(w, lot)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want an error containing %q", err, c.want)
		}
	}
}

// Breakages the counts check would catch first, so oneHousehold is called
// directly to prove it catches them on its own.
func TestOneHousehold(t *testing.T) {
	cases := []struct {
		want   string
		break_ func(w *world.World)
	}{
		{"household 1 holds person 1, who is not alive", func(w *world.World) {
			w.Households[1].Members = []world.PersonID{2, 1}
		}},
		{"person 2 is in 0 households", func(w *world.World) { w.Households[1].Members = nil }},
	}
	for _, c := range cases {
		w := tiny()
		c.break_(w)
		if err := oneHousehold(w); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want an error containing %q", err, c.want)
		}
	}
}

// A dissolved household (no members) is skipped, whatever its floor.
func TestDissolvedHouseholdSkipped(t *testing.T) {
	w := tiny()
	w.Households = append(w.Households, world.Household{ID: 2, Floor: world.None})
	if err := Check(w, lot); err != nil {
		t.Errorf("dissolved household checked: %v", err)
	}
}

// A dead person's stale values are not checked; they no longer take part.
func TestDeadPeopleNotRangeChecked(t *testing.T) {
	w := tiny()
	w.People[1].Health = 5
	if err := Check(w, lot); err != nil {
		t.Errorf("dead person's health was checked: %v", err)
	}
}

// spaced adds space state to tiny(): floor 1 Up Top, 2 Down Deep; person 0
// a retired Up Topper at home, person 2 a Mechanical worker whose
// department works on floor 2, where she lives.
func spaced() *world.World {
	w := tiny()
	w.Depts = []world.Department{{Name: "mechanical", Zone: world.DownDeep}, {Name: "porters", Zone: world.AllZones}}
	w.People[0].Stage = world.Retired
	w.People[2].Stage, w.People[2].Department = world.Worker, 0
	w.Space = &world.Space{
		Broken:     []bool{false},
		DeptFloors: [][]int{{2}, nil},
		Location:   []int{1, 0, 2},
		Stranded:   []bool{false, false, false},
	}
	return w
}

func TestSpace(t *testing.T) {
	if err := space(spaced()); err != nil {
		t.Fatalf("valid world failed: %v", err)
	}
	cases := []struct {
		want   string
		break_ func(w *world.World)
	}{
		{"person 0 is on floor 3, not a real floor", func(w *world.World) { w.Space.Location[0] = 3 }},
		{"person 0 is on floor 0, not a real floor", func(w *world.World) { w.Space.Location[0] = 0 }},
		{"person 2 has no location", func(w *world.World) { w.Space.Location = w.Space.Location[:2] }},
		{"person 0 is on floor 2 (stranded false), want home floor 1", func(w *world.World) { w.Space.Location[0] = 2 }},
		{"mechanical worker 2 is on floor 1 (stranded false)", func(w *world.World) {
			w.Households[1].Floor, w.Households[0].Floor = 1, 2 // she lives and stands on 1, not on her department's floor 2
			w.Space.Location[0], w.Space.Location[2] = 2, 1
		}},
		{"mechanical worker 2 is on floor 2 (stranded true)", func(w *world.World) { w.Space.Stranded[2] = true }},
		{"mechanical worker 2 is on floor 1 (stranded true)", func(w *world.World) { // stranded, but floor 2 is reachable
			w.Households[1].Floor, w.Households[0].Floor = 1, 2
			w.Space.Location[0], w.Space.Location[2], w.Space.Stranded[2] = 2, 1, true
		}},
		{"porter 2 is on floor 1, not connected to home floor 2", func(w *world.World) {
			w.People[2].Department, w.Space.Broken[0], w.Space.Location[2] = 1, true, 1
		}},
	}
	for _, c := range cases {
		w := spaced()
		c.break_(w)
		if err := space(w); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want %q", err, c.want)
		}
	}
	// Stranded at home when the stairs cut her off is valid.
	w := spaced()
	w.Households[1].Floor, w.Households[0].Floor = 1, 2
	w.Space.Location[0], w.Space.Location[2], w.Space.Stranded[2] = 2, 1, true
	w.Space.Broken[0] = true
	if err := space(w); err != nil {
		t.Errorf("stranded worker rejected: %v", err)
	}
}
