# Architecture — silosim

How the ODD spec (`docs/odd-spec.md`) becomes Go code. The spec says *what* the model does; this file says *where* it lives and *how* the code is organised. If the two disagree, the spec wins and this file gets fixed.

## Goals that shape the design

1. **Reproducible.** Same scenario file + same seed → byte-identical output on the same machine.
2. **Modular.** The baseline is five modules. Later experiments (beliefs, dissent, cleaning, rebellion, Silo 1) are new modules switched on in a scenario file, without touching the core.
3. **Checkable.** Invariants and a golden test catch silent bugs, because a broken simulation still produces plausible numbers.
4. **Fast enough.** 250 simulated years (50 warm-up + 200 measured) of ~10,000 people in under a minute on a laptop, so 100-seed batches are practical. With every part-2 module on, 350 years (50 + 300) in under 4 minutes (5 at most, if the full model needs it).
5. **Boring dependencies.** Go standard library + `gopkg.in/yaml.v3`. Nothing else without asking.
6. **Baseline frozen.** With every part-2 module off, `scenarios/baseline.yaml` writes `daily.csv`, `yearly.csv` and `events.jsonl` byte-identical to tag `baseline-v1`. `run.json` is not compared: it holds the runtime and the resolved scenario, which gains part-2 sections.

## Project layout

```text
silosim/
├── CLAUDE.md                 # rules for Claude Code
├── README.md
├── Makefile
├── go.mod                    # module github.com/skrilexxx/silo-sim
├── cmd/silosim/main.go       # CLI: run, batch, report
├── internal/
│   ├── config/               # scenario loading, `extends`, strict validation
│   ├── rng/                  # seeded random streams (PCG), one per module
│   ├── world/                # state: Person, Household, Floor, Department, Silo; indexes
│   ├── engine/               # tick loop, schedule.go (THE step order), module registry, agenda.go
│   ├── initpop/              # day-1 generator (ODD §5)
│   ├── modules/
│   │   ├── demography/       # 6.1 mortality, 6.3 partnering + housing, 6.4 lottery + reproduction
│   │   ├── health/           # 6.2 health and Medical
│   │   ├── work/             # 6.5 life stages, shadowing, skill, output, retraining
│   │   ├── resources/        # 6.6 generator, power, water, air, food, parts; 6.7 rations
│   │   ├── governance/       # 6.8 Mayor
│   │   ├── space/            # M8: 6.10 location, trips, stair segments
│   │   ├── network/          # M9: 6.11 ties and daily contacts
│   │   ├── beliefs/          # M10: 6.12 legitimacy, knowledge, relics, Vitamin D+
│   │   ├── control/          # M11: 6.13 surveillance, Judicial, sentences, cleaning
│   │   ├── rebellion/        # M12: 6.14 grievance, activation, collective action, collapse
│   │   ├── regime/           # M13: 6.15 policy and response strategies
│   │   └── silo1/            # M14: 6.16 Silo 1
│   ├── invariants/           # ODD §8 invariant checks
│   ├── output/               # daily.csv, yearly.csv, events.jsonl, run.json, rebellions.jsonl, frames.csv.gz
│   └── report/               # calibration targets → pass/fail table; canon.go: C1–C4 over the batch (ODD §1.2)
├── scenarios/
│   ├── baseline.yaml         # the complete default parameter set (part-2 modules off)
│   ├── part2-off.yaml        # every part-2 module listed but off; must equal baseline (T7.5)
│   └── canon.yaml            # part 2 at the show's settings; calibration target (§1.2)
├── testdata/golden/          # golden-run hashes
├── docs/
│   ├── odd-spec.md           # source of truth for model behaviour
│   ├── lore-reference.md     # show canon behind the spec
│   ├── architecture.md       # this file
│   ├── tasks.md              # build plan, part 1 (baseline, done)
│   ├── tasks-part2.md        # build plan, part 2 (rebellion)
│   └── calibration-log.md    # calibration runs and the spec changes they caused
└── analysis/                 # optional Python plots; never imported by Go
```

`internal/` keeps everything private to this module. Only `cmd/silosim` is an entry point. The part-2 module folders are planned, one per milestone, and appear when their milestone starts.

## Core types (sketch, not final code)

