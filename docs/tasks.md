# Build plan — silosim baseline

One task = one Claude Code session = one branch = one commit (or a few). Do them in order: each task assumes the ones before it are done. Tick the box and add the date when a task is merged.

**How to run a task**

1. Start Claude Code in the repo, switch to plan mode, and say: `Do task T1.2 from docs/tasks.md.`
2. Read the plan it proposes. Check it names the ODD sections, the files, and the tests. Ask it why anything looks off.
3. Let it build. It must finish with `make check` passing and a short plain-language summary of the rule it implemented.
4. Read the summary and the test names. You should be able to explain the rule to someone else. If you can't, ask before merging.

Every task's **Done when** list includes, without repeating it: `make check` passes (build, vet, tests, invariants, golden test), no new dependencies, and this file is updated.

---

## Milestone 0 — Skeleton

The engine runs empty ticks deterministically and writes output. No model yet.

- [x] **T0.1 Repo setup.** (2026-10-28) `go mod init`, folder layout from `docs/architecture.md`, `Makefile` (`build`, `test`, `check`, `run`), `.gitignore` (`runs/`, `bin/`), CLI skeleton with `run` that prints the resolved config.
  - *Done when:* `make check` runs; `silosim run --scenario scenarios/baseline.yaml --dry-run` prints every parameter; a typo in a YAML key fails with the key's name.
- [x] **T0.2 Config.** (2026-10-28) Typed `Scenario` struct mirroring `scenarios/baseline.yaml`, strict loading, `extends`, range validation (probabilities in 0–1, ages ordered, shares summing to 1).
  - *Spec:* ODD §5 (input), §7. *Done when:* tests cover unknown key, missing key, bad range, and an `extends` override.
- [x] **T0.3 RNG.** (2026-10-28) `internal/rng` with PCG streams per module name (fixed constants), plus helpers: `Bernoulli(p)`, `IntRange`, `WeightedIndex`, `Shuffle`.
  - *Done when:* same seed → same sequence; different stream → different sequence; a test proves adding a draw on stream A does not change stream B.
- [x] **T0.4 World and engine.** (2026-10-28) `world` types (ODD §2), `engine` with `schedule.go`, the `Module` interface, tick loop with `Every`, warm-up handling, shuffle of `Alive`.
  - *Done when:* a fake test module registering steps at phases 7 and 2 runs them in order 2, 7; a 30-day step runs on days 30, 60…; two modules on the same phase fail at startup.
- [x] **T0.5 Output and invariants framework.** (2026-10-28) `output` writers (`daily.csv` with day + population only for now, `run.json`), `invariants` package with the checks that already make sense (counts agree, 0–1 ranges), golden-test harness with `-update`.
  - *Done when:* a 10-year empty run writes files; two runs with the same seed give identical `daily.csv`.

## Milestone 1 — People, births and deaths

Demography only. Until later milestones, stub what other modules provide: health = 1, food intake = need, Medical coverage = 1.

- [x] **T1.1 Day-1 population.** (2026-10-28) `initpop`: floors and zones, 10,000 people with ages from the stationary distribution of the mortality rule, zones, households, couples, children placed with plausible mothers, implants in.
  - *Spec:* ODD §5 steps 1–4, 7. *Done when:* invariants pass on day 1; zone split within 1% of config; age histogram within 2 percentage points of the stationary distribution in every 5-year band; every child's mother was 18–40 at the birth (lottery limits; ODD §5 step 4).
- [x] **T1.2 Mortality.** (2026-10-28) Gompertz hazard with infant term and health multiplier; death handling (partner freed, orphans rehomed, person removed from `Alive`). Job vacancy moved to T2.1, since no jobs exist before it.
  - *Spec:* ODD §6.1. *Done when:* statistical test: 100,000 simulated lives with health 1 have median age at death 65–69; orphan rehoming keeps every child under 18 in a household in the same zone.
- [x] **T1.3 Partnering and housing.** (2026-10-28) Search rate, weighted choice, monthly housing moves, home capacity queue.
  - *Spec:* ODD §6.3, §5 step 1 (home capacity). *Done when:* partnerships always symmetric; no floor over capacity; unit test of the weights (same-zone pair chosen ~4× as often as a cross-zone pair with equal age gap).
- [x] **T1.4 Lottery and reproduction.** (2026-10-28) Monthly permits by the formula, eligibility, uniform draw, 365-day window, daily conception chance, 270-day pregnancy, birth into the household, implant back in.
  - *Spec:* ODD §6.4. *Done when:* statistical test: share of windows ending in pregnancy is 78–82% over 10,000 windows; no pregnancy without a window (invariant); permit formula unit-tested on hand-picked numbers.
- [x] **T1.5 Milestone check.** (2026-10-28) Run 10 seeds × 200 years (after warm-up) with demography only.
  - *Done when:* population stays within 9,000–11,000 in at least 9 of 10 seeds; median age 30–40. If not, stop and discuss the rule before tuning numbers. Write the result into `docs/calibration-log.md`.

## Milestone 2 — Work and health

