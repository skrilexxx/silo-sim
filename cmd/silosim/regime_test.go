package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
)

// regimeSilo is a 1,000-person silo with every module through the regime
// on (or through rebellion with regime false), invariants every 30 days,
// and a cleaning failure in year 5 so unrest and rebellions fire the
// triggers.
func regimeSilo(t *testing.T, regime bool, years int) *config.Scenario {
	t.Helper()
	golden := loadInto(t, goldenScenario)
	cfg := loadInto(t, "../../scenarios/baseline.yaml")
	cfg.Seed, cfg.WarmupYears, cfg.Years, cfg.Population = golden.Seed, 0, years, golden.Population
	cfg.Lottery.TargetPopulation = cfg.Population
	cfg.Output.InvariantEvery = 30
	cfg.Rebellion.PointOfNoReturn = 0 // P_min is for a silo of 10,000
	cfg.Modules = append(cfg.Modules, "space", "network", "beliefs", "control", "rebellion")
	if regime {
		cfg.Modules = append(cfg.Modules, "regime")
	}
	cfg.Events = []config.Event{{Day: 5*365 + 100, Type: "cleaningFailure"}}
	return cfg
}

// ODD §6.15, Off and null: with an empty playbook, no emergency line, no
// policy events and no election lift, the regime module changes nothing:
// daily.csv, events.jsonl and rebellions.jsonl are byte-identical to a run
// without it, and yearly.csv differs only by its two columns at the end.
func TestRegimeNull(t *testing.T) {
	off := mustSimulate(t, regimeSilo(t, false, 15))
	null := regimeSilo(t, true, 15)
	null.Regime.ElectionLift = 0
	on := mustSimulate(t, null)
	for _, f := range []string{"daily.csv", "events.jsonl", "rebellions.jsonl"} {
		if !bytes.Equal(readFile(t, filepath.Join(off, f)), readFile(t, filepath.Join(on, f))) {
			t.Errorf("%s differs with the regime module at its null setting", f)
		}
	}
	a := strings.Split(string(readFile(t, filepath.Join(off, "yearly.csv"))), "\n")
	b := strings.Split(string(readFile(t, filepath.Join(on, "yearly.csv"))), "\n")
	if len(a) != len(b) {
		t.Fatalf("yearly.csv: %d lines without the module, %d with", len(a), len(b))
	}
	for i := range a {
		if a[i] != "" && !strings.HasPrefix(b[i], a[i]+",") || strings.Count(b[i], ",") != strings.Count(a[i], ",")+2*min(1, len(a[i])) {
			t.Errorf("yearly.csv line %d: %q is not %q plus two columns", i, b[i], a[i])
		}
	}
	ev := string(readFile(t, filepath.Join(on, "events.jsonl")))
	t.Logf("%d unrest spells and %d rebellions fired triggers with nothing to start",
		strings.Count(ev, `"type":"unrest"`), strings.Count(ev, `"type":"rebellionStart"`))
	if !strings.Contains(ev, `"type":"rebellionStart"`) {
		t.Error("no rebellion: the null setting was never put to the test")
	}
	if s := sum(column(t, on, "emergencyDays")) + sum(column(t, on, "policies")); s != 0 {
		t.Errorf("emergency days and running entries add up to %d, want 0", s)
	}

	// An entry that sets σ to its own value starts with each rebellion and
	// lapses the day it ends, and changes nothing else.
	same := regimeSilo(t, true, 15)
	same.Regime.ElectionLift = 0
	same.Regime.Playbook = []config.Entry{{On: "rebellionStart", Hold: "quiet", Set: config.Levers{Surveillance: &same.Control.Surveillance}}}
	dir := mustSimulate(t, same)
	if !bytes.Equal(readFile(t, filepath.Join(on, "daily.csv")), readFile(t, filepath.Join(dir, "daily.csv"))) {
		t.Error("an entry that changes no lever changed daily.csv")
	}
	ev = string(readFile(t, filepath.Join(dir, "events.jsonl")))
	rebellions, ends := strings.Count(ev, `"type":"rebellionStart"`), strings.Count(ev, `"type":"rebellionEnd"`)
	starts, lapses := strings.Count(ev, `"type":"policy","trigger":"rebellionStart"`), strings.Count(ev, `"type":"policyEnd","trigger":"rebellionStart"`)
	if starts != rebellions || lapses != ends {
		t.Errorf("%d entries started and %d lapsed, for %d rebellions started and %d ended", starts, lapses, rebellions, ends)
	}
}

