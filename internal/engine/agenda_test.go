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

// planner is a test module that schedules agenda entries in Init and logs
// them, next to its own step, as they fire.
type planner struct {
	fake
	agenda *Agenda
	plan   func(p *planner) error
}

func (p *planner) UseAgenda(a *Agenda) { p.agenda = a }
func (p *planner) Init(*world.World, *config.Scenario, *rng.Stream) error {
	return p.plan(p)
}

// at schedules an entry that logs label when it fires.
func (p *planner) at(day world.Tick, phase Phase, label string) (Handle, error) {
	return p.agenda.Schedule(day, phase, func(w *world.World) error {
		*p.log = append(*p.log, fmt.Sprintf("day %d %s", w.Day, label))
		return nil
	})
}

func runPlanner(t *testing.T, days int, p *planner) []string {
	t.Helper()
	var log []string
	p.log = &log
	if p.name == "" {
		p.name = "demography"
	}
	e := mustNew(t, scenario(0, 1), &world.World{}, p)
	for range days {
		if err := e.Tick(); err != nil {
			t.Fatal(err)
		}
	}
	return log
}

func TestAgendaOrderByDayThenPhase(t *testing.T) {
	log := runPlanner(t, 3, &planner{
		fake: fake{phases: []Phase{PhaseDeaths}},
		plan: func(p *planner) error {
			// Scheduled out of order on purpose.
			for _, s := range []struct {
				day   world.Tick
				phase Phase
				label string
			}{
				{3, PhaseGenerator, "c"},
				{2, PhaseGovernance, "b2"},
				{2, PhaseFood, "b1"},
				{1, PhaseDeaths, "a"},
			} {
				if _, err := p.at(s.day, s.phase, s.label); err != nil {
					return err
				}
			}
			return nil
		},
	})
	// An entry fires before its phase's step; governance's step is not due
	// on day 2, but its entry still fires.
	want := []string{
		"day 1 a", "day 1 phase 700",
		"day 2 b1", "day 2 phase 700", "day 2 b2",
		"day 3 c", "day 3 phase 700",
	}
	if !slices.Equal(log, want) {
		t.Errorf("got %v, want %v", log, want)
	}
}

func TestAgendaSameTickTiesInScheduleOrder(t *testing.T) {
	log := runPlanner(t, 1, &planner{
		plan: func(p *planner) error {
			for _, label := range []string{"first", "second", "third", "fourth"} {
				if _, err := p.at(1, PhaseHealth, label); err != nil {
					return err
				}
			}
			return nil
		},
	})
	want := []string{"day 1 first", "day 1 second", "day 1 third", "day 1 fourth"}
	if !slices.Equal(log, want) {
		t.Errorf("got %v, want %v", log, want)
	}
}

func TestAgendaCancel(t *testing.T) {
	log := runPlanner(t, 2, &planner{
		plan: func(p *planner) error {
			if _, err := p.at(1, PhaseHealth, "kept"); err != nil {
				return err
			}
			h, err := p.at(1, PhaseHealth, "cancelled")
			if err != nil {
				return err
			}
			// An entry can cancel another due later.
			later, err := p.at(2, PhaseHealth, "cancelled by an entry")
			if err != nil {
				return err
			}
			_, err = p.agenda.Schedule(1, PhaseDeaths, func(*world.World) error {
				p.agenda.Cancel(later)
				return nil
			})
			p.agenda.Cancel(h)
			p.agenda.Cancel(h)        // twice does nothing
			p.agenda.Cancel(Handle{}) // nor does an empty handle
			return err
		},
	})
	if want := []string{"day 1 kept"}; !slices.Equal(log, want) {
		t.Errorf("got %v, want %v", log, want)
	}
}

func TestAgendaEntryCanScheduleLater(t *testing.T) {
	log := runPlanner(t, 2, &planner{
		plan: func(p *planner) error {
			_, err := p.agenda.Schedule(1, PhaseHealth, func(*world.World) error {
				if _, err := p.at(1, PhaseDeaths, "same day, later phase"); err != nil {
					return err
				}
				_, err := p.at(2, PhaseGenerator, "next day")
				return err
			})
			return err
		},
	})
	if want := []string{"day 1 same day, later phase", "day 2 next day"}; !slices.Equal(log, want) {
		t.Errorf("got %v, want %v", log, want)
	}
}

func TestAgendaRefusesThePast(t *testing.T) {
	var errs []error
	runPlanner(t, 2, &planner{
		plan: func(p *planner) error {
			_, err := p.agenda.Schedule(2, PhaseHealth, func(*world.World) error {
				for _, s := range []struct {
					day   world.Tick
					phase Phase
				}{{2, PhaseHealth}, {2, PhaseFood}, {1, PhaseGovernance}} {
					_, err := p.at(s.day, s.phase, "never")
					errs = append(errs, err)
				}
				return nil
			})
			return err
		},
	})
	if len(errs) != 3 {
		t.Fatalf("entry ran %d checks, want 3", len(errs))
	}
	for i, err := range errs {
		if err == nil || !strings.Contains(err.Error(), "is not after day 2, phase 600") {
			t.Errorf("case %d: got %v, want a past-schedule error", i, err)
		}
	}
	if _, err := (&Agenda{}).Schedule(1, 750, nil); err == nil || !strings.Contains(err.Error(), "unknown phase 750") {
		t.Errorf("unknown phase: got %v", err)
	}
}

