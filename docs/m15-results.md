# M15 results (lean plan)

Run overnight 2026-11-04/05 by `analysis/m15.sh`: seeds 1–30 in every scenario. Canon runs 300 measured years; the experiments run 50 years of warm-up and 150 measured. The plan, and the metric for each question, were written down first in `docs/experiments.md`. The raw reports are in `runs/m15/<name>.txt` (git-ignored) and the run folders sit next to them.

As in the plan, a difference counts as a result only when the two 95% intervals don't overlap. Every claim holds only under the model's rules (ODD §1.2).

Each batch took about 31 minutes on 8 cores, including its two reports; the whole run took about 13 hours. The thermal watcher never paused it: the package stayed below 95 °C and `tempwatch` logged no WARN (`runs/m15-thermal.log`).

## The main finding: rebellions don't end

All the experiments are shaped by one pattern. Before the scripted cleaning failure, the silo is almost perfectly quiet. After it, the silo is almost never at peace again.

| Batch | Rebellions per century before the failure | After it |
|---|---|---|
| canon (failure in year 150) | 0.09 (4 in 4,500 years) | 3.38 (152 in 4,500 years) |
| base (failure in year 75) | 0.04 (1 in 2,250 years) | 4.00 (90 in 2,250 years) |

In base, 43% of measured time is spent in a rebellion, which is about 87% of the time after the failure. In 27 of 30 runs a rebellion is still going on when the run ends. T13.2 had already named this "the standing rebellion", and C4 was set aside until it is understood. This run shows that it also decides C1, C4 and E3 (open question 1).

## T15.3 Canon validation (C1–C5)

`scenarios/canon.yaml`, 30 runs × 300 years, 156 rebellions. C5 comes from E4.

| Pattern | Target | Measured | Verdict |
|---|---|---|---|
| C1 Long peace | 0.3–2 rebellions per century, pooled | 1.73 (1.47–2.02) | PASS, but only because the runs are pooled (see below) |
| C2 Visible failures trigger revolts | ≥ 3× as likely within 5 years of a scripted failure | 29/30 after vs 4/999 without (241×) | PASS |
| C3 Revolts start Down Deep | ≥ 60% Down Deep, ≥ 50% of those with a Mechanical strike | 140/156 (90%) Down Deep; 13/140 (9%) with a strike | FAIL on the strike half |
| C4 Crushed rebellions are forgotten | < 10% know the history 40 years on, in ≥ 90% | 6/105 (6%) | FAIL |
| C5 A rebellion can end a silo | 5–50% end in collapse, Silo 1 on | 14/40 (35%), from E4 | PASS, but only through the Safeguard (see E4) |

- **C1 passes on canon only because it pools 150 quiet years with 150 restless ones** (0.09 and 3.38 per century). Each half on its own is outside 0.3–2. Every 150-year experiment with a failure fails C1, at 2.0–2.5.
- **C3 and C4 are reported as they are, as Adam decided on 2026-11-04.**
  - C3: the Down Deep half holds in every batch (80–100%). The strike half holds in none (9–30%).
  - C4: once a rebellion has started, the true history spreads instead of being forgotten. 40 years after a suppressed rebellion, at least 10% of adults know it in 94% of cases.
- **Baseline §8 targets on canon:** six of seven hold. *Balanced resources* fails: 22/30 runs ration on more than 5% of days, which happens during rebellions.
- **Collapses:** canon now has 0 in 30 runs, against 73 of 99 runs in the T12.8 canon batch (`runs/t12.8-canon-2`). The model has changed since then (T13.2, The Order), so the two are not comparable.

## E1 Peace without shocks (Q1)

`e1-noshock` against base: 4 rebellions in 4,500 years, **0.09 per century (0.02–0.20)**, and all 4 in one run (seed 9, the first in year 72.9). The other 29 runs have no rebellion in 150 years, even though there are 201 spells of unrest across the batch.

- Survival to the first rebellion: S = 0.967 (0.902–1.000) from year 75 to year 150. The median is not reached.
- Survival to collapse: 1.000 throughout.

**Answer:** under the model's rules, peace without a visible failure lasts longer than 150 years in 29 runs of 30. The model is quieter than C1's lower bound (0.3 per century).