// ODD §6.15: a policy event held for good on day d pulls levers that
// other modules read. From then on the Down Deep is blamed harder (H
// rises there), fed half rations (its daily hardship is positive), the
// Mayor is elected (a "policy" Mayor change at the next governance step),
// the lottery is fair (no denied wins), the water is dosed, and the lift
// shows at the next snapshot. The event is logged on its day with no
// end.
func TestPolicyEventPullsLevers(t *testing.T) {
	cfg := regimeSilo(t, true, 10)
	half, blame, fair, election := config.Zones{UpTop: 1, Mids: 1, DownDeep: 0.5}, 0.6, false, "election"
	day := 5*365 + 10
	cfg.Events = append(cfg.Events, config.Event{Day: day, Type: "policy", Hold: "forever", Lift: 0.1, WaterDosing: true,
		Set: config.Levers{RationWeights: &half, ScapegoatHardship: &blame, Denial: &fair, Governance: &election}})
	dir := mustSimulate(t, cfg)

	if h := floats(t, dir, "meanHDownDeep"); h[5] < h[4]+0.2 {
		t.Errorf("Down Deep H %.3f in year 5, %.3f in year 6; want it up by the extra blame", h[4], h[5])
	}
	if l := floats(t, dir, "lift"); l[5] < 0.03 {
		t.Errorf("lift %.3f at the snapshot after the event, want the event's 0.1 half gone at most", l[5])
	}
	if d := column(t, dir, "dosed"); d[5] == 0 {
		t.Error("nobody dosed at the snapshot after the water dosing started")
	}
	for y, n := range column(t, dir, "deniedWins")[6:] {
		if n != 0 {
			t.Errorf("year %d: %d denied wins with a fair lottery", y+7, n)
		}
	}

	daily := strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(dir, "daily.csv")))), "\n")
	col := strings.Index(daily[0], "hardshipDownDeep")
	idx := strings.Count(daily[0][:col], ",")
	after := strings.Split(daily[day+1], ",") // row i is day i: the header is row 0
	if after[0] != strconv.Itoa(day+1) || after[idx] == "0.0000" || strings.HasPrefix(after[idx], "-") {
		t.Errorf("day %s: Down Deep hardship %s, want positive on half rations", after[0], after[idx])
	}

	events := string(readFile(t, filepath.Join(dir, "events.jsonl")))
	for _, want := range []string{`{"day":1835,"type":"policy","trigger":"scripted","start":1835}`, `"reason":"policy","mode":"election"`} {
		if !strings.Contains(events, want) {
			t.Errorf("events.jsonl has no %s", want)
		}
	}
	if strings.Contains(events, `"policyEnd"`) {
		t.Error("an entry held forever ended")
	}
}

// regimeAs is regimeSilo with a strategy file's playbook.
func regimeAs(t *testing.T, strategy string, years int) *config.Scenario {
	t.Helper()
	cfg := regimeSilo(t, true, years)
	cfg.Regime.Playbook = loadInto(t, "../../scenarios/"+strategy+".yaml").Regime.Playbook
	return cfg
}

