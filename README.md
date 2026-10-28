# silosim

An agent-based simulation of Silo 18 from the TV series *Silo*, written in Go.

The first goal is a **baseline silo**: about 10,000 people living under the Pact for 200 years, with births controlled by the lottery, departments staffed through shadowing, and a closed loop of power, water, air and food. It answers one question: can the Pact's rules keep a closed silo stable? Every later experiment (dissent, cleaning, rebellion, different regime policies) is measured against it.

## Status

Planning done; implementation starts with task T0.1 in [`docs/tasks.md`](docs/tasks.md).

## Documents

- [`docs/odd-spec.md`](docs/odd-spec.md) — the model, described with the ODD protocol (Grimm et al. 2020)
- [`docs/architecture.md`](docs/architecture.md) — how the code is organised
- [`docs/tasks.md`](docs/tasks.md) — build plan and progress
- [`docs/lore-reference.md`](docs/lore-reference.md) — show canon used for the model (spoiler-safe through S3E9)

## Running (after milestone 0)

```bash
make build
bin/silosim run --scenario scenarios/baseline.yaml --seed 42
```

Each run writes `daily.csv`, `yearly.csv`, `events.jsonl` and `run.json` to `runs/<scenario>-<seed>/`.
