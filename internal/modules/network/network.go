// Package network keeps the social network (ODD §6.11, draft): strong
// ties read from the records, weak ties drawn and redrawn.
package network

import (
	"fmt"
	"slices"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

const numWeak = int(world.NumTieKinds - world.TieCoworker)

// key is what a person's weak ties depend on. A day on which nobody's key
// changes leaves every weak tie valid and every draw as fruitless as the
// day before, so the step only works on what changed (T9.1: 1 ms a day
// for 10,000 people when it rechecked everything).
type key struct {
	alive, porter   bool
	floor, work     int32 // neighbour bucket (home floor − 1, if 14+) and coworker bucket (department × floors + work floor − 1); −1 for none
	master, partner world.PersonID
	household       world.HouseholdID
}

type Module struct {
	k                      int
	neighbourAge, adultAge world.Tick // ages in days: neighbours, and the adults porters tie to
	porters, floors        int        // Porters' department index (or None); number of floors

	keys    []key
	cnt     [][numWeak]uint8 // weak ties held, by kind
	members []world.PersonID // alive at the last step, by ID
	known   int              // people already in the Children index

	byFloor, byWork     [][]world.PersonID // bucket members (everyone in it, short of ties or not)
	dirtyFloor          []bool             // buckets whose draws may succeed today
	dirtyWork           []bool
	dirtyF, dirtyW      []int
	porterDraw          []world.PersonID // porters who may need porter ties today
	changed             []world.PersonID
	drawers, cand, keep []world.PersonID
	stamp               []int // last update a person was listed as a drawer
	update              int
	ties                []world.Tie

	touched []world.PersonID // whose ties may have changed since the last step: changed keys, both ends of a weak tie drawn or ended
}

func New() *Module { return &Module{porters: world.None} }

func (m *Module) Name() string { return "network" }

// Requires space: coworkers share a work floor (ODD §6.10).
func (m *Module) Requires() []string { return []string{"space"} }

// ODD §6.11: on day 1 everyone draws their weak ties, in random order.
func (m *Module) Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error {
	if w.Space == nil {
		return fmt.Errorf("network: no space state; list space before network in modules")
	}
	m.k = cfg.Network.WeakTiesPerKind
	m.neighbourAge = world.Tick(cfg.Work.ShadowAge * engine.DaysPerYear)
	m.adultAge = world.Tick(cfg.Work.WorkerAge * engine.DaysPerYear)
	for d, dep := range w.Depts {
		if dep.Name == "porters" {
			m.porters = d
		}
	}
	m.floors = len(w.Floors)
	m.byFloor, m.dirtyFloor = make([][]world.PersonID, m.floors), make([]bool, m.floors)
	m.byWork, m.dirtyWork = make([][]world.PersonID, len(w.Depts)*m.floors), make([]bool, len(w.Depts)*m.floors)
	talks, err := rng.NewKeyed(cfg.Seed, "talk")
	if err != nil {
		return err
	}
	n := &world.Network{Talks: talks}
	for kind := range world.NumTieKinds {
		n.Chance[kind] = cfg.Network.StrongTieDailyChance
		if kind.Weak() {
			n.Chance[kind] *= cfg.Network.WeakTieStrength
		}
	}
	w.Network = n
	m.step(w, r)
	return nil
}

func (m *Module) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseNetwork, Run: m.step}}
}

// ODD §6.11 / §3 step 10.5: weak ties ended and drawn again; the ties
// cache and the connected runs of floors brought up to date, for the
// conversations other modules ask about today.
func (m *Module) step(w *world.World, r *rng.Stream) error {
	m.run(w, r)
	m.connect(w)
	w.Network.Forget(w, m.touched, m.ties)
	m.touched = m.touched[:0]
	return nil
}

// connect labels each floor with the run of connected floors it is in: a
// broken segment starts a new run.
func (m *Module) connect(w *world.World) {
	n := w.Network
	n.Part = append(n.Part[:0], 0)
	for f := 1; f < m.floors; f++ {
		run := n.Part[f-1]
		if w.Space.Broken[f-1] {
			run++
		}
		n.Part = append(n.Part, run)
	}
}

func (m *Module) run(w *world.World, r *rng.Stream) {
	n := w.Network
	for len(n.Weak) < len(w.People) {
		n.Weak, n.Children, n.Shadows = append(n.Weak, nil), append(n.Children, nil), append(n.Shadows, nil)
		m.keys = append(m.keys, key{floor: -1, work: -1, master: world.None, partner: world.None})
		m.cnt, m.stamp = append(m.cnt, [numWeak]uint8{}), append(m.stamp, 0)
	}
	// Kin: newborns join their parents' children.
	for ; m.known < len(w.People); m.known++ {
		p := &w.People[m.known]
		for _, parent := range [2]world.PersonID{p.Mother, p.Father} {
			if parent != world.None {
				n.Children[parent] = append(n.Children[parent], p.ID)
			}
		}
	}
	m.update++
	m.scan(w)
	for _, id := range m.changed {
		m.endTies(w, id)
	}
	m.drawAll(w, r)
}

