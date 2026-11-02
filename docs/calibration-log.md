# Calibration log

Every calibration run, in order: what was run, on which code, what came out, and what it means. Numbers here are never edited after the fact; a new run gets a new entry.

## 2026-10-28 — Milestone 1 check (T1.5): demography only

**Code:** branch `t1.5`, on top of `b2adb97` (T1.4).
**Setup:** `scenarios/baseline.yaml` unchanged, but `modules: [demography]` only (deaths, partnering, housing, lottery, reproduction). Health stays 1, so mortality is the pure Gompertz rule. Seeds 1–10; 50 warm-up years, then 200 measured years.
**Command:** `make milestone1` (the test `TestMilestone1` in `internal/modules/demography/milestone_test.go`).

**Targets (docs/tasks.md T1.5):** population within 9,000–11,000 in at least 9 of 10 seeds; median age 30–40.

| Seed | Population (min–max, measured years) | Final | Median age (yearly, min–max) | Births per year (min–max) | Largest yearly change in births |
| ---: | --- | ---: | --- | --- | ---: |
| 1 | 9,719–9,835 | 9,794 | 31.7–33.5 | 124–197 | 35.9% |
| 2 | 9,708–9,820 | 9,762 | 31.8–33.5 | 120–191 | 32.6% |
| 3 | 9,723–9,827 | 9,793 | 31.6–33.2 | 116–189 | 51.7% |
| 4 | 9,724–9,813 | 9,778 | 31.9–33.1 | 119–186 | 47.9% |
| 5 | 9,724–9,818 | 9,779 | 31.7–33.1 | 122–194 | 29.8% |
| 6 | 9,736–9,823 | 9,754 | 31.8–33.6 | 123–178 | 26.5% |
| 7 | 9,721–9,821 | 9,773 | 31.5–33.7 | 122–194 | 46.8% |
| 8 | 9,731–9,825 | 9,801 | 32.0–33.2 | 120–191 | 37.1% |
| 9 | 9,732–9,829 | 9,766 | 31.8–33.8 | 120–190 | 35.8% |
| 10 | 9,729–9,842 | 9,777 | 31.7–33.4 | 113–189 | 39.8% |

**Result: both T1.5 targets pass in 10 of 10 seeds.** No tuning was done.

**Observations**

1. *The population settles about 2% below the target, at ≈ 9,780.* This is how the permit rule (ODD §6.4) is built, not noise: it closes the gap T − P − E, so it aims for population + expected births = 10,000. Births in the pipeline are always about 109 pregnancies (148 births a year × 270/365) plus 0.8 × ≈ 115 open windows ≈ 183, which predicts P ≈ 9,817. That is close to the observed ≈ 9,780. The ODD §1 pattern (within ±10% of 10,000) holds easily.
2. *The ODD §8 target "largest year-to-year change in births under 30%" fails in 8 of 10 seeds (26.5–51.7%).* This is randomness, not booms. With about 148 births a year, even births drawn as pure Poisson noise around a perfectly steady mean give a 200-year maximum yearly change of 36.8% (median of 1,000 simulated runs; 10th–90th percentile 30.8–46.4%). Only 6.5% of such runs stay under 30%, so the target cannot reach 90 of 100 seeds by any tuning at this population size. Per ODD §8 ("if no setting works, the rule is wrong"), the metric needs a spec change before milestone 6. Open question for Adam.
3. *Runtime:* about 60 s for one 250-year run with demography only, when it has a core to itself (171 s with 8 runs sharing 8 cores). The T5.4 budget is under 60 s with all five modules, so T5.4 will need to profile. The obvious first candidate is the daily `math.Exp` in every death draw.

## 2026-10-28 — Births-shock metric changed (ODD §8), milestone 1 rerun

**Why:** the entry above showed that "largest year-to-year change in births under 30%" fails on Poisson noise alone. Agreed with Adam: ODD §8 now measures the *largest change between consecutive 5-year totals of births*, still under 30%. Pure Poisson noise at 148 births a year gives a median of 12.3% on this metric, 19.9% at the 99th percentile, and 25.9% at most over 1,000 simulated 200-year runs, so it fails only on real booms or collapses.

**Code:** branch `spec-births-shock`, on top of `1084e8d` (T1.5). Same setup and command as the entry above (`make milestone1`); the test now also logs the new metric.

| Seed | 5-year births shock | Yearly change (old metric) |
| ---: | ---: | ---: |
| 1 | 10.1% | 35.9% |
| 2 | 11.8% | 32.6% |
| 3 | 10.1% | 51.7% |
| 4 | 11.6% | 47.9% |
| 5 | 13.9% | 29.8% |
| 6 | 11.7% | 26.5% |
| 7 | 13.6% | 46.8% |
| 8 | 14.3% | 37.1% |
| 9 | 16.6% | 35.8% |
| 10 | 12.0% | 39.8% |

**Result: the new births-shock target passes in 10 of 10 seeds (10.1–16.6%).** That range matches what pure noise predicts, so the lottery adds no booms of its own. Population and median age came out identical to the run above, seed for seed, which also confirms that runs are reproducible.

## 2026-10-29 — Resource loop collapses after the first generator failure (T3.2)

**Code:** branch `t3.2`, on top of `e024fa8` (T3.1).
**Setup:** `scenarios/baseline.yaml` unchanged, modules demography, health, work and resources (no rations yet, so no hunger). Seed 42, no warm-up, 100 years, invariants every tick. A throwaway probe test, not kept in the repo.

**What happened:** the silo runs steadily for 44 years (condition 0.800, powerFactor 1, every coverage 1, about 9,780 people). Then the first generator failure hits on day 16,085: c drops to 0.70, followed by a 9-day outage on backup power.

| Days after failure | powerFactor | Air | Avg health | Mechanical cov. | Medical cov. |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 0 | 0.265 | 0.755 | 0.996 | 1.000 | 0.970 |
| 10 | 0.831 | 0.594 | 0.790 | 0.876 | 0.779 |
| 30 | 0.689 | 0.703 | 0.667 | 0.733 | 0.657 |
| 50 | 0.408 | 0.448 | 0.362 | 0.415 | 0.376 |
| 70 | 0.000 | 0.005 | 0.001 | 0.000 | 0.000 |

After that, health stays at 0 for good. The population falls to 643 by year 100. The 1,000-person golden scenario does the same, from its first failure in year 28.

**Why:** ODD §6.6 sets up a feedback loop with gain above 1: health → Mechanical output → power → air → health. After a failure, supply ≤ 1.3 × 0.7 × mechanicalCoverage = 0.91 × coverage, and coverage ≈ min(1, 1.1 × health). Health is in balance where airDamage × (1 − powerFactor) = recoveryRate × medicalCoverage. With 0.05 and 0.01 that gives 0.05 × (1 − 1.0 h) = 0.011 h, so h* ≈ 0.82. Below that, health only falls. An outage of about 8 days or more on backup power (air → 0.26) pushes average health below 0.82. At c = 0.8 the tipping point is h* ≈ 0.73. Because people at health 0 die only 3× faster (k_h = 2), the silo lingers for decades instead of ending at once.

**Open, agreed with Adam:** fix before T3.3, since hunger adds one more loss of health to the same loop. Candidates: airDamage and recoveryRate (both marked "Calibrate"), capacityMargin (a design choice), or a rule change to the loop.

