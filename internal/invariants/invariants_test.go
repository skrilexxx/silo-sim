package invariants

import (
	"math"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/world"
)

// tiny builds a valid world: 3 people (one dead) in 2 households, Up Top
// and Down Deep.
func tiny() *world.World {
	w := &world.World{
		Floors: []world.Floor{{Number: 1, Zone: world.UpTop}, {Number: 2, Zone: world.DownDeep}},
		Silo:   world.Silo{GeneratorCondition: 0.8, WaterSupply: 1, AirQuality: 1},
	}
	for i := range 3 {
		w.People = append(w.People, world.Person{ID: world.PersonID(i), Alive: i != 1, Health: 1, Skill: 0.5})
	}
	w.Alive = []world.PersonID{2, 0}
	w.Households = []world.Household{
		{ID: 0, Floor: 1, Members: []world.PersonID{0}},
		{ID: 1, Floor: 2, Members: []world.PersonID{2}},
	}
	return w
}

func TestValidWorldPasses(t *testing.T) {
	for _, w := range []*world.World{{}, tiny()} {
		if err := Check(w); err != nil {
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
		{"silo air quality = 1.2, outside 0–1", func(w *world.World) { w.Silo.AirQuality = 1.2 }},
		{"silo generator condition = -1", func(w *world.World) { w.Silo.GeneratorCondition = -1 }},
	}
	for _, c := range cases {
		w := tiny()
		c.break_(w)
		err := Check(w)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want an error containing %q", err, c.want)
		}
	}
}

// A dead person's stale values are not checked; they no longer take part.
func TestDeadPeopleNotRangeChecked(t *testing.T) {
	w := tiny()
	w.People[1].Health = 5
	if err := Check(w); err != nil {
		t.Errorf("dead person's health was checked: %v", err)
	}
}