- [x] **T2.1 Departments and life stages.** (2026-10-28) Department table from config, job vacated on death (ODD §6.1), child → shadow at 14 → worker at 18 → retired at 65 or on long ill health; shadow placement (parent's trade, understaffed departments, cross-zone chance); masters; day-1 skill and daily skill growth (moved here from T2.2: masters need skill).
  - *Spec:* ODD §6.5, §5 steps 5–6. *Done when:* after 50 years each department is within ±15% of target; ~60% of placements follow a parent's department (test with tolerance; the share of workers in a parent's department is logged as an outcome); a master never has more than 2 shadows.
- [x] **T2.2 Output, coverage, retraining.** (2026-10-28) Department output, the shared coverage formula (0.6 × target headcount), monthly retraining from General services.
  - *Spec:* ODD §6.5, coverage rule in §6.6. *Done when:* a fully staffed department at average skill and health has output / need ≈ 1.1, so coverage 1 (unit test); retraining lifts a department from 70% to above 80% of target within a few months.
- [x] **T2.3 Health and Medical.** (2026-10-28) Daily health changes (hunger and air still stubbed at "fine"), injuries by department, illness, recovery scaled by Medical coverage, maternal death risk, implant-removal queue when coverage < 1. Mortality now uses real health.
  - *Spec:* ODD §6.2. *Done when:* with Medical removed (target 0) average health falls and deaths rise (test); with full Medical, average health stays above 0.9; the queue delays windows when coverage is 0.5.

## Milestone 3 — Resources

- [x] **T3.1 Generator and power.** (2026-10-29) Condition wear, repair × (1 − c), parts use, failures, outages with backup power, capacity set on day 1, powerFactor.
  - *Spec:* ODD §6.6. *Done when:* with full Mechanical coverage condition settles at 0.80 ± 0.02; failure count over 100 simulated centuries at c = 0.8 averages 2.5–3.5 per century; with no parts, condition falls steadily.
- [x] **T3.2 Water, air, food, parts.** (2026-10-29) Water and air from power and Mechanical coverage, food production, Supply making parts.
  - *Spec:* ODD §6.6. *Done when:* a 14-day outage drops air below 0.3 and it recovers within ~10 days after repair (test); food stock is conserved exactly every tick (invariant).
- [x] **T3.2a Resource-loop stability.** (2026-10-29) One generator failure currently tips the silo into a health collapse (calibration log, 2026-10-29). Find the cause in the loop health → Mechanical → power → air → health, agree a parameter or rule change with Adam, record it.
  - *Spec:* ODD §6.2, §6.6. *Done when:* 10 seeds × 100 years with demography, health, work and resources all survive every generator failure (average health returns above 0.9); the change is in the spec and the calibration log.
- [ ] **T3.3 Rations.** Distribution by zone weights, children's need, rationing below 30 days of stock, intake/need per person feeding health (remove the hunger stub), hardship per zone logged.
  - *Spec:* ODD §6.7. *Done when:* with equal weights every zone's intake/need is equal; with Up Top weight 1.5 and a food shortage, Down Deep hardship is higher (test).

## Milestone 4 — Governance

- [ ] **T4.1 Mayor.** Appointment and election, `mayorTerm: 5y | life`, new Mayor when the term ends or the Mayor dies, logged to `events.jsonl` and `yearly.csv`.
  - *Spec:* ODD §6.8. *Done when:* with `life` the Mayor changes only on death; with elections and 80% own-zone voting the Mids candidate wins most elections (largest zone).

## Milestone 5 — Outputs and tools

- [ ] **T5.1 Full outputs.** All `daily.csv` columns, `yearly.csv`, `events.jsonl` per ODD §8; complete invariant list.
- [ ] **T5.2 Batch runner.** `silosim batch` with `--seeds 1-100` and `--workers`, one goroutine per run, one folder per seed.
  - *Done when:* batch output for seed 7 is identical to a single `run` with seed 7.
- [ ] **T5.3 Report.** `silosim report` computes every ODD §8 calibration target per seed and prints pass/fail with the share of seeds passing.
- [ ] **T5.4 Performance.** Benchmark a 250-year run; profile; fix hot spots.
  - *Done when:* one full run under 60 seconds on your laptop.

## Milestone 6 — Calibration

- [ ] **T6.1 Calibrate.** Sweep the parameters marked "Calibrate" in ODD §7, 100 seeds per setting, record every attempt and result in `docs/calibration-log.md`.
  - *Done when:* every calibration target passes in ≥ 90 of 100 seeds. If a target can't be met by tuning, the rule is wrong: change the spec first, then the code.
- [ ] **T6.2 Freeze baseline v1.** Update `scenarios/baseline.yaml` with calibrated values, update ODD §7, regenerate the golden hash, tag `baseline-v1`.

---

## After the baseline (not planned in detail yet)

Each becomes its own spec section before any code: beliefs and information spread, relics and the Flamekeepers, surveillance and covert lottery denial, cleaning as punishment and shock, rebellion (Epstein-style grievance and legitimacy), regime policy response, stair travel time, Silo 1 kill switch.