## 2026-10-29 — Outage length set by Mechanical skill and parts (T3.2a)

**Why:** the entry above. The silo collapses after any outage longer than about 8 days. Adam's decision, a rule change rather than tuning: an outage lasts as long as Mechanical needs to fix it, ⌈outageMinDays / (average Mechanical worker skill × partsShare)⌉ days, capped at outageMaxDays. outageMinDays stays 3 (now: the repair time with perfect skill and full parts), and outageMaxDays goes from 14 to 10. partsShare = min(1, parts / partsPerFailure). No other parameter changed. In the show, the generator can stop for only a few days.

**Code:** branch `t3.2a`, on top of `844fb60` (T3.2).
**Setup:** `scenarios/baseline.yaml`, modules demography, health, work and resources. Seeds 1–10, no warm-up, 100 years.
**Command:** `make stability` (the test `TestStability` in `internal/modules/resources/stability_test.go`).

**Target (docs/tasks.md T3.2a):** after every failure, average health is back above 0.9 within a year.

| Seed | Failures | Longest outage | Lowest avg health | Lowest air | Final condition | Final population |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 2 | 4 days | 0.932 | 0.410 | 0.800 | 9,767 |
| 2 | 2 | 4 days | 0.933 | 0.410 | 0.800 | 9,761 |
| 3 | 5 | 4 days | 0.932 | 0.410 | 0.800 | 9,754 |
| 4 | 5 | 4 days | 0.927 | 0.410 | 0.800 | 9,784 |
| 5 | 5 | 4 days | 0.927 | 0.410 | 0.800 | 9,748 |
| 6 | 5 | 4 days | 0.931 | 0.410 | 0.797 | 9,784 |
| 7 | 6 | 4 days | 0.914 | 0.401 | 0.798 | 9,776 |
| 8 | 1 | 4 days | 0.933 | 0.410 | 0.789 | 9,784 |
| 9 | 1 | 4 days | 0.933 | 0.410 | 0.800 | 9,775 |
| 10 | 3 | 4 days | 0.921 | 0.408 | 0.798 | 9,777 |

**Result: every seed survives every failure (10 of 10).** That's 35 failures in 1,000 years, 3.5 per century, inside the ODD §8 target of 1–6.

**Observations**

1. *Every outage lasted 4 days.* The average Mechanical skill stays near 0.85, and Supply keeps parts above 30. So in the baseline the rule behaves like a fixed 4-day outage. It matters only when Mechanical is green (after retraining or heavy losses) or when parts run short. That is when it should matter: a badly run silo gets longer outages and can still tip over (10 days is above the ~8-day tipping point).
2. *The tipping point is still in the loop.* Average health below about 0.82 after a failure still cannot recover. The rule keeps the baseline away from it but doesn't remove it. T3.3 adds hunger to the same loop, so the stability check should be rerun after T3.3.
3. *Runtime:* about 60 s per 100-year run with four modules, 10 runs in parallel.

## 2026-10-29 — Stability rerun with rations and hunger (T3.3)

**Code:** branch `t3.3`, on top of `52bd199` (T3.2a). Rations (ODD §6.7) hand out food, and hunger now feeds health (§6.2).
**Setup and command:** as in the T3.2a entry (`make stability`). The test now also logs rationing days, the lowest intake/need of any zone, and the final food stock in days.

**Result: 10 of 10 seeds survive every failure, with numbers identical to the T3.2a entry, seed for seed** (failures, outages, lowest health, population). Food never runs short. There are 0 rationing days in all 10 seeds, the lowest intake/need is 1.000, and the food stock ends at 13,904–17,164 days.

**Observations**

1. *Food production is far above need.* The stock grows by about 150 days' worth a year, so farms make about 40% more than the silo eats. ODD §7 wants about a 5% surplus (kcalPerFarmUnit, marked "Calibrate"; scaling 16,000 by 1.05 / 1.41 suggests about 12,000). With this surplus, rations and hunger never come into play in the baseline, so the stability check doesn't test the hunger part of the loop yet. That's left for T6.1. Lowering the surplus will put hunger into the loop health → Mechanical → power → food, so `make stability` should be rerun then.
2. *Runtime:* 42–43 s for a 100-year run when it has a core to itself (38 s before rations), and about 195 s with 8 runs sharing the machine.

## 2026-10-29 — First full report: 10 seeds, all modules (T5.3)

**Code:** branch `t5.3` (T5.3, on top of T5.2). **Setup:** `scenarios/baseline.yaml` unchanged, all five modules, seeds 1–10, 50 warm-up + 200 measured years.
**Command:** `bin/silosim batch --scenario scenarios/baseline.yaml --seeds 1-10 --out runs/batch-t53`, then `bin/silosim report --batch runs/batch-t53`.

| Seed | Population | Median age | Births shock | Rationing | Failures / century | Longest < 50% | Mechanical Down Deep (min) |
| ---: | --- | --- | ---: | ---: | ---: | --- | ---: |
| 1 | 9,728–9,824 | 31.5–33.6 | 11.9% | 0.0% | 2.0 | 0 d | 79.9% |
| 2 | 9,720–9,817 | 31.8–33.3 | 9.0% | 0.0% | 2.0 | 0 d | 81.3% |
| 3 | 9,721–9,825 | 31.9–33.1 | 11.7% | 0.0% | 3.0 | 0 d | 80.7% |
| 4 | 9,719–9,828 | 31.5–33.5 | 11.8% | 0.0% | 3.5 | 0 d | 80.5% |
| 5 | 9,721–9,823 | 32.1–33.2 | 9.6% | 0.0% | 4.5 | 0 d | 80.8% |
| 6 | 0–9,817 | 0.0–78.1 | 100% | 0.0% | 52.5 | 949 d (sheriff) | 0.0% |
| 7 | 9,722–9,827 | 31.6–33.7 | 14.0% | 0.0% | 3.0 | 0 d | 80.5% |
| 8 | 9,710–9,831 | 31.9–33.3 | 12.7% | 0.0% | 2.5 | 0 d | 80.5% |
| 9 | 9,715–9,818 | 31.2–34.1 | 9.9% | 0.0% | 2.5 | 0 d | 81.9% |
| 10 | 9,719–9,828 | 31.9–33.2 | 10.2% | 0.0% | 3.5 | 0 d | 81.2% |

**Result:** six patterns pass in 9 or 10 of 10 seeds. "Persistent zones" fails in all 10.

**Observations**

