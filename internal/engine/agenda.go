package engine

import (
	"container/heap"
	"fmt"

	"github.com/skrilexxx/silo-sim/internal/rng"
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

// EventType is a kind of scripted event (ODD §5) that a module defines:
// on the event's day, Fire runs just before Phase's step, with the
// engine's events stream, so its draws shift no module's.
type EventType struct {
	Name  string
	Phase Phase
	Fire  func(w *world.World, r *rng.Stream) error
}

// EventSource is a module that defines scripted event types.
type EventSource interface {
	EventTypes() []EventType
}

// scheduleEvents puts every scripted event of the scenario on the agenda,
// in file order, so events on the same day and phase fire in that order.
// An event whose type no module on defines stops the run before it
// starts: a misspelt or switched-off shock must not pass silently.
func (e *Engine) scheduleEvents(modules []Module) error {
	types := map[string]EventType{} // looked up by name only, never ranged over
	owner := map[string]string{}
	for _, m := range modules {
		src, ok := m.(EventSource)
		if !ok {
			continue
		}
		for _, t := range src.EventTypes() {
			if t.Phase.slot() < 0 {
				return fmt.Errorf("module %q defines event type %q at unknown phase %d", m.Name(), t.Name, t.Phase)
			}
			if o, dup := owner[t.Name]; dup {
				return fmt.Errorf("modules %q and %q both define event type %q", o, m.Name(), t.Name)
			}
			types[t.Name], owner[t.Name] = t, m.Name()
		}
	}
	stream, err := rng.New(e.cfg.Seed, "events")
	if err != nil {
		return err
	}
	for i, ev := range e.cfg.Events {
		t, ok := types[ev.Type]
		if !ok {
			return fmt.Errorf("events[%d]: no module that is on defines event type %q", i, ev.Type)
		}
		_, err := e.agenda.Schedule(world.Tick(ev.Day), t.Phase, func(w *world.World) error {
			w.Events = append(w.Events, world.ScriptedEvent{
				EventHeader: world.EventHeader{Day: w.Day, Type: "scripted"}, Event: t.Name})
			if err := t.Fire(w, stream); err != nil {
				return fmt.Errorf("event %q: %w", t.Name, err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("events[%d]: %w", i, err)
		}
	}
	return nil
}
