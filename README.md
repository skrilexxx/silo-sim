```text
  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░╔═╗░░░░░░  the outside
  ███████╗██╗██╗      ██████╗      ║▒║  Up Top      floors 1–48
  ██╔════╝██║██║     ██╔═══██╗     ║▒║
  ███████╗██║██║     ██║   ██║     ║▒║  Mids        floors 49–96
  ╚════██║██║██║     ██║   ██║     ║▒║
  ███████║██║███████╗╚██████╔╝     ║▒║  Down Deep   floors 97–144
  ╚══════╝╚═╝╚══════╝ ╚═════╝      ╚═╝
           18  ·  10,000 people  ·  200 years  ·  one Pact
```

# silosim

An agent-based simulation of Silo 18 from the TV series *Silo*, written in Go.

The **baseline silo** holds about 10,000 people living under the Pact for 200 years. Births are controlled by the lottery, departments are staffed through shadowing, and power, water, air and food form a closed loop. It answers one question: can the Pact's rules keep a closed silo stable? Every later experiment (dissent, cleaning, rebellion, different regime policies) is measured against it.

## Status

**Baseline v1 is done and frozen** at git tag `baseline-v1`: the model in [`docs/odd-spec.md`](docs/odd-spec.md) v1.0 with the parameters in [`scenarios/baseline.yaml`](scenarios/baseline.yaml). Milestones 0–6 in [`docs/tasks.md`](docs/tasks.md) are complete. The modules after the baseline (beliefs, cleaning, rebellion, …) are not started; each gets its own spec section first.

Calibration on 100 seeds × (50 warm-up + 200 measured) years, from [`docs/calibration-log.md`](docs/calibration-log.md), 2026-10-30:

| Pattern (ODD §8) | Target | Seeds passing |
| --- | --- | ---: |
| Stable population | 9,000–11,000 every day | 100 / 100 |
| Plausible ages | median age 30–40 every year | 100 / 100 |
| No demographic shocks | 5-year births change < 30% | 100 / 100 |
| Balanced resources | rationing < 5% of days | 100 / 100 |
| Rare crises | 1–6 generator failures per century | 96 / 100 |
| Staffed departments | never below 50% of target for > 1 year | 100 / 100 |
| Persistent zones | Mechanical workers living Down Deep > 75% | 100 / 100 |

A pattern holds when at least 90 of 100 seeds pass.

## Requirements

Go 1.27 or newer. The only dependency outside the standard library is `gopkg.in/yaml.v3`.

## Running

```bash
make build                                   # bin/silosim
bin/silosim run --scenario scenarios/baseline.yaml --seed 42
bin/silosim run --scenario scenarios/baseline.yaml --dry-run      # print the resolved parameters
bin/silosim batch --scenario scenarios/baseline.yaml --seeds 1-100 --out runs/batch-001 [--workers 8]
bin/silosim report --batch runs/batch-001    # ODD §8 targets: per seed, then share of seeds passing
```

One run of 250 years takes about a minute (`make bench`). A batch runs one seed per CPU core at a time, so 100 seeds take roughly an hour on an 8-core laptop and use the whole machine.

Other scenarios start with `extends: baseline.yaml` and override only what changes.

## Output

Each run writes to `<output dir>/<scenario>-<seed>/` (`runs/` by default):

| File | Content |
| --- | --- |
| `daily.csv` | population by zone, births, deaths, permits, windows, food stock, rationing, power, generator, water, air, health and hardship by zone, Medical and Mechanical coverage |
| `yearly.csv` | median age, age pyramid by sex, workers and target per department, Mechanical workers in their zone, zone moves, the Mayor |
| `events.jsonl` | generator failures and repairs, rationing on and off, Mayor changes, departments below half their target |
| `run.json` | the resolved scenario, seed, git commit, Go version, runtime |

Warm-up years are not written. The same seed always gives byte-identical files.

## Development

```bash
make check          # build, gofmt, vet, all tests including the golden test
make update-golden  # only when behaviour changes on purpose; give the reason in the commit
make bench          # time one full baseline run
make milestone1     # T1.5 check: demography only, 10 seeds (minutes)
make stability      # T3.2a check: 10 seeds × 100 years (minutes)
```

The rules for working on the code (determinism, numbers from config, one task at a time) are in [`CLAUDE.md`](CLAUDE.md).

## Documents

- [`docs/odd-spec.md`](docs/odd-spec.md): the model, described with the ODD protocol (Grimm et al. 2020). The source of truth for behaviour.
- [`docs/architecture.md`](docs/architecture.md): how the code is organised.
- [`docs/tasks.md`](docs/tasks.md): the build plan and progress.
- [`docs/calibration-log.md`](docs/calibration-log.md): every calibration run and spec decision, in order.
- [`docs/lore-reference.md`](docs/lore-reference.md): show canon used for the model (spoiler-safe through S3E9).
