package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
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