```go
// world/world.go
type Tick int32                    // 1 tick = 1 day
type PersonID int32                // index into World.People; never reused

type World struct {
    Day        Tick
    People     []Person            // every person ever born; dead ones stay (Alive=false)
    Alive      []PersonID          // living people; rebuilt or patched on birth/death
    Households []Household
    Floors     [144]Floor
    Depts      []Department        // fixed order = ODD §6.5 table order
    Silo       SiloState           // food, generator, power, water, air, parts, mayor…
    Events     *output.EventLog
}
```

Rules for state:

- Person IDs are slice indexes. Lookups are O(1); there are no maps of people.
- Optional references (partner, mother, master) use `-1` for "none", not pointers. This keeps the state flat, copyable and easy to snapshot.
- Ages are derived from `BirthDay` and `Day`. Never store age.
- Every value in 0–1 (health, skill, condition, air, water) is clamped by one helper, `world.Clamp01`, and checked by the invariants.

### Part-2 state

Part-2 modules add state (legitimacy, knowledge flags, location, ties, sentences…) that other modules must read: the rebellion module reads legitimacy, the control module reads location. The rule "modules share state only through `World`" still holds, so that state lives on `World`, typed:

```go
// world/beliefs.go — the type lives in world, the beliefs module owns and writes it
type Beliefs struct {
    Legitimacy []float64           // indexed by PersonID
    Knows      [][numFlags]bool    // indexed by PersonID
}

// world/world.go
type World struct {
    ...
    Beliefs *Beliefs               // nil when the beliefs module is off
}

// a reader in another module
if w.Beliefs != nil { l := w.Beliefs.Legitimacy[id] }
```

- **One pointer field per part-2 module, nil when it is off.** The owning module creates it in `Init` and is the only writer; others read it and check for nil. Types are checked by the compiler; there is no lookup by name and no type assertion.
- **Per-person data are slices indexed by `PersonID`, never new fields on `Person`.** `Person` is read by every daily step and its layout is tuned for the cache (T5.4); the baseline's struct stays as frozen as its output.
- **Births:** at the start of each of its steps, the owning module extends its slices to `len(w.People)` and initialises the newcomers by its own rule (for example legitimacy from the parents). There are no birth hooks. Dead people keep their entries, as they keep theirs in `People`.
- **Deaths, moves, job changes:** a module that must react compares the world with what it last recorded (for example the household it stored for each person). If that gets expensive, a world-level change log (births, deaths, moves per tick) is added in T9.0, where the network first needs it, and not before.
- **Policy levers (M13):** modules copy their settings at `Init`. With the regime module on, `World.Policy` is the scenario in force (ODD §6.15): `config.Scenario.InForce` applies the running entries' levers and Pact rows. The regime module publishes a new pointer whenever an entry starts or lapses and never changes a published one, so a module that reads a lever re-reads it from `World.Policy` at the start of its step (control only when the pointer changed, as it rebuilds its rule index). Nil when the module is off, so the baseline reads exactly what it read before.
- **Reading the network (M9):** use `World.Network.TiesOf(w, id)` (cached as of the last network step; `Ties` rebuilds from scratch) and `World.Network.Talk(w, a, b, kind)` to ask whether a tie is used today. Iterate over the people who have something to pass on and ask about their ties; never draw conversations for the whole silo.

## Engine: modules, steps and the schedule

The ODD schedule (§3) interleaves submodels: generator → life support → food → rations → work → health → deaths → … A module therefore does not "run" as one block. It registers **steps**, and the engine runs all steps in the fixed order defined in one file.

```go
// engine/schedule.go — the ONLY place the order and frequency live. Phase = 100 × ODD §3 step number.
type Phase int
const (
    PhaseGenerator    Phase = 100
    PhaseLifeSupport  Phase = 200
    PhaseFood         Phase = 300
    PhaseRations      Phase = 400
    PhaseWork         Phase = 500
    PhaseHealth       Phase = 600
    PhaseDeaths       Phase = 700
    PhaseReproduction Phase = 800
    PhaseLifeStages   Phase = 900
    PhasePartnering   Phase = 1000
    PhaseLogDaily     Phase = 1100
    PhaseLottery      Phase = 1200 // every 30 days
    PhaseHousing      Phase = 1300 // every 30 days
    PhaseStaffing     Phase = 1400 // every 30 days
    PhaseSnapshot     Phase = 1500 // every 365 days
    PhaseGovernance   Phase = 1600 // every 365 days
)

// engine/module.go
type Module interface {
    Name() string
    Requires() []string                       // e.g. health requires work (Medical output)
    Init(w *world.World, cfg *config.Scenario, r *rng.Stream) error
    Steps() []Step
}

type Step struct {
    Phase Phase                                // how often it runs comes from the phase (schedule.go)
    Run   func(w *world.World, r *rng.Stream) error
}
```

