package governance

import (
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/modules/work"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const year = engine.DaysPerYear

func load(t *testing.T, path string) *config.Scenario {
	t.Helper()
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Output.InvariantEvery = 1
	return cfg
}

// Appointment picks the most skilled worker of IT, Judicial and
// Administration. More skilled people elsewhere, retired, or still
// shadowing don't count; a tie goes to the lower ID.
func TestAppointment(t *testing.T) {
	cfg := load(t, "../../../scenarios/baseline.yaml")
	w := &world.World{Floors: []world.Floor{{Number: 1, Zone: world.UpTop}}, Households: []world.Household{{Floor: 1}}}
	for _, d := range cfg.Work.Departments {
		w.Depts = append(w.Depts, world.Department{Name: d.Name})
	}
	idx := func(name string) int {
		for d, dep := range w.Depts {
			if dep.Name == name {
				return d
			}
		}
		t.Fatalf("no department %s", name)
		return 0
	}
	for _, p := range []struct {
		stage world.Stage
		dept  string
		skill float64
	}{
		{world.Worker, "farms", 0.99},
		{world.Worker, "it", 0.70},
		{world.Worker, "judicial", 0.90}, // id 2: the pick
		{world.Retired, "administration", 0.95},
		{world.Shadow, "judicial", 0.97},
		{world.Worker, "administration", 0.90}, // id 5: same skill, higher ID
	} {
		id := world.PersonID(len(w.People))
		w.People = append(w.People, world.Person{ID: id, Alive: true, Stage: p.stage, Department: idx(p.dept), Skill: p.skill})
		w.Alive = append(w.Alive, id)
	}
	// Put the higher-ID tie first, so the pick can't come from the order.
	w.Alive[2], w.Alive[5] = w.Alive[5], w.Alive[2]
	m := New()
	if err := m.Init(w, cfg, nil); err != nil {
		t.Fatal(err)
	}
	if w.Silo.Mayor != 2 {
		t.Errorf("appointed %d, want 2", w.Silo.Mayor)
	}
	if len(w.Events) != 1 || w.Events[0].(world.MayorEvent).Reason != "start" || w.Events[0].When() != 0 {
		t.Errorf("events = %+v, want one start event on day 0", w.Events)
	}
}

// run plays the 1,000-person golden scenario for 100 years with
// demography, work and governance, and returns the Mayor events.
func run(t *testing.T, mode, term string) (*world.World, []world.MayorEvent) {
	t.Helper()
	cfg := load(t, "../../../testdata/golden/scenario.yaml")
	cfg.Governance.Mode, cfg.Governance.Term, cfg.Years = mode, term, 100
	r, _ := rng.New(cfg.Seed, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	e, err := engine.New(cfg, w, []engine.Module{demography.New(), work.New(), New()})
	if err != nil {
		t.Fatal(err)
	}
	var events []world.MayorEvent
	for w.Day < world.Tick(cfg.Years*year) {
		if err := e.Tick(); err != nil {
			t.Fatal(err)
		}
		for _, ev := range w.Events {
			events = append(events, ev.(world.MayorEvent))
		}
		w.Events = w.Events[:0]
	}
	return w, events
}

// With a life term the Mayor changes only when the Mayor has died.
func TestLifeTerm(t *testing.T) {
	w, events := run(t, "appointment", "life")
	changes := 0
	for _, e := range events[1:] { // [0] is the day-1 start
		if e.Reason != "died" || w.People[e.Previous].Alive {
			t.Errorf("day %d: Mayor %d → %d for %q; under a life term only death ends a term", e.Day, e.Previous, e.Mayor, e.Reason)
		}
		changes++
	}
	t.Logf("%d Mayors after the first in 100 years", changes)
	if changes == 0 {
		t.Error("no Mayor died in 100 years, so the test proves nothing")
	}
}

// With 5-year terms a new choice comes when the term runs out or the
// Mayor dies, and the sitting Mayor may be chosen again.
func TestFiveYearTerm(t *testing.T) {
	w, events := run(t, "appointment", "5y")
	since := world.Tick(0)
	again := 0
	minAge, maxAge := 1000.0, 0.0
	for _, e := range events {
		if e.Mayor != world.None {
			a := float64(e.Day-w.People[e.Mayor].BirthDay) / year
			minAge, maxAge = min(minAge, a), max(maxAge, a)
		}
	}
	for _, e := range events[1:] { // [0] is the day-1 start
		switch e.Reason {
		case "term":
			if e.Day-since != 5*year {
				t.Errorf("day %d: term ended after %d days, want %d", e.Day, e.Day-since, 5*year)
			}
			if e.Mayor == e.Previous {
				again++
			}
		case "died":
			if w.People[e.Previous].Alive {
				t.Errorf("day %d: Mayor %d replaced for death but alive", e.Day, e.Previous)
			}
		default:
			t.Errorf("day %d: unexpected reason %q", e.Day, e.Reason)
		}
		since = e.Day
	}
	t.Logf("%d choices after the start in 100 years, %d of them re-picked the sitting Mayor; ages when chosen %.1f–%.1f",
		len(events)-1, again, minAge, maxAge)
}

// ODD §6.8 done-when (T4.1): with elections and 80% own-zone voting, the
// Mids candidate (the largest zone) wins most elections.
func TestElectionsMidsWins(t *testing.T) {
	_, events := run(t, "election", "5y")
	wins := map[string]int{}
	for _, e := range events {
		wins[e.Zone]++
	}
	t.Logf("%d elections: Up Top %d, Mids %d, Down Deep %d; last votes %v",
		len(events), wins["upTop"], wins["mids"], wins["downDeep"], events[len(events)-1].Votes)
	if wins["mids"]*2 <= len(events) {
		t.Errorf("Mids won %d of %d elections, want most", wins["mids"], len(events))
	}
}

// Votes: own-zone candidate with ownZoneVote, else uniform over all
// candidates. With 3 candidates a voter backs their own with 0.8 + 0.2/3.
func TestVoteShares(t *testing.T) {
	cfg := load(t, "../../../scenarios/baseline.yaml")
	cfg.Governance.Mode = "election"
	r, _ := rng.New(1, "initpop")
	w, err := initpop.Build(cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	wk := work.New()
	if err := wk.Init(w, cfg, r); err != nil {
		t.Fatal(err)
	}
	m := New()
	if err := m.Init(w, cfg, r); err != nil {
		t.Fatal(err)
	}
	var adults [world.NumZones]float64
	for _, id := range w.Alive {
		if w.Day-w.People[id].BirthDay >= world.Tick(cfg.Partnering.MinAge*year) {
			adults[homeZone(w, id)]++
		}
	}
	total := adults[0] + adults[1] + adults[2]
	start := w.Events[0].(world.MayorEvent)
	votes := start.Votes
	own := cfg.Governance.OwnZoneVote
	for z := range world.NumZones {
		want := adults[z]*own + total*(1-own)/3
		if got := float64(votes[z]); got < 0.97*want || got > 1.03*want {
			t.Errorf("zone %d: %v votes, want ≈ %.0f", z, got, want)
		}
	}
	t.Logf("day-1 election, 10,000 people: votes %v (Up Top, Mids, Down Deep), winner's zone %s", votes, start.Zone)
}