func TestAgendaErrorStopsTick(t *testing.T) {
	var log []string
	p := &planner{
		fake: fake{name: "demography", log: &log},
		plan: func(p *planner) error {
			_, err := p.agenda.Schedule(1, PhaseLottery, func(*world.World) error { return errors.New("boom") })
			return err
		},
	}
	e := mustNew(t, scenario(0, 1), &world.World{}, p)
	want := "day 1, phase 1200 (lottery), agenda: boom"
	if err := e.Tick(); err == nil || err.Error() != want {
		t.Errorf("got %v, want %q", err, want)
	}
}

// shocks is a test module that defines event types and logs them as they
// fire, next to its own steps.
type shocks struct {
	fake
	types []EventType
}

func (s shocks) EventTypes() []EventType { return s.types }

// logShock returns a type that logs its name when it fires.
func logShock(name string, phase Phase, log *[]string) EventType {
	return EventType{Name: name, Phase: phase, Fire: func(w *world.World, _ *rng.Stream, _ config.Event) error {
		*log = append(*log, fmt.Sprintf("day %d %s", w.Day, name))
		return nil
	}}
}

func TestScriptedEventsFireOnTheirDayInFileOrder(t *testing.T) {
	var log []string
	cfg := scenario(0, 1)
	cfg.Events = []config.Event{{Day: 2, Type: "b"}, {Day: 2, Type: "a"}, {Day: 1, Type: "b"}, {Day: 2, Type: "b"}}
	w := &world.World{}
	e := mustNew(t, cfg, w, shocks{
		fake:  fake{name: "health", phases: []Phase{PhaseHealth}, log: &log},
		types: []EventType{logShock("a", PhaseHealth, &log), logShock("b", PhaseHealth, &log)},
	})
	for range 3 {
		if err := e.Tick(); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{
		"day 1 b", "day 1 phase 600",
		"day 2 b", "day 2 a", "day 2 b", "day 2 phase 600",
		"day 3 phase 600",
	}
	if !slices.Equal(log, want) {
		t.Errorf("got %v, want %v", log, want)
	}
	if len(w.Events) != 4 {
		t.Fatalf("logged %d events, want 4", len(w.Events))
	}
	if got, want := w.Events[1], (world.ScriptedEvent{EventHeader: world.EventHeader{Day: 2, Type: "scripted"}, Event: "b"}); got != want {
		t.Errorf("second event %+v, want %+v", got, want)
	}
}

// Scripted events draw from their own stream, never the module's.
func TestScriptedEventsUseTheEventsStream(t *testing.T) {
	var got float64
	cfg := scenario(0, 1)
	cfg.Events = []config.Event{{Day: 1, Type: "draw"}}
	e := mustNew(t, cfg, &world.World{}, shocks{
		fake: fake{name: "health"},
		types: []EventType{{Name: "draw", Phase: PhaseHealth, Fire: func(_ *world.World, r *rng.Stream, _ config.Event) error {
			got = r.Float64()
			return nil
		}}},
	})
	if err := e.Tick(); err != nil {
		t.Fatal(err)
	}
	events, _ := rng.New(cfg.Seed, "events")
	health, _ := rng.New(cfg.Seed, "health")
	if want := events.Float64(); got != want || got == health.Float64() {
		t.Errorf("drew %v, want %v from the events stream", got, want)
	}
}

func TestScriptedEventSetupErrors(t *testing.T) {
	var log []string
	shock := logShock("shock", PhaseHealth, &log)
	cases := []struct {
		want   string
		events []config.Event
		mods   []Module
	}{
		{`events[1]: no module that is on defines event type "shok"`,
			[]config.Event{{Day: 1, Type: "shock"}, {Day: 1, Type: "shok"}},
			[]Module{shocks{fake: fake{name: "health"}, types: []EventType{shock}}}},
		{`no module that is on defines event type "shock"`, // its module is off
			[]config.Event{{Day: 1, Type: "shock"}}, nil},
		{`modules "health" and "work" both define event type "shock"`, nil, []Module{
			shocks{fake: fake{name: "health"}, types: []EventType{shock}},
			shocks{fake: fake{name: "work"}, types: []EventType{shock}}}},
		{`module "health" defines event type "bad" at unknown phase 750`, nil, []Module{
			shocks{fake: fake{name: "health"}, types: []EventType{{Name: "bad", Phase: 750}}}}},
	}
	for _, c := range cases {
		cfg := scenario(0, 1)
		cfg.Events = c.events
		_, err := New(cfg, &world.World{}, c.mods)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want an error containing %q", err, c.want)
		}
	}
}
