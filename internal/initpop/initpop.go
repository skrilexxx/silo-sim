// Package initpop builds the day-1 silo from the scenario (ODD §5).
// Jobs, skill and stocks (§5 steps 5, 6, 8) are added by later tasks.
package initpop

import (
	"fmt"
	"math"
	"sort"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// builder holds the working state while the world is assembled.
type builder struct {
	cfg    *config.Scenario
	r      *rng.Stream
	w      *world.World
	zoneOf []world.Zone       // per person
	hhZone []world.Zone       // per household
	byZone [][]world.PersonID // people per zone, in ID order
}

// couple is a day-1 couple, with the children placed with them so far.
type couple struct {
	woman world.PersonID
	hh    world.HouseholdID
	kids  []world.PersonID
}

// Build returns the day-1 world. The day counter is 0; a person aged d
// days has BirthDay −d.
func Build(cfg *config.Scenario, r *rng.Stream) (*world.World, error) {
	// ODD §5 step 8, the part that exists so far: water and air start at 1.
	// Food, parts and the generator arrive with resources (T3.1, T3.2).
	b := &builder{cfg: cfg, r: r, w: &world.World{Silo: world.Silo{Mayor: world.None, WaterSupply: 1, AirQuality: 1, Intake: [3]float64{1, 1, 1}}}}
	b.w.Floors = floors(cfg.Init)
	b.people()
	b.zones()
	for z := range world.NumZones {
		if err := b.households(world.Zone(z)); err != nil {
			return nil, err
		}
	}
	b.homes()
	for i := range b.w.People {
		b.w.Alive = append(b.w.Alive, world.PersonID(i))
	}
	return b.w, nil
}

// ODD §5 step 1: floors. Each zone is a block of floorsPerZone floors, top
// to bottom: Up Top, Mids, Down Deep. Farm floors are spread evenly through
// the Mids, the lowest floors are Mechanical, and the rest are residential.
// Homes are added later, once the number of households is known.
func floors(in config.Init) []world.Floor {
	n := world.NumZones * in.FloorsPerZone
	fl := make([]world.Floor, n)
	for i := range fl {
		fl[i] = world.Floor{Number: i + 1, Zone: world.Zone(i / in.FloorsPerZone), Function: world.Residential}
	}
	for i := range in.MechanicalFloors {
		fl[n-1-i].Function = world.Mechanical
	}
	midsTop := int(world.Mids) * in.FloorsPerZone
	for i := range in.FarmFloors {
		fl[midsTop+i*in.FloorsPerZone/in.FarmFloors].Function = world.Farm
	}
	return fl
}

// ODD §5 step 2: people. Exactly half are women, chosen at random. Ages
// come from the steady-state age distribution of the mortality rule at
// health 1. Every woman starts with the implant in (step 7: the zero value).
func (b *builder) people() {
	n := b.cfg.Population
	sexes := make([]world.Sex, n)
	for i := n / 2; i < n; i++ {
		sexes[i] = world.Male
	}
	b.r.Shuffle(n, func(i, j int) { sexes[i], sexes[j] = sexes[j], sexes[i] })

	cum := stationaryCum(b.cfg.Mortality)
	total := cum[len(cum)-1]
	b.w.People = make([]world.Person, n)
	for i := range b.w.People {
		u := b.r.Float64() * total
		age := sort.Search(len(cum), func(d int) bool { return cum[d] > u })
		b.w.People[i] = world.Person{
			ID:         world.PersonID(i),
			Sex:        sexes[i],
			BirthDay:   world.Tick(-age),
			Alive:      true,
			Household:  world.None,
			Partner:    world.None,
			Mother:     world.None,
			Father:     world.None,
			Department: world.None,
			Master:     world.None,
			BabyFather: world.None,
			Health:     1,
		}
	}
}

// stationaryCum returns the running total of survival to each age in days.
// With a constant number of births, the number of people aged d is
// proportional to the chance of surviving d days, so this is the age
// distribution the mortality rule keeps in balance. The table ends where
// survival reaches exactly 0 in floating point (around age 150 at baseline
// values); gompertzA and gompertzB > 0 guarantee that it does.
func stationaryCum(m config.Mortality) []float64 {
	var cum []float64
	total, survival := 0.0, 1.0
	for d := 0; survival > 0; d++ {
		total += survival
		cum = append(cum, total)
		survival *= 1 - demography.DeathChance(float64(d)/engine.DaysPerYear, 1, m)
	}
	return cum
}

// ODD §5 step 3: zones. Each zone gets its share of people, rounded to whole
// people by largest remainder, assigned at random and so independent of
// age and sex.
func (b *builder) zones() {
	n := len(b.w.People)
	split := b.cfg.Init.ZoneSplit
	shares := [world.NumZones]float64{split.UpTop, split.Mids, split.DownDeep}
	var counts [world.NumZones]int
	given := 0
	for z, s := range shares {
		counts[z] = int(s * float64(n))
		given += counts[z]
	}
	for ; given < n; given++ {
		best := 0
		for z := range shares {
			if frac(shares[z], n, counts[z]) > frac(shares[best], n, counts[best]) {
				best = z
			}
		}
		counts[best]++
	}

	b.zoneOf = make([]world.Zone, 0, n)
	for z, c := range counts {
		for range c {
			b.zoneOf = append(b.zoneOf, world.Zone(z))
		}
	}
	b.r.Shuffle(n, func(i, j int) { b.zoneOf[i], b.zoneOf[j] = b.zoneOf[j], b.zoneOf[i] })
	b.byZone = make([][]world.PersonID, world.NumZones)
	for i, z := range b.zoneOf {
		b.byZone[z] = append(b.byZone[z], world.PersonID(i))
	}
}

// frac is how far a zone's count still falls short of its exact share.
func frac(share float64, n, count int) float64 { return share*float64(n) - float64(count) }

// ageDays is a person's age on day 0.
func (b *builder) ageDays(id world.PersonID) int { return int(-b.w.People[id].BirthDay) }

func (b *builder) years(y int) int { return y * engine.DaysPerYear }

// ODD §5 step 4: households in one zone. Couples form first, then each
// child joins a couple that could be its parents, then everyone else
// lives alone.
func (b *builder) households(z world.Zone) error {
	couples := b.pair(z)
	if err := b.placeChildren(z, couples); err != nil {
		return err
	}
	adult := b.years(b.cfg.Partnering.MinAge)
	for _, id := range b.byZone[z] {
		p := &b.w.People[id]
		if p.Household == world.None && b.ageDays(id) >= adult {
			b.newHousehold(z, id)
		}
	}
	return nil
}

// pair forms round(pairedShare × adults 20+ / 2) woman–man couples. Women
// are taken in random order; each picks an unpaired man with weight
// e^(−|age gap| / ageGapScaleYears), the age-gap rule of ODD §6.3.
func (b *builder) pair(z world.Zone) []couple {
	minAge := b.years(b.cfg.Init.PairedMinAge)
	var women, men []world.PersonID
	for _, id := range b.byZone[z] {
		if b.ageDays(id) < minAge {
			continue
		}
		if b.w.People[id].Sex == world.Female {
			women = append(women, id)
		} else {
			men = append(men, id)
		}
	}
	k := int(math.Round(b.cfg.Init.PairedShareOfAdults * float64(len(women)+len(men)) / 2))
	k = min(k, len(women), len(men))
	b.r.Shuffle(len(women), func(i, j int) { women[i], women[j] = women[j], women[i] })

	couples := make([]couple, 0, k)
	weights := make([]float64, 0, len(men))
	scale := b.cfg.Partnering.AgeGapScaleYears * engine.DaysPerYear
	for _, woman := range women[:k] {
		weights = weights[:0]
		for _, man := range men {
			gap := math.Abs(float64(b.ageDays(woman) - b.ageDays(man)))
			weights = append(weights, math.Exp(-gap/scale))
		}
		j := b.r.WeightedIndex(weights)
		man := men[j]
		men[j] = men[len(men)-1]
		men = men[:len(men)-1]

		b.w.People[woman].Partner = man
		b.w.People[man].Partner = woman
		hh := b.newHousehold(z, woman, man)
		couples = append(couples, couple{woman: woman, hh: hh})
	}
	return couples
}

// placeChildren gives each child under the adult age, in random order, to
// a random couple whose woman could have had them under the lottery rules
// (ODD §6.4): aged motherAgeMin–motherAgeMax at the birth, fewer than
// maxChildren children, and no other child closer in age than
// minYoungestAgeYears. If no couple fits, building fails rather than
// bending the rule.
func (b *builder) placeChildren(z world.Zone, couples []couple) error {
	adult := b.years(b.cfg.Partnering.MinAge)
	lot := b.cfg.Lottery
	var kids []world.PersonID
	for _, id := range b.byZone[z] {
		if b.ageDays(id) < adult {
			kids = append(kids, id)
		}
	}
	b.r.Shuffle(len(kids), func(i, j int) { kids[i], kids[j] = kids[j], kids[i] })

	fits := make([]int, 0, len(couples))
	for _, kid := range kids {
		fits = fits[:0]
		for i := range couples {
			if b.canParent(&couples[i], kid, lot) {
				fits = append(fits, i)
			}
		}
		if len(fits) == 0 {
			return fmt.Errorf("initpop: no couple in zone %d can be the parents of person %d (aged %d days)", z, kid, b.ageDays(kid))
		}
		c := &couples[fits[b.r.IntRange(0, len(fits)-1)]]
		mother := &b.w.People[c.woman]
		child := &b.w.People[kid]
		child.Mother, child.Father = c.woman, mother.Partner
		child.Household = c.hh
		mother.Children++
		if mother.Children == 1 || child.BirthDay > mother.LastBirth {
			mother.LastBirth = child.BirthDay
		}
		c.kids = append(c.kids, kid)
		hh := &b.w.Households[c.hh]
		hh.Members = append(hh.Members, kid)
	}
	return nil
}

func (b *builder) canParent(c *couple, kid world.PersonID, lot config.Lottery) bool {
	atBirth := b.ageDays(c.woman) - b.ageDays(kid)
	if atBirth < b.years(lot.MotherAgeMin) || atBirth > b.years(lot.MotherAgeMax) {
		return false
	}
	if b.w.People[c.woman].Children >= lot.MaxChildren {
		return false
	}
	gap := b.years(lot.MinYoungestAgeYears)
	for _, other := range c.kids {
		d := b.ageDays(other) - b.ageDays(kid)
		if d < gap && -d < gap {
			return false
		}
	}
	return true
}

func (b *builder) newHousehold(z world.Zone, members ...world.PersonID) world.HouseholdID {
	id := world.HouseholdID(len(b.w.Households))
	b.w.Households = append(b.w.Households, world.Household{ID: id, Floor: world.None, Members: members})
	b.hhZone = append(b.hhZone, z)
	for _, m := range members {
		b.w.People[m].Household = id
	}
	return id
}

// ODD §5 step 1, homes: each zone gets ⌈homeCapacityMargin × its
// households⌉ homes, spread as evenly as possible over its residential
// floors (higher floors take the remainder), and each household takes a
// random free home in its zone.
func (b *builder) homes() {
	for z := range world.NumZones {
		var hhs []world.HouseholdID
		for i, hz := range b.hhZone {
			if int(hz) == z {
				hhs = append(hhs, world.HouseholdID(i))
			}
		}
		var res []int // floor indexes
		for i, f := range b.w.Floors {
			if int(f.Zone) == z && f.Function == world.Residential {
				res = append(res, i)
			}
		}
		capacity := int(math.Ceil(b.cfg.Init.HomeCapacityMargin * float64(len(hhs))))
		var slots []int // one floor number per home
		for k, i := range res {
			homes := capacity / len(res)
			if k < capacity%len(res) {
				homes++
			}
			b.w.Floors[i].Homes = homes
			for range homes {
				slots = append(slots, b.w.Floors[i].Number)
			}
		}
		b.r.Shuffle(len(slots), func(i, j int) { slots[i], slots[j] = slots[j], slots[i] })
		for k, h := range hhs {
			b.w.Households[h].Floor = slots[k]
		}
	}
}