## E2 The trigger effect of a visible failure (Q1, C2)

The share of runs with a rebellion starting in the 5 years from day 45,625 (measured year 75), with Wilson intervals, from `rebellions.jsonl`:

| Arm | Runs | Share (95%) |
|---|---|---|
| base (cleaning failure on that day) | 29/30 | 97% (83–99%) |
| e1-noshock (none) | 0/30 | 0% (0–11%) |

**Answer:** the intervals are far apart. A cleaner who walks out of sight sets off a rebellion within 5 years in almost every run. Without one, no run has a rebellion in that window. C2 in base's report: 29/30 after vs 2/503 without (243×), PASS.

## E3 Regime response (Q3; M13's milestone check)

| Strategy | Rebellions per century | Time in a rebellion | Collapses | Rebellions before the failure |
|---|---|---|---|---|
| none | 2.15 (1.74–2.60) | 51.2% (45.6–56.8) | 1/30 (air) | 5/30 |
| **The Order (base)** | 2.02 (1.63–2.46) | **43.3% (42.2–44.5)** | 0/30 | 1/30 |
| crackdown | 2.24 (1.83–2.70) | 50.4% (45.3–55.6) | 0/30 | 4/30 |
| reform | 2.53 (2.09–3.02) | 52.2% (47.4–57.0) | 0/30 | 5/30 |
| memory wipe | 2.15 (1.74–2.60) | 46.8% (41.4–52.2) | 1/30 (air) | 5/30 |
| mixed | 2.16 (1.75–2.61) | 52.7% (48.2–57.2) | 0/30 | 5/30 |

How the rebellions ended:
- suppressed in 68–74% of them, unfinished in 26–31%;
- one ended in collapse each under none and memory wipe;
- none ended in a regime change or a split.

The survival curves to the first rebellion are the same in every strategy: median 75.0 years, which is the scripted failure. The survival curves to collapse are flat at 0.97–1.00.

**Answer:**
- **The rate of rebellions and survival to collapse don't differ by more than seed noise.** Every pair of intervals overlaps.
- **Time in a rebellion is the one result.** The Order spends less time in a rebellion than none, crackdown, reform and mixed; those intervals don't overlap, though the gaps are small (44.5% against 45.3–48.2%). Against memory wipe they overlap. This result rests on base's unusually narrow interval (open question 7).
- **A hint, not a result:** The Order's runs had 1 rebellion before the failure where the others had 4–5. The Wilson intervals overlap.

So on M13's check, the strategies mostly differ by no more than noise: no strategy ends the standing rebellion.

## E4 Silo 1 (Q2, Q4, C5)

| | base (Silo 1 off) | e4-silo1 (full) |
|---|---|---|
| Collapses | 0/30 (0–11%) | **30/30 (89–100%), all by the Safeguard** |
| Survival to collapse, median | not reached | **75.1 years** |
| Rebellions per century | 2.02 (1.63–2.46) | 1.68 (1.20–2.23), over 2,387 measured years |
| Time in a rebellion | 43.3% (42.2–44.5) | 2.9% (0.3–5.5) |
| Rebellions ending in collapse | 0/91 | 14/40 (35%, 22–50%), so C5 passes |

**The Safeguard ends every run.** In 21 of 30 runs it was set off within 0.07 years (about 25 days) of the scripted failure. In six runs it took 3–20 years, and in two 25–47 years. One went off before the failure, in year 68.

The second escape was always a failed noViolence cleaning. During a rebellion many rebels are cleaned, and each cleaning fails with p\_fail = 0.05, so a second escape within 5 years is near certain (open question 3). The lower time in a rebellion comes from runs that end within weeks, not from a quieter silo.

**Answer:** under these rules, Silo 1's threat makes Silo 18 much more fragile: once the first cleaner walks out of sight, the silo is usually gone within a month.

## E5 Governance (Q3)

| Mayor | Rebellions per century | Time in a rebellion | Collapses |
|---|---|---|---|
| appointed, 5 years (base) | 2.02 (1.63–2.46) | 43.3% (42.2–44.5) | 0/30 |
| appointed, for life | 2.02 (1.63–2.46) | 43.3% (42.2–44.5) | 0/30 |
| **elected, 5 years** | 2.13 (1.71–2.59) | **37.0% (32.3–41.6)** | 3/30 (air 1, water 2) |
| elected, for life | 2.16 (1.75–2.61) | 43.1% (39.7–46.5) | 0/30 |

