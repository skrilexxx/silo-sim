package space

import (
	"slices"
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

// shaft is a 144-floor silo of homes (48 per zone) with one person, a
// child living on floor home, and only General services as a department.
func shaft(t *testing.T, home int) (*world.World, *Module, *rng.Stream) {
	t.Helper()
	w := &world.World{Depts: []world.Department{{Name: "general", Zone: world.AllZones}}}
	for f := 1; f <= 144; f++ {
		w.Floors = append(w.Floors, world.Floor{Number: f, Zone: world.Zone((f - 1) / 48)})
	}
	w.People = []world.Person{{Alive: true, Stage: world.Child, Department: world.None}}
	w.Alive = []world.PersonID{0}
	w.Households = []world.Household{{Floor: home, Members: []world.PersonID{0}}}
	cfg := &config.Scenario{}
	cfg.Space.FullDescentHours, cfg.Space.WalkingHoursPerDay = 12, 8
	r, _ := rng.New(1, "space")
	m := New()
	if err := m.Init(w, cfg, r); err != nil {
		t.Fatal(err)
	}
	return w, m, r
}

// endDay runs the location step at the end of the current day, then
// starts the next.
func endDay(t *testing.T, w *world.World, m *Module, r *rng.Stream) {
	t.Helper()
	if err := m.locate(w, r); err != nil {
		t.Fatal(err)
	}
	w.Day++
}

func TestTripDays(t *testing.T) {
	w, _, _ := shaft(t, 1)
	for _, c := range []struct{ a, b, days int }{
		{1, 144, 2}, {144, 1, 2}, {1, 48, 1}, {97, 30, 1}, {120, 1, 2}, {5, 5, 1},
	} {
		if got := w.Space.TripDays(c.a, c.b); got != c.days {
			t.Errorf("%d → %d takes %d days, want %d", c.a, c.b, got, c.days)
		}
	}
}

// T8.2: floor 1 → 144 takes 2 days: on the stairs on the departure day
// and the next, on floor 144 from the day after; away, so off work, until
// a trip home.
func TestTripFromTopToBottom(t *testing.T) {
	w, m, r := shaft(t, 1)
	w.Day = 1
	s := w.Space
	if err := s.StartTrip(0, w.Day, 144); err != nil {
		t.Fatal(err)
	}
	var where []int
	for range 3 {
		where = append(where, s.Location[0])
		endDay(t, w, m, r)
	}
	where = append(where, s.Location[0])
	if want := []int{0, 0, 144, 144}; !slices.Equal(where, want) {
		t.Errorf("days 1–4 on floors %v, want %v (0 = on the stairs)", where, want)
	}
	if s.Away[0] != 144 || !s.OffWork[0] {
		t.Errorf("after arriving: away %d, off work %v; want away on 144", s.Away[0], s.OffWork[0])
	}

	// Home again: setting off ends being away, and the 2-day trip ends home.
	if err := s.StartTrip(0, w.Day, 1); err != nil {
		t.Fatal(err)
	}
	if s.Away[0] != 0 {
		t.Errorf("on the stairs, still away on %d", s.Away[0])
	}
	endDay(t, w, m, r)
	endDay(t, w, m, r)
	if s.Location[0] != 1 || s.Away[0] != 0 || s.OffWork[0] {
		t.Errorf("back home: floor %d, away %d, off work %v", s.Location[0], s.Away[0], s.OffWork[0])
	}
}

// T8.2: with segment 90 (floors 90–91) broken, no trip from the Down Deep
// reaches Up Top, while trips on either side still work.
func TestBrokenSegmentSplitsTheSilo(t *testing.T) {
	w, _, _ := shaft(t, 1)
	s := w.Space
	s.Broken[89] = true
	try := func(from, to int) error {
		s.Location[0], s.Dest[0] = from, 0
		return s.StartTrip(0, 1, to)
	}
	for from := 97; from <= 144; from++ {
		for to := 1; to <= 48; to++ {
			if err := try(from, to); err == nil {
				t.Fatalf("trip %d → %d allowed across the broken segment", from, to)
			}
		}
	}
	for _, c := range [][2]int{{90, 1}, {48, 60}, {91, 144}, {144, 91}} {
		if err := try(c[0], c[1]); err != nil {
			t.Errorf("trip %d → %d refused: %v", c[0], c[1], err)
		}
	}
}

// A segment that breaks during the trip stops the traveller next to the
// break, on the side they left from, on the planned day.
func TestBreakDuringTrip(t *testing.T) {
	for _, c := range []struct{ home, to, segment, stop int }{
		{1, 144, 100, 100}, // going down, segment 100–101 breaks: stop on 100
		{144, 1, 50, 51},   // going up, segment 50–51 breaks: stop on 51
	} {
		w, m, r := shaft(t, c.home)
		w.Day = 1
		if err := w.Space.StartTrip(0, w.Day, c.to); err != nil {
			t.Fatal(err)
		}
		w.Space.Broken[c.segment-1] = true
		endDay(t, w, m, r)
		endDay(t, w, m, r)
		if got := w.Space.Location[0]; got != c.stop || w.Space.Away[0] != c.stop {
			t.Errorf("%d → %d, segment %d broken: on floor %d (away %d), want %d",
				c.home, c.to, c.segment, got, w.Space.Away[0], c.stop)
		}
	}
}

func TestTripRefusals(t *testing.T) {
	w, _, _ := shaft(t, 1)
	s := w.Space
	for _, c := range []struct {
		to   int
		want string
	}{{0, "floor 0 is not in the silo"}, {145, "floor 145 is not in the silo"}} {
		if err := s.StartTrip(0, 1, c.to); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("trip to %d: got %v, want %q", c.to, err, c.want)
		}
	}
	if err := s.StartTrip(0, 1, 50); err != nil {
		t.Fatal(err)
	}
	if err := s.StartTrip(0, 1, 60); err == nil || !strings.Contains(err.Error(), "already on the stairs") {
		t.Errorf("second trip: got %v", err)
	}
}