// scan compares everyone alive at the last step, and everyone born since,
// with their key, in ID order, and keeps the buckets, the shadows index
// and the members list in step.
func (m *Module) scan(w *world.World) {
	m.changed, m.keep = m.changed[:0], m.keep[:0]
	visit := func(id world.PersonID) {
		old := m.keys[id]
		now := m.keyOf(w, id, old)
		if now.alive {
			m.keep = append(m.keep, id)
		}
		if now == old {
			return
		}
		m.keys[id] = now
		m.changed = append(m.changed, id)
		m.touched = append(m.touched, id)
		m.move(m.byFloor, m.dirtyFloor, &m.dirtyF, id, old.floor, now.floor)
		m.move(m.byWork, m.dirtyWork, &m.dirtyW, id, old.work, now.work)
		if old.master != now.master {
			sh := w.Network.Shadows
			if old.master != world.None {
				sh[old.master] = slices.DeleteFunc(sh[old.master], func(s world.PersonID) bool { return s == id })
			}
			if now.master != world.None {
				sh[now.master] = append(sh[now.master], id)
			}
		}
		if now.porter {
			m.porterDraw = append(m.porterDraw, id)
		}
	}
	last := world.PersonID(-1)
	for _, id := range m.members {
		visit(id)
		last = id
	}
	for id := last + 1; int(id) < len(w.People); id++ {
		if w.People[id].Alive {
			visit(id)
		}
	}
	m.members, m.keep = m.keep, m.members
}

func (m *Module) keyOf(w *world.World, id world.PersonID, old key) key {
	p := &w.People[id]
	k := key{floor: -1, work: -1, master: world.None, partner: world.None}
	if !p.Alive {
		return k
	}
	k.alive, k.partner, k.household = true, p.Partner, p.Household
	home := w.Households[p.Household].Floor
	if w.Day-p.BirthDay >= m.neighbourAge {
		k.floor = int32(home - 1)
	}
	working := p.Stage == world.Worker || p.Stage == world.Shadow || p.Stage == world.SickLeave
	if f := w.Space.WorkFloor; int(id) < len(f) && f[id] != 0 {
		k.work = int32(p.Department*m.floors + f[id] - 1)
	} else if working && old.work >= 0 && int(old.work)/m.floors == p.Department {
		// ODD §6.11: cut off from every floor of their department by the
		// stairs, they keep the work floor they had, and its coworkers.
		k.work = old.work
	}
	k.porter = p.Department == m.porters && working
	if p.Stage == world.Shadow {
		k.master = p.Master
	}
	return k
}

// move takes a person out of one bucket and into another, and marks both
// for drawing.
func (m *Module) move(buckets [][]world.PersonID, dirty []bool, list *[]int, id world.PersonID, from, to int32) {
	if from == to {
		if to >= 0 {
			m.mark(dirty, list, to)
		}
		return
	}
	if from >= 0 {
		b := buckets[from]
		i := slices.Index(b, id)
		b[i] = b[len(b)-1]
		buckets[from] = b[:len(b)-1]
		m.mark(dirty, list, from)
	}
	if to >= 0 {
		buckets[to] = append(buckets[to], id)
		m.mark(dirty, list, to)
	}
}

func (m *Module) mark(dirty []bool, list *[]int, b int32) {
	if !dirty[b] {
		dirty[b] = true
		*list = append(*list, int(b))
	}
}

// ODD §6.11: a weak tie ends when either dies, or when the two no longer
// share a home floor (neighbours), a department and work floor
// (coworkers), or a porter (porter ties). Both sides drop it, and the
// other side's buckets may now draw.
func (m *Module) endTies(w *world.World, id world.PersonID) {
	n := w.Network
	kept := n.Weak[id][:0]
	for _, t := range n.Weak[id] {
		if m.holds(id, t) {
			kept = append(kept, t)
			continue
		}
		o := t.Other
		n.Weak[o] = slices.DeleteFunc(n.Weak[o], func(u world.Tie) bool { return u.Other == id })
		m.touched = append(m.touched, id, o)
		m.cnt[id][t.Kind-world.TieCoworker]--
		m.cnt[o][t.Kind-world.TieCoworker]--
		ko := m.keys[o]
		if ko.floor >= 0 {
			m.mark(m.dirtyFloor, &m.dirtyF, ko.floor)
		}
		if ko.work >= 0 {
			m.mark(m.dirtyWork, &m.dirtyW, ko.work)
		}
		if ko.porter {
			m.porterDraw = append(m.porterDraw, o)
		}
	}
	n.Weak[id] = kept
}

func (m *Module) holds(a world.PersonID, t world.Tie) bool {
	ka, kb := m.keys[a], m.keys[t.Other]
	if !ka.alive || !kb.alive {
		return false
	}
	switch t.Kind {
	case world.TieNeighbour:
		return ka.floor >= 0 && ka.floor == kb.floor
	case world.TieCoworker:
		return ka.work >= 0 && ka.work == kb.work
	default:
		return ka.porter || kb.porter
	}
}