**Spaced phases (part 2).** Phase numbers are 100 × the ODD §3 step, so a part-2 step fits between two baseline steps without renumbering them: a step after deaths and before reproduction is phase 750, and ODD §3 lists it as step 7.5. The schedule is a table sorted by phase (not an array indexed by it), still one step per phase, still the only place order and frequency live. The baseline phases keep their relative order, so the spacing alone does not move the golden hash (T7.3). Each new phase is added to ODD §3 in its milestone's spec task before any code uses it.

The engine:

1. Loads the scenario, builds the module list from `modules:`, checks `Requires`.
2. Runs `initpop`, then each module's `Init`.
3. Collects all steps, sorts them by `Phase` (two modules may not register the same phase), and loops over ticks. Days run 1…N; a phase that runs every 30 or 365 days runs on days that are multiples of that number. Frequency belongs to the phase, not the step, because ODD §3 fixes it per phase.
4. At the start of each tick, shuffles `Alive` with the engine's own stream (ODD §3: random order each tick).
5. Runs invariants every `output.invariantEvery` ticks (every tick in tests).
6. Skips the output phases (logging, snapshot) during warm-up unless `output.logWarmup: true`.
7. Ends the run after the tick in which a collapse (ODD §1.2) is recorded: the rebellion module sets one `World` field with the day and cause, and the engine stops unless `output.runAfterCollapse` is true. The output module writes the cause to `run.json`.

A step that is disabled because its module is off simply does not run. Modules must not reach into other modules' internals; they share state only through `world.World`. If a step needs a value from a module that is off (for example food intake when `resources` is off), the scenario is invalid and loading fails. The baseline always runs all five.

## Agenda: scheduled actions

Part 2 has actions that take days: a prison sentence (M11), a dosing period (M10), the Safeguard's delay (M14). They go on the **agenda**, a queue of actions due on a later day. (It is not called "events": `World.Events` already means lines for `events.jsonl`.)

Trips (M8) do not use it. They are state on `World.Space` (destination and arrival day per person), because other modules start them through `Space.StartTrip` and cannot reach the engine, and because the location step already visits everyone every evening and can move each trip on. Cancelling a trip needs nothing either: a dead traveller leaves `World.Alive`.

```go
// engine/agenda.go
type AgendaUser interface { UseAgenda(a *Agenda) }   // a module that schedules; given the agenda before Init

func (a *Agenda) Schedule(day world.Tick, phase Phase, run func(w *world.World) error) (Handle, error)
func (a *Agenda) Cancel(h Handle)
```

