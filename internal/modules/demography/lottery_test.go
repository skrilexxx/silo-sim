package demography

import (
	"math"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/invariants"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// Hand-computed cases of the ODD §6.4 permit formula, with baseline
// T = 10,000, Nmax = 40, w = 0.8.
func TestPermits(t *testing.T) {
	lot := config.Lottery{TargetPopulation: 10000, MaxPermitsPerMonth: 40, OpenWindowWeight: 0.8}
	cases := []struct {
		what                         string
		deaths, alive, preg, open, n int
	}{
		// 8 + (10000 − 9900 − (20 + 0.8·10)) / 12 = 8 + 72/12 = 14
		{"below target", 8, 9900, 20, 10, 14},
		// 5 + (−100 − 50) / 12 = −7.5 → −8 → 0
		{"above target", 5, 10100, 50, 0, 0},
		// 10 + 1000 / 12 = 93.3 → 40
		{"big deficit capped", 10, 9000, 0, 0, 40},
		// 0 + 6 / 12 = 0.5 → 1 (half away from zero)
		{"half rounds up", 0, 9994, 0, 0, 1},
		// 3 + (0 − 0.8·30) / 12 = 3 − 2 = 1
		{"open windows count at 0.8", 3, 10000, 0, 30, 1},
	}
	for _, c := range cases {
		if got := Permits(c.deaths, c.alive, c.preg, c.open, lot); got != c.n {
			t.Errorf("%s: Permits = %d, want %d", c.what, got, c.n)
		}
	}
}

// couples builds n woman–man couples, each in its own household on one
// big Mids floor. Women are aged womanAge, men 30.
func couples(t *testing.T, n, womanAge int) (*world.World, *Module) {
	t.Helper()
	w := &world.World{Day: 0, Floors: []world.Floor{{Number: 1, Zone: world.Mids, Homes: n}}}
	add := func(sex world.Sex, age int, hh world.HouseholdID) world.PersonID {
		id := world.PersonID(len(w.People))
		w.People = append(w.People, world.Person{ID: id, Sex: sex, Alive: true, BirthDay: world.Tick(-age * year),
			Household: hh, Partner: world.None, Mother: world.None, Father: world.None,
			Department: world.None, Master: world.None, BabyFather: world.None, Health: 1})
		w.Alive = append(w.Alive, id)
		return id
	}
	for i := range n {
		hh := world.HouseholdID(i)
		a, b := add(world.Female, womanAge, hh), add(world.Male, 30, hh)
		w.People[a].Partner, w.People[b].Partner = b, a
		w.Households = append(w.Households, world.Household{ID: hh, Floor: 1, Members: []world.PersonID{a, b}})
	}
	m := New()
	if err := m.Init(w, loadBaseline(t), nil); err != nil {
		t.Fatal(err)
	}
	return w, m
}

func TestEligibility(t *testing.T) {
	cases := []struct {
		what  string
		setup func(w *world.World) // on woman 0 of one couple, day 1000
		want  bool
	}{
		{"eligible", func(w *world.World) {}, true},
		{"aged 17", func(w *world.World) { w.People[0].BirthDay = 1000 - 17*year - 364 }, false},
		{"aged 18", func(w *world.World) { w.People[0].BirthDay = 1000 - 18*year }, true},
		{"aged 40 and 364 days", func(w *world.World) { w.People[0].BirthDay = 1000 - 40*year - 364 }, true},
		{"aged 41", func(w *world.World) { w.People[0].BirthDay = 1000 - 41*year }, false},
		{"3 children", func(w *world.World) { w.People[0].Children, w.People[0].LastBirth = 3, 0 }, false},
		{"last birth 1 year ago", func(w *world.World) { w.People[0].Children, w.People[0].LastBirth = 1, 1000-year }, false},
		{"last birth 2 years ago", func(w *world.World) { w.People[0].Children, w.People[0].LastBirth = 1, 1000-2*year }, true},
		{"window open", func(w *world.World) { w.People[0].Implant = world.WindowOpen }, false},
		{"pregnant", func(w *world.World) { w.People[0].Implant = world.Pregnant }, false},
		{"no partner", func(w *world.World) { w.People[0].Partner, w.People[1].Partner = world.None, world.None }, false},
		{"female partner", func(w *world.World) { w.People[1].Sex = world.Female }, false},
		{"a man", func(w *world.World) { w.People[0].Sex = world.Male }, false},
	}
	for _, c := range cases {
		w, m := couples(t, 1, 25)
		w.Day = 1000
		w.People[0].BirthDay = 1000 - 25*year
		c.setup(w)
		if got := Eligible(w, 0, w.Day, m.cfg.Lottery); got != c.want {
			t.Errorf("%s: Eligible = %v, want %v", c.what, got, c.want)
		}
	}
}

// With 1 permit and 3 eligible couples, each wins about a third of draws.
func TestWinnersDrawnUniformly(t *testing.T) {
	w, m := couples(t, 3, 25)
	m.cfg.Lottery.TargetPopulation = len(w.Alive) + 12 // (T − P) / 12 = 1 permit
	r := stream(t)
	wins := make([]int, 3)
	const trials = 30_000
	for range trials {
		if err := m.lottery(w, r); err != nil {
			t.Fatal(err)
		}
		for c := range 3 {
			if p := &w.People[2*c]; p.Implant == world.WindowOpen {
				wins[c]++
				p.Implant = world.ImplantIn
			}
		}
	}
	t.Logf("wins per couple over %d draws of 1 permit: %v", trials, wins)
	for c, n := range wins {
		if math.Abs(float64(n)-trials/3) > 0.05*trials/3 {
			t.Errorf("couple %d won %d times, want about %d", c, n, trials/3)
		}
	}
}

func TestPermitsBeyondEligibleAllWin(t *testing.T) {
	w, m := couples(t, 5, 25)
	w.Day = 30
	if err := m.lottery(w, stream(t)); err != nil {
		t.Fatal(err)
	}
	for c := range 5 {
		p := &w.People[2*c]
		if p.Implant != world.WindowOpen || p.WindowEnds != 30+365 {
			t.Errorf("woman %d: implant %d, window ends %d; want open until day 395", 2*c, p.Implant, p.WindowEnds)
		}
	}
	if err := invariants.Check(w, m.cfg); err != nil {
		t.Error(err)
	}
}

// openAll gives every woman a window drawn on day 0.
func openAll(w *world.World, m *Module) {
	for i := range w.People {
		if p := &w.People[i]; p.Sex == world.Female {
			p.Implant, p.WindowEnds, p.PermitDay = world.WindowOpen, world.Tick(m.cfg.Lottery.WindowDays), 0
		}
	}
}

// days runs the reproduction step for n days, checking invariants daily.
func days(t *testing.T, w *world.World, m *Module, n int) {
	t.Helper()
	r := stream(t)
	for range n {
		w.Day++
		if err := m.reproduction(w, r); err != nil {
			t.Fatal(err)
		}
		if err := invariants.Check(w, m.cfg); err != nil {
			t.Fatalf("day %d: %v", w.Day, err)
		}
	}
}

// ODD §6.4 done-when: over 10,000 windows, 78–82% end in a pregnancy
// (daily chance 0.0044 for 365 days: 1 − 0.9956^365 = 80.0%).
func TestWindowsEndingInPregnancy(t *testing.T) {
	const n = 10_000
	w, m := couples(t, n, 25)
	openAll(w, m)
	days(t, w, m, m.cfg.Lottery.WindowDays+1)
	conceived := 0
	for i := range 2 * n {
		p := &w.People[i]
		if p.Sex == world.Female && (p.Children > 0 || p.Implant == world.Pregnant) {
			conceived++
		}
		if p.Implant == world.WindowOpen {
			t.Fatalf("woman %d still has a window after it ended", i)
		}
	}
	share := float64(conceived) / n
	t.Logf("%d of %d windows ended in pregnancy: %.2f%% (expected %.2f%%)", conceived, n, 100*share,
		100*(1-math.Pow(1-m.cfg.Lottery.ConceptionChance, float64(m.cfg.Lottery.WindowDays))))
	if share < 0.78 || share > 0.82 {
		t.Errorf("share %.4f, want 0.78–0.82", share)
	}
}

func TestWindowClosesOnLastDay(t *testing.T) {
	w, m := couples(t, 1, 25)
	m.cfg.Lottery.ConceptionChance = 0
	openAll(w, m)
	days(t, w, m, 364)
	if w.People[0].Implant != world.WindowOpen {
		t.Fatal("window closed before its last day")
	}
	days(t, w, m, 1)
	if w.People[0].Implant != world.ImplantIn {
		t.Error("window still open after its last day")
	}
}

func TestConceptionAndBirth(t *testing.T) {
	w, m := couples(t, 1, 25)
	m.cfg.Lottery.ConceptionChance = 1
	openAll(w, m)
	days(t, w, m, 1)
	mo := w.People[0]
	if mo.Implant != world.Pregnant || mo.DueDay != 1+270 || mo.BabyFather != 1 {
		t.Fatalf("after conception on day 1: %+v", mo)
	}
	days(t, w, m, 269)
	if len(w.People) != 2 {
		t.Fatal("born before the due day")
	}
	days(t, w, m, 1)
	if len(w.People) != 3 {
		t.Fatal("no birth on the due day")
	}
	kid, mo := w.People[2], w.People[0]
	if kid.BirthDay != 271 || kid.Mother != 0 || kid.Father != 1 || kid.Household != 0 || !kid.Alive || kid.Health != 1 {
		t.Errorf("child: %+v", kid)
	}
	if mo.Implant != world.ImplantIn || mo.Children != 1 || mo.LastBirth != 271 {
		t.Errorf("mother after birth: %+v", mo)
	}
	if hh := w.Households[0].Members; len(hh) != 3 || hh[2] != 2 {
		t.Errorf("household %v, want the child added", hh)
	}
}

// A widow's open window closes; a pregnancy goes on, and the dead partner
// is still the father.
func TestPartnerDeathAndWindows(t *testing.T) {
	w, m := couples(t, 2, 25)
	openAll(w, m)
	// Woman 2 is pregnant: window drawn day −215, conceived day −170, due 100.
	w.Day = 50
	w.People[2].Implant, w.People[2].WindowEnds, w.People[2].PermitDay = world.Pregnant, 150, -215
	w.People[2].DueDay, w.People[2].BabyFather = 100, 3
	w.People[0].WindowEnds, w.People[0].PermitDay = 300, -65 // woman 0's window: drawn day −65
	for _, id := range []world.PersonID{1, 3} {
		if err := m.die(w, stream(t), id); err != nil {
			t.Fatal(err)
		}
	}
	w.Alive = []world.PersonID{0, 2}
	if w.People[0].Implant != world.ImplantIn {
		t.Error("widow's open window did not close")
	}
	if w.People[2].Implant != world.Pregnant {
		t.Error("widow's pregnancy ended")
	}
	days(t, w, m, 50)
	if len(w.People) != 5 || w.People[4].Father != 3 {
		t.Errorf("child of the dead father: %+v", w.People[len(w.People)-1])
	}
}

// withMedical gives the world a Medical department at the given coverage.
func withMedical(w *world.World, cov float64) {
	w.Depts = []world.Department{{Name: "medical", Coverage: cov}}
}

// T2.3 done-when: at Medical coverage 0.5 the queue delays windows. Ten
// winners: ⌊0.5 × 10⌋ = 5 get their implant out on the draw day; of the 5
// left, ⌊2.5⌋ = 2 next month, and so on. Each window starts on its
// removal day. At coverage 1 nobody waits.
func TestRemovalQueue(t *testing.T) {
	w, m := couples(t, 10, 25)
	withMedical(w, 0.5)
	w.Day = 30
	r := stream(t)
	opened := func() (n int) {
		for c := range 10 {
			if w.People[2*c].Implant == world.WindowOpen {
				n++
			}
		}
		return n
	}
	var perMonth []int
	for range 4 {
		before := opened()
		if err := m.lottery(w, r); err != nil {
			t.Fatal(err)
		}
		perMonth = append(perMonth, opened()-before)
		if err := invariants.Check(w, m.cfg); err != nil {
			t.Fatal(err)
		}
		w.Day += 30
	}
	t.Logf("windows opened per month at coverage 0.5: %v", perMonth)
	if perMonth[0] != 5 || perMonth[1] != 2 || perMonth[2] != 1 || perMonth[3] != 1 {
		t.Errorf("opened per month %v, want [5 2 1 1]", perMonth)
	}
	waiting := 0
	for c := range 10 {
		p := w.People[2*c]
		switch {
		case p.Implant == world.Queued:
			waiting++
		case p.PermitDay != 30 || p.WindowEnds < 30+365 || (p.WindowEnds-365)%30 != 0:
			t.Errorf("woman %d: permit day %d, window ends %d; want permit 30 and a window from her removal day", 2*c, p.PermitDay, p.WindowEnds)
		}
	}
	if waiting != 1 {
		t.Errorf("%d winners still waiting after 4 months, want 1", waiting)
	}

	w, m = couples(t, 10, 25)
	withMedical(w, 1)
	w.Day = 30
	m.lottery(w, r)
	if n := opened(); n != 10 {
		t.Errorf("at coverage 1, %d of 10 windows opened on the draw day", n)
	}
}

// Waiting winners count as open windows in E: with 10 waiting (E = 8) and
// a gap of 12 people, round((12 − 8) / 12) = 0 new permits, not 1.
func TestQueuedCountInE(t *testing.T) {
	w, m := couples(t, 11, 25)
	withMedical(w, 0)
	for c := range 10 {
		w.People[2*c].Implant, w.People[2*c].PermitDay = world.Queued, 0
	}
	m.cfg.Lottery.TargetPopulation = len(w.Alive) + 12
	w.Day = 30
	m.lottery(w, stream(t))
	if w.People[20].Implant != world.ImplantIn {
		t.Error("a permit was issued although the waiting winners cover the gap")
	}
}

func TestPartnerDeathCancelsQueuedPermit(t *testing.T) {
	w, m := couples(t, 1, 25)
	w.People[0].Implant = world.Queued
	if err := m.die(w, stream(t), 1); err != nil {
		t.Fatal(err)
	}
	if w.People[0].Implant != world.ImplantIn {
		t.Error("a widow kept her place in the removal queue")
	}
}

// ODD §6.2: maternal death risk is maternalDeathBase × (2 − coverage). With
// a base of 0.5 and no Medical the risk is 1: the mother dies at the birth
// and the child lives, staying with the father.
func TestMaternalDeath(t *testing.T) {
	w, m := couples(t, 1, 25)
	withMedical(w, 0)
	m.cfg.Health.MaternalDeathBase = 0.5
	w.People[0].Implant, w.People[0].DueDay, w.People[0].BabyFather = world.Pregnant, 1, 1
	w.People[0].WindowEnds = 100 // drawn day −265, conceived day −269: inside
	w.People[0].PermitDay = -265
	days(t, w, m, 1)
	if w.People[0].Alive || !w.People[2].Alive {
		t.Fatalf("mother alive %v, child alive %v; want the mother dead and the child alive", w.People[0].Alive, w.People[2].Alive)
	}
	if h := w.Households[0].Members; len(h) != 2 || h[0] != 1 || h[1] != 2 {
		t.Errorf("household %v, want the father and the child", h)
	}

	// At full coverage and the baseline base the risk is 0.002: 1,000
	// births give about 2 deaths.
	w, m = couples(t, 1000, 25)
	withMedical(w, 1)
	for c := range 1000 {
		p := &w.People[2*c]
		p.Implant, p.DueDay, p.BabyFather, p.WindowEnds, p.PermitDay = world.Pregnant, 1, world.PersonID(2*c+1), 100, -265
	}
	days(t, w, m, 1)
	dead := 0
	for c := range 1000 {
		if !w.People[2*c].Alive {
			dead++
		}
	}
	t.Logf("maternal deaths in 1,000 births at full coverage: %d (expected %.0f)", dead, 1000*m.cfg.Health.MaternalDeathBase)
	if dead > 10 {
		t.Errorf("%d maternal deaths in 1,000 births at risk 0.002", dead)
	}
}