// ODD §6.11: anyone short of k ties of a weak kind draws, in random order.
// Only people in a bucket that changed today can find someone new, so only
// they are listed; the rest would draw nothing.
func (m *Module) drawAll(w *world.World, r *rng.Stream) {
	m.drawers = m.drawers[:0]
	add := func(id world.PersonID) {
		if m.keys[id].alive && m.stamp[id] != m.update {
			m.stamp[id] = m.update
			m.drawers = append(m.drawers, id)
		}
	}
	for _, b := range m.dirtyF {
		for _, id := range m.byFloor[b] {
			if int(m.cnt[id][1]) < m.k {
				add(id)
			}
		}
		m.dirtyFloor[b] = false
	}
	for _, b := range m.dirtyW {
		for _, id := range m.byWork[b] {
			if int(m.cnt[id][0]) < m.k {
				add(id)
			}
		}
		m.dirtyWork[b] = false
	}
	for _, id := range m.porterDraw {
		if int(m.cnt[id][2]) < m.k {
			add(id)
		}
	}
	m.dirtyF, m.dirtyW, m.porterDraw = m.dirtyF[:0], m.dirtyW[:0], m.porterDraw[:0]
	slices.Sort(m.drawers)
	r.Shuffle(len(m.drawers), func(i, j int) { m.drawers[i], m.drawers[j] = m.drawers[j], m.drawers[i] })
	for _, id := range m.drawers {
		k := m.keys[id]
		if k.work >= 0 {
			m.fill(w, r, id, world.TieCoworker, m.byWork[k.work])
		}
		if k.floor >= 0 {
			m.fill(w, r, id, world.TieNeighbour, m.byFloor[k.floor])
		}
		if k.porter {
			m.fillPorter(w, r, id)
		}
	}
}

// fill draws, one at a time and uniformly, from the bucket's people also
// short of this kind and not yet tied to id, until id has k or nobody is
// left.
func (m *Module) fill(w *world.World, r *rng.Stream, id world.PersonID, kind world.TieKind, bucket []world.PersonID) {
	i := int(kind - world.TieCoworker)
	if int(m.cnt[id][i]) >= m.k {
		return
	}
	m.ties = w.Network.Ties(w, id, m.ties[:0])
	m.cand = m.cand[:0]
	for _, o := range bucket {
		if o != id && int(m.cnt[o][i]) < m.k && !tied(m.ties, o) {
			m.cand = append(m.cand, o)
		}
	}
	for int(m.cnt[id][i]) < m.k && len(m.cand) > 0 {
		j := r.IntRange(0, len(m.cand)-1)
		o := m.cand[j]
		m.cand[j] = m.cand[len(m.cand)-1]
		m.cand = m.cand[:len(m.cand)-1]
		m.tie(w, id, o, kind)
	}
}

// fillPorter draws porter ties: adults living in another zone, short of k
// porter ties, not yet tied to the porter. Drawing a living person at
// random until one is eligible is uniform over the eligible; after as many
// misses as there are people, the eligible are listed instead.
func (m *Module) fillPorter(w *world.World, r *rng.Stream, id world.PersonID) {
	const i = int(world.TiePorter - world.TieCoworker)
	zone := w.Floors[w.Households[w.People[id].Household].Floor-1].Zone
	ok := func(o world.PersonID) bool {
		q := &w.People[o]
		return o != id && w.Day-q.BirthDay >= m.adultAge && int(m.cnt[o][i]) < m.k &&
			w.Floors[w.Households[q.Household].Floor-1].Zone != zone && !tied(m.ties, o)
	}
	for int(m.cnt[id][i]) < m.k {
		m.ties = w.Network.Ties(w, id, m.ties[:0])
		found := world.PersonID(world.None)
		for range len(w.Alive) {
			if o := w.Alive[r.IntRange(0, len(w.Alive)-1)]; ok(o) {
				found = o
				break
			}
		}
		if found == world.None {
			m.cand = m.cand[:0]
			for _, o := range w.Alive {
				if ok(o) {
					m.cand = append(m.cand, o)
				}
			}
			if len(m.cand) == 0 {
				return
			}
			found = m.cand[r.IntRange(0, len(m.cand)-1)]
		}
		m.tie(w, id, found, world.TiePorter)
	}
}

func (m *Module) tie(w *world.World, a, b world.PersonID, kind world.TieKind) {
	n := w.Network
	n.Weak[a] = append(n.Weak[a], world.Tie{Other: b, Kind: kind})
	n.Weak[b] = append(n.Weak[b], world.Tie{Other: a, Kind: kind})
	m.cnt[a][kind-world.TieCoworker]++
	m.cnt[b][kind-world.TieCoworker]++
	m.touched = append(m.touched, a, b)
	m.ties = append(m.ties, world.Tie{Other: b, Kind: kind})
}

func tied(ties []world.Tie, o world.PersonID) bool {
	for _, t := range ties {
		if t.Other == o {
			return true
		}
	}
	return false
}
