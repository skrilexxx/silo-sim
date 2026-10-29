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
