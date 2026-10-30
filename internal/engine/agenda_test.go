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