// T13.3, ODD §6.15: the memory wipe in a 1,000-person silo. While a
// rebellion runs, convicts are dosed; once it ends, the water is, and the
// history its ending taught everyone is hidden from most adults at the
// next snapshot.
func TestMemoryWipe(t *testing.T) {
	dir := mustSimulate(t, regimeAs(t, "strategy-wipe", 15))
	ev := string(readFile(t, filepath.Join(dir, "events.jsonl")))
	if !strings.Contains(ev, `"type":"policy","trigger":"rebellionStart"`) {
		t.Fatal("no rebellion started the wipe")
	}
	ended := strings.Contains(ev, `"type":"policy","trigger":"suppressed"`) || strings.Contains(ev, `"type":"policy","trigger":"split"`)
	dosed, hidden := column(t, dir, "dosed"), floats(t, dir, "hiddenAny")
	most, peak := 0, 0.0
	for y := range dosed {
		most, peak = max(most, dosed[y]), max(peak, hidden[y])
	}
	t.Logf("ended %v; most dosed at a snapshot %d, highest share with a hidden flag %.3f", ended, most, peak)
	if most == 0 {
		t.Error("nobody dosed at any snapshot")
	}
	if ended && peak < 0.5 {
		t.Errorf("after a wipe, at most %.3f of adults had a flag hidden; want most", peak)
	}
}

// T13.4, ODD §6.9, §6.15: a Pact rewrite by policy event. Switching off the
// rule against forbidden talk stops trials for it: with surveillance
// raised so talk is seen, the year after the rewrite has no warnings; a
// rewrite that costs legitimacy (lift −0.1) shows as a lower lift S at
// the next snapshot.
func TestPactRewrite(t *testing.T) {
	run := func(active bool, lift float64) string {
		cfg := regimeSilo(t, true, 8)
		cfg.Events = nil
		i := slices.IndexFunc(cfg.Pact, func(r config.Rule) bool { return r.ID == "noForbiddenTalk" })
		row := cfg.Pact[i]
		row.Active = active
		sigma := 1.0
		cfg.Events = []config.Event{{Day: 5*365 + 10, Type: "revealHistory"},
			{Day: 5*365 + 20, Type: "policy", Hold: "forever", Set: config.Levers{Surveillance: &sigma}, Pact: []config.Rule{row}, Lift: lift}}
		return mustSimulate(t, cfg)
	}
	on, off := run(true, 0), run(false, 0)
	if w := column(t, on, "warnings")[6]; w == 0 {
		t.Fatalf("with the rule in force and σ = 1, no warnings in year 7; the test sees nothing")
	}
	if w := column(t, off, "warnings")[6]; w != 0 {
		t.Errorf("rule switched off: %d warnings in year 7, want 0", w)
	}
	cost := run(false, -0.1)
	if l0, l1 := floats(t, off, "lift")[5], floats(t, cost, "lift")[5]; l1 > l0-0.03 {
		t.Errorf("lift at the snapshot: %.3f without the cost, %.3f with −0.1; want it clearly lower", l0, l1)
	}
}

// T14.1, ODD §6.16: Silo 1 in a 1,000-person silo. Two scripted cleaning
// failures a year apart are two escapes within 5 years: each brings Silo
// 1's order through the Head of IT, and the Safeguard goes off 30 days
// after the second, a collapse with cause safeguard that ends the run.
func TestSafeguard(t *testing.T) {
	cfg := regimeSilo(t, true, 10)
	cfg.Modules = append(cfg.Modules, "silo1")
	first, second := 5*365+100, 6*365+100
	cfg.Events = []config.Event{{Day: first, Type: "cleaningFailure"}, {Day: second, Type: "cleaningFailure"}}
	dir := mustSimulate(t, cfg)
	ev := string(readFile(t, filepath.Join(dir, "events.jsonl")))
	if n := strings.Count(ev, `"type":"policy","trigger":"silo1"`); n < 2 {
		t.Errorf("%d Silo 1 orders, want one per escape", n)
	}
	want := fmt.Sprintf(`{"day":%d,"type":"collapse","cause":"safeguard"}`, second+30)
	if !strings.Contains(ev, want) {
		t.Errorf("events.jsonl has no %s", want)
	}
	if rj := string(readFile(t, filepath.Join(dir, "run.json"))); !strings.Contains(rj, `"collapseCause": "safeguard"`) {
		t.Error("run.json lacks the safeguard cause")
	}
}