An entry is a day, a phase and a closure. The closure carries what the action needs (the person, the module's own random stream), so the engine needs no table of action kinds and no payload format.

- **Order:** a min-heap (`container/heap`) keyed by day, then phase, then insertion number. Two entries due at the same day and phase fire in the order they were scheduled, so ties are deterministic.
- **Firing:** before a phase's step runs, the engine fires every entry due today at that phase, in key order. Entries fire on their own day whether or not the phase's step is due that day, and a phase with no step still fires them. An entry must be due later than the phase that schedules it (a later day, or later the same day); scheduling into the past is an error, so nothing is silently moved to another day.
- **Cancellation:** `Schedule` returns a handle; cancelling marks the entry, and a marked entry is dropped when it reaches the top (lazy deletion), so cancelling is O(1).
- **Scripted events** (`events:` in the scenario, ODD §5) are agenda entries added when the engine starts, in file order. A module defines its types by implementing `EventSource` (`EventTypes() []EventType`: a name, a phase, a `Fire` function). The engine refuses a type that no running module defines, draws for every event from its own `events` stream, and logs each one to `events.jsonl` as `{"day":…,"type":"scripted","event":…}` before it fires.
- The agenda is engine state, not world state: modules schedule and cancel through it, never read it. A module that does not implement `AgendaUser` never sees it, which is every baseline module.

## Randomness

- `math/rand/v2` with `rand.NewPCG(seed, streamID)`.
- **One seed, one stream per module.** `streamID` is a fixed constant per module name (written in `rng/streams.go`, never computed from map order). The engine has its own stream for shuffling.
- New part-2 modules get stream IDs 9 and up, appended at the end of `streams.go`. Existing IDs never change.
- **Keyed draws** (`rng.Keyed`, T9.2): a number that depends only on the run seed, a stream name and the keys asked about (for conversations: day, lower person ID, higher person ID), not on how often or in what order it is asked. It mixes the keys with the splitmix64 finalizer. Use it where several modules must agree on one draw without storing it, and where drawing everything would cost too much: whether a tie is used today (ODD §6.11). It is still seeded per run and per stream, so determinism and common random numbers hold.
- Why streams instead of one generator: switching on a new module (say, dissent) must not shift every random draw in demography. With separate streams the baseline parts of two scenarios see the same random numbers, so differences between scenarios come from the new module, not from noise. This is the "common random numbers" technique.
- Banned: the global `math/rand` functions, `time.Now()` anywhere in the simulation, and ranging over a Go map in simulation logic (map order is random). Sort keys first if a map is unavoidable.
- One simulation runs in one goroutine. `silosim batch` runs many seeds in parallel, one goroutine each.

## Configuration

- `scenarios/baseline.yaml` is the **complete** parameter set; every number in ODD §7 appears there with a comment pointing to the ODD section.
- Other scenarios start with `extends: baseline.yaml` and override only what changes. The path is relative to the child file. Sections merge key by key, so `rations: { weights: { upTop: 1.5 } }` changes one weight and keeps the rest; lists (`modules`, `departments`) replace the parent's list whole. Chains are allowed; loops fail.
- Loading is strict: unknown keys fail (`yaml.Decoder.KnownFields(true)`, checked in each file), missing keys fail (checked after merging), out-of-range values fail with a message naming the key. Range rules are listed in ODD §7. All problems are reported at once.
- **A module is on if and only if it is listed in `modules:`.** Part-2 parameters live in `baseline.yaml` with every part-2 module left out of `modules:`, so the loader still sees one complete parameter set; `canon.yaml` and the experiment scenarios extend it. A module's section is loaded and validated even when the module is off. `part2-off.yaml` keeps the baseline module list and differs only in part-2 sections, and must give the same files (T7.5).
- Each part-2 module also has a **null setting** (surveillance 0, no relics, Silo 1 `off`): on, but doing nothing that reaches the baseline state. It is a separate test per module, not part of the frozen-baseline check.
- Modules read parameters only from the typed config struct. **No numbers in module code** except 0, 1 and unit conversions (365 days a year).

## Outputs

One folder per run: `runs/<scenario>-<seed>/`

| File | Written | Content |
| --- | --- | --- |
| `daily.csv` | every tick after warm-up | ODD §8 daily columns |
| `yearly.csv` | every 365 ticks | median age, age pyramid, department headcount vs target, Mechanical workers in their zone, zone changes, Mayor |
| `events.jsonl` | on event | generator failure/repair, rationing on/off, Mayor change, department below 50% |
| `run.json` | start and end | resolved scenario (after `extends`), seed, git commit, Go version, runtime; part 2 adds outcome fields (below) |
| `rebellions.jsonl` | when a rebellion ends (or at the end of the run), only with the rebellion module on | one line per rebellion: start, trigger, starting zone, peak share of active adults, duration, collective actions, deaths, arrests, damage, ending (fields and window in ODD §6.14, Outputs). The module keeps a day log (`Rebellion.Days`) and folds it into a `RebellionRecord`; the output module writes the records |
| `frames.csv.gz` | every `output.frameEvery` days (default 30), only if switched on | per-floor snapshot in long format: `day,floor,metric,value` |

**Part-2 outcome fields in `run.json`** (`outcomes`, with the rebellion module on): day of the first rebellion, number of rebellions and unrest spells, day and cause of collapse. A run that ends without one writes `null` (censored, ODD §1.2), so survival curves can be built from `run.json` alone. The output module counts them from the events it writes, so they always agree with `events.jsonl`.

**Frames** are off by default (`output.frames`) and switched on for runs meant to be replayed (M16). They are written with `compress/gzip` from the standard library, in the logging phase on measured days that are multiples of `output.frameEvery`, and never draw random numbers. A metric is an `output.FrameMetric`: a name and `Fill(w *world.World, v []float64)`, which writes one value per floor into `v` (`v[i]` is floor i+1). The output module holds the baseline's three (population, households, average health); a part-2 module adds its own by implementing `output.FrameSource` (`FrameMetrics() []FrameMetric`), which `run1` in `cmd/silosim` collects in module order, so row order is fixed. Size: a 250-year run with the three baseline metrics writes 4.4 MB (T7.6); each extra metric adds about 1.5 MB.

**Columns added by part-2 modules** to `daily.csv` or `yearly.csv` appear only when that module is on, after the baseline columns. That is what keeps the baseline's bytes frozen.

Floats are written with a fixed format so files diff cleanly: 4 decimals (`strconv.AppendFloat(b, x, 'f', 4, 64)`), or 1 for food stock in days and department targets. That's finer than any change the model makes in a day, and keeps a 250-year `daily.csv` near 10 MB. JSON in `events.jsonl` uses Go's shortest exact form.

## CLI

```text
silosim run    --scenario scenarios/baseline.yaml [--seed 42] [--years 200] [--out runs/]
silosim batch  --scenario scenarios/baseline.yaml --seeds 1-100 [--workers 8] --out runs/batch-001
silosim report --batch runs/batch-001            # calibration targets, pass/fail per pattern; C1–C4 when the rebellion module was on
```

A progress line in the terminal is enough.

**Viewer rule.** Everything a UI shows is written to files during the run. No viewer, live or after the fact, reads the simulation's memory or changes a run: `silosim serve` (M16) and any later terminal viewer read only files under `runs/`. A run with a viewer open is therefore the same run as one without.

## Testing

| Layer | What | When |
| --- | --- | --- |
| Unit tests | Each submodel on a tiny hand-built world (5–50 people), checking the rule in ODD §6 | Every task |
| Statistical tests | Rules with known expected rates, e.g. conception within a 365-day window ≈ 80% over 10,000 trials; median death age 65–69 with health 1 | When the submodel is written |
| Invariants | ODD §8 list, as functions `func(w *world.World) error` | Every tick in tests, every year in runs |
| Golden test | 1,000 people, 20 years, seed 1: SHA-256 of `daily.csv`, `yearly.csv` and `events.jsonl` stored in `testdata/golden/` (the last two since T7.5, checked equal to `baseline-v1`) | Every `make check`; update only with `make update-golden` (runs `go test ./cmd/silosim -run TestGolden -update`; `go test ./... -update` fails in packages without the flag) and a reason in the commit message |
| Calibration | 100 seeds, `silosim report` | Milestone 6 (baseline); T12.8 and T15.4 (canon patterns C1–C5) |
| Frozen baseline | `part2-off.yaml` gives the same `daily.csv`, `yearly.csv` and `events.jsonl` as `baseline.yaml`, at the golden size; with the golden test this ties every part-2 build to `baseline-v1` | Every `make check`, from T7.5 |
| Module alone | Each part-2 module on its own: off, null setting, on; then combined | Every part-2 module task |
| Agenda | Ordering by day and phase, same-tick ties in insertion order, cancellation | T7.3 |

The golden test is the main guard against silent changes: any change to model behaviour changes the hash, so it can never happen by accident.

## Performance budget

Rough cost: 10,000 people × 91,250 ticks ≈ 0.9 billion person-days, each with a mortality draw and a few float updates. With flat slices and no allocation inside the tick loop this is seconds to tens of seconds in Go. Rules that keep it there:

- No allocation per person per tick (reuse buffers; `Alive` is patched, not rebuilt every tick).
- Partner search only for the few people who start searching today (ODD §6.3), not a scan of everyone every day.
- Department output and coverage computed once per tick, not per person.
- Profile with `go test -bench` and `pprof` before optimising anything.

**Part 2: ≤ 4 minutes** for a full run (50 + 300 years) with every module on; 5 minutes is the hard ceiling if the full model can't fit in 4. Raised from 3 minutes on 2026-10-31: the baseline plus space and network already took 177 s (T9.2). The final experiments use 1,000 seeds per scenario (ODD §1.2): at 4 minutes a run on 8 workers that is about 8 hours per scenario (10 at 5 minutes), which is what T15.5 plans with. The hot spot is conversations: they are drawn per tie only when a module asks (ODD §6.11), and modules ask only for the people with something to pass on; nothing scans all ties or all pairs of people each day. The same rules apply as above: no allocation per person per tick, module slices grown on birth and not rebuilt, and per-floor or per-department totals computed once per tick.