- **Appointment, 5 years vs life:** the results are identical. `rebellions.jsonl` is byte-for-byte the same; only the Mayor events differ (731 term ends in base, none for life). The plan expected this: the term has no other effect in this model, so there is no defect.
- **Election with a 5-year term vs appointment:** the time in a rebellion is lower and the intervals don't overlap, because an election every 5 years lifts L by λ\_elect. The rate of rebellions doesn't move. The 3 collapses against 0 are not a result, since the intervals overlap (3–26% against 0–11%).
- **Election, 5 years vs life:** the intervals just overlap (41.6% against 39.7%).

**Answer:** frequent elections shorten the time spent in a rebellion but don't make rebellions rarer. Appointment for life changes nothing.

## T15.2 Sensitivity, one at a time

*Running: 4 of 12 cells are done; this section will be completed when the rest finish.*

Every cell is base with one parameter at −50% and at +50%. Base: rate 2.02 (1.63–2.46), time 43.3% (42.2–44.5), 0/30 collapses. A value in bold has an interval that doesn't overlap base's.

| Parameter (base) | Cell | Rebellions per century | Time in a rebellion | Collapses |
|---|---|---|---|---|
| L\_edu (0.8) | 0.9 | 1.67 (1.30–2.08) | **34.5% (29.2–39.8)** | 4/30 (air) |
| | 0.7 | 1.47 (1.14–1.85) | **59.8% (49.2–70.4)** | 1/30 (air) |
| δ (0.15) | 0.075 | **0.76 (0.52–1.03)** | **49.2% (48.8–49.7)** | 0/30 |
| | 0.225 | 2.43 (1.99–2.93) | 42.2% (35.0–49.5) | 3/30 (air 2, water 1) |

What the finished cells show so far:
- **L\_edu matters** for the time spent in a rebellion, in both directions.
- **Halving δ gives fewer but endless rebellions.** There are 34 rebellions in all, and 29 runs end with one still going.

## Open questions for Adam

1. **The standing rebellion.** After the first triggered rebellion, the silo is in a rebellion about 87% of the remaining time, and 27–30 runs in 30 end with one unfinished. It drives C1's failure in the experiments, C4's failure, and the lack of difference between the E3 strategies. Is this recurrence intended, or does something stop L from recovering or a rebellion from ending? Nothing in the model has been changed for this.
2. **How to judge C1.** Without a visible failure the model gives 0.09 rebellions per century, and after one 3.4–4.0, so both are outside 0.3–2. Canon passes only by pooling the two halves. Should C1 be judged on canon pooled (as now), or per phase?
3. **The Safeguard in E4.** Every failed cleaning counts as an escape (D8), and there are many cleanings during a rebellion. As a result, 30 of 30 silos die, most within a month of the first failure. C5 passes only through this. Is that the intended reading of D8, or should only cleanings outside a rebellion count, or should the threshold be higher?
4. **Baseline §8 targets in part-2 runs.** *Balanced resources* fails in canon and most experiments (73–87% of runs, against 90% needed), because of rationing during rebellions. *Rare crises* fails in e3-none and e4 (87%); in e4 the runs are only a few decades long. Should `silosim report` keep judging the §8 targets on runs with part 2 on, or only show them?
5. **T15.2's wording.** `docs/tasks-part2.md` says the sensitivity runs "on the canon scenario", but `docs/experiments.md` runs it on `base.yaml`, which is what was run. I propose changing T15.2 to "on `scenarios/experiments/base.yaml`".
6. **An output quirk, not fixed.** The first yearly row's control counts include the warm-up, because `o.tally` is only set at yearly rows in `output.go`. The chronicle leaves arrests out because of it, and none of the metrics above use those counts.
7. **The interval for time in a rebellion.** It is a normal interval on a bounded share. Base's runs bunch near 43–48%, which makes the interval narrow (42.2–44.5). The Order's E3 result and part of E5's rest on that. Should these be checked with a nonparametric interval, such as a bootstrap, before the thesis uses them?

T15.5 (freeze model v2) has not been started: it waits for Adam's answers above.
