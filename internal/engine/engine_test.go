package engine

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// fake is a test module that records which of its steps ran on which day.
type fake struct {
	name     string
	requires []string
	phases   []Phase
	log      *[]string
	fail     Phase // this phase returns an error
}

func (f fake) Name() string       { return f.name }
func (f fake) Requires() []string { return f.requires }
func (f fake) Init(*world.World, *config.Scenario, *rng.Stream) error {
	return nil
}
func (f fake) Steps() []Step {
	var steps []Step
	for _, p := range f.phases {
		steps = append(steps, Step{Phase: p, Run: func(w *world.World, _ *rng.Stream) error {
			if p == f.fail {
				return errors.New("boom")
			}
			*f.log = append(*f.log, fmt.Sprintf("day %d phase %d", w.Day, p))
			return nil
		}})
	}
	return steps
}

func scenario(warmupYears, years int) *config.Scenario {
	cfg := &config.Scenario{Seed: 42, WarmupYears: warmupYears, Years: years}
	cfg.Output.InvariantEvery = 1 // tests check every tick
	return cfg
}

func mustNew(t *testing.T, cfg *config.Scenario, w *world.World, mods ...Module) *Engine {
	t.Helper()
	e, err := New(cfg, w, mods)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// days lists the days on which entries in log mention phase p.
func days(log []string, p Phase) []int {
	var out []int
	suffix := fmt.Sprintf(" phase %d", p)
	for _, entry := range log {
		if rest, ok := strings.CutSuffix(entry, suffix); ok {
			var d int
			fmt.Sscanf(rest, "day %d", &d)
			out = append(out, d)
		}
	}
	return out
}

func TestStepsRunInScheduleOrder(t *testing.T) {
	var log []string
	e := mustNew(t, scenario(0, 1), &world.World{},
		fake{name: "demography", phases: []Phase{PhaseDeaths, PhaseLifeSupport}, log: &log})
	if err := e.Tick(); err != nil {
		t.Fatal(err)
	}
	want := []string{"day 1 phase 2", "day 1 phase 7"}
	if !slices.Equal(log, want) {
		t.Errorf("got %v, want %v", log, want)
	}
}

func TestStepsFromSeveralModulesInterleave(t *testing.T) {
	var log []string
	e := mustNew(t, scenario(0, 1), &world.World{},
		fake{name: "demography", phases: []Phase{PhaseDeaths}, log: &log},
		fake{name: "resources", phases: []Phase{PhaseGenerator, PhaseFood}, log: &log},
		fake{name: "health", phases: []Phase{PhaseHealth}, log: &log})
	if err := e.Tick(); err != nil {
		t.Fatal(err)
	}
	want := []string{"day 1 phase 1", "day 1 phase 3", "day 1 phase 6", "day 1 phase 7"}
	if !slices.Equal(log, want) {
		t.Errorf("got %v, want %v", log, want)
	}
}

func TestFrequencies(t *testing.T) {
	var log []string
	e := mustNew(t, scenario(0, 2), &world.World{},
		fake{name: "demography", phases: []Phase{PhaseDeaths, PhaseLottery, PhaseGovernance}, log: &log})
	if err := e.Run(); err != nil {
		t.Fatal(err)
	}
	if got := len(days(log, PhaseDeaths)); got != 2*DaysPerYear {
		t.Errorf("daily step ran %d times in 2 years, want %d", got, 2*DaysPerYear)
	}
	monthly := days(log, PhaseLottery)
	if len(monthly) != 24 || monthly[0] != 30 || monthly[1] != 60 || monthly[23] != 720 {
		t.Errorf("30-day step ran on days %v, want 30, 60, …, 720", monthly)
	}
	if yearly := days(log, PhaseGovernance); !slices.Equal(yearly, []int{365, 730}) {
		t.Errorf("365-day step ran on days %v, want [365 730]", yearly)
	}
}

func TestOutputSkippedDuringWarmup(t *testing.T) {
	for _, logWarmup := range []bool{false, true} {
		var log []string
		cfg := scenario(1, 1)
		cfg.Output.LogWarmup = logWarmup
		e := mustNew(t, cfg, &world.World{},
			fake{name: "demography", phases: []Phase{PhaseDeaths, PhaseLogDaily, PhaseSnapshot}, log: &log})
		if err := e.Run(); err != nil {
			t.Fatal(err)
		}
		logged, snaps := days(log, PhaseLogDaily), days(log, PhaseSnapshot)
		if got := len(days(log, PhaseDeaths)); got != 2*DaysPerYear {
			t.Errorf("logWarmup=%v: model step ran %d times, want %d", logWarmup, got, 2*DaysPerYear)
		}
		if logWarmup {
			if len(logged) != 2*DaysPerYear || logged[0] != 1 || !slices.Equal(snaps, []int{365, 730}) {
				t.Errorf("logWarmup=true: logging ran %d times from day %v, snapshots %v", len(logged), logged[:1], snaps)
			}
		} else {
			if len(logged) != DaysPerYear || logged[0] != DaysPerYear+1 || !slices.Equal(snaps, []int{730}) {
				t.Errorf("logWarmup=false: logging ran %d times from day %v, snapshots %v", len(logged), logged[:1], snaps)
			}
		}
	}
}

func TestSetupErrors(t *testing.T) {
	var log []string
	cases := []struct {
		want string
		mods []Module
	}{
		{`modules "demography" and "health" both registered phase 7 (deaths)`, []Module{
			fake{name: "demography", phases: []Phase{PhaseDeaths}, log: &log},
			fake{name: "health", phases: []Phase{PhaseDeaths}, log: &log}}},
		{`module "health" requires module "work", which is off`, []Module{
			fake{name: "health", requires: []string{"work"}, log: &log}}},
		{`module "health" listed twice`, []Module{
			fake{name: "health", log: &log}, fake{name: "health", log: &log}}},
		{`module "health" registered unknown phase 17`, []Module{
			fake{name: "health", phases: []Phase{17}, log: &log}}},
		{`module "health" registered unknown phase 0`, []Module{
			fake{name: "health", phases: []Phase{0}, log: &log}}},
		{`no stream for "dissent"`, []Module{fake{name: "dissent", log: &log}}},
	}
	for _, c := range cases {
		_, err := New(scenario(0, 1), &world.World{}, c.mods)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want an error containing %q", err, c.want)
		}
	}
}

