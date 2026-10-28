// Package engine runs the tick loop: every module's steps, in the fixed
// order of schedule.go, once per simulated day.
package engine

import (
	"fmt"
	"slices"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// Module is one switchable part of the model. Modules share state only
// through the World, never through each other's internals.
type Module interface {
	Name() string       // also selects the module's random stream
	Requires() []string // modules that must also be on
	Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error
	Steps() []Step
}

// Step is one piece of a module's work at one phase of the schedule.
type Step struct {
	Phase Phase
	Run   func(w *world.World, r *rng.Stream) error
}

type boundStep struct {
	Step
	module string
	stream *rng.Stream
}

type Engine struct {
	w          *world.World
	steps      []boundStep // sorted by phase
	shuffle    *rng.Stream // the engine's own stream, for the order of Alive
	days       world.Tick
	warmupDays world.Tick
	logWarmup  bool
}

// New checks the module set, runs each module's Init in the given order,
// and collects the steps. Any problem with the setup fails here, before a
// single tick runs.
func New(cfg *config.Scenario, w *world.World, modules []Module) (*Engine, error) {
	shuffle, err := rng.New(cfg.Seed, "engine")
	if err != nil {
		return nil, err
	}
	e := &Engine{
		w:          w,
		shuffle:    shuffle,
		days:       world.Tick((cfg.WarmupYears + cfg.Years) * DaysPerYear),
		warmupDays: world.Tick(cfg.WarmupYears * DaysPerYear),
		logWarmup:  cfg.Output.LogWarmup,
	}

	var names []string
	for _, m := range modules {
		if slices.Contains(names, m.Name()) {
			return nil, fmt.Errorf("module %q listed twice", m.Name())
		}
		names = append(names, m.Name())
	}
	for _, m := range modules {
		for _, req := range m.Requires() {
			if !slices.Contains(names, req) {
				return nil, fmt.Errorf("module %q requires module %q, which is off", m.Name(), req)
			}
		}
	}

	// owner[p] is the module that registered phase p. One step per phase
	// keeps the order inside a tick fully defined by schedule.go.
	var owner [len(schedule)]string
	for _, m := range modules {
		stream, err := rng.New(cfg.Seed, m.Name())
		if err != nil {
			return nil, err
		}
		if err := m.Init(w, cfg, stream); err != nil {
			return nil, fmt.Errorf("init %s: %w", m.Name(), err)
		}
		for _, s := range m.Steps() {
			if !s.Phase.valid() {
				return nil, fmt.Errorf("module %q registered unknown phase %d", m.Name(), s.Phase)
			}
			if owner[s.Phase] != "" {
				return nil, fmt.Errorf("modules %q and %q both registered phase %d (%s)",
					owner[s.Phase], m.Name(), s.Phase, s.Phase)
			}
			owner[s.Phase] = m.Name()
			e.steps = append(e.steps, boundStep{Step: s, module: m.Name(), stream: stream})
		}
	}
	slices.SortFunc(e.steps, func(a, b boundStep) int { return int(a.Phase - b.Phase) })
	return e, nil
}

// Run plays every tick: warm-up, then the measured years.
func (e *Engine) Run() error {
	for e.w.Day < e.days {
		if err := e.Tick(); err != nil {
			return err
		}
	}
	return nil
}

// Tick advances one day and runs the steps due today.
func (e *Engine) Tick() error {
	w := e.w
	w.Day++
	// ODD §3: people are processed in a new random order every tick.
	e.shuffle.Shuffle(len(w.Alive), func(i, j int) { w.Alive[i], w.Alive[j] = w.Alive[j], w.Alive[i] })

	warmup := w.Day <= e.warmupDays
	for _, s := range e.steps {
		sched := schedule[s.Phase]
		if int(w.Day)%sched.every != 0 {
			continue
		}
		if sched.output && warmup && !e.logWarmup {
			continue
		}
		if err := s.Run(w, s.stream); err != nil {
			return fmt.Errorf("day %d, phase %d (%s), module %s: %w", w.Day, s.Phase, s.Phase, s.module, err)
		}
	}
	return nil
}
