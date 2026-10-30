# Build plan, part 2: from a stable silo to rebellion

The baseline (part 1, tag `baseline-v1`) answers one question: can the Pact's rules keep a closed silo stable for 200 years? Yes. Part 2 adds what the baseline left out on purpose, so the model can answer the questions this project exists for:

- **Q1 Rebellion:** how often does a rebellion start, and what sets it off?
- **Q2 Death:** can a rebellion kill the silo, and by which route: damage to life support, collapse, or Silo 1?
- **Q3 Survival:** which regime response keeps the silo alive afterwards: crackdown, reform, or wiping memory?
- **Q4 Silo 1:** does Silo 1's threat make Silo 18 more stable or more fragile?

**Every module below exists because one of these questions needs it.** If a feature can't name its question, it doesn't go in. That rule matters more in part 2 than in part 1. With 60+ new parameters, a model can be tuned to produce any rebellion rate you like, and a thesis examiner will ask exactly that. The defences are built into this plan: spec before code, canon patterns as the only calibration targets, and sensitivity analysis before conclusions.

Canon for part 2: the show through the Season 3 finale (S3E10), no books. See `docs/lore-reference.md`, section "Season 3: the system behind the silos".

---

## How part 2 differs from part 1

1. **Spec first, always.** Every milestone starts with a spec task (`T*.0`). It writes the new ODD sections and changes no code. You approve it before any code task in that milestone starts. Spec tasks can be done in Claude chat or in Claude Code plan mode. Either way, the output is a commit to `docs/odd-spec.md` and `scenarios/baseline.yaml`.
2. **The baseline is frozen.** With every part-2 module switched off, `scenarios/baseline.yaml` must produce byte-identical output to `baseline-v1`, so the golden hash doesn't move. New modules get their own RNG streams. New per-person state lives in module-owned slices, not in the `Person` struct.
3. **Every module has an off switch and a null setting** (surveillance 0, no relics, Silo 1 off), and is tested alone before it's combined with the others.
4. **Calibration targets are canon patterns, never outcomes.** The patterns are listed below. Rebellion frequency, death rates and survival are what experiments *measure*. They are never tuned toward a number you'd like to see. (C1 only checks that the rebellion rate is plausible, within a wide band; see ODD §1.2.)
5. **Parameter budget.** Each milestone's summary reports how many parameters it added and how many have a basis other than "guess". Prefer one parameter that does the job over three that look realistic.

Task naming continues from part 1: milestones 7–16, tasks `T7.1` and so on. "How to run a task" and the definition of done from `docs/tasks.md` still apply.

---

## Decisions for Adam

Each one is settled in the spec task named. Decided ones point to the spec; for the rest, my default is in brackets.