func TestStepErrorStopsRun(t *testing.T) {
	var log []string
	e := mustNew(t, scenario(0, 1), &world.World{},
		fake{name: "health", phases: []Phase{PhaseHealth, PhaseLottery}, fail: PhaseLottery, log: &log})
	err := e.Run()
	want := "day 30, phase 12 (lottery), module health: boom"
	if err == nil || err.Error() != want {
		t.Errorf("got %v, want %q", err, want)
	}
	if got := len(days(log, PhaseHealth)); got != 30 {
		t.Errorf("ran %d days before stopping, want 30", got)
	}
}

func TestAliveShuffledEveryTickAndReproducible(t *testing.T) {
	run := func() [][]world.PersonID {
		w := &world.World{Floors: []world.Floor{{Number: 1, Zone: world.Mids}}}
		var members []world.PersonID
		for i := range 20 {
			w.People = append(w.People, world.Person{ID: world.PersonID(i), Alive: true, Health: 1, Partner: world.None})
			w.Alive = append(w.Alive, world.PersonID(i))
			members = append(members, world.PersonID(i))
		}
		w.Households = []world.Household{{Floor: 1, Members: members}}
		e := mustNew(t, scenario(0, 1), w)
		var orders [][]world.PersonID
		for range 3 {
			if err := e.Tick(); err != nil {
				t.Fatal(err)
			}
			orders = append(orders, slices.Clone(w.Alive))
		}
		return orders
	}
	a, b := run(), run()
	for day := range a {
		if !slices.Equal(a[day], b[day]) {
			t.Errorf("tick %d: same seed gave different orders", day+1)
		}
		sorted := slices.Clone(a[day])
		slices.Sort(sorted)
		for i, id := range sorted {
			if id != world.PersonID(i) {
				t.Fatalf("tick %d: shuffle lost or duplicated people: %v", day+1, a[day])
			}
		}
	}
	if slices.Equal(a[0], a[1]) || slices.Equal(a[1], a[2]) {
		t.Error("order did not change between ticks")
	}
}

// breaker is a module whose health step pushes a person's health above 1
// on one day.
type breaker struct{ on world.Tick }

func (b breaker) Name() string                                           { return "health" }
func (b breaker) Requires() []string                                     { return nil }
func (b breaker) Init(*world.World, *config.Scenario, *rng.Stream) error { return nil }
func (b breaker) Steps() []Step {
	return []Step{{Phase: PhaseHealth, Run: func(w *world.World, _ *rng.Stream) error {
		if w.Day == b.on {
			w.People[0].Health = 1.5
		}
		return nil
	}}}
}

func TestInvariantsRunDuringTheRun(t *testing.T) {
	newWorld := func() *world.World {
		return &world.World{
			People:     []world.Person{{Alive: true, Health: 1, Partner: world.None}},
			Alive:      []world.PersonID{0},
			Floors:     []world.Floor{{Number: 1, Zone: world.Mids}},
			Households: []world.Household{{Floor: 1, Members: []world.PersonID{0}}},
		}
	}

	e := mustNew(t, scenario(0, 1), newWorld(), breaker{on: 3})
	err := e.Run()
	want := "day 3: invariant broken: person 0 health = 1.5, outside 0–1"
	if err == nil || err.Error() != want {
		t.Errorf("checking every tick: got %v, want %q", err, want)
	}

	// Checked only every 30 days, the same break is found on day 30.
	cfg := scenario(0, 1)
	cfg.Output.InvariantEvery = 30
	e = mustNew(t, cfg, newWorld(), breaker{on: 3})
	if err := e.Run(); err == nil || !strings.HasPrefix(err.Error(), "day 30:") {
		t.Errorf("checking every 30 days: got %v, want an error on day 30", err)
	}
}

func TestInvariantIntervalMustBePositive(t *testing.T) {
	cfg := scenario(0, 1)
	cfg.Output.InvariantEvery = 0
	if _, err := New(cfg, &world.World{}, nil); err == nil {
		t.Error("invariantEvery = 0 accepted")
	}
}