1. *Seed 6 collapses in year 107 (day 39,026).* Two failures came 515 days apart, on days 38,511 and 39,026. The first left c at 0.69, and it had only climbed back to 0.75 when the second dropped it to 0.65. Mechanical coverage had also slipped to 0.955, so even a well-kept generator was giving slightly less power than needed. After the second 4-day outage: powerFactor 0.80 → air 0.73 → health 0.87 → Mechanical 0.84 → powerFactor 0.67, and so on. Health hit 0 by day 39,105. 90 days later the entire workforce took sick retirement (§6.5) on day 39,181, which can't be undone, and the silo emptied. This is the tipping point noted in the T3.2a entry. The 100-year stability check stopped before year 107, so it never saw it. A fix needs a rule or parameter change: Mechanical staffing margin, recovery vs air damage, capacity margin, or whether a whole workforce can sick-retire.
2. *Mechanical workers living Down Deep: about 80–82%, not above 90%.* The share is 100% on day 1, about 82% by the end of warm-up, and stays at 80–85% for the whole run. My unverified explanation is the housing rule (§6.3). A couple whose partners both have zoned departments picks one of the two zones at random, so about half of the Mechanical workers who partner with, say, a Farms worker move out of Down Deep. Cross-zone shadows (5%) add a little more. This needs a spec decision, either on the housing rule or on the target.
3. *No rationing in any seed.* This is the food surplus from the T3.3 entry.
4. *Runtime:* 325 s per run with 10 runs on 8 cores, against 86 s for a run that has a machine to itself. This is for T5.4.

## 2026-10-29 — Stability experiments: capacity margin, air damage, recovery

**Code:** branch `stability`, on top of T5.3, with the two new rules (health 0 means death, sick leave instead of permanent sick retirement; commit `90dd132`).
**Setup:** five settings, each `scenarios/baseline.yaml` with one value changed, seeds 1–10, 50 warm-up + 200 measured years, all modules. Every setting uses the same seeds and random streams, so the settings differ only by the changed value (common random numbers), and seed 6's crisis starts the same way in each.
**Command:** `bin/silosim batch … --seeds 1-10`, then `bin/silosim report`. Margins come from `daily.csv`: the lowest population-weighted average health and the lowest Mechanical coverage over the measured years.

| Setting | Change | Seeds passing patterns 1–6 | Seed 6 | Lowest avg health (other seeds) | Lowest Mechanical coverage |
| --- | --- | ---: | --- | --- | --- |
| A | none (new rules only) | 9 / 10 | collapses, run stops on day 39,100 | 0.911–0.933 | 0.955 |
| B | capacityMargin 1.3 → 1.4 | 10 / 10 | survives (lowest health 0.938) | 0.933–0.941 | 1.000 |
| C | capacityMargin 1.3 → 1.5 | 10 / 10 | survives (0.943) | 0.942–0.944 | 1.000 |
| D | airDamage 0.05 → 0.02 | 10 / 10 | survives (0.996) | 0.996 | 0.979 |
| E | recoveryRate 0.01 → 0.02 | 10 / 10 | survives (0.982) | 0.978–0.982 | 0.949 |

"Persistent zones" fails in every seed of every setting (about 80%), as in the entry above. It doesn't depend on these values.

**Observations**

1. *The two new rules alone don't save seed 6.* Health 0 now kills, so the collapse is fast. The run stops on day 39,100, because so many adults die at once that orphans in Down Deep have no adult left in their zone (§6.1 says the run then stops with an error). Sick leave can't help when nobody recovers.
2. *Capacity margin works through the condition a failure leaves.* After a failure c ≈ 0.7, so the silo keeps full power only if capacityMargin × 0.7 ≥ 1, i.e. capacityMargin ≥ 1.43. With 1.4 a failure still leaves power at 0.98 of demand; with 1.5, at 1.05. It changes no health rule.
3. *Air damage 0.02 almost removes outages from the model:* lowest health 0.996 everywhere, so a failure barely shows in health. This weakens the loop the spec describes (§6.6: "short outages are survivable and long ones are not"), rather than making it stable.
4. *Recovery 0.02 helps health, but Mechanical coverage still dips to 0.95 in three seeds.* Those dips look like staffing, not health, and setting A shows them too.
5. *Only 10 seeds.* Seed 6 is the only one that came near the tipping point. A choice should be checked on 100 seeds (about an hour) before T6.1.

**Open, for Adam:** which knob to change, if any (B or C keep the health rules as they are). And should a mass death that leaves orphans with no adult in their zone still stop the run (§6.1), now that health 0 kills?

## 2026-10-29 — Agreed changes: capacity 1.5, orphans to any zone, Mechanical couples stay Down Deep

**Decisions (Adam):**
- **Capacity margin 1.3 → 1.5** (setting C above). backupPower goes 0.2 → 0.173 to keep backup at its old strength, about 26% of day-1 demand.
- **Orphans:** they go to a household with an adult in any zone when their own zone has none (§6.1).
- **Mechanical couples:** a couple split between Mechanical and another zone's department lives Down Deep always when Mechanical is below its target, and otherwise with `work.mechanicalStayChance` = 0.8 (§6.3).

**Why the zone rule:** a probe at year 50 (seed 42) found 160 of 999 Mechanical workers living outside Down Deep. 155 of them were partnered with a worker of another zoned department (77 Farms, 20 Judicial, 17 Education, 13 IT, 13 Administration, …), and the 50/50 pick had sent them away. Mechanical was at its target then (999 workers, target 985), so the "below target" clause alone rarely fires; the 0.8 does most of the work.

**Code:** branch `stability`. **Setup:** `scenarios/baseline.yaml` with the changes, seeds 1–10, 50 + 200 years. **Command:** `bin/silosim batch … --out runs/exp-final`, then `bin/silosim report --batch runs/exp-final`.

| Seed | Population | Median age | Births shock | Rationing | Failures / century | Longest < 50% | Mechanical Down Deep (min) | Lowest avg health |
| ---: | --- | --- | ---: | ---: | ---: | --- | ---: | ---: |
| 1 | 9,723–9,821 | 31.4–33.8 | 11.8% | 0.0% | 2.0 | 0 d | 92.2% | 0.937 |
| 2 | 9,727–9,816 | 31.8–33.7 | 11.9% | 0.0% | 2.0 | 0 d | 91.4% | 0.937 |
| 3 | 9,723–9,825 | 31.8–33.2 | 17.2% | 0.0% | 3.0 | 0 d | 92.4% | 0.935 |
| 4 | 9,727–9,822 | 31.7–33.3 | 14.3% | 0.0% | 3.5 | 0 d | 92.3% | 0.934 |
| 5 | 9,720–9,823 | 31.8–33.3 | 12.7% | 0.0% | 4.5 | 0 d | 91.9% | 0.937 |
| 6 | 9,725–9,819 | 32.0–33.1 | 11.7% | 0.0% | 4.0 | 0 d | 92.6% | 0.933 |
| 7 | 9,728–9,818 | 31.8–33.3 | 14.2% | 0.0% | 3.0 | 0 d | 92.6% | 0.934 |
| 8 | 9,735–9,821 | 31.8–33.0 | 10.8% | 0.0% | 2.5 | 0 d | 92.4% | 0.936 |
| 9 | 9,730–9,818 | 31.7–33.7 | 15.5% | 0.0% | 2.5 | 0 d | 92.6% | 0.934 |
| 10 | 9,724–9,829 | 31.4–33.5 | 16.7% | 0.0% | 3.5 | 0 d | 91.6% | 0.935 |

**Result: every ODD §8 pattern holds in 10 of 10 seeds.** Seed 6 now survives its two close failures, with lowest average health 0.933.

**Observations**

1. *Persistent zones passes with a thin margin: 91.4–92.6% against the 90% target.* A stay chance of 0.85–0.9 would give more room if 100 seeds show some dips below 90%.
2. *Still no rationing anywhere.* The food surplus from the T3.3 entry remains, so hunger is still untested in the loop. That's for T6.1.
3. *Only 10 seeds.* The ODD §1 definition of done asks for 90 of 100.

