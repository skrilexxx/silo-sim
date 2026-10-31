package beliefs

import (
	"math"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const year = engine.DaysPerYear

// tiny is a one-floor world; people are added with add, then start runs Init.
type tiny struct {
	t *testing.T
	w *world.World
	m *Module
	r *rng.Stream
}

func newTiny(t *testing.T) *tiny {
	// The first step is a recompute day (ODD §6.12, Upbringing).
	w := &world.World{Day: 1200*engine.DaysPerMonth - 1, Floors: []world.Floor{{Number: 1, Homes: 9}}, Network: &world.Network{}}
	r, _ := rng.New(1, "beliefs")
	return &tiny{t: t, w: w, m: New(), r: r}
}

// add puts a person of the given age in years and stage in their own
// household and returns their ID.
func (s *tiny) add(ageYears int, stage world.Stage) world.PersonID {
	w := s.w
	id, hh := world.PersonID(len(w.People)), world.HouseholdID(len(w.Households))
	w.People = append(w.People, world.Person{ID: id, Alive: true, BirthDay: w.Day - world.Tick(ageYears*year), Household: hh,
		Stage: stage, Partner: world.None, Mother: world.None, Father: world.None, Master: world.None, BabyFather: world.None})
	w.Alive = append(w.Alive, id)
	w.Households = append(w.Households, world.Household{ID: hh, Floor: 1, Members: []world.PersonID{id}})
	return id
}

func (s *tiny) start() {
	s.t.Helper()
	cfg := &config.Scenario{}
	cfg.Work.WorkerAge = 18
	cfg.Beliefs.TaughtLegitimacy, cfg.Beliefs.EducationWeight, cfg.Beliefs.LegitimacyCostPerFlag = 0.8, 0.5, 0.15
	if err := s.m.Init(s.w, cfg, s.r); err != nil {
		s.t.Fatal(err)
	}
}

func (s *tiny) step() {
	s.t.Helper()
	s.w.Day++
	if err := s.m.step(s.w, s.r); err != nil {
		s.t.Fatal(err)
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

// ODD §6.12, Upbringing: base = ε L_edu + (1 − ε) × mean L of the living
// parents and, for a shadow, the master.
func TestUpbringing(t *testing.T) {
	s := newTiny(t)
	mother, father := s.add(40, world.Worker), s.add(42, world.Worker)
	child := s.add(5, world.Child)
	shadow := s.add(15, world.Shadow)
	orphan := s.add(10, world.Child)
	deadDad := s.add(50, world.Worker)
	halfOrphan := s.add(8, world.Child)
	p := s.w.People
	p[child].Mother, p[child].Father = mother, father
	p[shadow].Mother, p[shadow].Master = mother, father
	p[halfOrphan].Mother, p[halfOrphan].Father = mother, deadDad
	p[deadDad].Alive = false
	s.w.Alive = []world.PersonID{halfOrphan, mother, child, shadow, orphan, father} // not in ID order
	s.start()

	b := s.w.Beliefs
	b.Known[mother] = 1<<world.FlagHistory | 1<<world.FlagOutside // L = 0.8 − 2 × 0.15 = 0.5
	s.step()
	for _, c := range []struct {
		who  world.PersonID
		want float64
	}{
		{mother, 0.8}, {father, 0.8}, // adults keep the taught base
		{child, 0.4 + 0.5*(0.5+0.8)/2},
		{shadow, 0.4 + 0.5*(0.5+0.8)/2}, // the master counts like a parent
		{orphan, 0.8},
		{halfOrphan, 0.4 + 0.5*0.5}, // the dead father doesn't count
	} {
		if got := b.Base[c.who]; !near(got, c.want) {
			t.Errorf("person %d: base %v, want %v", c.who, got, c.want)
		}
	}
	if got := b.L(mother); !near(got, 0.5) {
		t.Errorf("mother's L %v, want 0.5", got)
	}
}

// ODD §6.12: the base is recomputed every 30 days, not in between, until
// worker age, then fixed for life at its last value.
func TestBaseMonthlyAndFixedAtWorkerAge(t *testing.T) {
	s := newTiny(t)
	mother := s.add(40, world.Worker)
	teen := s.add(18, world.Shadow)
	s.w.People[teen].BirthDay += 10 // reaches worker age 9 days after the first step
	s.w.People[teen].Mother = mother
	s.start()
	b := s.w.Beliefs
	b.Known[mother] = 1 << world.FlagLottery
	s.step() // a recompute day
	want := 0.4 + 0.5*0.65
	if !near(b.Base[teen], want) {
		t.Fatalf("base %v, want %v", b.Base[teen], want)
	}
	b.Known[mother] = 0
	for range engine.DaysPerMonth - 1 {
		s.step()
		if !near(b.Base[teen], want) {
			t.Fatalf("day %d: base changed to %v between recompute days", s.w.Day, b.Base[teen])
		}
	}
	s.step() // the next recompute day, after worker age
	if !near(b.Base[teen], want) {
		t.Errorf("base changed to %v after worker age, want it fixed at %v", b.Base[teen], want)
	}
}

// ODD §6.12, Legitimacy: L = max(0, base − δ × flags known).
func TestLegitimacyNeverBelowZero(t *testing.T) {
	b := &world.Beliefs{Base: []float64{0.2, 0.8}, Known: []world.Flags{0b10101, 0b11111}, Cost: 0.15}
	if got := b.L(0); got != 0 {
		t.Errorf("L %v, want 0", got)
	}
	if got := b.L(1); !near(got, 0.05) {
		t.Errorf("L %v, want 0.05", got)
	}
}

// ODD §6.12, State: risk aversion is drawn once per person, in ID order,
// from the module's stream, and newborns get theirs at the next step.
func TestRiskDrawnOnceInIDOrder(t *testing.T) {
	s := newTiny(t)
	s.add(30, world.Worker)
	s.add(3, world.Child)
	s.start()
	s.w.People = append(s.w.People, world.Person{ID: 2, Alive: true, BirthDay: s.w.Day + 1, Household: 0,
		Mother: world.None, Father: world.None, Master: world.None, Partner: world.None, BabyFather: world.None})
	s.w.Alive = append(s.w.Alive, 2)
	s.step()
	s.step()
	r, _ := rng.New(1, "beliefs")
	for id, got := range s.w.Beliefs.Risk {
		if want := r.Float64(); got != want {
			t.Errorf("person %d: R %v, want %v", id, got, want)
		}
	}
	if len(s.w.Beliefs.Risk) != 3 {
		t.Errorf("%d risk values, want 3", len(s.w.Beliefs.Risk))
	}
}

func TestNeedsNetwork(t *testing.T) {
	s := newTiny(t)
	s.w.Network = nil
	if err := s.m.Init(s.w, &config.Scenario{}, s.r); err == nil {
		t.Error("started without the network")
	}
}
