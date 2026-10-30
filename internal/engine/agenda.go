package engine

import (
	"container/heap"
	"fmt"

	"github.com/skrilexxx/silo-sim/internal/world"
)

// Agenda holds actions due on a later day: a trip's arrival, the end of a
// sentence, the Safeguard's delay (docs/architecture.md, Agenda). It is
// not called "events": World.Events already means lines of events.jsonl.
//
// Entries fire just before their phase's step, ordered by day, then phase,
// then the order they were scheduled in, so ties are deterministic.
type Agenda struct {
	queue entries
	seq   int64
	day   world.Tick // the agenda's clock: what "the past" means to Schedule
	phase Phase
}

// AgendaUser is a module that schedules actions on later days. The engine
// hands it the agenda before its Init, so Init can schedule too.
type AgendaUser interface {
	UseAgenda(a *Agenda)
}

// Handle refers to a scheduled entry, so it can be cancelled.
type Handle struct{ e *entry }

type entry struct {
	day       world.Tick
	phase     Phase
	seq       int64
	run       func(w *world.World) error
	cancelled bool
}

// Schedule adds an action due on day, just before phase's step. It must be
// due after the phase that schedules it: scheduling into the past is an
// error, so nothing is silently moved to another day. run uses whatever it
// captured, normally the owning module's random stream.
func (a *Agenda) Schedule(day world.Tick, phase Phase, run func(w *world.World) error) (Handle, error) {
	if phase.slot() < 0 {
		return Handle{}, fmt.Errorf("agenda: unknown phase %d", phase)
	}
	if day < a.day || day == a.day && phase <= a.phase {
		return Handle{}, fmt.Errorf("agenda: day %d, phase %d (%s) is not after day %d, phase %d",
			day, phase, phase, a.day, a.phase)
	}
	e := &entry{day: day, phase: phase, seq: a.seq, run: run}
	a.seq++
	heap.Push(&a.queue, e)
	return Handle{e}, nil
}

// Cancel drops the entry; it is removed when it reaches the front.
// Cancelling twice, or after it fired, does nothing.
func (a *Agenda) Cancel(h Handle) {
	if h.e != nil {
		h.e.cancelled = true
	}
}

// fire runs every entry due at day and phase, in key order. The engine
// visits every phase every day, so nothing due earlier is left behind, and
// an entry cannot add another for the same day and phase (Schedule refuses
// it), so the loop ends.
func (a *Agenda) fire(w *world.World, day world.Tick, phase Phase) error {
	a.day, a.phase = day, phase
	for len(a.queue) > 0 {
		e := a.queue[0]
		if e.day != day || e.phase != phase {
			return nil
		}
		heap.Pop(&a.queue)
		if e.cancelled {
			continue
		}
		if err := e.run(w); err != nil {
			return err
		}
	}
	return nil
}

// entries is a min-heap by day, phase, then insertion order.
type entries []*entry

func (q entries) Len() int { return len(q) }
func (q entries) Less(i, j int) bool {
	a, b := q[i], q[j]
	if a.day != b.day {
		return a.day < b.day
	}
	if a.phase != b.phase {
		return a.phase < b.phase
	}
	return a.seq < b.seq
}
func (q entries) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *entries) Push(x any)   { *q = append(*q, x.(*entry)) }
func (q *entries) Pop() any {
	old := *q
	e := old[len(old)-1]
	old[len(old)-1] = nil
	*q = old[:len(old)-1]
	return e
}
