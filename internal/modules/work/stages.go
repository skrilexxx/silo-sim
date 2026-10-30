package work

import (
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// ODD §6.5: life stages, daily. In the tick's shuffled order: a child
// turning shadowAge is placed as a shadow; a shadow turning workerAge
// becomes a worker in the same department; a worker retires at retireAge,
// and goes on sick leave after sickLeaveDays days in a row with health
// below sickLeaveHealth. Someone on sick leave returns to work in the same
// department once health is back at sickLeaveHealth, or retires at
// retireAge. Afterwards, every shadow whose master died, retired or left
// (sick leave included) has a new master found.
func (m *Module) lifeStages(w *world.World, r *rng.Stream) error {
	m.recount(w)
	for _, id := range w.Alive {
		p := &w.People[id]
		a := age(w, id)
		switch p.Stage {
		case world.Child:
			if a >= m.days(m.c.ShadowAge) {
				m.place(w, r, id)
			}
		case world.Shadow:
			if a >= m.days(m.c.WorkerAge) {
				m.unlink(w, id)
				p.Stage = world.Worker
				w.Depts[p.Department].Shadows--
				m.countIn(w, id)
			}
		case world.Worker:
			if p.Health < m.c.SickLeaveHealth {
				p.LowHealth++
			} else {
				p.LowHealth = 0
			}
			switch {
			case a >= m.days(m.c.RetireAge):
				p.Stage = world.Retired
				m.countOut(w, id)
			case p.LowHealth >= m.c.SickLeaveDays:
				p.Stage, p.LowHealth = world.SickLeave, 0
				m.countOut(w, id)
			}
		case world.SickLeave:
			switch {
			case a >= m.days(m.c.RetireAge):
				p.Stage = world.Retired
			case p.Health >= m.c.SickLeaveHealth:
				p.Stage = world.Worker
				m.countIn(w, id)
			}
		}
	}
	m.remaster(w, r)
	return nil
}

// countIn and countOut keep a department's headcount in step with recount
// when a person starts or stops being a worker: someone off work because
// of where they are (ODD §6.10) is not counted.
func (m *Module) countIn(w *world.World, id world.PersonID) {
	if !w.Space.Off(id) {
		w.Depts[w.People[id].Department].Headcount++
	}
}

func (m *Module) countOut(w *world.World, id world.PersonID) {
	if !w.Space.Off(id) {
		w.Depts[w.People[id].Department].Headcount--
	}
}

func (m *Module) unlink(w *world.World, id world.PersonID) {
	if mst := w.People[id].Master; mst != world.None {
		m.load[mst]--
		w.People[id].Master = world.None
	}
}

// remaster gives a new master to every shadow whose master is no longer a
// living worker of their department, or who has none: the free master with
// the fewest shadows, if there is one.
func (m *Module) remaster(w *world.World, r *rng.Stream) {
	for _, id := range w.Alive {
		p := &w.People[id]
		if p.Stage != world.Shadow {
			continue
		}
		if mst := p.Master; mst != world.None {
			q := &w.People[mst]
			if q.Alive && q.Stage == world.Worker && q.Department == p.Department {
				continue
			}
			m.load[mst]--
			p.Master = world.None
		}
		if mst := m.freeMaster(w, r, p.Department); mst != world.None {
			p.Master = mst
			m.load[mst]++
		}
	}
}

// ODD §6.5: shadow placement at shadowAge. With followParentChance the
// child follows the department of a parent picked at random from those who
// have one (alive or not), if that department has room: a free master.
// Otherwise, with crossZoneChance, they pick from departments tied to
// another zone, else from their own zone's departments plus the all-zone
// ones; departments without room are skipped, and the pick is weighted by
// the monthly shortfall (target shares if every shortfall is 0). If no
// department in reach has room, the child still shadows the department the
// same weights pick, with no master until one is free.
func (m *Module) place(w *world.World, r *rng.Stream, id world.PersonID) {
	p := &w.People[id]
	dept := world.None
	if r.Bernoulli(m.c.FollowParentChance) {
		if d := m.parentDept(w, r, p); d != world.None && m.freeMaster(w, nil, d) != world.None {
			dept = d
		}
	}
	if dept == world.None {
		zone := homeZone(w, id)
		cross := r.Bernoulli(m.c.CrossZoneChance)
		m.candidates(w, zone, cross)
		if len(m.cands) == 0 { // a config with no department in another zone
			m.candidates(w, zone, false)
		}
		dept = m.pick(w, r, zone, true)
		if dept == world.None {
			dept = m.pick(w, r, zone, false)
		}
	}

	p.Stage, p.Department = world.Shadow, dept
	w.Depts[dept].Shadows++
	if mst := m.freeMaster(w, r, dept); mst != world.None {
		p.Master = mst
		m.load[mst]++
	}
}

// candidates fills m.cands with the departments of the child's own zone
// plus the all-zone ones, or, if cross, those tied to another zone.
func (m *Module) candidates(w *world.World, zone world.Zone, cross bool) {
	m.cands = m.cands[:0]
	for d, dep := range w.Depts {
		own := dep.Zone == zone || dep.Zone == world.AllZones
		if own != cross {
			m.cands = append(m.cands, d)
		}
	}
}

// parentDept returns the department of a parent picked at random from
// those who have one, or None.
func (m *Module) parentDept(w *world.World, r *rng.Stream, p *world.Person) int {
	var ds [2]int
	n := 0
	for _, par := range []world.PersonID{p.Mother, p.Father} {
		if par != world.None && w.People[par].Department != world.None {
			ds[n] = w.People[par].Department
			n++
		}
	}
	if n == 0 {
		return world.None
	}
	return ds[r.IntRange(0, n-1)]
}

// pick chooses among m.cands by shortfall (if all are 0, by each
// department's share of jobs in the child's zone), only departments with a
// free master if needRoom. None if nothing fits.
func (m *Module) pick(w *world.World, r *rng.Stream, zone world.Zone, needRoom bool) int {
	m.ws = m.ws[:0]
	total := 0.0
	for _, d := range m.cands {
		x := 0.0
		if !needRoom || m.freeMaster(w, nil, d) != world.None {
			x = m.shortfall(w, d)
		}
		m.ws = append(m.ws, x)
		total += x
	}
	if total == 0 {
		for i, d := range m.cands {
			if !needRoom || m.freeMaster(w, nil, d) != world.None {
				m.ws[i] = m.zoneShare(d, zone)
			}
		}
	}
	i := r.WeightedIndex(m.ws)
	if i < 0 {
		return world.None
	}
	return m.cands[i]
}

// zoneShare is department d's share of the jobs in zone z: its target
// share, split by allZoneSplit if it works in every zone, as on day 1.
func (m *Module) zoneShare(d int, z world.Zone) float64 {
	share := m.c.Departments[d].Share
	if parseZone(m.c.Departments[d].Zone) != world.AllZones {
		return share
	}
	split := [world.NumZones]float64{m.c.AllZoneSplit.UpTop, m.c.AllZoneSplit.Mids, m.c.AllZoneSplit.DownDeep}
	return share * split[z]
}

// freeMaster returns a worker of department d with skill above
// masterMinSkill and fewer than maxShadowsPerMaster shadows, choosing among
// those with the fewest shadows at random. With r == nil it only reports
// whether one exists (returning the first found), drawing nothing.
func (m *Module) freeMaster(w *world.World, r *rng.Stream, d int) world.PersonID {
	best, found := m.c.MaxShadowsPerMaster, world.PersonID(world.None)
	m.people = m.people[:0]
	for _, id := range w.Alive {
		q := &w.People[id]
		if q.Stage != world.Worker || q.Department != d || q.Skill <= m.c.MasterMinSkill || m.load[id] >= m.c.MaxShadowsPerMaster {
			continue
		}
		if r == nil {
			return id
		}
		switch {
		case m.load[id] < best:
			best, found = m.load[id], id
			m.people = append(m.people[:0], id)
		case m.load[id] == best:
			m.people = append(m.people, id)
		}
	}
	if found == world.None {
		return world.None
	}
	return m.people[r.IntRange(0, len(m.people)-1)]
}