## 2026-10-29 — Performance (T5.4)

Not a calibration run. Recorded here because batch times set how many seeds calibration can afford.

**Changes, none of which changes a result:** every output file of a 250-year seed-42 run is byte-identical before and after, and the golden hash is unchanged.
1. **Death chance at health 1 from a table by age in days:** the same formula, computed once per age. About 97% of people are at health exactly 1 on any day, so most of the daily `exp` calls go away.
2. **The dead are removed from `Alive` only on days someone died.**
3. **Health skips the per-person zone lookup when no zone is short of food.**
4. **Work output and skill growth in one pass instead of two.**
5. **`Person` fields reordered** so the ones read every day (health, skill, department, birthday, household, partner, master, flags) share the first 44 bytes, and the struct went from 120 to 96 bytes. Every step visits people in shuffled order, so memory access, not arithmetic, sets the pace.

Tried and dropped: profile-guided optimisation (`default.pgo`), no measurable gain.

**Timings** (one full run, 50 + 200 years, seed 42; `make bench`). This laptop runs the `powersave` governor alongside other programs, so the same binary varies by ±30% (67 s and 98 s back to back).

| When | Old code | New code |
| --- | ---: | ---: |
| Same machine state, back to back | 132 s | 70 s |
| 60-year runs, alternated 3× | 25–36 s | 16–23 s |

The new code is about 1.5–1.9× faster. Against the 90 s the old code took on a quieter machine earlier today, that is about 48–60 s. It still needs one timing on a quiet machine to confirm the under-60 s target.

## 2026-10-29 — T6.1 step 2: food surplus to 5% (kcalPerFarmUnit)

**Why:** ODD §7 wants about a 5% food surplus, but 16,000 kcal per unit of farm output gave 31–39% (stock drift in the 10-seed batch above). So rationing and hunger never came into play.

**Setup:** `scenarios/baseline.yaml` with only `kcalPerFarmUnit` changed, seeds 1–8, 10 warm-up + 50 years (short runs: the surplus settles within years). The surplus is the report's new information column: the drift of the food stock in days over the measured years, production ÷ need − 1, with outage losses included.
**Command:** `bin/silosim batch --scenario <copy> --seeds 1-8 --out runs/cal-kcal-<v>`, then `bin/silosim report`.

| kcalPerFarmUnit | Food surplus, seeds 1–8 | Mean | Rationing days |
| ---: | --- | ---: | ---: |
| 11,900 | 5.3, 4.8, 3.5, 5.9, 4.3, 6.2, 3.0, 4.7% | **4.7%** | 0% in all |
| 12,000 | 6.2, 5.7, 4.4, 6.8, 5.2, 7.1, 3.8, 5.6% | 5.6% | 0% |
| 12,500 | 10.6, 10.1, 8.8, 11.2, 9.6, 11.6, 8.1, 10.0% | 10.0% | 0% |
| 13,000 | 15.1, 14.5, 13.1, 15.7, 13.9, 16.0, 12.5, 14.5% | 14.4% | 0% |

**Chosen: 11,900** (4.7%, closest to 5%). The surplus rises about 0.9 percentage points per 100 kcal, almost linearly.

**Observations**

1. *Still no rationing.* With any surplus above 0 the stock grows year after year, and a 4-day outage costs only about 3 days of food. Rationing (stock below 30 days) can only start after a long run of deficits. So hunger stays out of the loop in a normal run. This is how §6.6–6.7 are built, not a calibration miss.
2. *In these short runs "Rare crises" fails in 4 of 8 seeds:* 50 years hold 0–5 failures, which scales to 0–10 per century. This is a short-run artefact; the 100-seed check uses the full 200 measured years.
3. Lowest average health is the same as at 16,000 (0.934–0.998), so food doesn't touch the stability margin.

## 2026-10-29 — T6.1 step 3: 100 seeds — rationing fails, caused by the Mechanical stay rule

**Code:** branch `t6.1` at `4337c44` (kcalPerFarmUnit 11,900). **Setup:** `scenarios/baseline.yaml`, seeds 1–100, 50 + 200 years. **Command:** `bin/silosim batch --scenario scenarios/baseline.yaml --seeds 1-100 --out runs/cal-100`, then `bin/silosim report --batch runs/cal-100`. All 100 runs finished.

| Pattern | Seeds passing |
| --- | ---: |
| Stable population | 100 / 100 |
| Plausible ages | 100 / 100 |
| No demographic shocks | 100 / 100 |
| **Balanced resources (rationing < 5% of days)** | **89 / 100: FAIL** |
| Rare crises (1–6 failures per century) | 96 / 100 |
| Staffed departments | 100 / 100 |
| Persistent zones | 100 / 100 |

The 11 rationing seeds ration on 5.8–18.0% of days. The food surplus over the full measured run averages 0.8% (−1.4% to 5.5%), not the 4.7% of the step-2 sweep. Lowest average health stays at 0.917 or above in every seed, so hunger now enters the loop without tipping it. The 4 failure-rate misses (0.5, 6.5, 7.0, 7.0 per century) are Poisson noise around 3.5.

**Why the surplus is lower: Farms loses workers to Mechanical over the decades.**

1. *The step-2 sweep measured years 10–60 only.* Production per need falls later in a run. At 16,000 kcal (the earlier `exp-final` batch, no rationing) the surplus was about 40% in years 60–110 and about 30% in years 160–250. The short window overstated the long-run surplus. That was a mistake in step 2.
2. *The fall comes from staffing.* Farms workers slide from about 1,860 (year 60) to 1,670–1,740 (year 250) while the target stays near 1,870. Meanwhile Mechanical grows to 1,257–1,378 against a target of about 1,000. In the batch without the stay rule (capacity 1.5, `exp-C`) both departments stay at their targets: Farms 1,878–2,017, Mechanical 977–1,039 at year 250.
3. *Down Deep grows.* With the stay rule, Down Deep goes from 22% of the population to 31–33% by year 250, and Up Top and the Mids shrink. Without it Down Deep stays at 20.5–22%.

**Mechanism:** the stay rule keeps 80% of split couples Down Deep even when Mechanical is well over target. Their children grow up there, and a Down Deep child's placement can only choose Mechanical, General and Porters (apart from the 5% cross-zone chance and following a parent), so most go to Mechanical. Each generation of Down Deep families is larger than the one before. The rule fixed "Mechanical workers live Down Deep" but drains Farms and inflates Down Deep.

**Open, for Adam:** how to change the stay rule (see the T6.1 discussion). The food value should be calibrated again on the full 200 measured years once staffing is fixed.

## 2026-10-29 — Stay rule only while Mechanical is short: staffing fixed, zones fail again

**Change (Adam):** a couple split between Mechanical and another zone's department goes Down Deep only while Mechanical is below its target; otherwise 50/50 like any other couple. `work.mechanicalStayChance` is removed. kcalPerFarmUnit stays 11,900.
**Setup:** seeds 1–10, 50 + 200 years; `bin/silosim batch --scenario scenarios/baseline.yaml --seeds 1-10 --out runs/cal-stay`.

