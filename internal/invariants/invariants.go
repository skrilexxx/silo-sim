// Package invariants holds the ODD §8 checks that must hold on every tick.
// A broken simulation still produces plausible numbers, so these are the
// main guard against silent bugs. More checks join as the state they
// cover gets built.
package invariants

import (
	"fmt"
	"slices"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// Check runs every invariant and returns the first violation. The ODD §8
// list maps onto them as follows:
//
//   - counts agree: countsAgree
//   - one household each, no floor over capacity: oneHousehold, homeCapacity
//   - partnerships symmetric and alive: partnerships
//   - no pregnancy without a window, windows only for the eligible: reproduction
//   - food conserved exactly: foodConserved
//   - every 0–1 variable within 0–1: unitRanges (stocks and outage days ≥ 0 too)
//   - nobody acts after the tick they died: every step acts on World.Alive,
//     which countsAgree keeps to the living; households (oneHousehold),
//     partners (partnerships) and masters (work) never point to the dead.
//     The Mayor may be dead until the next yearly step (ODD §6.8).
//   - same seed → identical files, different seeds → different files:
//     tested in cmd/silosim, not checked per tick.
//
// Part 2 (draft): every living person is on a real floor, and workers on
// their department's floors (space, ODD §6.10); weak ties are symmetric
// and between the living (network, ODD §6.11); base, L and R lie in 0–1
// (beliefs, ODD §6.12).
func Check(w *world.World, cfg *config.Scenario) error {
	for _, check := range []func(*world.World) error{countsAgree, oneHousehold, homeCapacity, partnerships, unitRanges, foodConserved} {
		if err := check(w); err != nil {
			return err
		}
	}
	if err := reproduction(w, cfg.Lottery); err != nil {
		return err
	}
	if err := work(w, cfg.Work); err != nil {
		return err
	}
	if err := space(w); err != nil {
		return err
	}
	if err := network(w, cfg.Network.WeakTiesPerKind); err != nil {
		return err
	}
	if err := beliefs(w, cfg.Beliefs.Relics); err != nil {
		return err
	}
	if err := control(w, len(cfg.Pact)); err != nil {
		return err
	}
	return rebellion(w)
}

// ODD §6.14 (draft): H and G lie in 0–1; everyone not calm has a
// grievance (so is an adult), is free and is not the Head of IT.
func rebellion(w *world.World) error {
	b := w.Rebellion
	if b == nil {
		return nil
	}
	for _, id := range w.Alive {
		if h, g := b.H[id], b.G[id]; h < 0 || h > 1 || g < 0 || g > 1 {
			return fmt.Errorf("day %d: person %d has H = %v, G = %v, outside 0–1", w.Day, id, h, g)
		}
		if b.Stage[id] != world.Calm && (b.G[id] == 0 || w.Control.Held(id) || w.Control.IsHead(id)) {
			return fmt.Errorf("day %d: person %d at stage %d has G = %v, held %v, Head %v", w.Day, id, b.Stage[id], b.G[id], w.Control.Held(id), w.Control.IsHead(id))
		}
	}
	return nil
}

// ODD §6.13 (draft): the Head of IT, if any, is alive, of IT (at any
// stage: the post is for life), free, knows all five flags and holds no
// Flamekeeper seat; the successor is alive; every open case is against a living
// person not in custody, with charges only under rules of the Pact; the
// people in custody are charged; everyone held is alive and holds no
// seat, and a detention has a last day.
func control(w *world.World, rules int) error {
	c := w.Control
	if c == nil {
		return nil
	}
	if h := c.Head; h != world.None {
		p := &w.People[h]
		all := world.Flags(1<<world.NumFlags - 1)
		if !p.Alive || w.Depts[p.Department].Name != "it" || c.Held(h) {
			return fmt.Errorf("day %d: Head of IT %d is not a free, living member of IT", w.Day, h)
		}
		if w.Beliefs.Known[h] != all || w.Beliefs.IsMember[h] {
			return fmt.Errorf("day %d: Head of IT %d knows %05b or holds a Flamekeeper seat", w.Day, h, w.Beliefs.Known[h])
		}
	}
	if c.Fear < 0 || c.Fear > 1 || w.Beliefs.Lift < 0 {
		return fmt.Errorf("day %d: fear %v, lift %v", w.Day, c.Fear, w.Beliefs.Lift)
	}
	for _, d := range c.Doctors {
		if p := &w.People[d]; !p.Alive || w.Depts[p.Department].Name != "medical" || p.Stage != world.Worker && p.Stage != world.SickLeave {
			return fmt.Errorf("day %d: implant doctor %d is not a living Medical worker", w.Day, d)
		}
	}
	if s := c.Successor; s != world.None && !w.People[s].Alive {
		return fmt.Errorf("day %d: successor %d is dead", w.Day, s)
	}
	charged := 0
	for i, ch := range c.Charges {
		id := world.PersonID(i)
		if ch>>rules != 0 {
			return fmt.Errorf("day %d: person %d charged under a rule beyond the Pact's %d", w.Day, id, rules)
		}
		if ch != 0 && c.Hold[id] != world.Custody {
			charged++
		}
		if c.Hold[id] == world.Free {
			continue
		}
		if !w.People[id].Alive || w.Beliefs.IsMember[id] {
			return fmt.Errorf("day %d: %d held by Judicial, but dead or a Flamekeeper", w.Day, id)
		}
		if c.Hold[id] == world.Custody && ch == 0 || c.Hold[id] != world.Custody && c.Until[id] == 0 {
			return fmt.Errorf("day %d: %d in custody with no charge, or detained with no last day", w.Day, id)
		}
	}
	if charged != len(c.Cases) {
		return fmt.Errorf("day %d: %d people charged and not in custody, %d open cases", w.Day, charged, len(c.Cases))
	}
	for _, id := range c.Cases {
		if c.Charges[id] == 0 || !w.People[id].Alive || c.Hold[id] == world.Custody {
			return fmt.Errorf("day %d: open case of %d, who is dead, in custody or charged with nothing", w.Day, id)
		}
	}
	return nil
}

// ODD §6.12 (draft): every living person's base, L and risk aversion lie
// in 0–1, they know only the five flags, never one both known and hidden,
// and what they lived they know or have hidden;
// all the relics are there, each on a real floor, held by nobody or by a
// living person; every Flamekeeper is alive and knows the history.
func beliefs(w *world.World, relics int) error {
	b := w.Beliefs
	if b == nil {
		return nil
	}
	taken := 0
	if w.Control != nil {
		taken = w.Control.Tally.Taken
	}
	if len(b.Relics)+taken != relics { // ODD §6.13: confiscated relics leave the silo
		return fmt.Errorf("%d relics and %d taken, want %d", len(b.Relics), taken, relics)
	}
	for _, id := range b.Members {
		if !w.People[id].Alive || !b.IsMember[id] || !b.Known[id].Has(world.FlagHistory) {
			return fmt.Errorf("Flamekeeper %d: alive %v, listed %v, knows %b; want alive and knowing the history",
				id, w.People[id].Alive, b.IsMember[id], b.Known[id])
		}
	}
	for i, r := range b.Relics {
		if r.Floor < 1 || r.Floor > len(w.Floors) {
			return fmt.Errorf("relic %d on floor %d", i, r.Floor)
		}
		if r.Holder != world.None && !w.People[r.Holder].Alive {
			return fmt.Errorf("relic %d held by person %d, who is dead", i, r.Holder)
		}
	}
	for _, id := range w.Alive {
		if x := b.Base[id]; x < 0 || x > 1 {
			return fmt.Errorf("person %d: base legitimacy %v outside 0–1", id, x)
		}
		if x := b.L(id); x < 0 || x > 1 {
			return fmt.Errorf("person %d: legitimacy %v outside 0–1", id, x)
		}
		if x := b.Risk[id]; x < 0 || x > 1 {
			return fmt.Errorf("person %d: risk aversion %v outside 0–1", id, x)
		}
		if k := b.Known[id]; k >= 1<<world.NumFlags {
			return fmt.Errorf("person %d: knows flags %b, beyond the %d facts", id, k, world.NumFlags)
		}
		if both := b.Known[id] & b.Hidden[id]; both != 0 || b.Hidden[id] >= 1<<world.NumFlags {
			return fmt.Errorf("person %d: flags %b both known and hidden, or hidden %b beyond the five", id, both, b.Hidden[id])
		}
		if extra := b.Lived[id] &^ (b.Known[id] | b.Hidden[id]); extra != 0 {
			return fmt.Errorf("person %d: flags %b lived but neither known nor hidden", id, extra)
		}
	}
	return nil
}

// ODD §6.11 (draft): every weak tie of a living person joins two different
// living people, appears once in each one's list with the same kind, and
// nobody holds more than k of a kind. Strong ties follow from the records,
// which the other checks cover.
func network(w *world.World, k int) error {
	n := w.Network
	if n == nil {
		return nil
	}
	for _, a := range w.Alive {
		var count [world.NumTieKinds]int
		for i, t := range n.Weak[a] {
			b := t.Other
			switch {
			case b == a:
				return fmt.Errorf("person %d has a %s tie with themselves", a, t.Kind)
			case !w.People[b].Alive:
				return fmt.Errorf("person %d has a %s tie with dead person %d", a, t.Kind, b)
			case !t.Kind.Weak():
				return fmt.Errorf("person %d has a stored %s tie, but only weak ties are stored", a, t.Kind)
			}
			for _, u := range n.Weak[a][:i] {
				if u.Other == b {
					return fmt.Errorf("person %d has two weak ties with %d", a, b)
				}
			}
			back := 0
			for _, u := range n.Weak[b] {
				if u.Other == a && u.Kind == t.Kind {
					back++
				}
			}
			if back != 1 {
				return fmt.Errorf("person %d has a %s tie with %d, who has it %d times", a, t.Kind, b, back)
			}
			if count[t.Kind]++; count[t.Kind] > k {
				return fmt.Errorf("person %d has more than %d %s ties", a, k, t.Kind)
			}
		}
	}
	return nil
}

// ODD §6.10 (draft), checked after the location step, the last of the
// tick: every living person is on a real floor, except travellers, who are
// on the stairs (location 0). A person away is on the floor they reached.
// Otherwise workers and shadows are on a floor of their department
// connected to home (General services at home, Porters anywhere connected
// to home), or stranded at home when no department floor is connected,
// and everyone else is at home. Off work = stranded, travelling or away.
func space(w *world.World) error {
	s := w.Space
	if s == nil {
		return nil
	}
	general, porters := world.None, world.None
	for i, d := range w.Depts {
		switch d.Name {
		case "general":
			general = i
		case "porters":
			porters = i
		}
	}
	for _, id := range w.Alive {
		if int(id) >= len(s.Location) {
			return fmt.Errorf("person %d has no location", id)
		}
		p, loc := &w.People[id], s.Location[id]
		home := w.Households[p.Household].Floor
		travelling, away := s.Dest[id] != 0, s.Away[id] != 0
		if s.OffWork[id] != (s.Stranded[id] || travelling || away) {
			return fmt.Errorf("person %d: off work %v, but stranded %v, travelling %v, away %v",
				id, s.OffWork[id], s.Stranded[id], travelling, away)
		}
		switch {
		case travelling && (loc != 0 || away || s.Stranded[id]):
			return fmt.Errorf("person %d is travelling but on floor %d (away %v, stranded %v)", id, loc, away, s.Stranded[id])
		case travelling:
			continue
		case loc < 1 || loc > len(w.Floors):
			return fmt.Errorf("person %d is on floor %d, not a real floor", id, loc)
		case away && (loc != s.Away[id] || s.Stranded[id]):
			return fmt.Errorf("person %d is away on floor %d but on floor %d", id, s.Away[id], loc)
		case away:
			continue
		}
		working := p.Stage == world.Worker || p.Stage == world.Shadow
		switch {
		case !working || p.Department == general:
			if loc != home || s.Stranded[id] {
				return fmt.Errorf("person %d is on floor %d (stranded %v), want home floor %d", id, loc, s.Stranded[id], home)
			}
		case p.Department == porters:
			if !s.Connected(home, loc) || s.Stranded[id] {
				return fmt.Errorf("porter %d is on floor %d, not connected to home floor %d", id, loc, home)
			}
		default:
			floors := s.DeptFloors[p.Department]
			reachable := slices.ContainsFunc(floors, func(f int) bool { return s.Connected(home, f) })
			ok := reachable && !s.Stranded[id] && slices.Contains(floors, loc) && s.Connected(home, loc)
			if s.Stranded[id] {
				ok = !reachable && loc == home
			}
			if !ok {
				return fmt.Errorf("%s worker %d is on floor %d (stranded %v), home floor %d, department floors %v",
					w.Depts[p.Department].Name, id, loc, s.Stranded[id], home, floors)
			}
		}
	}
	return nil
}

// ODD §8: people alive = sum over zones = sum over households.
func countsAgree(w *world.World) error {
	alive := 0
	for i := range w.People {
		if w.People[i].Alive {
			alive++
		}
	}

	// The Alive list must hold each living person exactly once.
	seen := make([]bool, len(w.People))
	for _, id := range w.Alive {
		if int(id) < 0 || int(id) >= len(w.People) {
			return fmt.Errorf("Alive holds unknown person %d", id)
		}
		if !w.People[id].Alive {
			return fmt.Errorf("Alive holds dead person %d", id)
		}
		if seen[id] {
			return fmt.Errorf("Alive holds person %d twice", id)
		}
		seen[id] = true
	}
	if len(w.Alive) != alive {
		return fmt.Errorf("%d people alive, but Alive lists %d", alive, len(w.Alive))
	}

	var inHouseholds int
	var inZone [3]int // UpTop, Mids, DownDeep
	for _, h := range w.Households {
		if len(h.Members) == 0 {
			continue // dissolved
		}
		if h.Floor < 1 || h.Floor > len(w.Floors) {
			return fmt.Errorf("household %d is on floor %d, which does not exist", h.ID, h.Floor)
		}
		z := w.Floors[h.Floor-1].Zone
		if z < world.UpTop || z > world.DownDeep {
			return fmt.Errorf("floor %d has no residential zone (%d)", h.Floor, z)
		}
		inHouseholds += len(h.Members)
		inZone[z] += len(h.Members)
	}
	if inHouseholds != alive {
		return fmt.Errorf("%d people alive, but households hold %d", alive, inHouseholds)
	}
	if sum := inZone[world.UpTop] + inZone[world.Mids] + inZone[world.DownDeep]; sum != alive {
		return fmt.Errorf("%d people alive, but zones hold %d", alive, sum)
	}
	return nil
}

// ODD §8: every living person is in exactly one household, the one their
// Household field names, and households hold only the living.
func oneHousehold(w *world.World) error {
	count := make([]int, len(w.People))
	for _, h := range w.Households {
		for _, id := range h.Members {
			if int(id) < 0 || int(id) >= len(w.People) || !w.People[id].Alive {
				return fmt.Errorf("household %d holds person %d, who is not alive", h.ID, id)
			}
			if w.People[id].Household != h.ID {
				return fmt.Errorf("household %d holds person %d, whose household is %d", h.ID, id, w.People[id].Household)
			}
			count[id]++
		}
	}
	for _, id := range w.Alive {
		if count[id] != 1 {
			return fmt.Errorf("person %d is in %d households", id, count[id])
		}
	}
	return nil
}

// ODD §8: no floor holds more households than it has homes.
func homeCapacity(w *world.World) error {
	used := make([]int, len(w.Floors))
	for _, h := range w.Households {
		if len(h.Members) > 0 {
			used[h.Floor-1]++ // floor validity is checked by countsAgree
		}
	}
	for i, f := range w.Floors {
		if used[i] > f.Homes {
			return fmt.Errorf("floor %d holds %d households but has %d homes", f.Number, used[i], f.Homes)
		}
	}
	return nil
}

// ODD §8: partnerships are symmetric and both partners are alive.
func partnerships(w *world.World) error {
	for _, id := range w.Alive {
		q := w.People[id].Partner
		if q == world.None {
			continue
		}
		if q == id {
			return fmt.Errorf("person %d is their own partner", id)
		}
		if int(q) < 0 || int(q) >= len(w.People) || !w.People[q].Alive {
			return fmt.Errorf("person %d's partner %d is not alive", id, q)
		}
		if w.People[q].Partner != id {
			return fmt.Errorf("person %d's partner %d has partner %d", id, q, w.People[q].Partner)
		}
	}
	return nil
}

// ODD §8: no pregnancy without an open window; windows only for eligible
// couples. A window's start is WindowEnds − windowDays (the day of the
// draw), and conception is DueDay − pregnancyDays. Eligibility is
// re-implemented here on purpose, independent of the lottery's own code.
func reproduction(w *world.World, lot config.Lottery) error {
	year := world.Tick(365) // days per year; engine.DaysPerYear would be an import cycle
	for _, id := range w.Alive {
		p := &w.People[id]
		if p.Implant == world.ImplantIn {
			continue
		}
		if p.Sex != world.Female {
			return fmt.Errorf("person %d is male but has implant state %d", id, p.Implant)
		}
		start := p.WindowEnds - world.Tick(lot.WindowDays)
		eligible := func() error {
			years := int((p.PermitDay - p.BirthDay) / year)
			spaced := p.Children == 0 || p.PermitDay-p.LastBirth >= world.Tick(lot.MinYoungestAgeYears)*year
			if years < lot.MotherAgeMin || years > lot.MotherAgeMax || p.Children >= lot.MaxChildren || !spaced {
				return fmt.Errorf("person %d won a permit on day %d while not eligible (aged %d, %d children)", id, p.PermitDay, years, p.Children)
			}
			if p.Partner == world.None || w.People[p.Partner].Sex != world.Male {
				return fmt.Errorf("person %d has a permit without a male partner", id)
			}
			return nil
		}
		switch p.Implant {
		case world.Queued:
			if p.PermitDay > w.Day {
				return fmt.Errorf("person %d: permit day %d is after today %d", id, p.PermitDay, w.Day)
			}
			if err := eligible(); err != nil {
				return err
			}
		case world.WindowOpen:
			if w.Day < start || w.Day >= p.WindowEnds || start < p.PermitDay {
				return fmt.Errorf("person %d: window %d–%d (permit day %d) is open on day %d", id, start, p.WindowEnds, p.PermitDay, w.Day)
			}
			if err := eligible(); err != nil {
				return err
			}
		case world.Pregnant:
			conceived := p.DueDay - world.Tick(lot.PregnancyDays)
			if conceived <= start || conceived > p.WindowEnds || conceived > w.Day || p.DueDay <= w.Day {
				return fmt.Errorf("person %d: pregnancy conceived day %d, due %d, outside window %d–%d (today %d)",
					id, conceived, p.DueDay, start, p.WindowEnds, w.Day)
			}
		}
	}
	return nil
}

// ODD §6.5: workers and shadows have a department; a shadow's master is a
// living worker of the same department with at most maxShadowsPerMaster
// shadows. Children have no department yet.
func work(w *world.World, c config.Work) error {
	load := make([]int, len(w.People))
	for _, id := range w.Alive {
		p := &w.People[id]
		switch p.Stage {
		case world.Child:
			if p.Department != world.None {
				return fmt.Errorf("child %d has department %d", id, p.Department)
			}
		case world.Shadow, world.Worker, world.SickLeave:
			if p.Department < 0 || p.Department >= len(w.Depts) {
				return fmt.Errorf("person %d (stage %d) has no valid department (%d)", id, p.Stage, p.Department)
			}
		}
		if p.Stage != world.Shadow && p.Master != world.None {
			return fmt.Errorf("person %d is not a shadow but has master %d", id, p.Master)
		}
		if p.Stage == world.Shadow && p.Master != world.None {
			q := &w.People[p.Master]
			if !q.Alive || q.Stage != world.Worker || q.Department != p.Department {
				return fmt.Errorf("shadow %d's master %d is not a living worker of department %d", id, p.Master, p.Department)
			}
			load[p.Master]++
			if load[p.Master] > c.MaxShadowsPerMaster {
				return fmt.Errorf("master %d has %d shadows, more than %d", p.Master, load[p.Master], c.MaxShadowsPerMaster)
			}
		}
	}
	return nil
}

// ODD §8: every 0–1 variable stays within 0–1. Written as !(0 ≤ x ≤ 1) so
// NaN fails too.
func unitRanges(w *world.World) error {
	for _, id := range w.Alive {
		p := &w.People[id]
		if !in01(p.Health) {
			return fmt.Errorf("person %d health = %v, outside 0–1", id, p.Health)
		}
		if !in01(p.Skill) {
			return fmt.Errorf("person %d skill = %v, outside 0–1", id, p.Skill)
		}
	}
	for _, d := range w.Depts {
		if !in01(d.Coverage) {
			return fmt.Errorf("department %s coverage = %v, outside 0–1", d.Name, d.Coverage)
		}
	}
	s := &w.Silo
	for _, v := range []struct {
		name string
		x    float64
	}{
		{"generator condition", s.GeneratorCondition},
		{"power factor", s.PowerFactor()},
		{"water supply", s.WaterSupply},
		{"air quality", s.AirQuality},
	} {
		if !in01(v.x) {
			return fmt.Errorf("silo %s = %v, outside 0–1", v.name, v.x)
		}
	}
	for _, v := range []struct {
		name string
		x    float64
	}{
		{"food stock", s.FoodStock},
		{"parts stock", s.PartsStock},
		{"generator down days", float64(s.GeneratorDown)},
		{"power supply", s.PowerSupply},
	} {
		if !(v.x >= 0) {
			return fmt.Errorf("silo %s = %v, below 0", v.name, v.x)
		}
	}
	for z, x := range s.Intake {
		if !(x >= 0) {
			return fmt.Errorf("zone %d intake/need = %v, below 0", z, x)
		}
	}
	return nil
}

func in01(x float64) bool { return x >= 0 && x <= 1 }

// ODD §8: food is conserved exactly: today's stock = the opening stock +
// produced − handed out. Compared with ==, in the order the steps apply
// them, so any other change to the stock shows.
func foodConserved(w *world.World) error {
	s := &w.Silo
	if want := s.FoodOpening + s.FoodProduced - s.FoodHandedOut; s.FoodStock != want {
		return fmt.Errorf("food stock %v, but opening %v + produced %v − handed out %v = %v",
			s.FoodStock, s.FoodOpening, s.FoodProduced, s.FoodHandedOut, want)
	}
	return nil
}
