package demography

import (
	"math"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// lotteriesPerYear spreads the gap to the target over a year of monthly
// draws (ODD §6.4).
const lotteriesPerYear = 12

// ODD §6.4: lottery permits.
//
//	N = clamp(round(D30 + (T − P − E) / 12), 0, Nmax),  E = pregnant + w × openWindows
//
// D30 is deaths since the last lottery, T the target population, P the
// people alive. round is half away from zero (math.Round).
func Permits(deaths30, alive, pregnant, openWindows int, lot config.Lottery) int {
	expected := float64(pregnant) + lot.OpenWindowWeight*float64(openWindows)
	gap := float64(lot.TargetPopulation-alive) - expected
	n := int(math.Round(float64(deaths30) + gap/lotteriesPerYear))
	return max(0, min(n, lot.MaxPermitsPerMonth))
}

// ODD §6.4: the lottery, every 30 days. N permits go to couples drawn
// uniformly from the eligible ones (all of them if fewer). A winner's
// implant is removed at once and her window runs for windowDays from
// tomorrow. Medical coverage is taken as 1 until health (T2.3) adds the
// removal queue.
func (m *Module) lottery(w *world.World, r *rng.Stream) error {
	pregnant, open := 0, 0
	m.pool = m.pool[:0]
	for _, id := range w.Alive {
		switch w.People[id].Implant {
		case world.Pregnant:
			pregnant++
		case world.WindowOpen:
			open++
		}
		if Eligible(w, id, w.Day, m.cfg.Lottery) {
			m.pool = append(m.pool, id)
		}
	}
	n := min(Permits(m.deaths30, len(w.Alive), pregnant, open, m.cfg.Lottery), len(m.pool))
	m.deaths30 = 0

	// Partial Fisher–Yates: the first n of the pool become a uniform
	// random sample without replacement.
	for i := range n {
		j := r.IntRange(i, len(m.pool)-1)
		m.pool[i], m.pool[j] = m.pool[j], m.pool[i]
		p := &w.People[m.pool[i]]
		p.Implant = world.WindowOpen
		p.WindowEnds = w.Day + world.Tick(m.cfg.Lottery.WindowDays)
	}
	return nil
}

// ODD §6.4: an eligible couple is a woman and a man; she is aged
// motherAgeMin–motherAgeMax in whole years, has fewer than maxChildren
// children, her last birth was at least minYoungestAgeYears ago, and she
// has no open window or pregnancy. Whether the couple has a home yet does
// not matter. The invariants re-check this independently for every open
// window.
func Eligible(w *world.World, id world.PersonID, day world.Tick, lot config.Lottery) bool {
	p := &w.People[id]
	if p.Sex != world.Female || p.Implant != world.ImplantIn || p.Partner == world.None {
		return false
	}
	if w.People[p.Partner].Sex != world.Male {
		return false
	}
	years := int(day-p.BirthDay) / engine.DaysPerYear
	if years < lot.MotherAgeMin || years > lot.MotherAgeMax || p.Children >= lot.MaxChildren {
		return false
	}
	return p.Children == 0 || day-p.LastBirth >= world.Tick(lot.MinYoungestAgeYears*engine.DaysPerYear)
}

// ODD §6.4: reproduction, daily. A woman with an open window conceives
// with conceptionChance; birth follows pregnancyDays later. A window that
// reaches its last day without a pregnancy closes and the implant goes
// back in.
func (m *Module) reproduction(w *world.World, r *rng.Stream) error {
	lot := m.cfg.Lottery
	for _, id := range w.Alive { // newborns appended below are not visited
		switch p := &w.People[id]; p.Implant {
		case world.WindowOpen:
			if r.Bernoulli(lot.ConceptionChance) {
				p.Implant = world.Pregnant
				p.DueDay = w.Day + world.Tick(lot.PregnancyDays)
				p.BabyFather = p.Partner
			} else if w.Day >= p.WindowEnds {
				p.Implant = world.ImplantIn
			}
		case world.Pregnant:
			if w.Day == p.DueDay {
				m.birth(w, r, id)
			}
		}
	}
	return nil
}

// birth adds the child to the mother's household (ODD §6.4). The father
// is the partner at conception, even if he has died since.
func (m *Module) birth(w *world.World, r *rng.Stream, mother world.PersonID) {
	sex := world.Male
	if r.Bernoulli(m.cfg.Lottery.GirlShare) {
		sex = world.Female
	}
	id := world.PersonID(len(w.People))
	mo := &w.People[mother]
	child := world.Person{
		ID: id, Sex: sex, BirthDay: w.Day, Alive: true, Household: mo.Household,
		Partner: world.None, Mother: mother, Father: mo.BabyFather,
		Department: world.None, Master: world.None, BabyFather: world.None, Health: 1,
	}
	mo.Implant, mo.BabyFather = world.ImplantIn, world.None
	mo.Children++
	mo.LastBirth = w.Day
	hh := &w.Households[mo.Household]
	hh.Members = append(hh.Members, id)
	w.People = append(w.People, child) // may move People: mo is not used after this
	w.Alive = append(w.Alive, id)
}