| # | Decision | Spec task | Default or decision |
| --- | --- | --- | --- |
| D1 | What counts as a rebellion | T7.1 | **Decided 2026-10-30:** ≥ 5% of adults active for 7 days **and** a collective action; endings suppressed, regime change, split, collapse (ODD §1.2) |
| D2 | What counts as silo death | T7.1 | **Decided 2026-10-30:** collapse = air or water below 0.2 for 30 days, a Safeguard, or population below a *measured* point of no return P_min (T12.6); extinction logged separately (ODD §1.2) |
| D3 | Location: continuous walking or home and work floors | T8.0 | **Decided 2026-10-30:** home floor + work floor, one day location per person per day; trips as multi-day events, no step-by-step walking; a broken segment strands workers cut off from their department (ODD §6.10) |
| D4 | Beliefs: one number or several | T10.0 | [legitimacy (0–1) plus five knowledge flags; no free-text beliefs] |
| D5 | Vitamin D+ in the water: can memories come back? | T10.0 | [yes, when the dosing stops and a relic or a trusted contact cues them, as with Juliette] |
| D6 | How rebels and Judicial fight | T12.0 | [force ratio decides casualties (simple Lanchester rule); no individual combat] |
| D7 | Regime behaviour | T13.0 | [fixed response strategy per scenario (an experimental factor), not a learning regime] |
| D8 | Silo 1 as agents or as a rule set | T14.0 | [a rule set with three settings: `off`, `watch`, `full`; no second simulated population] |
| D9 | Several silos at once | — | [no; only as optional work after M15] |
| D10 | The Pact: prose or rules | T7.1 | **Decided 2026-10-30:** a rule table in the scenario file, no prose (ODD §6.9) |
| D11 | Run length and seeds for part 2 | T7.1 | **Decided 2026-10-30:** 50 + 300 years; 100 seeds for calibration, 1,000 for the final experiments (ODD §1.2) |
| D12 | Viewer technology | T16.0 | [`silosim serve`: Go's standard HTTP server with the page embedded in the binary, plus one vendored JavaScript chart library; read-only] |

---

## Canon patterns (calibration targets for part 2)

Five patterns from the show, checked on `scenarios/canon.yaml` (the regime at the show's settings). They are the only targets part-2 parameters are tuned to. The checks and thresholds live in **ODD §1.2**, in one place, so they can't drift apart:

- **C1 Long peace:** rebellions are rare, but they happen.
- **C2 Visible failures trigger revolts:** a cleaner who walks away matters more than slow hardship.
- **C3 Revolts start Down Deep and use the generator.**
- **C4 Crushed rebellions are forgotten.**
- **C5 A rebellion can end a silo.**

---

## Milestone 7: Foundations

Questions, definitions and the plumbing every later module needs. No model behaviour changes.

- [x] **T7.1 Research questions and definitions (spec).** (2026-10-30, drafted in Claude chat as ODD §1.2 and §6.9) Write ODD §1 v2: Q1–Q4 as measurable outputs, definitions D1–D2, canon patterns C1–C5 with thresholds, and the outcome metrics (time to first rebellion, rebellions per century, outcome of each rebellion, cause of death, survival curve).
  - *Done when:* each question names the output column or report table that answers it.
- [x] **T7.2 Architecture for part 2 (docs only).** (2026-10-30) Update `docs/architecture.md` and `CLAUDE.md` (add `docs/tasks-part2.md` to "Read first", plus the "baseline is frozen" rule). Design: module-owned per-person state (slices indexed by PersonID), spaced phase numbers (100, 200, …) so new steps fit between old ones, a scheduled-event queue for multi-day actions, new output files (`rebellions.jsonl`, outcome fields in `run.json`, `frames.csv.gz`), a performance budget of ≤ 3 minutes for a full run with every module on. **Viewer rule:** everything a UI will show is written to files during the run; nothing outside the simulation reads its memory or changes a run.
- [x] **T7.3 Engine refactor.** (2026-10-30) Spaced phases, the agenda (scheduled-action queue: min-heap by day, then phase, then insertion order, so ties are deterministic), typed part-2 state fields on `World` (nil when a module is off; `docs/architecture.md`, Part-2 state).
  - *Done when:* golden hash unchanged; event-queue tests cover ordering, cancellation and same-tick ties.
- [x] **T7.4 Scripted events in scenarios.** (2026-10-30) `events:` in the YAML injects a shock on a given day: for example `{day: 20000, type: cleaningFailure}` or `{day: 30000, type: vitaminDWater, days: 365}`. Types are added by later milestones; this task builds the mechanism. Shocks you control are how experiments separate cause from effect.
  - *Done when:* an event type registered by a test module fires on the right day, in a fixed phase, and is logged to `events.jsonl`.
- [x] **T7.5 Baseline regression guard.** (2026-10-30) `scenarios/part2-off.yaml` (every part-2 module listed but switched off) must equal `baseline.yaml` byte for byte. Runs in `make check`.
- [x] **T7.6 Frames output.** (2026-10-30) `frames.csv.gz`, a per-floor snapshot in long format (`day, floor, metric, value`), every `output.frameEvery` days (default 30), off by default. Modules register metrics; the baseline registers population, households and average health per floor. This is the data a replay viewer (M16) draws from.
  - *Done when:* with frames off, the golden hash is unchanged; with frames on, a 250-year run's file stays under 50 MB, and the per-floor population summed over floors equals `daily.csv`'s population on the same day.

## Milestone 8: Space and travel

*Answers Q1 and Q2.* Distance decides who meets whom, how fast rebels can gather, and how fast Judicial can reach the Down Deep.

- [x] **T8.0 Spec: location and travel (ODD §6.10).** (2026-10-30, approved) Decision D3. Each department gets work floors (farms and Mechanical already have them; the others get floors in their zone). Travel time = floors crossed × hours per floor (≈ 12 h top to bottom), turned into days at ~8 walking hours a day, so a full descent is about 2 days. Stair segments are intact or broken; a broken segment blocks travel, as at levels 90–92 in S2. Porters cross zones as part of their job.
- [x] **T8.1 Floors and daily location.** (2026-10-30) Work floors per department, each person's home and work floor, daily location (work floor on working days, otherwise home).
  - *Done when:* invariant: every living person has a location on a real floor; workers are on their department's floors.
- [x] **T8.2 Trips and stair segments.** (2026-10-30) Trips as scheduled events with arrival day; 143 stair segments with an intact/broken state; a trip through a broken segment is impossible.
  - *Done when:* 1 → 144 takes 2 days; breaking segment 90–91 makes every Down Deep → Up Top trip impossible (test).

## Milestone 9: Social network

*Answers Q1.* Beliefs spread person to person, so who knows whom decides what people hear.

- [ ] **T9.0 Spec: social network (ODD §6.11).** Ties: partner, household, close kin, master–shadow, coworkers (same department and work floor, up to k), neighbours (same home floor, up to k), and weak cross-zone ties for Porters. Each day a person talks with a few contacts, picked from their ties by tie strength and floor distance.
- [ ] **T9.1 Build and maintain the network.** Built on day 1, updated on birth, death, partnering, moving and job change.
  - *Done when:* invariants: ties are symmetric and only between living people; degree distribution logged yearly.
- [ ] **T9.2 Daily contacts.** k contacts per person per day, sampled by tie weight and distance.
  - *Done when:* the full run stays inside the T7.2 performance budget; contact counts by tie type logged.

## Milestone 10: Beliefs, knowledge and memory

*Answers Q1 and Q3.* Legitimacy is the core of the rebellion model. What people know lowers it; what they forget lets it recover. This is where relics, the Flamekeepers and **Vitamin D+** live.

> **About vitamin D.** In the show, "Vitamin D+" isn't nutrition. It's Silo 1's memory suppressant, disguised as vitamins (S3). Modelling vitamin D as a vitamin would add a health parameter that changes none of the four answers. Modelling it as the memory drug gives the regime its strongest lever. This plan does the second.

- [ ] **T10.0 Spec: beliefs and memory (ODD §6.12).** Decisions D4–D5. Per person:
  - **legitimacy L** (0–1): how much they believe the Pact and the leaders are right
  - **knowledge flags:** the outside may be survivable; the cleaning view is a lie; the true history; other silos exist; the lottery is rigged
  - **risk aversion R** (a fixed trait)

  Rules: L moves down with knowledge and hardship and up with propaganda (Freedom Day, Education); children inherit L partly from their parents and partly from Education; knowledge spreads in contacts, faster between kin; knowledge fades, and dies with the people who held it.
- [ ] **T10.1 Belief state and legitimacy.** Day-1 beliefs, the legitimacy rule, socialisation of children and shadows.
  - *Done when:* with nothing to learn, L stays steady in every zone for 200 years (no drift from the rule itself).
- [ ] **T10.2 Information spread.** Telling and believing in daily contacts; the chance to believe falls with the listener's L.
  - *Done when:* one seeded "knower" gives an S-shaped spread curve; spread through Porters crosses zones faster than without them (test).
- [ ] **T10.3 Memory and forgetting.** A forgetting rate per flag (lived experience fades slower than hearsay); generational turnover.
  - *Done when:* knowledge half-life matches the parameter (statistical test); pattern C4 measured on a scripted crushed rebellion.
- [ ] **T10.4 Relics.** Relics hidden in floors, found by digging and clearing flats, held and shared, confiscated later by Judicial (M11). Holding one raises the chance of learning the true history. A relic also **cues memory** after Vitamin D+, as the Pez dispenser and the watch did for Juliette.
- [ ] **T10.5 Flamekeepers.** A small secret group that keeps the true history alive. It recruits trusted, low-L contacts and loses members to arrests and deaths.
  - *Done when:* without any regime, the group survives generations; with arrests on (M11), it can die out.
- [ ] **T10.6 Vitamin D+.** Two forms: a **targeted dose** for named people, and **water dosing** for everyone, for a set period. While dosed, people lose knowledge flags at a daily rate, and their L is pulled toward the regime's target. After dosing stops, flags can return when cued by a relic or a contact who still knows. Triggered only by scripted events for now; the regime uses it in M13 and Silo 1 orders it in M14.
  - *Done when:* a scripted year of water dosing removes ≥ 90% of a flag; memory comes back after dosing only with cues (test with and without relics).

## Milestone 11: Regime control

*Answers Q1 and Q3.* The deterrence half of the rebellion model: how likely being caught is, and what follows.

- [ ] **T11.0 Spec: surveillance, Judicial, punishment, cleaning (ODD §6.13).** Offences and penalties come from the Pact table (ODD §6.9); this task fixes its exact YAML keys.
  - **Offences:** holding a relic, spreading forbidden knowledge, organising, violence, contact attempts.
  - **Detection:** a chance per offence that rises with IT coverage and the surveillance level (a policy lever).
  - **Judicial and Sheriff:** capacity from coverage; arrests are trips (M8), so a Down Deep arrest takes days.
  - **Sentences:** warning, prison, the mines (forced labour), cleaning.
  - **Covert lottery denial:** flagged people "win" the lottery but their implant stays in.
- [ ] **T11.1 Surveillance and detection.**
- [ ] **T11.2 Arrests, trials and sentences.** Prison and the mines take people out of work and out of contact.
- [ ] **T11.3 Cleaning.** Sentenced cleanings and "I want to go out". A cleaning raises fear in everyone and slightly raises L when the cleaner cleans. A cleaner who refuses and walks out of sight is the **cleaning-failure shock**: everyone watching learns that the outside may be survivable, and L drops across the silo. It is also a scripted event type for experiments (pattern C2).
  - *Done when:* the shock's effect on L and knowledge matches the spec; a normal cleaning lowers dissent for a while (test).
- [ ] **T11.4 Covert lottery denial.** Flagged couples never conceive; their family lines shrink.
  - *Done when:* the share of births to flagged families falls as the spec predicts; logged as an outcome, not tuned.

## Milestone 12: Grievance and rebellion

*Answers Q1 and Q2. This is the core of the thesis.*

- [ ] **T12.0 Spec: grievance, activation and collective action (ODD §6.14).** Decision D6. Based on Epstein's civil violence model (2002), extended with networks and thresholds:
  - **hardship H** from zone hardship (rations), health, losses in the family (denied lottery, cleaned relatives) and being in the Down Deep (scapegoating, S2)
  - **grievance G** = H × (1 − L)
  - **active** when G − R × P(arrest) > threshold, where P(arrest) is estimated from officers and active rebels among the person's contacts and floor
  - **joining** (Granovetter): an active person joins a collective action once enough of their ties have joined
  - **escalation:** unrest → protest or strike → uprising
  - **actions:** strikes in a department (a Mechanical strike cuts power), occupying floors, barricading or breaking stair segments, sabotaging the generator or water pump, clashes with Judicial (force ratio decides casualties)
  - **endings:** suppressed, regime change, split silo, silo death (D2)
- [ ] **T12.1 Hardship and grievance.**
- [ ] **T12.2 Activation and arrest estimate.**
  - *Done when:* reproduces the qualitative result of Epstein's model in a test world: with low legitimacy, outbursts come in bursts when the officer ratio drops.
- [ ] **T12.3 Collective action and thresholds.** Groups per floor and department, joining by threshold, escalation states.
- [ ] **T12.4 Strikes and leverage.** A strike stops a department's output; a Mechanical strike lowers power through the existing loop (§6.6).
- [ ] **T12.5 Clashes and casualties.** Judicial versus rebels by force ratio; deaths, injuries and arrests.
- [ ] **T12.6 Sabotage and collapse.** Stair segments, generator, water pump; the collapse checks of ODD §1.2; the run records the cause and stops (or keeps going to extinction with `output.runAfterCollapse`). **Measure P_min:** start runs from damaged states (population cut to 9,000, 8,000, 7,000 …; Mechanical cut by 10%, 20% …), 100 seeds each; P_min is the highest population at which fewer than 5% of seeds recover above 9,000 within 50 years. Write it into ODD §1.2 and the calibration log.
- [ ] **T12.7 Rebellion log and outcomes.** `rebellions.jsonl`: start, trigger, zone, peak size, actions, casualties, damage, ending.
- [ ] **T12.8 First-results checkpoint.** `scenarios/canon.yaml` with modules 8–12 on, 100 seeds. Check patterns C1–C4. **Stop here if they fail.** Fix rules in the spec, not numbers, then continue. This is the first moment the thesis has real results; write them into the calibration log.

## Milestone 13: Regime response and adaptation

*Answers Q3.* What the regime does during and after a rebellion, and which choice keeps the silo alive.

- [ ] **T13.0 Spec: policy and response (ODD §6.15).** Decision D7.
  - **Policy vector:** surveillance, punishment severity, cleaning rate, ration weights (§6.7), relic confiscation, Vitamin D+ (targeted or water), governance mode and term (§6.8), lottery fairness.
  - **The Order** (S2) as the default playbook: blame Mechanical, raids, propaganda.
  - **Response strategies:** crackdown, reform (raise L, lower hardship, elections), memory wipe (water dosing plus relic destruction), mixed.
  - **Pact rewrite** = a policy change with a legitimacy cost or benefit.
- [ ] **T13.1 Policy vector and scheduled policy changes.**
- [ ] **T13.2 Response strategies,** triggered by rebellion state.
- [ ] **T13.3 Memory-wipe strategy,** using T10.4 and T10.6.
- [ ] **T13.4 Pact rewrites** (ODD §6.9): rows switched, penalties changed, `travelPermits` and `curfew` added, and their effect on L.
  - *Milestone done when:* each strategy runs on 100 seeds, and the survival curves differ by more than seed noise (or the report says plainly that they don't; that's a result too).

## Milestone 14: Silo 1

*Answers Q2 and Q4.* Silo 1 is an outside controller. Its trigger is **contact between silos or escape**, not unrest as such (S3).

- [ ] **T14.0 Spec: Silo 1 (ODD §6.16).** Decision D8. Three settings:
  - `off` — no Silo 1
  - `watch` — Silo 1 sees through IT and gives orders through the Head of IT (more surveillance, doubled Vitamin D+, cleanings), and the Head of IT complies with some probability
  - `full` — adds the Directive and the Safeguard

  Contact attempts are rebel actions: radio (needs someone who knows how), digging toward another silo (months of progress in the mines), and walking out. If threat > threshold, Silo 1 triggers the **Safeguard** (gas through a pipe) after a delay. Capping the pipe defeats it, if someone knows about it and Mechanical acts in time. **Drones** kill anyone outside. After a defeated Safeguard, Silo 1 offers the **deal** from S3E10: no contact, no Safeguard.
- [ ] **T14.1 Algorithm and Head of IT.** Silo 1's view and orders through the Head of IT.
- [ ] **T14.2 Contact attempts.** Radio, digging, escape.
- [ ] **T14.3 Safeguard, capping and drones.** A Safeguard death counts as silo death (D2) with its own cause.
- [ ] **T14.4 The deal.** The offer, the silo's answer (decided by its leaders' L and knowledge), and breaking it.
  - *Milestone done when:* pattern C5 measured; Q4 answered with `off` vs `watch` vs `full` on 100 seeds.

## Milestone 15: Experiments and thesis

- [ ] **T15.1 Outcome report.** `silosim report --outcomes`: per seed and across seeds, with confidence intervals: time to first rebellion, rebellions per century, endings, causes of death, survival curves (Kaplan–Meier).
- [ ] **T15.2 Sweeps.** `silosim sweep` runs a scenario matrix (full factorial or chosen cells) with common random numbers, so differences come from the factor, not noise.
- [ ] **T15.3 Sensitivity analysis.** Morris screening over the part-2 parameters. Which few drive the outcomes? Results go in the calibration log and the thesis.
- [ ] **T15.4 Canon validation.** C1–C5 on 100 seeds per pattern, reported pass or fail.
- [ ] **T15.5 Experiments.** Write each one down before running it: question, scenarios, seeds, the metric that answers it.
  - **E1:** beliefs and control on, no shocks. How long does peace last? (Q1)
  - **E2:** cleaning-failure shock at a fixed day. Trigger effect (Q1, C2)
  - **E3:** crackdown vs reform vs memory wipe after the first rebellion (Q3)
  - **E4:** Silo 1 `off` vs `watch` vs `full` (Q2, Q4)
  - **E5:** appointment vs election, 5-year vs life term (Q3)
- [ ] **T15.6 Freeze model v2.** ODD v2 complete, `scenarios/canon.yaml` frozen, golden hashes regenerated, tag `model-v2`. Thesis figures generated from `runs/` by scripts in `analysis/`.

---

## Milestone 16: Results viewer

A UI for looking at results. It answers no research question by itself, but it's how you'll *see* a rebellion happen, and it makes the thesis defence far easier. It only reads files from `runs/` (the viewer rule in T7.2), so it can be built any time after T7.6. **Start it after T12.8**, when there's something worth looking at.

- [ ] **T16.0 Spec: views and data contract.** Decision D12. Which files each view reads, which metrics and colours; no simulation code.
- [ ] **T16.1 `silosim serve --runs runs/`.** Go standard HTTP server, page embedded in the binary, one vendored chart library (an agreed exception to the dependency rule). Lists runs and batches.
- [ ] **T16.2 Run replay.** The silo as a 144-floor cross-section, each floor coloured by a chosen metric (population, legitimacy, active rebels, arrests, damage); a time slider with play and pause; rebellions and events marked on the timeline; broken stair segments drawn.
- [ ] **T16.3 Run dashboard.** Time series from `daily.csv` (population, health, power, food, hardship and legitimacy by zone) with events overlaid.
- [ ] **T16.4 Experiment view.** For a batch or a sweep: survival curves, rebellions per century and endings with confidence intervals, scenarios side by side.
  - *Milestone done when:* a run with frames on can be replayed from start to collapse or end; the experiment view shows the same numbers as `silosim report --outcomes`.

## Optional, after M15

None of these answers Q1–Q4, so none of them goes in before the experiments are done.

- A live terminal viewer (Bubble Tea) that reads `frames` while a run is going
- An LLM narrator that turns `events.jsonl` and `rebellions.jsonl` into a story of one run
- Several silos at once (Silo 17 and 18 as populations, Silo 1 as an agent)
- The hidden elevator, goods logistics for Porters, chits and a black-market economy

## Left out on purpose

- **Vitamin D as nutrition.** The show's Vitamin D+ is the memory drug (M10, T10.6). Nutritional vitamin D would change no answer.
- **Named characters.** No Juliette agent. Canon events become scripted shocks, not characters.
- **The outside world's physics** (gray goo, the nuclear blast). For people in the silo, the outside is one belief: "it may be survivable".
