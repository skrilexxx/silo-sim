# Architecture — silosim baseline

How the ODD spec (`docs/odd-spec.md`) becomes Go code. The spec says *what* the model does; this file says *where* it lives and *how* the code is organised. If the two disagree, the spec wins and this file gets fixed.

## Goals that shape the design

1. **Reproducible.** Same scenario file + same seed → byte-identical output on the same machine.
2. **Modular.** The baseline is five modules. Later experiments (beliefs, dissent, cleaning, rebellion, Silo 1) are new modules switched on in a scenario file, without touching the core.
3. **Checkable.** Invariants and a golden test catch silent bugs, because a broken simulation still produces plausible numbers.
4. **Fast enough.** 250 simulated years (50 warm-up + 200 measured) of ~10,000 people in under a minute on a laptop, so 100-seed batches are practical.
5. **Boring dependencies.** Go standard library + `gopkg.in/yaml.v3`. Nothing else without asking.

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
│   ├── engine/               # tick loop, schedule.go (THE step order), module registry
│   ├── initpop/              # day-1 generator (ODD §5)
│   ├── modules/
│   │   ├── demography/       # 6.1 mortality, 6.3 partnering + housing, 6.4 lottery + reproduction
│   │   ├── health/           # 6.2 health and Medical
│   │   ├── work/             # 6.5 life stages, shadowing, skill, output, retraining
│   │   ├── resources/        # 6.6 generator, power, water, air, food, parts; 6.7 rations
│   │   └── governance/       # 6.8 Mayor
│   ├── invariants/           # ODD §8 invariant checks
│   ├── output/               # daily.csv, yearly.csv, events.jsonl, run.json
│   └── report/               # calibration targets → pass/fail table
├── scenarios/
│   └── baseline.yaml         # the complete default parameter set
├── testdata/golden/          # golden-run hashes
├── docs/
│   ├── odd-spec.md           # source of truth for model behaviour
│   ├── lore-reference.md     # show canon behind the spec
│   ├── architecture.md       # this file
│   ├── tasks.md              # build plan, one task per Claude Code session
│   └── calibration-log.md    # created in milestone 6
└── analysis/                 # optional Python plots; never imported by Go
```

`internal/` keeps everything private to this module. Only `cmd/silosim` is an entry point.

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

## Engine: modules, steps and the schedule

The ODD schedule (§3) interleaves submodels: generator → life support → food → rations → work → health → deaths → … A module therefore does not "run" as one block. It registers **steps**, and the engine runs all steps in the fixed order defined in one file.

```go
// engine/schedule.go — the ONLY place the order and frequency live. Mirrors ODD §3 step numbers.
type Phase int
const (
    PhaseGenerator    Phase = 1
    PhaseLifeSupport  Phase = 2
    PhaseFood         Phase = 3
    PhaseRations      Phase = 4
    PhaseWork         Phase = 5
    PhaseHealth       Phase = 6
    PhaseDeaths       Phase = 7
    PhaseReproduction Phase = 8
    PhaseLifeStages   Phase = 9
    PhasePartnering   Phase = 10
    PhaseLogDaily     Phase = 11
    PhaseLottery      Phase = 12 // every 30 days
    PhaseHousing      Phase = 13 // every 30 days
    PhaseStaffing     Phase = 14 // every 30 days
    PhaseSnapshot     Phase = 15 // every 365 days
    PhaseGovernance   Phase = 16 // every 365 days
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

The engine:

1. Loads the scenario, builds the module list from `modules:`, checks `Requires`.
2. Runs `initpop`, then each module's `Init`.
3. Collects all steps, sorts them by `Phase` (two modules may not register the same phase), and loops over ticks. Days run 1…N; a phase that runs every 30 or 365 days runs on days that are multiples of that number. Frequency belongs to the phase, not the step, because ODD §3 fixes it per phase.
4. At the start of each tick, shuffles `Alive` with the engine's own stream (ODD §3: random order each tick).
5. Runs invariants every `output.invariantEvery` ticks (every tick in tests).
6. Skips the output phases (logging, snapshot) during warm-up unless `output.logWarmup: true`.

A step that is disabled because its module is off simply does not run. Modules must not reach into other modules' internals; they share state only through `world.World`. If a step needs a value from a module that is off (for example food intake when `resources` is off), the scenario is invalid and loading fails. The baseline always runs all five.

## Randomness

- `math/rand/v2` with `rand.NewPCG(seed, streamID)`.
- **One seed, one stream per module.** `streamID` is a fixed constant per module name (written in `rng/streams.go`, never computed from map order). The engine has its own stream for shuffling.
- Why streams instead of one generator: switching on a new module (say, dissent) must not shift every random draw in demography. With separate streams the baseline parts of two scenarios see the same random numbers, so differences between scenarios come from the new module, not from noise. This is the "common random numbers" technique.
- Banned: the global `math/rand` functions, `time.Now()` anywhere in the simulation, and ranging over a Go map in simulation logic (map order is random). Sort keys first if a map is unavoidable.
- One simulation runs in one goroutine. `silosim batch` runs many seeds in parallel, one goroutine each.

## Configuration

- `scenarios/baseline.yaml` is the **complete** parameter set; every number in ODD §7 appears there with a comment pointing to the ODD section.
- Other scenarios start with `extends: baseline.yaml` and override only what changes. The path is relative to the child file. Sections merge key by key, so `rations: { weights: { upTop: 1.5 } }` changes one weight and keeps the rest; lists (`modules`, `departments`) replace the parent's list whole. Chains are allowed; loops fail.
- Loading is strict: unknown keys fail (`yaml.Decoder.KnownFields(true)`, checked in each file), missing keys fail (checked after merging), out-of-range values fail with a message naming the key. Range rules are listed in ODD §7. All problems are reported at once.
- Modules read parameters only from the typed config struct. **No numbers in module code** except 0, 1 and unit conversions (365 days a year).

## Outputs

One folder per run: `runs/<scenario>-<seed>/`

| File | Written | Content |
| --- | --- | --- |
| `daily.csv` | every tick after warm-up | ODD §8 daily columns |
| `yearly.csv` | every 365 ticks | median age, age pyramid, department headcount vs target, Mechanical workers in their zone, zone changes, Mayor |
| `events.jsonl` | on event | generator failure/repair, rationing on/off, Mayor change, department below 50% |
| `run.json` | start and end | resolved scenario (after `extends`), seed, git commit, Go version, runtime |

Floats are written with a fixed format so files diff cleanly: 4 decimals (`strconv.AppendFloat(b, x, 'f', 4, 64)`), or 1 for food stock in days and department targets. That's finer than any change the model makes in a day, and keeps a 250-year `daily.csv` near 10 MB. JSON in `events.jsonl` uses Go's shortest exact form.

## CLI

```text
silosim run    --scenario scenarios/baseline.yaml [--seed 42] [--years 200] [--out runs/]
silosim batch  --scenario scenarios/baseline.yaml --seeds 1-100 [--workers 8] --out runs/batch-001
silosim report --batch runs/batch-001            # calibration targets, pass/fail per pattern
```

A progress line in the terminal is enough. A live terminal dashboard (for example Bubble Tea) can come later as its own task; it must only read state, never change it.

## Testing

| Layer | What | When |
| --- | --- | --- |
| Unit tests | Each submodel on a tiny hand-built world (5–50 people), checking the rule in ODD §6 | Every task |
| Statistical tests | Rules with known expected rates, e.g. conception within a 365-day window ≈ 80% over 10,000 trials; median death age 65–69 with health 1 | When the submodel is written |
| Invariants | ODD §8 list, as functions `func(w *world.World) error` | Every tick in tests, every year in runs |
| Golden test | 1,000 people, 20 years, seed 1: SHA-256 of `daily.csv` stored in `testdata/golden/` | Every `make check`; update only with `make update-golden` (runs `go test ./cmd/silosim -run TestGolden -update`; `go test ./... -update` fails in packages without the flag) and a reason in the commit message |
| Calibration | 100 seeds, `silosim report` | Milestone 6 |

The golden test is the main guard against silent changes: any change to model behaviour changes the hash, so it can never happen by accident.

## Performance budget

Rough cost: 10,000 people × 91,250 ticks ≈ 0.9 billion person-days, each with a mortality draw and a few float updates. With flat slices and no allocation inside the tick loop this is seconds to tens of seconds in Go. Rules that keep it there:

- No allocation per person per tick (reuse buffers; `Alive` is patched, not rebuilt every tick).
- Partner search only for the few people who start searching today (ODD §6.3), not a scan of everyone every day.
- Department output and coverage computed once per tick, not per person.
- Profile with `go test -bench` and `pprof` before optimising anything.