| | 80% stay rule (100-seed run) | Stay only while short (this run) |
| --- | --- | --- |
| Farms workers / target, year 250 | 1,670–1,740 / ~1,870 | 1,823–2,026 / ~1,870 |
| Mechanical workers / target, year 250 | 1,257–1,378 / ~1,000 | 992–1,067 / ~1,000 |
| Down Deep share of population, year 250 | 31–33% | 21.3–25.7% |
| Food surplus over the full run | 0.8% | 5.5% (2.1–9.1%) |
| Seeds with rationing < 5% of days | 89 / 100 | 10 / 10 (0 days) |
| Mechanical living Down Deep (min) | 91–93% | 80.5–82.5%: fails 10 / 10 |

Every other pattern passes in 10/10.

**Reading:** with departments balanced, kcalPerFarmUnit 11,900 gives the intended ~5% surplus over the full run. But Mechanical is rarely below target, so the rule hardly ever fires and the share falls back to about 81%, as under the plain 50/50 pick.

**The tension:** keeping Mechanical's people Down Deep means their families live there, and Down Deep children can mostly only become Mechanical, General or Porters (§6.5 placement). When no department in reach is short, a child's pick falls back to the departments' shares in their own zone, so a growing Down Deep overfills Mechanical. Free partnering (§6.3) and zone-bound placement (§6.5) together cap the share near 81% unless Down Deep grows.

**Open, for Adam:** change placement so children go where the jobs are, or keep the rule and revise the 90% target.

## 2026-10-29 — 80% stay rule plus "children go where the jobs are" placement (not committed)

**Change (Adam):** the 80% stay rule restored. In §6.5 placement, a child not following a parent whose reachable departments are all at or above target now picks a short department (with room) in the other set: tied to another zone, or their own. Only when nothing anywhere is short do their own zone's job shares weigh the pick.
**Setup:** seeds 1–10, 50 + 200 years, kcalPerFarmUnit 11,900; `--out runs/cal-place`.

**Report:** every §8 pattern passes in 10/10. Mechanical living Down Deep: 91.6–93.0%. Rationing: 0% in 8 seeds, 0.7% and 2.0% in two.

| Year 250 | 80% stay rule alone (100-seed run) | 80% stay + placement change |
| --- | --- | --- |
| Mechanical vs target | +26% to +38% | +19% to +30% |
| Farms vs target | −7% to −11% | −4% to −10% |
| Down Deep share of population | 31–33% | 28.3–30.7% |
| Food surplus, full run | 0.8% | −1.1% to 2.5% (mean 0.6%) |

**Reading:** the placement change only affects the 40% of children who don't follow a parent. Mechanical still overfills, through children following a Mechanical parent (60%, whenever a free master exists, and an overfull Mechanical has many). The §8 targets don't measure overfilling, so they pass anyway. `TestFiftyYears` (T2.1, demography and work only) now ends at Porters +16.5%, outside its ±15% band.

**Open, for Adam:** also limit following a parent to departments below target, or go back to "stay only while short" and revise the 90% target.

## 2026-10-29 — Decision: balanced staffing, persistent-zones target revised to 75%

**Decision (Adam):** keep "stay only while Mechanical is short" (committed in `977f78d`) and drop the 80% stay rule and the placement experiment. Change the ODD §8 persistent-zones target from "above 90%" to "above 75%" of Mechanical workers living Down Deep.

**Why:** partners are chosen freely (§6.3), so about one Mechanical worker in five lives with a partner from another zone, and the share settles near 81%. Every rule tried that pushed it above 90% (80% stay; 80% stay plus job-seeking placement) did so by growing the Down Deep to 28–33% of the silo and overfilling Mechanical by 19–38% at Farms' expense, which in turn starved the food supply. With the balanced rule, both departments stay on target, the Down Deep stays near 22%, and the 5% food surplus holds without rationing. 75% still means Mechanical clearly stays a Down Deep department.

**Next:** the 100-seed confirmation of this version (T6.1 step 3, repeated).

## 2026-10-30 — T6.1 done: 100 seeds, every pattern holds

**Code:** branch `t6.1` at `c80cb10`: health 0 means death, sick leave, capacity 1.5 with backup 0.173, orphans to any zone, Mechanical couples stay Down Deep while Mechanical is short, kcalPerFarmUnit 11,900, persistent-zones target 75%. **Setup:** `scenarios/baseline.yaml`, seeds 1–100, 50 + 200 years. **Command:** `bin/silosim batch --scenario scenarios/baseline.yaml --seeds 1-100 --out runs/cal-100b`, then `bin/silosim report --batch runs/cal-100b`. All 100 runs finished. The laptop was suspended once during the batch; that only lengthens the wall-clock runtimes in `run.json`, not the results.

| Pattern | Target | Seeds passing |
| --- | --- | ---: |
| Stable population | 9,000–11,000 every day | 100 / 100 |
| Plausible ages | median age 30–40 every year | 100 / 100 |
| No demographic shocks | 5-year births change < 30% | 100 / 100 |
| Balanced resources | rationing < 5% of days | 100 / 100 |
| Rare crises | 1–6 failures per century | 96 / 100 |
| Staffed departments | never < 50% of target for > 1 year | 100 / 100 |
| Persistent zones | Mechanical living Down Deep > 75% every year | 100 / 100 |

**Across the 100 seeds:**
- **Mechanical living Down Deep:** 77.9–83.8% (mean 81.1%).
- **Food surplus over the full run:** mean 4.3% (−0.8% to 9.1%). One seed rationed, on 2.6% of days.
- **Failures:** 3.07 per century on average. The four misses (0.5, 6.5, 7.0, 7.0) are what Poisson noise around 3 gives over 200 years.
- **Lowest average health in any seed:** 0.925, far from the ~0.82 tipping point.

**Result: T6.1 done.** Every ODD §8 calibration target holds in at least 96 of 100 seeds.

**Parameters marked "Calibrate" and left at their defaults,** because every target passes with them: hungerDamage 0.02, airDamage 0.05, recoveryRate 0.01, needFactor 0.84, searchRate 0.0009, sameSexWeight 0.05, generatorWear 0.0004, repairRate 0.002, failureBase 0.002. The one tuned value is kcalPerFarmUnit (16,000 → 11,900, 5% surplus). The other changes made during calibration were rule or design decisions, each recorded in the entries above: health 0 means death, sick leave, capacity 1.5 and backup 0.173, orphans to any zone, the Mechanical stay rule, and the 75% target.

## 2026-10-31 — Pattern C4 measured on a scripted crushed rebellion (T10.3)

**Code:** T10.3 (beliefs: upbringing, spread, forgetting), on top of `26c38d4`.
**Setup:** `scenarios/baseline.yaml` with `modules` plus `space, network, beliefs` at their defaults (π = 0.002, T_heard 5 years, T_lived 20 years, p_s 0.5); 5 warm-up years, 45 measured; `revealHistory` on day 2000 (the first measured year): everyone aged 14+ learns the history as lived. No regime: nothing is punished, dosed or confiscated (M11, M13 not built). Seeds 1–5.

**Target (ODD §1.2, C4):** 40 years after a suppressed rebellion, fewer than 10% of adults know the history.

