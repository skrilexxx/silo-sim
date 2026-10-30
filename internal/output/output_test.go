package output

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
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

// A department still understaffed when warm-up ends is logged again on
// the first measured day, because its warm-up event is dropped.
func TestUnderstaffedCarriesOverWarmup(t *testing.T) {
	o, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()
	cfg := &config.Scenario{WarmupYears: 1, Output: config.Output{UnderstaffedShare: 0.5}}
	w := &world.World{Depts: []world.Department{{Name: "it", Target: 10, Headcount: 2}}}
	if err := o.Init(w, cfg, nil); err != nil {
		t.Fatal(err)
	}
	for w.Day = 1; w.Day <= engine.DaysPerYear+2; w.Day++ {
		if err := o.logDaily(w, nil); err != nil {
			t.Fatal(err)
		}
	}
	// Logged on day 1 (warm-up, dropped), then again on day 366, the first
	// measured day.
	if len(w.Events) != 0 || !o.under[0] {
		t.Fatalf("events left %v, under %v", o.w.Events, o.under)
	}
	o.events.Flush()
	data, _ := os.ReadFile(filepath.Join(filepath.Dir(o.files[2].Name()), "events.jsonl"))
	if got := strings.TrimSpace(string(data)); got != `{"day":366,"type":"understaffed","department":"it","workers":2,"target":10}` {
		t.Errorf("events.jsonl = %q, want one understaffed event on day 366", got)
	}
}

// shocker is a test module that defines one scripted event type and
// records the days it fired on.
type shocker struct{ fired *[]world.Tick }

func (s shocker) Name() string                                           { return "health" }
func (s shocker) Requires() []string                                     { return nil }
func (s shocker) Init(*world.World, *config.Scenario, *rng.Stream) error { return nil }
func (s shocker) Steps() []engine.Step                                   { return nil }
func (s shocker) EventTypes() []engine.EventType {
	return []engine.EventType{{Name: "testShock", Phase: engine.PhaseHealth,
		Fire: func(w *world.World, _ *rng.Stream) error {
			*s.fired = append(*s.fired, w.Day)
			return nil
		}}}
}

// T7.4: a scripted event fires on its day and is logged to events.jsonl.
func TestScriptedEventLogged(t *testing.T) {
	dir := t.TempDir()
	o, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Scenario{Seed: 1, Years: 1, Events: []config.Event{{Day: 3, Type: "testShock"}}}
	cfg.Output.InvariantEvery = 1
	var fired []world.Tick
	e, err := engine.New(cfg, &world.World{}, []engine.Module{shocker{&fired}, o})
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		if err := e.Tick(); err != nil {
			t.Fatal(err)
		}
	}
	if err := o.Close(); err != nil {
		t.Fatal(err)
	}
	if len(fired) != 1 || fired[0] != 3 {
		t.Errorf("fired on days %v, want [3]", fired)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	if got := strings.TrimSpace(string(data)); got != `{"day":3,"type":"scripted","event":"testShock"}` {
		t.Errorf("events.jsonl = %q", got)
	}
}

func TestAppendTrimmed(t *testing.T) {
	for x, want := range map[float64]string{0: "0", 73: "73", 100: "100", 0.5: "0.5", 0.91234: "0.9123", 0.00004: "0"} {
		if got := string(appendTrimmed(nil, x)); got != want {
			t.Errorf("%v: got %q, want %q", x, got, want)
		}
	}
}

// Baseline frame metrics: dissolved households and empty floors count as 0.
func TestBaselineFrameMetrics(t *testing.T) {
	w := &world.World{
		People: []world.Person{{Health: 1}, {Health: 0.5}, {Health: 0.2}},
		Floors: make([]world.Floor, 3),
		Households: []world.Household{
			{Floor: 1, Members: []world.PersonID{0, 1}},
			{Floor: world.None}, // dissolved
			{Floor: 3, Members: []world.PersonID{2}},
		},
	}
	want := map[string][]float64{
		"population": {2, 0, 1},
		"households": {1, 0, 1},
		"health":     {0.75, 0, 0.2},
	}
	for _, m := range baselineMetrics {
		v := make([]float64, 3)
		m.Fill(w, v)
		if !slices.Equal(v, want[m.Name]) {
			t.Errorf("%s = %v, want %v", m.Name, v, want[m.Name])
		}
	}
}
