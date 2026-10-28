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