| Years after the reveal | Adults who know the history, seeds 1–5 | Mean | Mean L of adults |
| ---: | --- | ---: | ---: |
| 0.5 | 0.978 0.982 0.982 0.983 0.980 | 0.981 | 0.652 |
| 5.5 | 0.908 0.905 0.908 0.904 0.903 | 0.906 | 0.656 |
| 10.5 | 0.863 0.863 0.867 0.859 0.869 | 0.864 | 0.655 |
| 20.5 | 0.824 0.817 0.817 0.818 0.822 | 0.820 | 0.648 |
| 30.5 | 0.793 0.794 0.796 0.792 0.798 | 0.795 | 0.637 |
| 40.5 | 0.796 0.800 0.794 0.799 0.798 | 0.798 | 0.620 |

**Result: C4 fails, as expected without a regime.** Knowledge of the history settles near 80% instead of dying out. Every knower passes it on to about π × (1 − L) × conversations a day ≈ 0.002 × 0.35 × 3.5 ≈ 0.0025 people a day, against losing it at 0.00038 a day (heard) or 0.0001 (lived): each knower replaces themself several times over, so the history is endemic. Mean L drifts down (0.65 → 0.62) as children grow up with doubting parents, the same in all three zones.

**What it means:** in this model, forgetting alone doesn't erase a rebellion; it takes the regime (arrests and the speech rule, M11; dosing and confiscation, M13) or a much smaller π. C4 is the target for π and p_s together in T12.8, with the regime on. Nothing was tuned here.

**Cost:** 80% of the silo knowing a flag adds about 150 µs a day to a run (about 19 s over 350 years), after drawing the rare passing events first (T10.3); asking every tie whether it was used had cost 1 ms a day.

## 2026-10-31 — Relics alone, no regime (T10.4)

**Code:** T10.4 (relics found, held hidden, handed over, inherited), on top of `39ac777`.
**Setup:** `scenarios/baseline.yaml` plus `space, network, beliefs` at their defaults (100 relics, T_find 50 years, T_keep 10 years, ρ 0.05, π 0.002); 5 warm-up years, 150 measured; no scripted events, no regime. Seed 1.

| Measured year | Relics hidden / held | Adults who know the history | Mean L of adults |
| ---: | --- | ---: | ---: |
| 1 | 93 / 7 | 0.003 | 0.800 |
| 11 | 78 / 22 | 0.031 | 0.795 |
| 21 | 60 / 40 | 0.234 | 0.763 |
| 31 | 54 / 46 | 0.632 | 0.696 |
| 51 | 44 / 56 | 0.752 | 0.651 |
| 101 | 30 / 70 | 0.807 | 0.585 |
| 150 | 31 / 69 | 0.824 | 0.560 |

**What it means:** a few found relics are enough to start the history spreading; with nothing to stop it, it is endemic within about 30 years (as after a reveal, T10.3) and settles near 80%. Relics settle at about two thirds held: finds (2 a year at first) against relics left in empty flats at death. Mean L keeps falling for a century, as each generation is raised by doubting parents: the base converges to about 0.8 − 0.15 × (flags its parents know). Nothing was tuned; the regime (M11, M13) is what should hold this down.

## 2026-10-31 — A year of water dosing after a reveal (T10.6)

**Code:** T10.6 (Vitamin D+), on top of `eb1f04f`.
**Setup:** as the C4 run of T10.3 (`scenarios/baseline.yaml` plus `space, network, beliefs` at their defaults, 5 + 45 years), with `revealHistory` on day 2000 and `waterDosing` (365 days, d = 0.01, κ = 0.05) on day 3825, five years later. No regime otherwise. Seeds 1–3.

| Years after the reveal | Adults who know the history | Adults with a hidden flag | Dosed | Flamekeepers |
| ---: | --- | --- | ---: | ---: |
| 4.5 | 0.919 0.912 0.917 | 0 | 0 | 15 |
| 5.5 (dosing) | 0.131 0.133 0.129 | 0.769 0.767 0.773 | 9,779 | 7 |
| 6.5 | 0.894 0.894 0.898 | 0.001 0 0 | 0 | 0 |
| 10.5 | 0.864 0.869 0.869 | 0 | 0 | 0 |
| 40.5 | 0.793 0.804 0.795 | 0 | 0 | 0 |

**What it means:** water dosing alone is a short-lived wipe. Within half a year of its end, the few who still knew (those whose flag was never hidden, relic holders, the newly told) cue everyone else, and recall spreads like news, but at κ = 0.05 a conversation, 25 times faster than π. The level then follows the same course as without dosing. The Flamekeepers do not come back: their history was hidden, they lost their seats, and with no members left the group is gone for good. So, in this model, a wipe that lasts needs the cues removed too (relics confiscated, knowers silenced), which is what the memory-wipe strategy of M13 combines; and dosing is the one lever that ends the Flamekeepers. Nothing was tuned.

## 2026-10-31 — The same wipe, with the Flamekeepers warned by insiders

