# CLAUDE.md — silosim

Agent-based simulation of Silo 18 (from the TV series *Silo*) in Go. The **baseline** (part 1, tag `baseline-v1`) is done and frozen: a silo of ~10,000 people running normally under the Pact for 200 years, with no dissent or rebellion. The current goal is **part 2**: modules for space, social ties, beliefs, regime control, rebellion, regime response and Silo 1, built on top of the baseline without changing it. This is also a likely university thesis, so the owner (Adam) must be able to explain every rule in the model.

## Read first

| File | Role |
| --- | --- |
| `docs/odd-spec.md` | **Source of truth for model behaviour** (ODD protocol). Section numbers like §6.4 refer to it. |
| `docs/architecture.md` | Where code lives, engine and module design, randomness, testing. |
| `docs/tasks-part2.md` | The build plan for part 2 (current). Work on exactly one task at a time. |
| `docs/tasks.md` | The build plan for part 1 (done); its "how to run a task" still applies. |
| `scenarios/baseline.yaml` | The complete parameter set and the real config shape. The YAML snippet in ODD §5 is only illustrative. |
| `docs/lore-reference.md` | Show canon behind the spec. Background only; never overrides the spec. |

## Rules

### The spec decides behaviour
- Implement only what the task and the spec say. No extra mechanics, no "realism" additions, no plot details from the show.
- If the spec is ambiguous, contradictory, or a rule can't work as written: **stop and ask**. Propose the exact spec wording to change. Never pick an interpretation silently.
- Every submodel function starts with a comment naming its ODD section, e.g. `// ODD §6.4: lottery permits`.
- When a task changes model behaviour, the same commit updates `docs/odd-spec.md` if the spec was wrong or incomplete.

### The baseline is frozen
- With every part-2 module off, `scenarios/baseline.yaml` gives `daily.csv`, `yearly.csv` and `events.jsonl` byte-identical to `baseline-v1`. Part-2 work never moves the golden hash.
- Each new module gets its own RNG stream, appended in `internal/rng/streams.go`; existing stream IDs never change.
- New per-person state lives in the module's own slices on `World` (indexed by PersonID, nil when the module is off), never as new fields on `Person`. See `docs/architecture.md`, Part-2 state.
- Each milestone starts with a spec task (`T*.0`) that changes only docs and the scenario file. Adam approves it before any code task in that milestone starts.

### Numbers come from config
- No numeric constants in module code except 0, 1 and unit conversions (365 days a year). Every rate, age, share and threshold is a field in `scenarios/baseline.yaml` and the typed config.
- New parameter → add it to `baseline.yaml` with a comment and an ODD § reference, and to ODD §7.

### Determinism (non-negotiable)
- Randomness only through the `rng.Stream` given to the module. Never the global `math/rand`, never `time.Now()` in simulation code.
- Never range over a Go map in simulation logic. Use slices, or sort the keys.
- No goroutines inside a single run. Parallelism only across seeds in `batch`.
- The step order lives only in `internal/engine/schedule.go`. Do not reorder phases without a spec change.

### Tests
- Each submodel gets unit tests on a tiny hand-built world, and a statistical test where the spec gives an expected rate (see the task's "Done when").
- Invariants (ODD §8) run every tick in tests.
- The golden test must pass. Never regenerate the golden hash unless the task changes behaviour on purpose; when you do, say why in the commit message and in your summary.
- `make check` must pass before a task is done.

### Scope and dependencies
- Standard library + `gopkg.in/yaml.v3` only. Ask before adding anything else.
- Keep diffs to the task. Note unrelated problems you spot in your summary instead of fixing them.
- Code style: `gofmt`, `go vet` clean, small functions, plain names. Comments explain *why*, not *what*.

## Finishing a task

1. `make check` passes.
2. Tick the task in `docs/tasks.md` with the date.
3. Commit: `T1.4: lottery and reproduction (ODD §6.4)` plus a short body.
4. End with a summary for Adam, in plain language:
   - the rule implemented, in 2–4 sentences, as if explaining it to a thesis examiner;
   - test results, including any statistical numbers;
   - any spec change or open question;
   - for part 2: how many parameters the task added, and how many have a basis other than "guess".

## Commands

```bash
make build                                   # bin/silosim
make test                                    # go test ./...
make check                                   # build + vet + test + golden
make update-golden                           # regenerate golden hash (only with a reason)
bin/silosim run --scenario scenarios/baseline.yaml --seed 42
bin/silosim batch --scenario scenarios/baseline.yaml --seeds 1-100 --out runs/batch-001
bin/silosim report --batch runs/batch-001
```
