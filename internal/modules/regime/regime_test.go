package regime

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// tiny is a world with just what the regime step reads: the rebellion's
// open spell and triggers, the Head of IT, the lift and air.
type tiny struct {
	t   *testing.T
	w   *world.World
	m   *Module
	cfg *config.Scenario
}

func newTiny(t *testing.T, playbook ...config.Entry) *tiny {
	t.Helper()
	cfg, err := config.Load("../../../scenarios/baseline.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Regime.Playbook = playbook
	w := &world.World{Silo: world.Silo{AirQuality: 1}, Beliefs: &world.Beliefs{},
		Rebellion: &world.Rebellion{UnrestStart: world.None, RebellionStart: world.None}, Control: &world.Control{Head: 0}}
	s := &tiny{t: t, w: w, m: New(), cfg: cfg}
	if err := s.m.Init(w, cfg, nil); err != nil {
		t.Fatal(err)
	}
	return s
}

// day runs one regime step with these triggers fired.
func (s *tiny) day(fired ...string) {
	s.t.Helper()
	s.w.Day++
	s.w.Rebellion.Fired = fired
	if err := s.m.step(s.w, nil); err != nil {
		s.t.Fatal(err)
	}
}

func (s *tiny) sigma() float64 { return s.w.Policy.Control.Surveillance }

func f(x float64) *float64 { return &x }

// events lists the day and type of the policy events logged.
func (s *tiny) events() []string {
	var out []string
	for _, e := range s.w.Events {
		if p, ok := e.(world.PolicyEvent); ok {
			out = append(out, fmt.Sprintf("%s@%d", p.Type, p.Day))
		}
	}
	return out
}

// ODD §6.15: a policy event held 3 days puts its lever in force from its
// regime step through the step 3 days later, when it lapses: every step
// in between reads it on exactly 3 days.
func TestHoldDays(t *testing.T) {
	s := newTiny(t)
	peace := s.sigma()
	s.day()
	if err := s.m.EventTypes()[0].Fire(s.w, nil, config.Event{Type: "policy", Hold: "3", Set: config.Levers{Surveillance: f(0.5)}}); err != nil {
		t.Fatal(err)
	}
	var got []float64
	for range 5 {
		s.day()
		got = append(got, s.sigma())
	}
	if want := []float64{0.5, 0.5, 0.5, peace, peace}; !slices.Equal(got, want) {
		t.Errorf("σ on days 2–6: %v, want %v", got, want)
	}
	if ev := s.events(); !slices.Equal(ev, []string{"policy@2", "policyEnd@5"}) {
		t.Errorf("events %v", ev)
	}
}

// ODD §6.15: an entry started by unrest and held quiet runs while the
// spell is open, through the rebellion it becomes, and lapses at the first
// step with neither open. A later entry wins a lever both set; when it
// lapses, the earlier one's value comes back.
func TestQuietHoldAndOverlap(t *testing.T) {
	s := newTiny(t,
		config.Entry{On: "unrestStart", Hold: "quiet", Set: config.Levers{Surveillance: f(0.2)}},
		config.Entry{On: "rebellionStart", Hold: "2", Set: config.Levers{Surveillance: f(0.4)}})
	b := s.w.Rebellion
	b.UnrestStart = 1
	s.day("unrestStart")
	if s.sigma() != 0.2 {
		t.Fatalf("σ %v after unrest, want 0.2", s.sigma())
	}
	b.UnrestStart, b.RebellionStart = world.None, 1
	s.day("rebellionStart")
	if s.sigma() != 0.4 {
		t.Errorf("σ %v after the rebellion started, want the later entry's 0.4", s.sigma())
	}
	s.day()
	s.day()
	if s.sigma() != 0.2 {
		t.Errorf("σ %v once the 2-day entry lapsed, want the quiet entry's 0.2", s.sigma())
	}
	b.RebellionStart = world.None
	s.day("suppressed")
	if s.sigma() != s.cfg.Control.Surveillance || len(s.w.Regime.Running) != 0 {
		t.Errorf("σ %v with %d running after the rebellion ended, want peacetime", s.sigma(), len(s.w.Regime.Running))
	}
}

// ODD §6.15: a regime change ends every running entry and starts the
// regimeChange entries for good; the old playbook's triggers start
// nothing from then on, nor does a second regime change. While the Head
// of IT's post is vacant the old regime starts nothing.
func TestRegimeChangeAndVacantHead(t *testing.T) {
	s := newTiny(t,
		config.Entry{On: "unrestStart", Hold: "forever", Set: config.Levers{Surveillance: f(0.2)}},
		config.Entry{On: "regimeChange", Hold: "3", Set: config.Levers{Surveillance: f(0)}})
	s.w.Control.Head = world.None
	s.day("unrestStart")
	if len(s.w.Regime.Running) != 0 {
		t.Fatal("a headless regime started an entry")
	}
	s.w.Control.Head = 0
	s.day("unrestStart")
	s.day("regimeChange")
	for range 5 {
		s.day("unrestStart")
	}
	s.day("regimeChange")
	r := s.w.Regime.Running
	if !s.w.Regime.Changed || len(r) != 1 || r[0].Trigger != "regimeChange" || s.sigma() != 0 {
		t.Errorf("after the change: changed %v, running %+v, σ %v; want only the new regime's entry, held for good", s.w.Regime.Changed, r, s.sigma())
	}
}

// ODD §6.15: a lift is added once, either sign; a water dosing is ordered
// for the next beliefs step.
func TestLiftAndWater(t *testing.T) {
	s := newTiny(t, config.Entry{On: "suppressed", Hold: "1", Lift: -0.25, WaterDosing: true})
	s.day("suppressed")
	s.day()
	if s.w.Beliefs.Lift != -0.25 || !s.w.Beliefs.WaterOrdered {
		t.Errorf("lift %v, water ordered %v", s.w.Beliefs.Lift, s.w.Beliefs.WaterOrdered)
	}
}

// ODD §6.15: two entries each fine alone whose rows together let two
// active rules list one offence stop the run on the day the later starts.
func TestPactCombinationFails(t *testing.T) {
	s := newTiny(t)
	rule := func(id string, active bool, offences ...string) config.Rule {
		return config.Rule{ID: id, Active: active, Offences: offences, Enforcer: "judicial"}
	}
	s.cfg.Regime.Playbook = []config.Entry{
		{On: "unrestStart", Hold: "quiet", Pact: []config.Rule{rule("noViolence", false, "violence", "sabotage"), rule("extra", true, "violence")}},
		{On: "rebellionStart", Hold: "quiet", Pact: []config.Rule{rule("noViolence", false, "violence", "sabotage"), rule("noOrganising", true, "organise", "violence")}},
	}
	s.w.Rebellion.UnrestStart = 1
	s.day("unrestStart")
	s.w.Day++
	s.w.Rebellion.Fired = []string{"rebellionStart"}
	if err := s.m.step(s.w, nil); err == nil || !strings.Contains(err.Error(), "violence") {
		t.Errorf("error %v, want one about violence listed twice", err)
	}
}

// ODD §6.15, Emergency staffing: air below the line in force declares an
// emergency, logged when it starts and ends, its days counted.
func TestEmergency(t *testing.T) {
	s := newTiny(t)
	s.cfg.Regime.EmergencyAir = 0.8
	s.w.Policy = s.cfg.InForce(nil)
	for _, air := range []float64{0.9, 0.7, 0.75, 0.85} {
		s.w.Silo.AirQuality = air
		s.day()
	}
	var kinds []string
	for _, e := range s.w.Events {
		kinds = append(kinds, e.(world.EventHeader).Type)
	}
	if g := s.w.Regime; g.Emergency || g.EmergencyDays != 2 || !slices.Equal(kinds, []string{"emergency", "emergencyEnd"}) {
		t.Errorf("emergency %v, days %d, events %v", g.Emergency, g.EmergencyDays, kinds)
	}
}