**Code:** T10.6 plus the warning rule (ODD §6.12, Warning the Flamekeepers; Adam's choice): on the day a water dosing starts, the group is warned if a member works in IT or Medical, or has a strong tie to a worker or shadow there who knows the history; warned, members and their households are not dosed by the water.
**Setup:** as the previous entry, seeds 1–3.

| Years after the reveal | Adults who know the history | Flamekeepers |
| ---: | --- | --- |
| 4.5 | 0.919 0.912 0.917 | 15 15 15 |
| 5.5 (dosing) | 0.136 0.126 0.125 | 15 15 15 |
| 6.5 | 0.895 0.899 0.893 | 15 15 15 |
| 40.5 | 0.796 0.793 0.805 | 15 15 15 |

**What it means:** with the history known by 92% when the dosing starts, some member always has a doubting insider among their strong ties, so the group is warned in every seed and comes through whole. Where the history is rare, as C4 wants after a crushed rebellion, such insiders are rare too, and a wipe can end the group; that case is the one T12.8 and M13 will see.

## 2026-10-31 — Regime control at the canon settings, and what broke on the way (M11)

**Code:** T11.1–T11.4, with the four rule changes they led to (all Adam's choices, ODD §6.9, §6.13): an arrested Flamekeeper names one member at most; the Head of IT holds the post until death; fear F silences talk over weak ties; forbidden talk flags only on a repeat offence, the flag covers the offender and their partner, and a denied win uses up no permit.
**Setup:** `scenarios/baseline.yaml` with `space, network, beliefs, control` at their defaults (σ = 0.05), 60 years, no warm-up, seed 1, 10,000 people; then 1,000-person silos (target 1,000) at other σ, seed 42.

What each rule change fixed, measured before it:

| Rule as first written | What happened |
| --- | --- |
| A caught Flamekeeper named each other member with p\_name = 0.3 | About 4 of 14 named per arrest: all 15 gone in year 1 of every run |
| A retiring Head of IT became an ordinary teller | All five flags at about 90% of adults within 50 years; mean L 0.06 |
| Telling unaffected by punishment | History at 94% of adults, hundreds of offences a year |
| A first warning for talk flagged the whole household for life | 51% of the silo flagged in 56 years (full size) |
| A denied win used up a permit | σ = 0.3: no births after year 50, the 1,000-person silo extinct in years 61–70 |

With all the changes, full size, σ = 0.05 (population 9,770–9,790 throughout):

| Year | Arrests a year | Held | Fear F | Know history | Know lottery | Mean L | Flagged |
| ---: | ---: | ---: | --- | --- | --- | --- | --- |
| 20 | 36 | 3 | 0.15 | 0.13 | 0.005 | 0.78 | 0.6% |
| 40 | 152 | 21 | 0.66 | 0.63 | 0.45 | 0.62 | 6.3% |
| 60 | 137 | 23 | 0.59 | 0.81 | 0.81 | 0.48 | 14.6% |

1,000-person silo, 100 years: at σ = 0.1 a third of the people are flagged, births to flagged parents are 1.9% of births, and the population never falls below 964. At σ = 0.3 two thirds are flagged, too few couples can have children, the population falls to about 735, Mechanical coverage drifts below 0.75 (staffing needs stay sized for 1,000), and power, water and air fail together in year 87. At σ = 1 so many workers are in prison that Farms fail: food runs out, and the silo starves in year 42.

**What it means:** fear slows the spread of knowledge but does not stop it: the history still reaches 80% in 60 years, and the lottery secret, starting from the two implant doctors, spreads as fast. Holding knowledge down (C4) is π's and forgetting's job, to calibrate in T12.8, and the regime's responses' (M13). A crackdown can kill the silo two ways the baseline already allowed for: imprisoned workers starve it, and denied births shrink it below what its departments need. Both are results for Q3, not bugs. A collapse still ends a run with demography's "no household with an adult" error, once the last adults die; the collapse rule of §1.2 (T12.6) should stop the run first. Nothing was tuned.

## 2026-11-01 — Point of no return P_min (T12.6)

**Code:** `7293b6b` (T12.6: sabotage, stair repairs and collapse).
**Setup:** `scenarios/baseline.yaml` with the baseline modules only (demography, health, work, resources, governance), so dissent plays no part. 50 warm-up years; on the first measured day (18,251) a scripted `cull` (ODD §6.1) kills a share of everyone, or of Mechanical's workers, drawn uniformly; then 50 measured years. 100 seeds per state (1–100).
**Command:** `analysis/pmin.sh` (1,600 runs, about 4 hours on 8 cores).
**Rule (ODD §1.2):** a seed *recovers* if the population is above 9,000 on some day of the 50 years; P\_min is the highest population at which fewer than 5% of seeds recover.

Population cuts:

| Cut | Population after (mean) | Recovered | Died out | Mean lowest | Mean final |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 10% | 8,794 | 100% | 1% | 8,694 | 9,683 |
| 20% | 7,817 | 99% | 30% | 5,464 | 6,858 |
| 25% | 7,328 | 80% | 49% | 3,730 | 5,001 |
| 30% | 6,840 | 42% | 58% | 2,868 | 4,124 |
| 35% | 6,351 | 22% | 78% | 1,395 | 2,162 |
| 40% | 5,863 | 10% | 90% | 586 | 984 |
| 50% | 4,885 | 0% | 100% | 0 | 0 |
| 60–90% | 3,909–977 | 0% | 100% | 0 | 0 |

Mechanical cuts (a robustness table, not part of P\_min; the population starts above 9,000, so "recovered" says nothing here):

| Mechanical workers cut | Population after (mean) | Died out | Mean lowest | Mean final |
| ---: | ---: | ---: | ---: | ---: |
| 10% | 9,669 | 0% | 9,656 | 9,774 |
| 20% | 9,568 | 4% | 9,170 | 9,384 |
| 30% | 9,466 | 20% | 7,558 | 7,818 |
| 40% | 9,364 | 44% | 5,231 | 5,473 |
| 50% | 9,261 | 100% | 0 | 0 |

**Result: P\_min = 4,900** (Adam, 2026-11-01): at 4,885 no seed recovers, at 5,863 10% do. The true value lies between the two; the grid was not refined. The recovery test of §1.2 is kept as written.

**Observations**

1. *The silo is far more fragile than its last few hundred people.* A cut of a fifth kills 30% of seeds within 50 years, half of Mechanical's workers (about 8% of the people) kills all of them. Every department's coverage need is fixed at its day-1 size (ODD §6.6), so a smaller silo can't staff Mechanical and Farms; power, food and air fall, health falls, and deaths outrun the lottery.
2. *Above P\_min lies a wide risk zone the test does not catch.* Many seeds pass 9,000 again and still die out later (cut by 20%: 99% recover, 30% die). Those deaths end as extinction, which always ends a run (ODD §1.2), not as a collapse.
3. *P\_min holds for a silo of 10,000.* Test silos of 1,000 set it to 0; another size needs its own measurement.

## 2026-11-01 — First results: the canon scenario, 100 seeds (T12.8)

**Code:** `8f0db73` (report with C1–C4), plus `scenarios/canon.yaml` and `analysis/canon.sh` (not yet committed).
**Setup:** `scenarios/canon.yaml`: the baseline plus space, network, beliefs, control and rebellion, every setting at its default (the show's settings: the canon Pact, σ = 0.05); 50 warm-up and 300 measured years; one scripted `cleaningFailure` on day 73,000 (measured year 150) for C2 (Adam). Seeds 1–100.
**Commands:** `bin/silosim batch --scenario scenarios/canon.yaml --seeds 1-100 --out runs/t12.8-canon`, then `bin/silosim report --batch runs/t12.8-canon` and `analysis/canon.sh runs/t12.8-canon`. About 3.4 hours on 8 cores: 17 minutes per run in the batch, 4 min 18 s for one run alone (the performance budget of `docs/architecture.md` is 3 minutes).

**Result: C1–C4 all fail, because nothing rebels.** No rebellion in 30,000 silo-years; 96 unrest spells; no collapse. The §8 baseline targets all still hold (population 9,700–9,850).

| Pattern | Measured | |
| --- | --- | --- |
| C1 rate 0.3–2 per century | 0 in 30,000 years | FAIL |
| C2 ≥ 3× after a scripted failure | 0/100 after vs 0/5,703 without | FAIL (nothing to compare) |
| C3 Down Deep, Mechanical strike | no rebellions | FAIL (nothing to judge) |
| C4 < 10% know the history 40 years on | no suppressed rebellions | FAIL (nothing to judge); the history is known by 73% of adults in measured year 1 and 89% at the end |

The peak share of the population active in a run: median 0.95%, highest 1.8% (about 1.3% and 2.5% of adults), against the rebellion line of 5% of adults. The peak usually falls on the day of the scripted failure. Almost nobody joins (at most 5 a day) and nobody turns violent.

Means over the 100 runs, from yearly.csv:

| Measured year | H Up Top / Mids / Down Deep | G Up Top / Mids / Down Deep | Fear F | Arrests a year | Know history | Mean L |
| ---: | --- | --- | ---: | ---: | ---: | ---: |
| 100 | 0.004 / 0.004 / 0.202 | 0.002 / 0.003 / 0.139 | 0.970 | 164 | 0.820 | 0.312 |
| 149 | 0.004 / 0.004 / 0.204 | 0.003 / 0.003 / 0.144 | 0.972 | 167 | 0.825 | 0.290 |
| 151 | 0.004 / 0.004 / 0.204 | 0.004 / 0.004 / 0.195 | 0.993 | 385 | 0.856 | 0.035 |
| 200 | 0.006 / 0.005 / 0.204 | 0.005 / 0.005 / 0.199 | 0.996 | 295 | 0.882 | 0.017 |
| 300 | 0.008 / 0.007 / 0.206 | 0.008 / 0.007 / 0.202 | 0.996 | 300 | 0.885 | 0.011 |

Cleanings: 0.03 a year (0.01–0.08 by run), inside σ's plausibility band of 0–2 (ODD §7); 51 failed cleanings in the 100 runs.

**Why nothing rebels (three rules, not numbers):**

1. *Fear is saturated, so Epstein's tipping is switched off.* F sits at 0.97–1.0 all run: about 165 trials a year (300 after the failure) × φ = 0.01 × the rules' visibility, against a half-life of 1 year, adds more than 1 a year. The net risk N = R × (1 − (1 − P)(1 − F)) is then R whatever the patrol, so the arrest estimate P, the mechanism by which more actives make it safer to join (§6.14), does nothing. Most trials are for talk, because the history is endemic (point 3).
2. *Grievance has nowhere to come from but the Down Deep.* In a fed silo H is 0 Up Top and in the Mids and s\_DD = 0.2 Down Deep (§6.14); losses are rare and rationing never happens. Even with L near 0 after the failure, G ≤ 0.2, so with N = R only Down Deep adults with R below about 0.1 are active: about a tenth of a fifth of the silo, 2% of adults at most. That is unrest, never the 5% a rebellion needs.
3. *The history is endemic before any rebellion.* From 15 Flamekeepers and 100 relics on day 1, it is known by 73% of adults after the warm-up (each knower tells about 5 others before forgetting, R₀ ≈ 5; T10.3, T10.4). A rebellion teaches nobody anything (§6.14), so C4 would measure this background level, not the forgetting of a rebellion, and it can't pass while R₀ > 1. M13's regime responses alone can't hold it down: from 1% it regrows to near 80% within about 30 years (T10.4's growth).

The three are linked: the endemic history keeps L low, but also keeps the talk offences coming, which keeps fear at its ceiling, which stops the low L from ever turning into a rebellion.

**What it means:** at the show's settings the model gives a silo that doubts everything and never rises: the regime's fear, not legitimacy, keeps the peace. Per T12.8, the run stops here; the fix is a rule change in the spec (Adam), not a number. Nothing was tuned.

## 2026-11-02 — Diagnostic: the canon scenario with fear off (T12.8)

**Setup:** `scenarios/canon.yaml` with `control.fearPerTrial: 0` (φ = 0, so F stays 0), seeds 1–8. A diagnostic, not a calibration: it bounds what any change to fear could do.

| | Canon (fear on) | Fear off |
| --- | --- | --- |
| Rebellions | 0 in 30,000 years | 0 in 2,400 years |
| Peak share of the population active, before / after the failure | ~0.2% / ~1.0% | 0.16–0.23% / 0.8–1.1% |
| Most joined on a day, most strikers on a day | 5, — | 35, 32 (a Mechanical strike needs about 95) |
| Adults knowing the history, mean L, at the end | 0.885, 0.011 | 0.934, 0.004 |

**Result:** fear is not what keeps the canon silo quiet. Without it the net risk is R × P, and P stays high because the Down Deep's patrol (about 90) outnumbers its actives (about 100): grievance is at most 0.2, and only Down Deep, so only its least risk-averse adults come out, too few to tip the patrol. The fear change drafted for T12.8 was withdrawn (Adam); the binding limits are hardship and an already-low legitimacy.

## 2026-11-02 — Smoke run of the T12.8 rule changes, seed 1

**Code:** uncommitted T12.8 rule changes (Adam, 2026-11-02): own hardship h\_own ~ U(0,1) in H (§6.14); only doubters tell, π × (1 − L\_teller) × (1 − L\_listener) (§6.12); everyone 14+ learns the history when a rebellion ends; C4 judged from M13 on. **Setup:** `scenarios/canon.yaml`, seed 1, 5.4 minutes.

| | Before the changes | After |
| --- | --- | --- |
| Know history / mean L, measured years 1–149 | 0.73–0.83 / 0.54–0.29 | 0.01–0.02 / 0.80 |
| Peak active before the failure | 0.2% | 0.4% |
| Rebellions | none | 3: day 73,000 (the failure, 251 days, suppressed), day 73,281 (425 days, suppressed), day 113,071 (no trigger, 13 years, collapse by air) |
| Fear, arrests a year before the failure | 0.97, 165 | 0.99, 170 |
| Cleanings a year | 0.03 | 1.2 |

Every rebellion started in the Mids (the largest zone), and no critical department struck: at most 295 strikers, spread over departments. The second rebellion was an *echo*: the first one's end taught everyone the history, L fell 0.15, and it started 30 days later. Adam moved the teaching rule to M13, to be settled with the memory wipe (T13.0). The third ended in the life-support spiral (T13.0), with 18,926 arrests over its 13 years.

## 2026-11-02 — Second canon batch, after the rule changes (T12.8)

**Code:** `79bae6c` (own hardship, doubting tellers, C4 from M13), plus the report fix for runs that collapse in warm-up. **Setup:** `scenarios/canon.yaml`, seeds 1–100, `runs/t12.8-canon-2`; about 3.5 hours (953 s per run in the batch). **Commands:** as the first batch.

| Pattern | Measured | |
| --- | --- | --- |
| C1 0.3–2 rebellions per century | 0.92 (223 in 24,292 measured years) | PASS |
| C2 ≥ 3× after a scripted failure | 98/98 windows after vs 22/3,997 without (182×) | PASS |
| C3 ≥ 60% start Down Deep | 0/223: every one starts in the Mids; a critical department struck in 16 (Mechanical 11) | FAIL |
| C4 (information until M13) | 83 of 108 judged under 10% (77%) | — |

**The peace breaks once and doesn't come back.** Before the scripted failure: 2 rebellions in 14,832 silo-years (0.013 per century), L 0.80, the history known by 1–2%. After it: 221 in 9,461 (2.3 per century), L 0.50 then about 0.6, as failed cleanings (652 in the batch, mostly after the shock) keep teaching the outside and the screen. C1's pooled 0.92 is the average of the two, not a long peace with rare revolts.

**Three silos in four die.** 74 collapses (air 66, water 7; one in warm-up year 23, seed 32, before any measured day), at a median measured year of 224, about 75 years after the failure; 26 runs reach year 300. 61 rebellions end in collapse, all but 3 of them later ones with no trigger. A rebellion lasts a median 245 days (90th percentile 1,485, longest 10,569) with about 2,750 arrests: workers held by Judicial take Mechanical below the air line, the spiral parked for M13 (T13.0). The §8 targets now fail on population (27%) and rationing (26%) for the same reason.

**Why C3 fails:** h\_own is the same everywhere, so the Down Deep's grievance is only 1.2 times the Mids' (G 0.30 vs 0.25 right after the failure), while the Mids have twice as many adults. The starting zone counts heads, so the Mids win every time.

Unrest: 6,644 spells, about 66 a run. Fear stays at 0.99. Cleanings 0.52 a year (0–1.13), inside σ's band. Nothing was tuned.
