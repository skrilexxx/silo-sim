# Silo Model — ODD Spec v2 draft (baseline-v1 + part 2)

Oct 30, 2026 · Adam Kopřiva · sections 1.1 and 2–8 are frozen with `scenarios/baseline.yaml` at tag `baseline-v1` (changes since v0.1 are in docs/calibration-log.md); sections marked "draft" belong to part 2 (`docs/tasks-part2.md`) and change milestone by milestone

## 1. Purpose and patterns

### 1.1 Baseline

The baseline simulates Silo 18 running normally under the Pact, with no dissent, no cleanings and no rebellion. It is the control group: every later experiment is measured against it.

**Question the baseline answers:** can the Pact's population and resource rules keep a closed silo of about 10,000 people stable for 200 years?

**Patterns it must reproduce** (calibration targets; details in section 8):

1. Population stays within ±10% of 10,000 for 200 years.
2. A plausible age structure: median age 30–40, no baby booms or collapses after warm-up.
3. Food, power, water and air stay balanced, with occasional short shortages but no collapse.
4. The generator has a few serious failures per century, not none and not constant ones.
5. Every department stays staffed; trades pass on through shadowing.
6. The zone structure persists: Mechanical stays in the Down Deep, IT and Judicial stay Up Top.

**Definition of done for v1 of the code:**

- [x] Patterns 1–6 hold in at least 90 of 100 runs with different seeds (96–100 of 100; calibration log, 2026-10-30)
- [x] The same seed gives byte-identical output
- [x] All invariant tests in section 8 pass
- [x] One run of 200 years finishes in under a minute on a laptop (60.7 s for 50 + 200 years with other programs open, T5.4)

**Out of scope for the baseline** (later modules): beliefs and dissent, relics, surveillance, cleaning, covert lottery denial, rebellion, Silo 1, other silos, money.

### 1.2 Part 2: rebellion (draft)

Part 2 adds the people, beliefs and institutions the baseline left out, so the model can answer the questions the project exists for. The baseline (§1.1) stays frozen: with every part-2 module off, `scenarios/baseline.yaml` gives byte-identical output to `baseline-v1`. The plan is in `docs/tasks-part2.md`; sections 6.10–6.16 are written milestone by milestone.

**Research questions.** Each one is answered by outputs listed under "Outcome metrics" below.

| # | Question | Answered by |
| --- | --- | --- |
| Q1 | How often does a rebellion start, and what sets it off? | Time to first rebellion; rebellions per century; trigger of each rebellion |
| Q2 | Can a rebellion kill the silo, and by which route? | Share of rebellions ending in collapse; cause of collapse |
| Q3 | Which regime response keeps the silo alive afterwards: crackdown, reform, or wiping memory? | Time to collapse and endings, compared across response strategies (§6.15) |
| Q4 | Does Silo 1's threat make Silo 18 more stable or more fragile? | Q1–Q3 metrics compared across Silo 1 settings `off`, `watch`, `full` (§6.16) |

The answers are **conditional and comparative**. The agents follow rules, not human judgment, so the model shows what follows from its assumptions ("if dissent works like Epstein's rule, reform keeps the silo alive more often than crackdown"), not what real people would do. Claims compare scenarios run on the same seeds; absolute numbers, such as "a rebellion every 112 years", are reported but not relied on.

**Definitions**

- **Active rebel:** an adult (18+) whom the activation rule of §6.14 makes active on a given day.
- **Unrest:** at least 1% of adults active on 3 days in a row, without meeting the rebellion test. Its start day is the first of the 3 days. It ends when fewer than 1% of adults are active for 30 days in a row, its end day the first of those 30 (Adam, 2026-10-31). An unrest spell during which a rebellion starts is that rebellion's run-up, counted as the rebellion and not as unrest, and no unrest is tracked while a rebellion runs (Adam, 2026-10-31). Logged; not a rebellion.
- **Rebellion:** starts when, on 7 days in a row, at least 5% of adults are active **and** at least one collective action (§6.14) is under way on one of those days: a strike in a critical department (Mechanical, Farms, Supply, Medical; at least a share q\_strike = 10% of its workers on strike), a fight with Judicial, or rebels holding a zone. Its start day is the first of the 7 days. It ends when fewer than 1% of adults are active for 30 days in a row, its end day the first of those 30, or at collapse. Only one rebellion runs at a time. The shares and day counts in these two definitions are scenario settings (`rebellion:`, §6.14).
- **Trigger:** the scripted event (`events:` in the scenario, T7.4) that happened within the 365 days before the start, or "none". With no scripted events, every rebellion is "none", and Q1's trigger question is answered by experiments that script shocks.
- **Endings** (one per rebellion; when several hold on the same day, the first in this order: collapse, regime change, split, suppressed):
  - *Suppressed:* it ends with the regime still in place.
  - *Regime change:* rebels hold Up Top, the zone of the IT floors and the Mayor's office, for 30 days in a row (§6.14), the 30 a scenario setting (`rebellion.regimeDays`). The regime is replaced: the old playbook ends and only the scenario's `regimeChange` entries apply (§6.15); with none, the silo runs on at its peacetime settings. The run continues.
  - *Split:* a stair segment broken while the rebellion runs (from its start day, the run-up included; by sabotage or a scripted `breakStairs`) is still broken 90 days after it broke, the rebellion is still running, and at least 1,000 living people have their home on each side of it (the floors above and below the segment). The rebellion ends that day. The silo runs on as before: the segment is repaired like any other (§6.14), and a new rebellion can start later. The 90 days and 1,000 people are scenario settings (`rebellion.splitDays`, `rebellion.splitPeople`; Adam, 2026-11-01).
  - *Collapse:* see below.
- **Collapse** (the silo is dead, and the run stops), on the first day any of these holds; the cause is recorded. It is checked only with the rebellion module on (§6.14), so the baseline never stops early:
  1. air or water below 0.2 for 30 days in a row;
  2. a Safeguard goes off (§6.16);
  3. population below the **point of no return** P\_min. If air or water is below 0.2 that day, the cause is recorded as air or water (in that order), not population: people die of bad air within days of it falling below 0.2, before test 1 can fire, so without this rule every life-support death would be recorded as a population collapse (T12.7, Adam, 2026-11-01).

  P\_min is measured, not guessed (T12.6): runs start from damaged states (population cut to 9,000, 8,000, 7,000 … and Mechanical cut by 10%, 20% …), and P\_min is the highest population at which fewer than 5% of seeds recover to above 9,000 within 50 years. **Measured: P\_min = 4,900** (T12.6, 2026-11-01; Adam accepted it and kept this recovery test). The baseline silo, warmed up for 50 years, was culled on the first measured day (`cull`, §6.1) and run 50 more years, 100 seeds per cut: cut by 40% (5,863 left) 10% of seeds recovered, cut by 50% (4,885 left) none did, and every one died out. The true value lies between those two; the grid stopped there. Above P\_min is a wide risk zone the test leaves to extinction: cut by 20%, 30% of seeds still died out within 50 years, many after recovering first (calibration log, 2026-11-01). It holds for a silo of 10,000; a scenario of another size needs its own. The fragility comes from the baseline: every department's coverage need is fixed at its day-1 size (§6.6), so a smaller silo can't staff its systems. The baseline already shows why the last person is the wrong line: in the T3.2 collapse, health was 0 and the silo still held 643 people in year 100. Canon agrees: Silo 17, with fewer than a hundred survivors and later one, is a dead silo.
- **Extinction:** the last person dies. It is logged and ends the run, whether or not a collapse came first (Adam, 2026-10-31): a silo can die out before any test fires, as when air and water stay above 0.2 but a strike leaves too few people to run it. Runs that set `output.runAfterCollapse: true` go on after a collapse to see how a dead silo empties.

**Canon patterns** (calibration and validation targets for part 2). They are checked on `scenarios/canon.yaml`, the regime at the show's settings, 300 measured years, 100 seeds. These are the only targets part-2 parameters are tuned to.

| # | Pattern | Source | Check |
| --- | --- | --- | --- |
| C1 | **Long peace:** rebellions are rare, but they do happen | The Rebellion ~140 years before S1; the next in S2 | Mean rate across seeds between 0.3 and 2 rebellions per century |
| C2 | **Visible failures trigger revolts:** a cleaner who walks away matters more than slow hardship | Juliette (S1E10); Silo 17's cleaner | A rebellion within 5 years after a scripted cleaning failure is at least 3× as likely as in a 5-year window without one |
| C3 | **Revolts start Down Deep and use the generator** | The S2 uprising | ≥ 60% of rebellions start with most active rebels living Down Deep; ≥ 50% of those include a Mechanical strike |
| C4 | **Crushed rebellions are forgotten** | Freedom Day, lost records, Vitamin D+ | 40 years after a suppressed rebellion, < 10% of adults hold the true-history flag (§6.12) |
| C5 | **A rebellion can end a silo** | Silo 17 | With Silo 1 on `full`, 5–50% of rebellions end in collapse |

**How the patterns are counted** (`silosim report`, which reads only the output files; Adam, 2026-11-01). Each is judged over the whole batch, not per seed, and only on measured days.

- *C1:* the pooled rate: all rebellions that started in the batch, divided by the measured years the silos ran (a run that collapses counts up to its last day), × 100. A mean of per-seed rates would let one silo that dies early with a rebellion dominate.
- *C2:* the treated windows are the 5 years from each scripted `cleaningFailure` (its own day included). The comparison windows are the measured period cut into 5-year blocks from its first day, keeping only blocks that overlap no 5-year window after any cleaning failure, scripted or a sentenced cleaning that failed. A window counts as having a rebellion if one starts in it. A window cut short by the end of the run counts only if a rebellion started in it, so a rebellion that ends the silo early is not lost. C2 holds when both kinds of window exist, at least one treated window has a rebellion, and the treated share is at least 3× the comparison share (or the comparison share is 0).
- *C3:* over all rebellions in the batch: the share whose starting zone (§6.14) is the Down Deep, and among those the share with Mechanical on strike on any of its days (`struck` in `rebellions.jsonl`).
- *C4:* each suppressed rebellion is judged at the first yearly snapshot at least 40 years after its end day, by the share of adults who know the history flag (`knowHistory`); one with no such snapshot is not judged. C4 holds when at least 90% of the judged rebellions are under 10%, the same bar as the §8 targets.

**C4 is judged from M13 on** (Adam, 2026-11-02). Its sources (Freedom Day, lost records, Vitamin D+) are the regime's memory wipe (§6.15), which the model does not have before M13; `silosim report` printed C4 as information only until `scenarios/canon.yaml` carried The Order's playbook, and judges it since T13.2. Since T13.0 everyone who lived through a rebellion learns the history at its end (§6.15), so C4 measures how a silo forgets a rebellion everyone saw.

C1 bounds the rebellion rate, which is also what Q1 measures. That is deliberate but limited: C1 is a wide plausibility check (no constant revolts, no eternal peace), never a target to tune toward a point. Q1's answer is what *changes* the rate, read from comparisons between scenarios.

**Outcome metrics**

- *Per run:* time to first rebellion (censored at the end of the run if none); number of rebellions and unrest spells; for each rebellion, its start, trigger, starting zone, peak share of active adults, duration, collective actions, deaths, arrests, damage, and ending; time and cause of collapse (censored if none).
- *Across seeds:* survival curves (Kaplan–Meier) for time to first rebellion and time to collapse; rebellions per century and the share of each ending, with 95% confidence intervals; the share of measured days spent in a rebellion, which tells rare long rebellions from frequent short ones (the standing rebellion of T12.8; §6.15).

**Run settings for part 2.** 50 warm-up years plus **300 measured years**. Rebellions are rare by design, so longer runs and more seeds are the price of saying anything about how they end. Calibration uses 100 seeds; the final experiments (T15.5) use **1,000 seeds per scenario**. `scenarios/baseline.yaml` keeps its 200 measured years.

**Outputs for viewers.** Everything a later UI needs is written to files during the run; a viewer only reads them and can never change a run. Part 2 adds `rebellions.jsonl`, outcome fields in `run.json`, and optional `frames.csv.gz`, a per-floor snapshot in long format (day, floor, metric, value), written every `output.frameEvery` days (default 30). Each module registers its own metrics. Frames are off by default and are switched on for runs meant to be replayed.

## 2. Entities, state variables and scales

Five kinds of entity: people, households, floors, departments and the silo itself. People are the only agents that make choices; the rest hold state.

### Person

| Variable | Type | Meaning |
| --- | --- | --- |
| id | int | Unique, never reused |
| sex | enum | F / M |
| birthDay | int | Tick of birth; age is derived |
| alive | bool | Dead people stay in the log, not in the active list |
| householdID | int | Where they live |
| partnerID | int or none | Chosen partner |
| motherID, fatherID | int or none | For family trees and inheritance of trades |
| stage | enum | child, shadow, worker, sick leave, retired |
| department | enum or none | Set when shadowing starts; kept as a record after retirement and death |
| masterID | int or none | The worker they shadow |
| skill | 0–1 | Grows with experience; drives output |
| health | 0–1 | 1 = healthy; drops with injury, illness, hunger, bad air |
| lowHealth | int | Days in a row a worker's health has been below 0.3 (sick leave, 6.5) |
| implant | enum | in, queued (permit won, waiting for Medical), removed (window open), pregnant |
| permitDay | int | Tick she won the lottery |
| windowEnds | int | Last tick of the 365-day conception window |
| dueDay | int | Tick of birth if pregnant |
| babyFather | int or none | Partner at conception, recorded as the child's father |
| children | int | Number of children born, alive or not |
| lastBirth | int | Tick of her most recent birth |

### Household

| Variable | Meaning |
| --- | --- |
| id | Unique |
| floor | 1–144; its zone follows from the floor |
| members | People living there: a couple and their children under 18, or a single adult |

### Floor

| Variable | Meaning |
| --- | --- |
| number | 1–144 |
| zone | Up Top, Mids or Down Deep |
| homes | Housing capacity in households (0 on pure work floors) |
| function | Residential, farm or mechanical (farm and mechanical floors are pure work floors) |

### Department

| Variable | Meaning |
| --- | --- |
| name | One of the departments in section 6.5 |
| zone | Where it works and where its workers live |
| targetShare | Share of the workforce it needs |
| headcount, shadows | Current workers and apprentices |
| output | Effective work this tick: worker skill × health, plus 0.3 × skill × health per shadow |
| coverage | 0–1: min(1, output / need), 6.6 |

### Silo (global state)

| Variable | Meaning |
| --- | --- |
| day | Current tick |
| foodStock | Stored food in kcal |
| generatorCondition | 0–1; 1 = like new |
| generatorDown | Ticks left until a failed generator is repaired |
| powerSupply, powerDemand | This tick's power balance, in abstract units |
| waterSupply, airQuality | 0–1 each |
| partsStock | Spare parts, in abstract units |
| rationShares | Share of distributed food per zone (policy) |
| governanceMode | appointment or election |
| mayorID | Current Mayor |
| rng | One seed; one random stream per module (see §4) |

### Scales

- **Time:** 1 tick = 1 day. A year is 365 ticks; monthly steps run every 30 ticks. A run is 50 years of warm-up plus 200 years of measurement.
- **Space:** 144 floors in three zones. Location is an attribute only: nobody walks the stairs in the baseline. Travel time arrives with the rebellion modules, where it starts to matter (§6.10).
- **Population:** about 10,000 people alive at once, about 40,000–50,000 over a full run.

## 3. Process overview and scheduling

Each tick runs the same fixed sequence: silo-wide systems first, then people. The order is part of the model; changing it changes results, so it is written down here and nowhere else.

### Every day

1. **Generator** — Supply's new parts added, then wear, repair, failure check, outage countdown → power supply (6.6)
2. **Life support** — water and air from power and Mechanical output (6.6)
3. **Food production** — farm output, scaled by power, added to the stock (6.6)
4. **Rations** — food split by zone, intake per person, stock reduced (6.7)
5. **Work** — department output, skill growth for workers and shadows (6.5)
6. **Health** — hunger, air, work injuries, illness, recovery with Medical (6.2)
7. **Deaths** — mortality draws; widowed partners, orphans and job vacancies handled (6.1)
8. **Reproduction** — conceptions in open windows, births due today, expired windows closed (6.4)
9. **Life stages** — child → shadow at 14, shadow → worker at 18, worker ↔ sick leave, worker → retired (6.5)
10. **Partnering** — unpartnered adults may pair up (6.3)
10.5. **Network** *(part 2, draft)* — weak ties ended and drawn again; the connected runs of floors for the day's conversations (6.11)
10.7. **Beliefs** *(part 2, draft)* — upbringing, forgetting, Vitamin D+, relics, the day's conversations that pass knowledge on, Flamekeepers (6.12)
10.8. **Control** *(part 2, draft)* — Head of IT, detection, releases, trials and cleanings, raids, arrests (6.13)
10.9. **Rebellion** *(part 2, draft)* — hardship and grievance, stages, fights, sabotage, stair repairs, unrest and rebellion, collapse (6.14)
10.95. **Regime** *(part 2, draft)* — the playbook's entries lapse and start, policy events, the emergency declared or lifted (6.15)
11. **Logging** — daily metrics written (8)

### Every 30 days

12. **Lottery** — permits issued and winners drawn (6.4)
13. **Housing** — new couples move to free homes; grown children leave home only by partnering (6.3)
14. **Staffing** — each department's target headcount set for the next shadow placements (6.5)

### Every 365 days

15. **Snapshot** — age pyramid and department table written (8)
16. **Governance** — if the Mayor died or the term ended, a new Mayor is chosen by appointment or election (6.8)

### Every day, last *(part 2, draft)*

16.5. **Location** — where every person is the next day, from the day's final state; travellers due tomorrow are placed at their destination; stranded workers (6.10). Day 1's locations are set when the module starts. It comes last, after the monthly and yearly steps, so a department or home that changed today counts from tomorrow

### Update rules

- People are processed one at a time in an order shuffled each tick by the seeded random generator, so no one always goes first.
- Changes take effect immediately (asynchronous updating). A person who dies in step 7 does not reproduce in step 8.
- One goroutine runs one simulation. Parallelism comes from running many seeds at once, never from splitting one tick across threads.
- *Part 2 (draft):* actions due on a later day, such as a trip's arrival, the end of a sentence or a scripted event (§5), run on their day just before the step they belong to, whether or not that step itself runs that day. Several due at the same step run in the order they were scheduled; scripted events in the order of the scenario file. An action can only be scheduled for a later step or a later day, never for a step that has already run. Part-2 steps get in-between numbers, such as 7.5 for a step between deaths and reproduction.

## 4. Design concepts

The baseline is deliberately simple: people follow fixed rules, and only population-level patterns should emerge. Learning, beliefs and strategy arrive with the dissent modules.

| Concept | In the baseline |
| --- | --- |
| Basic principles | Closed population with controlled reproduction; a closed resource loop where power drives food, water and air |
| Emergence | Age structure, population stability, shortage frequency, staffing gaps, family lines concentrating in departments |
| Imposed by rules | Life stages, lottery, staffing targets, ration shares |
| Adaptation | Minimal: departments attract shadows when understaffed; the lottery issues more permits when population falls |
| Objectives | None at the person level yet; the regime's implicit goal is a steady 10,000 |
| Learning | Skill grows with work experience; nothing else is learned. *Part 2:* people learn, forget and recall five facts, which set their legitimacy (§6.12) |
| Prediction | None |
| Sensing | People know their partner, household, master and department; nothing more |
| Interaction | Partnering, shadowing, shared rations within a zone, competition for lottery permits and homes |
| Stochasticity | Deaths, injuries, illness, conception, partner choice, generator failures, lottery draws; all from one seed, with one random stream per module so adding a module does not shift the others' draws. *Part 2:* whether a tie is used on a day (§6.11) comes from a counter-based stream, a function of the seed, the day and the pair, so it is the same whoever asks and costs nothing until asked |
| Collectives | Households, departments, zones |
| Observation | Daily silo-wide metrics, yearly snapshots, an event log; see section 8 |

**Why so little behavior:** if the baseline already contains choices and beliefs, nobody can tell later whether a rebellion came from the dissent rules or from a quirk in the base. Keeping the base mechanical keeps later results interpretable.

## 5. Initialization and input

Day 1 is generated from the scenario file, then the model runs 50 years of warm-up that are thrown away. Warm-up removes the artifacts of a hand-built starting population, such as everyone's skill or children's ages being too uniform.

### Generating day 1

1. **Floors:** 3 × 48 floors: 1–48 Up Top, 49–96 Mids, 97–144 Down Deep. The 20 farm floors are spread evenly through the Mids (farm floor i, for i = 0…19, is floor 49 + ⌊i × 48 / 20⌋, i.e. 49, 51, 53, 56, …); the lowest 12 floors are Mechanical; every other floor is residential. Farm and Mechanical floors are pure work floors with no homes. Each zone gets ⌈1.15 × its households⌉ homes, spread as evenly as possible over its residential floors, and each household takes a random free home in its zone, so moving is possible.
2. **People:** 10,000 people, exactly half women, chosen at random. Each age is drawn, in days, from the steady-state age distribution of the mortality rule (6.1) at health 1: the chance of being d days old is proportional to the chance of surviving d days. The start then already looks like a lived-in population.
3. **Zones:** exactly 32% Up Top, 46% Mids, 22% Down Deep (rounded to whole people), assigned at random and independent of age and sex. The split follows from where departments sit (6.5).
4. **Households:** in each zone, round(0.70 × adults aged 20+ / 2) couples form, each a woman and a man aged 20+. Women are taken in random order; each picks an unpaired man from her zone with weight e^(−|age gap| / 5), the age-gap rule of 6.3. Each child under 18 then, in random order, joins a random couple in their zone whose woman could have had them under the lottery rules (6.4): aged 18–40 at the birth, fewer than 3 children, and no other child less than 2 years older or younger. The woman becomes the mother and her partner the father. If no couple fits, initialization stops with an error instead of bending the rule; this never happened in 300 seeds at 10,000 people, but small test worlds (about 7% of seeds at 1,000 people) can run out of fitting couples, so tests use seeds that work. Every other adult (18+) lives alone.
5. **Jobs** (only when the work module is on): in each zone, adults 18–64 fill the departments of that zone plus General services and Porters by target share (an all-zone department's share split over the zones by allZoneSplit), rounded by largest remainder and assigned at random. People 65+ are retired with no department. 14- to 17-year-olds are placed by the shadowing rule of 6.5, in random order.
6. **Skill:** 1 − (1 − α)^(days since the 14th birthday, counted up to 65), the same curve as daily skill growth (6.5), since shadows learn too.
7. **Implants:** every woman starts with the implant in; no open windows, no pregnancies.
8. **Stocks:** food for 90 days of full rations (90 × the day-1 daily need of 6.7), parts for 365 days of expected use (365 × the parts one repair day uses, 6.6), generator condition 0.8, water and air at 1.

### Input data

No external data. Everything comes from one scenario file, so every run can be repeated from the file and a seed.

```yaml
name: baseline
seed: 42
warmupYears: 50
years: 200
population: 10000
zoneSplit: { upTop: 0.32, mids: 0.46, downDeep: 0.22 }
governance: appointment          # or: election
mayorTerm: 5y                     # or: life
rationWeights: { upTop: 1.0, mids: 1.0, downDeep: 1.0 }   # 1.0 everywhere = equal per person
modules: [demography, health, work, resources, governance]
params: {}                        # overrides for any value in section 7
```

**Scripted events (part 2, draft).** A scenario may list shocks under `events:`, each a day and a type, for example `{ day: 20000, type: cleaningFailure }`. They are how experiments separate cause from effect: the same seeds with and without the shock. The day counts from the run's first day, warm-up included, the same as the `day` column of the output files, and it must fall in the measured years, since warm-up is thrown away. Each type is defined by the module that uses it, in that module's section of 6, which also names the step of §3 it runs at. A type that no running module defines stops the run before it starts, so a misspelt or switched-off shock can't pass unnoticed. Every scripted event is logged to events.jsonl as type `scripted`. What it causes is logged separately, by its module. Scripted events draw random numbers from their own stream, so their draws never take numbers from any module's stream. The baseline has none.

## 6. Submodels

Each rule below is what Claude Code implements, one submodel per module. Every number is a starting value from section 7; calibration may change it, the rule shape stays.

### 6.1 Mortality

Each day each person dies with a probability from a Gompertz hazard (risk of death doubling roughly every 8 years of age), raised by poor health and by an extra risk in the first year of life.

```latex
h(a) = A e^{B a} + h_{infant}[a < 1], \quad p_{day} = 1 - \exp\left(-\frac{h(a)\,(1 + k_h (1 - health))}{365}\right)
```

With A = 0.0002 and B = 0.085, half of people reach about 67 before health effects. A person whose health reaches 0 dies that day: the health step (§3 step 6) comes just before deaths, so the death chance is 1 then. Deaths are drawn person by person in the tick's shuffled order and handled at once. On a death: the partner becomes unpartnered and may pair again; if an adult (18+) still lives in the household, the children under 18 stay with them (their other parent, or a step-parent); otherwise all the children in that household move together to a random household in the same zone with at least one living adult. If their zone has none (only after a mass death), they go to such a household in any zone. *Part 2:* when no household in the silo has an adult, the children stay in their own home (Adam, 2026-10-31); the silo is dead, and §1.2 ends the run at its collapse or extinction. The baseline never reaches this state. A household left with no members is dissolved, which frees its home. The job becomes a vacancy (6.5).

*Part 2, scripted event `cull`* (share, department optional; §5): a share of the living people, or of the department's workers, rounded to the nearest whole number and drawn uniformly, die just before that day's deaths step, each handled as any death. It starts runs from the damaged states that measure the point of no return P\_min (§1.2, T12.6). It belongs to this section, not to §6.14, so P\_min is measured on the baseline modules alone (Adam, 2026-10-31).

### 6.2 Health and Medical

Health runs from 0 to 1 and changes every day for every living person, in the tick's shuffled order: damage from hunger, bad air, injuries and illness, then recovery that depends on how well Medical is staffed, then a clamp to 0–1.

- **Hunger:** −d\_hunger × max(0, 1 − intake/need), with intake/need of the person's zone (6.7); food above need does not raise health
- **Bad air:** −d\_air × (1 − airQuality)
- **Work injury:** for workers and shadows, daily chance 0.0002 × the department factor (Mechanical ×2, Farms ×1, others ×0.5); an injury costs 0.3 health
- **Illness:** small daily chance; costs 0.15 health
- **Recovery:** +r × medicalCoverage, capped at 1

```latex
medicalCoverage = \min\left(1, \frac{MedicalOutput}{0.84 \times MedicalDay1TargetHeadcount}\right)
```

Without the resources module, intake/need stays 1 and air stays at its day-1 value of 1 (§5 step 8).

Medical also performs implant removals for lottery winners. Winners join a queue; right after each monthly draw, Medical removes the implants of the oldest ⌊medicalCoverage × waiting⌋ winners (oldest permit first), and the rest wait for the next month. At coverage 1 nobody waits; at 0.5 about half wait a month, which delays births when the silo is short of doctors. A winner whose partner dies while she waits loses the permit. Mothers face a death risk at each birth of 0.002 × (2 − medicalCoverage); the child lives. Where there is no Medical department (the work module is off), coverage counts as 1.

### 6.3 Partnering and housing

Partners are chosen freely. Each day, each unpartnered adult (18+), in the tick's shuffled order, starts looking with a small chance (on average about every 3 years) and at once pairs with someone picked from all other unpartnered adults in the silo, with weights:

- same zone ×4, same department ×2, same sex ×0.05
- an age-gap penalty: weight × e^(−|age gap| / 5)
- close relatives are never picked: parent, child, or sibling (sharing a mother or a father)

Any couple can form, so about 5% of new couples are same-sex; only a woman–man couple can enter the birth lottery. New couples join a housing queue. Each month the queue is handled oldest first. The first time a couple is handled it gets a target zone: the zone of a partner whose department has a zone (every department but General services and Porters); if both partners have one and they differ, one of the two at random; if neither, one of the two partners' current zones at random. Mechanical is the exception, because it runs the generator: while Mechanical has fewer workers than its target, a couple with one partner in Mechanical and the other in a department of another zone goes to Mechanical's zone (Down Deep). Otherwise such a couple is picked 50/50 like any other. Keeping them Down Deep with 80% even when Mechanical was full overfilled Mechanical (about 130% of target) and Down Deep (a third of the silo) at Farms' expense, because children who grow up Down Deep mostly become Mechanical (calibration log, 2026-10-29). The couple then moves into a random free home in that zone (a floor's homes minus the households living there), or keeps waiting, living apart, if none is free; a couple waiting for one zone never blocks couples waiting for another. A couple broken by a death leaves the queue. Moving starts a new household: from each household a partner leaves, children under 18 come along if they are a partner's own children, or if the partners were the only adults there, so no child is left without an adult. Homes left empty are free at once for the rest of the queue. Young adults stay in their parents' household until they partner. Partnering and job placement are the only ways a person changes zone, which makes zone mobility measurable.

### 6.4 Lottery and reproduction

Every 30 days the silo issues permits to replace recent deaths and close the gap to the target population, spread over a year so the population does not swing.

```latex
N = \mathrm{clamp}\left(\mathrm{round}\left(D_{30} + \frac{T - P - E}{12}\right),\ 0,\ N_{max}\right), \quad E = pregnant + 0.8 \times openWindows
```

D₃₀ = deaths in the last 30 days, T = target (10,000), P = people alive.

D₃₀ counts deaths since the last lottery; round is half away from zero. Unused permits are not carried over.

- **Eligible couples:** woman and man, woman aged 18–40 in whole years (up to her 41st birthday), fewer than 3 children born (alive or not), her last birth at least 2 years ago, no open window or pregnancy. Couples still waiting for a home are eligible.
- **Draw:** winners are picked uniformly at random, without replacement; if fewer couples are eligible than there are permits, all of them win. With the control module on, flagged couples win but stay childless: covert lottery denial (§6.13). Otherwise the draw is fair.
- **Window:** winners join Medical's removal queue (6.2); the couple has the 365 days after the removal. Women still waiting count as open windows in E. If her partner dies while she waits or while the window is open, the permit or window ends and the implant stays or goes back in.
- **Conception:** daily chance 0.0044, which gives about an 80% chance within the year. Her partner is recorded as the father.
- **Birth:** 270 days after conception the child is born into the mother's household: a girl with chance 0.5, health 1. The father is the partner recorded at conception, even if he has died since; a pregnancy goes on after his death. The implant goes back in. If the window ends without a pregnancy, the implant goes back in.

### 6.5 Work and shadowing

Departments have a home zone and a target share of the workforce. Their shares set where people live, which gives a zone split of about 32% Up Top, 46% Mids and 22% Down Deep.

| Department | Zone | Target share of workforce |
| --- | --- | --- |
| Farms | Mids | 30% |
| General services (cafeterias, crafts, cleaning) | All zones | 20% |
| Mechanical (incl. water plant) | Down Deep | 16% |
| Supply | Mids | 6% |
| Judicial and Security | Up Top | 6% |
| Porters | All zones | 5% |
| Education | Up Top | 5% |
| Medical | Up Top | 4% |
| IT | Up Top | 4% |
| Administration (Mayor's office) | Up Top | 3% |
| Sheriff | Up Top | 1% |

- **Targets:** each month (staffing, §3 step 14) every department's target headcount is its share × the current number of workers. Shadows and retired people are not workers.
- **Shadowing at 14:** with a 60% chance the child follows the department of a parent picked at random from those who have one (working, retired or dead), if it has room: a worker with skill above 0.5 and fewer than 2 shadows. Otherwise, with a 5% chance they pick among departments tied to another zone, else among their own zone's departments plus General services and Porters. Departments without room are skipped, and the pick is weighted by shortfall = max(0, target − workers − shadows), counted live so that every placement shrinks it at once. If every shortfall in reach is 0, the weights are each department's share of the jobs in the child's zone (an all-zone department's share × allZoneSplit for that zone). If no department in reach has room, the child shadows the department those weights pick, with no master until one is free. The master is the free worker of that department with the fewest shadows (ties at random). A shadow whose master dies, retires or leaves gets a new one the same day if one is free. A shadow keeps living at home; a job tied to another zone moves them only when they partner (6.3).
- **Worker at 18** in the same department; the master link ends. **Sick leave** after 90 days in a row with health below 0.3: the worker stops working (no output, not counted as a worker for targets, coverage or retraining) but keeps the department, and their shadows get new masters. They return to work in the same department, with their skill, on the first day their health is back at 0.3 or more. **Retired** at 65, from work or from sick leave; the department stays on record. Sick leave used to be permanent retirement. That made a silo-wide health crisis final, because a recovered workforce never came back (calibration log, 2026-10-29).
- **Skill:** grows each day by α × (1 − skill) for workers and shadows, so reaching 0.8 takes about 10 years.
- **Trade inheritance (outcome, not a rule):** about 80% of workers end up in a parent's department, more than the 60% follow chance, because the other placements often land in a parent's department too.
- **Output:** department output = the sum of skill × health over its workers, plus 0.3 × skill × health for each shadow. Computed daily (§3 step 5), before the day's skill growth.
- **Retraining:** each month, after the targets are set, each department other than General services with fewer than 80% of its target workers takes General workers living in its zone (any zone for Porters, which work everywhere), drawn at random, until it reaches 80%, as long as General keeps at least 80% of its own target. Departments are handled in table order. Retrained workers restart at skill 0.2, and their shadows get new masters. This keeps critical departments staffed after deaths.

### 6.6 Resources

The generator is the root of the loop: it powers the farms, the water pumps and the air, so a weak Mechanical department eventually shows up as hunger and illness everywhere.

- **Generator condition c:** loses 0.0004 per day (from new to broken in about 7 years without care). Repair adds ρ × mechanicalCoverage × (1 − c) and uses 1 part a day; with fewer parts in stock than one repair day uses there is no repair. Because repair slows as c nears 1, condition settles where wear and repair balance: c\* = 1 − wear / (ρ × coverage), which is 0.8 at full coverage with ρ = 0.002. Wear and repair come first each day, then the failure draw; c is kept within 0–1. The generator step runs before the work step (§3), so it uses the previous day's mechanicalCoverage (on day 1, the coverage from initialization).
- **Failure:** daily chance f₀ × (1 − c)², with f₀ = 0.002, which gives about 3 failures per century at c = 0.8. A failure drops c by 0.1 and takes ⌈3 / (skill × partsShare)⌉ days to fix, at most 10. Here skill is the average skill of Mechanical workers (shadows not counted), and partsShare = min(1, parts in stock / the 30 a failure uses), measured before the failure takes its parts. With no workers or no parts the cap applies. Repair time is the work divided by the rate Mechanical can do it at: 3 days with perfect skill and full parts, about 4 at the usual skill of 0.85, 6 with a green crew (skill 0.5) or half the parts. Short outages matter because air follows power: one of 8 days or more on backup drops average health below the point where the loop can recover (see the calibration log, 2026-10-29). A failure then consumes its parts, as many as are left if the stock is short. The day of the failure is the first day of the outage. While it is down, the condition is frozen: no wear, no repair, no parts used and no new failure draw; only the days left count down. Backup power then gives 17.3% of capacity (about 26% of day-1 demand), whatever c and Mechanical coverage are. When capacity went from 1.3 to 1.5, this share went from 20% to 17.3%, so outages stay as harsh as before.
- **Power:** supply = capacity × c × mechanicalCoverage (or backup while down); demand is fixed per person and per farm floor. Capacity is set on day 1 to 1.5 × day-1 demand. A failure leaves the condition at about 0.7, and 1.5 × 0.7 = 1.05, so the silo keeps full power after a repair; the earlier 1.3 left it at 0.91 and could tip the health loop into a collapse (calibration log, 2026-10-29). powerFactor = min(1, supply / demand).
- **Water:** powerFactor × mechanicalCoverage (today's powerFactor, the previous day's coverage). *Part 2:* 0 while the water pump is down after sabotage (§6.14).
- **Air:** moves toward powerFactor by a third of the gap each day, so short outages are survivable and long ones are not.
- **Food:** production = farm output × kcal per unit of farm output × powerFactor, added to the stock. Farm output is the previous day's, since the work step (§3 step 5) comes later.
- **Parts:** measured in days of normal use. Supply makes 1.2 units a day × supplyCoverage (scavenging and recycling), added at the start of the generator step (§3 step 1) every day, outage days included, so they can serve that day's repair; each repair day uses 1 unit; each failure uses 30 more.

mechanicalCoverage = min(1, Mechanical output / Mechanical need), the same shape as medicalCoverage. Every coverage uses the same need: 0.84 × the department's day-1 target headcount (its share of the day-1 workers). The need is fixed, so losing workers across the silo lowers every coverage, not only an imbalance between departments. The silo's average skill × health is about 0.92 (measured over 50 years with health 1), so a fully staffed department has output / need ≈ 1.1, coverage is capped at 1, and shortages start to bite at about 90% staffing.

```text
Mechanical (staff, parts) -> Generator (condition, outages) -> Power
Power -> Food (farms, then rations)
Power -> Water (pumps)
Power -> Air (follows power)
Food + Water + Air -> Health (hunger, air, Medical)
Health -> Deaths and work output -> back into Mechanical staffing
```

The loop closes through people: hunger and bad air raise deaths and lower output, which weakens Mechanical, which weakens the generator. Calibration has to keep this loop stable before any dissent is added.

### 6.7 Rations

Each day the silo hands out the day's total need from the stock. If the stock, after the day's production, is below 30 days of need, it hands out only 90%, which is rationing; it never hands out more than the stock holds. Food is split between zones by policy weights, then shared within each zone in proportion to need (children under 14 need 60% of an adult's 2,200 kcal). N\_z below is zone z's total need in kcal, not its headcount, so with equal weights every zone gets the same share of its need however many children it has.

```latex
share_z = \frac{w_z N_z}{\sum_{z'} w_{z'} N_{z'}}, \qquad \frac{intake_i}{need_i} = \frac{share_z \times distributed}{need_z}
```

This simplifies to intake/need = w\_z × distributed / Σ w × N. In the baseline every weight is 1.0, so everyone gets the same share of their need. The regime can change the weights later, for example to favor Up Top. A zone with a weight above 1 can then get more than its need. All the distributed food is still handed out, and the extra does not raise health (6.2), while the other zones get less than their need even when the stock is full. Each day the model logs hardship per zone = 1 − average intake/need (everyone in a zone has the same intake/need). Hardship is negative for a zone fed above its need. The dissent modules will use it as a source of grievance.

### 6.8 Governance

How a Mayor is chosen and how long they serve are two separate settings. In the baseline the office changes no rules; it is logged so the dissent modules have it ready. "Senior worker" here means any worker (stage worker, not shadow or retired). Skill grows with years worked, so the most skilled is nearly always among the oldest. Ties in skill go to the lower person ID. The first Mayor is chosen on day 1, at initialization. After that, the yearly governance step (§3 step 16) chooses a new Mayor if the Mayor has died since the last step, the term has run out, or the office is vacant. A death is therefore noticed only at the next yearly step. The sitting Mayor may be chosen again. A Mayor who retires keeps the office until death or the end of the term. If nobody can be chosen, the office stays vacant and is retried each year.

- **Appointment (default):** the heads of IT, Judicial and Administration (the most skilled worker in each) choose the most skilled senior worker from those three departments.
- **Election (scenario toggle):** each zone puts up its most skilled senior worker living there; every adult (18+) votes for their own zone's candidate with an 80% chance, otherwise for a candidate drawn uniformly from all of them, their own included (so about 87% back their own zone). A voter whose zone has no candidate always draws. Most votes wins; a tie is drawn at random. Without beliefs, this simply measures how zone size decides an election.
- **Term (scenario toggle):** a 5-year term (default), or for life, so a Mayor serves until death. A life term works with either way of choosing; it removes any regular, peaceful change of leader, which matters once dissent is modeled.

### 6.9 The Pact (draft)

The Pact is modelled as a **rule table**, not as prose. The agents can't read text, so a rule exists in the model only if a module detects its offences and punishes them. The table lives in the scenario file under `pact:`. A rule whose detecting or enforcing module is switched off does nothing, so the baseline is unaffected. The show never shows the Pact in full, so the table holds only rules the show establishes. The syndrome rule (S1E6) is left out because none of Q1–Q4 needs it.

Each row says what is forbidden, which offences break it, how hard it is to see, and what follows:

| Field | Meaning |
| --- | --- |
| id | Short name, unique |
| active | Whether the rule is in force (a Pact rewrite switches it) |
| offences | Offence types that break it. Modules produce offences: `holdRelic`, `tradeRelic`, `speakForbidden`, `askToGoOut`, `unpermittedConception`, `contactAttempt`, `organise`, `violence`, `sabotage` |
| detection | Multiplies the daily detection chance of §6.13 for these offences (1 = normal) |
| penalty | First offence and repeat offence, from the penalty list below |
| enforcer | Who carries out the penalty: Sheriff, Judicial, or Silo 1 |
| visibility | 0–1: how widely an enforcement is seen. It scales the effect on fear and legitimacy (§6.12, §6.14) |

**Penalties:** `warning`, `confiscate` (relics), `prison(days)` and `mines(days)` (no work, no conversations), `cleaning` (death, public), `flagFamily` (covert lottery denial for the offender and their partner, §6.13). A `contactAttempt` can also bring Silo 1's response (§6.16), which the table doesn't control.

**The canon Pact** (default for `scenarios/canon.yaml`):

| id | Forbids | Offences | Detection | Penalty: first / repeat | Enforcer | Visibility | Source |
| --- | --- | --- | --- | --- | --- | --- | --- |
| noRelics | Owning or trading objects from before the silo | holdRelic, tradeRelic | 1.0 | confiscate + prison(90) / confiscate + prison(365) | Judicial | 0.5 | Show: relics banned down to a Pez dispenser (S1) |
| noForbiddenTalk | Speaking about the outside, the time before, or other silos | speakForbidden | 0.5 | warning / prison(180) + flagFamily | Judicial | 0.3 | Show: questioning the rules can lead to cleaning (S1); people who ask questions are quietly denied children (lore reference) |
| goingOut | Saying "I want to go out" | askToGoOut | 1.0 | cleaning / — | Sheriff | 1.0 | Show: the request is binding (S1E1) |
| lottery | Conceiving without a permit or removing one's own implant | unpermittedConception | 0.8 | prison(180) + flagFamily / cleaning | Judicial | 0.6 | Show: implants and lottery (S1E1) |
| noContact | Radio contact, digging toward other silos, opening the doors below | contactAttempt | 0.7 | cleaning / — | Judicial | 0.2 | Show: Safeguard and the Directive (S2–S3) |
| noViolence | Attacking officers or damaging the silo's systems | violence, sabotage | 1.0 | mines(730) / cleaning | Judicial | 0.8 | Show: S2 uprising |
| noOrganising | Unauthorised gatherings and strikes | organise | 0.6 | prison(90) / mines(365) | Judicial | 0.7 | Show: Mechanical treated as rebels (S2) [Likely] |

Detection weights, penalties and visibilities are guesses, to be checked by the sensitivity analysis (T15.3). Forbidden talk flags the family only on a repeat offence: flagging it on the first warning flagged 51% of a full silo within 56 years (T11.3), which would leave most couples childless once covert denial runs (Adam, 2026-10-31). Calibration (C1–C5) may tune them as a group, never row by row toward a wanted outcome.

**Rewriting the Pact** is a change to rows: a rule switched on or off, a penalty raised or lowered, or a new rule added. It happens through a scripted `policy` event or a playbook entry (§6.15), which can carry a legitimacy cost or benefit (its `lift`). Season 3's rewrite under Camille is the canon example. Two rules the show suggests the regime could add are *left out* (Adam, 2026-11-03): no research question needs them, and each would need new offences and new effects on conversations (§6.11):

| id | Forbids | Effect it would have |
| --- | --- | --- |
| travelPermits | Crossing zones without a permit | Fewer cross-zone contacts (§6.11), slower rebel gathering |
| curfew | Being away from one's home floor at night | Fewer contacts outside the household |

**In the scenario file** (keys fixed in T11.0), one entry per row under `pact:`:

```yaml
pact:                             # ODD §6.9, §6.13; used only when `control` is in modules
  - id: noRelics
    active: true
    offences: [holdRelic, tradeRelic]
    detection: 1.0
    first:  { confiscate: true, prison: 90 }
    repeat: { confiscate: true, prison: 365 }
    enforcer: judicial
    visibility: 0.5
  - id: noForbiddenTalk
    active: true
    offences: [speakForbidden]
    detection: 0.5
    first:  {}                    # a warning
    repeat: { prison: 180, flagFamily: true }  # and the couple quietly flagged
    enforcer: judicial
    visibility: 0.3
```

A penalty is a map of `prison` and `mines` (days, integers ≥ 1), and `cleaning`, `confiscate` and `flagFamily` (true or absent); `{}` is a warning. `repeat` may be left out, and is then the same as `first`. Checked at load: ids are unique; every offence is one of the nine above, and no offence is listed by two active rules, so each act has one weight and one penalty; a penalty holds at most one of prison, mines and cleaning; detection is ≥ 0; visibility is in [0, 1]; enforcer is `judicial`, `sheriff` or `silo1`. Judicial and the Sheriff share one arrest capacity (§6.13), so the enforcer has no effect until Silo 1 (§6.16); it is kept so the table reads like the show.

### 6.10 Location and travel (draft)

*Answers Q1 and Q2.* Distance decides who meets whom (§6.11), how fast rebels can gather and how fast Judicial reaches the Down Deep (§6.13–6.14), and a broken stairwell can cut the silo in two. The baseline has no walking (§2 Scales); this module adds it as an attribute plus multi-day trips, not as step-by-step movement (decision D3).

**Stairs.** The one spiral stairwell is a line of 143 segments; segment s joins floors s and s + 1. Each segment is intact or broken. All are intact on day 1, and only other modules (sabotage, M12) or scripted events break or repair them; this module never does. Two floors are *connected* when every segment between them is intact.

**Floors a person uses.** Their *home floor* is their household's floor (§2). Workers and shadows also have a *work floor*:

| Department | Work floor |
| --- | --- |
| Farms, Mechanical | The department's pure work floors (§5) |
| Supply, Judicial, Education, Medical, IT, Administration, Sheriff | The department's office floors, listed in the scenario (`space.officeFloors`). Office floors are residential floors in the department's zone; floor 1 holds the cafeteria and the screen |
| General services | The home floor: cafeterias, crafts and cleaning are on every floor |
| Porters | A floor drawn each day, uniformly from all floors connected to the home floor: porters carry goods through the whole silo |

Where a department has several floors, a person works on the one nearest their home floor, the lower-numbered one on a tie, among those connected to it. The work floor follows from today's home floor, so it changes when a person moves (§6.3) or changes department (§6.5). Shadows use the same rule as their department's workers, whatever floor their master is on.

**Where people are.** At the end of each day, at the location step (§3 step 16.5), every living person gets their *day location* for the next day, from the day's final state; day 1's are set when the module starts. Coming last, the step sees every change of the day (a new shadow, a retrained worker, a move), and nothing happens between it and the next day's first step:

- a worker or shadow is on their work floor (the baseline has no rest days: output is computed every day);
- everyone else (children, retired, sick leave) is on their home floor;
- a person *away* (below) is on the floor they travelled to, or *on the stairs* while travelling.

**Stranded.** A worker or shadow whose department has floors but none connected to their home floor is *stranded* for the day. They stay on their home floor and don't work: no output, no skill growth, no work injuries (§6.2), and they don't count as workers for coverage, targets or retraining, as on sick leave (§6.5). They keep their department, skill and master, and work again on the first day a department floor is connected. With every segment intact nobody is ever stranded. When the stairs break, this is how a split silo loses the workers who live on the wrong side, and through the §6.6 loop its power: a break around levels 90–92, as in S2, would leave Mechanical (floors 133–144) only the workers who live below it.

**Trips.** A trip moves one person from their day location to a destination floor. It is started by another module (arrests, M11; rebels gathering, M12), never by this one, so with no such module nobody travels.

```latex
hours = |from - to| \times \frac{H_{descent}}{143}, \qquad days = \max\left(1, \left\lceil \frac{hours}{H_{walk}} \right\rceil\right)
```

With a full descent of H\_descent = 12 hours of walking and H\_walk = 8 walking hours a day, floor 1 → 144 takes 2 days, and any trip within one zone (at most 47 floors, 3.9 hours) takes 1 day. One day is the model's tick, so even a short trip takes a day.

- A trip can start only between connected floors, and only for a person who is not already on one. Otherwise it is refused, and the module that asked decides what that means.
- The traveller is on the stairs from the departure day until the arrival day, departure + days, and is on the destination floor from the arrival day on (placed there by the location step the evening before). While on the stairs they don't work, as if stranded.
- If a segment on the route breaks during the trip, the traveller stops on the floor next to the break, on the side they left from, and arrives there on the planned day. Where on the route they were when it broke is not tracked.
- After arriving, the person is *away*: their day location is the floor they reached, and they don't work, until a module sends them on another trip; setting off ends being away. A trip back to the home floor ends being away, except for someone held by Judicial (§6.13), who stays away wherever they are held.
- A death ends the trip.

**State** (owned by this module; §2 is unchanged): the 143 segment states; for each person, their day location, whether they are stranded, and their trip or away floor. Home and work floors are not stored, because they follow from the household and the department.

**Off and null.** With the module off, none of this exists and the baseline runs as before. With the module on, all stairs intact and no module starting trips (the null setting), nobody is stranded, travels or is away. The only new random draws are the porters' floors, on this module's own stream, so the baseline's output files are byte-identical to a run with the module off.

**Parameters:** full descent, walking per day and office floors, in §7 (M8 done, 2026-10-30).

### 6.11 Social network (draft)

*Answers Q1.* Beliefs spread person to person (§6.12), and rebels join when enough of their ties have joined (§6.14), so who knows whom decides what people hear and who moves together. The network is ties, which last, plus conversations, drawn from the ties each day. It needs the space module, because coworkers share a work floor and nobody talks across broken stairs (§6.10).

**Ties.** A tie joins two living people; it has no direction, and a pair has at most one, of the first kind in this list that applies:

| Kind | Who | Strength |
| --- | --- | --- |
| Partner | The two partners (§6.3) | 1 |
| Household | Everyone living in the same household | 1 |
| Kin | Parent and child, and siblings (sharing a mother or a father), wherever they live: the relatives of §6.3 | 1 |
| Master–shadow | A shadow and their master (§6.5) | 1 |
| Coworker | Workers and shadows of the same department with the same work floor (§6.10); up to k each | w\_weak |
| Neighbour | People of shadowing age (14, §6.5) or older with the same home floor; up to k each | w\_weak |
| Porter | A porter and an adult (worker age, 18, §6.5) living in another zone; up to k each | w\_weak |

The first four, the *strong* ties, follow from the household, partner, family and shadowing records and change with them at once. The last three, the *weak* ties, are drawn, which is how the network gets its randomness. Porters have no coworker ties (they have no fixed work floor); their porter ties are the silo's main bridges between zones, next to kin who moved away. Children under 14 have only strong ties.

**Drawing and losing weak ties.** At the network step (§3 step 10.5), in the tick's shuffled order, anyone with fewer than k ties of a weak kind draws new ones, one at a time and uniformly at random, from the people eligible for that kind who also have fewer than k of it, until they have k or nobody is left. Both sides count the new tie. On day 1 everyone draws, in random order. A weak tie ends when either person dies, and also:

- a neighbour tie, when the two no longer share a home floor;
- a coworker tie, when the two no longer share a department and work floor: one retired, changed department or got another work floor (moving home, or a change in the stairs, can do that). Sick leave, being away or on the stairs, and being stranded (§6.10) don't end it: a worker cut off from every floor of their department keeps the work floor they had, for coworker ties, until they change department or can reach one again;
- a porter tie, when neither of the two is a porter any more (the porter retired or left Porters). Moving zone doesn't end it.

Each rule looks only at the two people's state today, not at what changed.

The lost ties are drawn again at the next network step.

**Conversations.** A tie is *used* on a day, meaning the two talk, with a chance set by its kind: p\_s for a strong tie, w\_weak × p\_s for a weak one. Each tie and day gets one draw, independent of every other tie and day. It can only be used if the two can reach each other: neither is on the stairs, neither is in custody or detained (§6.13), and the stairs connect their day locations (§6.10). A conversation is between two people, and either may pass something to the other (§6.12). On average a person talks p\_s × (strong ties + w\_weak × weak ties) times a day: about 3.5 for an adult with 4 strong and 10 weak ties at the defaults.

The draw for a tie and a day depends only on the run's seed, the day and the pair, never on who asks or in what order (a counter-based random stream, §4). Conversations are therefore not generated for the whole silo each day. A module that needs one asks whether a given tie is used today, and gets the same answer from either end. The cost then follows the people who have something to pass on, not the population: with every tie drawn every day the network took more run time than the whole baseline (T9.2).

There is no distance weight. Distance acts through who a person's ties are (neighbours share a floor, coworkers a work floor) and through one hard limit, the stairs: when they break, kin and porter ties across the break go silent until they are repaired.

**Outputs** (only when the module is on, as extra columns of yearly.csv): the mean number of ties per person and its 10th, 50th and 90th percentiles at the snapshot, and the expected number of conversations that day by tie kind (partner, household, kin, master–shadow, coworker, neighbour, porter): the sum of the daily chances of the ties whose two people can reach each other. Conversations are drawn only when asked for, so their expected number is what is logged.

**Off and null.** With the module off, nothing of this exists. With it on, the network changes no baseline state and draws only from its own stream, so `daily.csv` and `events.jsonl` are byte-identical to a run without it; `yearly.csv` keeps its baseline columns and gains the network's.

**Parameters:** weak ties per kind k, strong tie daily chance p\_s and weak tie strength w\_weak, in §7 (M9 done, 2026-10-31).

### 6.12 Beliefs, knowledge and memory (draft)

*Answers Q1 and Q3.* Legitimacy is the core of the rebellion model (§6.14): what people know lowers it, and what they forget lets it recover. Knowledge passes in conversations (§6.11), fades, and dies with the people who hold it. Relics and the Flamekeepers keep the true history alive; Vitamin D+, Silo 1's memory drug disguised as vitamins (lore reference, Season 3), hides it. The module needs the network (§6.11), and through it the space module.

**State** (owned by this module; §2 is unchanged). For each person:

| Variable | Meaning |
| --- | --- |
| base | 0–1: the legitimacy they were raised with (upbringing, below) |
| flags | For each of five facts: unknown, known (heard or lived), or hidden by Vitamin D+ |
| dosedUntil | Last day of their Vitamin D+ dose, or none |
| risk aversion R | 0–1, drawn uniformly at birth (on day 1 for the starting population) and never changed; used by activation (§6.14), as in Epstein (2002) |
| member | Whether they are a Flamekeeper |

And for each relic: the floor it is hidden on, and the person holding it, if anyone does.

**The five facts** (decision D4: one number and five flags, no free-text beliefs):

| Flag | The person knows that … | Sources besides conversations |
| --- | --- | --- |
| outside | the outside may be survivable | a cleaning failure (§6.13) |
| screen | the view of the outside on the screen is a lie | a cleaning failure (§6.13) |
| history | the true history: the world before the silo and the rebellions the regime erased | Flamekeepers and relics (below); the scripted event `revealHistory` |
| silos | other silos exist | contact with other silos (§6.16) |
| lottery | the lottery is rigged | the implant doctors, who see each denied win (§6.13) |

Every flag spreads in conversations. With this module alone, only history has a source; outside, screen and lottery get theirs from M11 and silos from M14. Tests seed them by hand.

**Legitimacy.** L, 0–1, is how much a person believes the Pact and the leaders are right:

```latex
L = \min\left(1,\; \max\left(0,\; base - \delta \times \text{flags known} + S\right)\right)
```

S is the silo-wide lift from cleanings (§6.13); it is 0 without the control module. Hidden flags don't count. Learning a flag lowers L by δ at once; forgetting it, or having it hidden, gives the δ back.

*Upbringing.* Until worker age (18, §6.5), a person's base is recomputed every 30 days, on the days the monthly steps run (§3), from Education and from the adults who raise them:

```latex
base = \varepsilon \times L_{edu} + (1 - \varepsilon) \times \text{mean } L \text{ of their living parents and, for a shadow, their master}
```

with base = L\_edu when none of them is alive. A newborn starts with base = L\_edu until the next recompute. At worker age the base is fixed for life, at its last recomputed value. Upbringing is slow, so a month's lag changes nothing measured; recomputing it daily cost about 3% of a run's time (T10.1). L\_edu is the legitimacy the regime teaches, in Education and on Freedom Day: its propaganda level, a policy lever (§6.15). Everyone alive on day 1 starts with base = L\_edu, so with nothing known, L = L\_edu for everyone, for ever; the rule itself never drifts. Doubt runs in families: a child whose parents know something grows up with a lower base before learning anything, and the effect shrinks by the share ε with each generation that learns nothing.

Hardship is not part of L. It enters grievance, G = H × (1 − L) (§6.14), so it isn't counted twice. Later modules shift L silo-wide only through S, and say so in their own sections: cleanings (§6.13), and a playbook's lifts, Pact rewrites and elections (§6.15). L\_edu itself is a policy lever (§6.15).

**Telling and believing.** In a conversation (§6.11), each flag the teller knows and the listener doesn't passes to the listener, who knows it from then on as *heard*, with chance

```latex
\pi \times (1 - L_{teller}) \times (1 - L_{listener})
```

Telling takes doubt as much as believing does: someone who still trusts the regime keeps what they heard to themself. With the listener's doubt alone, every knower told about five others before forgetting (R₀ ≈ 5, T10.3), so the history was known by three quarters of adults before any rebellion, mean L was 0.3, and steady trials for talk kept fear at its ceiling (T12.8). With the teller's doubt too, R₀ is about 1 at L\_edu, and it rises as the silo loses faith, so a shock can still make the history spread (Adam, 2026-11-02).

Either of the two can be the teller, so a tie between two knowers works both ways. The chance is the same for every flag and every tie kind: kin and households pass things on faster only because strong ties talk more often (§6.11). With the control module on, fear F (§6.13) silences talk with acquaintances: over a weak tie (coworker, neighbour, porter) the teller dares to tell only with chance 1 − F, so the chance becomes π × (1 − F) × (1 − L\_teller) × (1 − L\_listener). Family is safe: over a strong tie fear changes nothing. People who believe the regime are hard to convince, and once someone doubts, the next fact is easier to believe, because every flag they learn lowers their L. A dosed listener learns nothing; a listener with the flag hidden may recall it instead (Vitamin D+, below). A flag learned today is passed on from the next day.

Only conversations that involve a knower are drawn (§6.11): the step visits the people who know at least one flag, in an order shuffled each day. The chances are the ones above, but they are drawn rarest first: whether a flag would pass at π, then at 1 − L, and only then whether the tie is used today. Almost nothing passes on a given day, so most knowers are passed over without reading their ties, and the cost follows the knowers, not the population or their ties (T10.3: asking every tie cost 1 ms a day with 80% of the silo knowing a flag).

**Forgetting.** A known flag is lost for good with a daily chance set by how it was learned:

```latex
P(\text{lost today}) = 1 - 2^{-1/(365\,T)}, \qquad T = T_{heard} \text{ or } T_{lived}
```

*Lived* knowledge comes from the person's own experience: a relic, an event they saw (`revealHistory`; later the cleaning failure), or being a Flamekeeper on day 1. It fades more slowly than *heard* knowledge. A lost flag is unknown again and can be learned again like any other. Flamekeepers never forget the history. All of a person's knowledge dies with them.

**Vitamin D+** (decision D5). While dosed, each day each known flag becomes *hidden* with chance d, and the person can't learn a flag or recall a hidden one. Their base is unchanged, so their L climbs back toward the base as flags are hidden. Two forms:

- *Water dosing:* the scripted event `waterDosing` doses everyone alive, and everyone born while it lasts, for D\_water days from the event's day, except those the Flamekeepers shield (below). A second event while one runs extends it to D\_water days from the later event.
- *Targeted:* another module (§6.13, §6.15, §6.16) doses named people until a day it sets. Scripted events can't name people, so this form has no event type.

*Warning the Flamekeepers.* Only a few insiders in IT and Medical know that Vitamin D+ exists and what it does. On the day a water dosing starts, the Flamekeepers are *warned* if a member works in IT or Medical (as a worker or shadow, §6.5), or has a strong tie (§6.11) to someone who does and who knows the history: a loyal insider doesn't tell. A warned group knows how to avoid the water, so for that dosing, members and everyone living in a member's household are not dosed by it; someone who joins or moves in while it lasts is shielded from then on, and someone who leaves is not. Each water dosing event is judged afresh on its own day. A targeted dose works on members as on anyone. An unwarned group is dosed like everyone else, and can be erased by it (below).

A hidden flag never fades on its own. Once the dose has ended, it comes back, known again as heard or lived as before, with chance κ per *cue*:

- each day the person holds a relic (one draw per hidden flag);
- each conversation with someone who knows that flag, with chance κ × (1 − L), L the person's own, in place of the telling chance: it is their own memory, so the teller's trust doesn't matter, but someone who trusts the regime again dismisses the reminder. With κ alone, 95% of a silo recalled a dosed-away history within a year of the dose ending, and no wipe could make a rebellion forgotten (T13.2, Adam, 2026-11-04).

The conversations are drawn as in telling (above), from the side of the people with a hidden flag, visited in the order they first had one hidden. This is how Juliette's memory came back (S3).

**Relics.** Relics are rare and hard to find, and whoever has one keeps it out of sight.

- *Hidden:* on day 1, N\_relic relics are hidden, each on a floor drawn uniformly from all 144. Each day each hidden relic is found with chance 1 / (365 × T\_find), so it stays hidden T\_find years on average, by a person drawn uniformly from those aged 14 or more (shadowing age, §6.5) whose day location (§6.10) is its floor; with nobody there it stays hidden. The finder holds it.
- *Held:* a holder keeps the relic hidden in their flat, so it lies on their home floor. They can use it every day, and nobody else sees it: the regime finds it only by searching or by catching it change hands (§6.13). A person can hold several.
- *Used:* each day, for a holder who isn't dosed, their relics work like a cue: each hidden flag comes back with chance κ, and if they don't know the history, they learn it as lived with chance κ. Holding several relics gives no more draws than holding one.
- *Handed over:* each day a holder hands each relic, with chance 1 / (365 × T\_keep), to someone aged 14 or more they talk with that day (§6.11), drawn uniformly from them; with nobody to talk to, they keep it. A relic is kept T\_keep years on average, and travels along the network, so the best-connected people, Porters among them, become the dealers. Selling and giving are the same act, since the model has no money, and both are the Pact's `tradeRelic` offence (§6.9).
- *Inherited:* when a holder dies, each relic goes to a living member of their household aged 14 or more, drawn uniformly; with none, it stays hidden in the flat, on its floor, to be found when the flat is cleared.

Relics are not lost or destroyed in this module. The regime can find and take them through Judicial searches and raids (§6.13) and destroy them as part of a memory wipe (§6.15); a relic taken leaves the silo's stock for good.

**Flamekeepers.** A secret group of N\_F seats that keeps the true history alive. On day 1, N\_F adults drawn uniformly take the seats and know the history as lived. Members never forget it, and tell what they know like anyone else. Each day, each empty seat goes to an adult who knows the history, isn't a member, and talks today (§6.11) with a member over a strong tie; one is drawn uniformly from all such people. With nobody who qualifies, the seat stays empty. A member who dies leaves a seat, and so does one whose history flag is hidden by Vitamin D+; later modules may remove members too (arrests, §6.13). With no members left, the group is gone for good.

*The Head of IT.* With the control module on, the Head of IT (§6.13) knows all five flags as lived, and while in office never forgets them, is never dosed, never tells anything, can't take a Flamekeeper seat and never warns the Flamekeepers. Telling and handing over a relic are offences there (§6.13), and people held by Judicial have no conversations and can't use their relics.

**Step.** The beliefs step (§3 step 10.7) runs after the network has drawn the day's ties, in this order:

1. newcomers get R and base = L\_edu; every 30 days, the bases of people under worker age are recomputed;
2. relics held by people who died today pass on;
3. forgetting, then dosing hides flags;
4. hidden relics are found, relics are handed over, and they work on their holders;
5. conversations: telling, believing and cued recall;
6. empty Flamekeeper seats are filled.

**Scripted events** (§5):

- `revealHistory`: everyone alive aged 14 or more and not dosed learns the history as lived; someone who had only heard it now knows it as lived. It stands in for a crushed rebellion that everyone saw, so pattern C4 can be measured before the rebellion modules exist (T10.3). Since T13.0 a real rebellion's ending teaches the same (§6.15).
- `waterDosing`: water dosing, above.

**Outputs** (only when the module is on):

- *yearly.csv:* mean L of adults (18+), overall and by zone of their home; the share of adults who know each flag; the share of adults with at least one hidden flag; people dosed; relics hidden and held; Flamekeepers.
- *frames.csv.gz:* per floor, the mean L of adults and the share of adults who know the history.

**Invariants** (§8): a flag is never both known and hidden; every Flamekeeper is a living adult who knows the history; every relic is hidden on a real floor or held by a living person, and there are N\_relic of them, less those taken by the regime (§6.13, §6.15); base and L lie in 0–1.

**Off and null.** With the module off, nothing of this exists. With it on, it changes no baseline state and draws only from its own stream, so `daily.csv` and `events.jsonl` are byte-identical to a run without it, whatever its settings; `yearly.csv` keeps its baseline columns and gains the module's. In the null setting (no Flamekeepers, no relics, no events) nobody ever knows anything and L = L\_edu for everyone.

**Parameters:** taught legitimacy L\_edu, Education weight ε, legitimacy cost per flag δ, pass chance π, the two half-lives, hiding while dosed d, water dosing length D\_water, cue chance κ, relics N\_relic, relic hiding and keeping times T\_find and T\_keep, and Flamekeeper seats N\_F, in §7 (M10 done, 2026-10-31). Risk aversion R is uniform on 0–1, as in Epstein (2002); it adds no parameter.

### 6.13 Regime control (draft)

*Answers Q1 and Q3.* The deterrence half of the rebellion model: what the regime sees, whom it takes, and what follows. Offences and penalties come from the Pact table (§6.9); this section says how an offence is seen, how the offender is brought in and tried, and what each penalty does. It also gives the silo its Head of IT (decision D13). The module needs the beliefs module (§6.12), and through it the network and space modules: an arrest is a trip (§6.10).

**State** (owned by this module; §2 is unchanged): the Head of IT and their successor; the implant doctors; the open cases; for each person, whether they are in custody or detained and until when, the Pact rules they were convicted under, and whether they are flagged; for each woman, her denied wins and the day the last one ends; the lift S and the fear F.

**Head of IT** (D13). The most powerful person in the silo: one IT worker who alone enters the vault, where the Algorithm tells them what the silo hides (lore reference, Season 3).

- *Knowledge.* On taking office the Head learns all five flags as lived. While in office they never forget them, are never dosed (by water or a targeted dose), never tell anything in a conversation, can't take a Flamekeeper seat and never warn the Flamekeepers (§6.12). They know everything and pass on nothing, for life: the post ends only with death (below), so there is never a former Head to tell what the vault holds. A Head whose successor told everything on retiring spread all five flags to 90% of the silo within 50 years (T11.3).
- *The first Head* is chosen on day 1: the most skilled IT worker, ties to the lower ID, as for the department heads of §6.8.
- *Successor.* The Head trains their own. A Head without a successor names their shadow (§6.5), the older of two (lower ID on a tie). A Head with no shadow takes one if they can be a master (§6.5): the youngest IT shadow (lower ID on a tie), whose master becomes the Head. The successor keeps the role after becoming a worker. It lapses if they die, retire or are arrested; a new Head starts without one.
- *Leaving office:* only death (Adam, 2026-10-31). The Head keeps the post through sick leave and past retirement age, can't be charged with any offence, and so is never arrested; a relic found in their flat is still taken. The post is filled at the next control step by the successor, whatever their stage, even a shadow; with none, by the most skilled IT worker. With no IT worker it stays empty and is retried each day. A Head who has stopped working (retired, or on sick leave) can't take on a new shadow, as no such person can be a master (§6.5).
- *What the Head decides.* In this module, nothing: surveillance, raids and the Pact are scenario settings. With the regime module, the playbook that changes them is the Head's, and a vacant post starts no response (§6.15); Silo 1 speaks to the Head in §6.16.

**Offences.** An offence is an *act* by one person, reported by the module where it happens, under the active Pact rule that lists it; an act no active rule lists is not an offence. Acts by children under 14 (shadow age, §6.5) are not offences. With this milestone, three exist:

| Offence | What counts | Reported by |
| --- | --- | --- |
| speakForbidden | Each *telling* (§6.12): a flag's draw at π lands for a listener who doesn't know it, and the two talk today, whether or not the listener believes it. Cued recall is not a telling | Beliefs |
| tradeRelic | Each handover (§6.12); one act by both giver and receiver | Beliefs |
| holdRelic | Holding is not an act: it is found only by searches (below) | — |

The rebellion module (§6.14) reports `organise` (each day a person is active), `violence` (each day a person is violent; a rebel arrested in a fight is charged with it without a detection draw) and `sabotage` (each sabotage). `contactAttempt` comes with Silo 1 (§6.16), and walking out with it, so no module produces `askToGoOut` or `unpermittedConception` yet, and their rows have no effect.

**Detection.** Each act is seen with chance

```latex
p = \min\left(1,\; \sigma \times c_{IT} \times w_{rule}\right)
```

where σ is the surveillance level (a policy lever, §6.15), c\_IT is IT's coverage today (§6.6), and w\_rule is the rule's detection weight (§6.9). The control step draws once for each act reported since the last control step, in the order they were reported. A seen act opens a *case* against the offender (for a handover, against both). With σ = 0 nothing is ever seen: the null setting.

**Cases.** A case is a person and the rules they are charged under. A person has at most one open case: a detection against someone whose case is open, or who is in custody awaiting trial, adds the rule to it. Against someone serving a sentence, it opens a case that waits until they are released.

**Arrests.** Each day Judicial arrests up to ⌊a × officers⌋ people, where officers are the workers of Judicial and the Sheriff who are not off work (§6.10). Open cases are taken oldest first, in the order they were opened. An arrest starts a trip (§6.10) to the *court*: the Judicial office floor nearest to where the person is, among those connected to it (lower on a tie). If the trip can't start (they are on the stairs, or no court floor is connected), or their day location is in a zone the rebels hold (§6.14), the case waits and uses no capacity. Rebels arrested in a fight (§6.14) are sent to the court the same way but use no capacity. From the arrest to the trial the person is *in custody*. On arrest:

- their flat is searched (below);
- a Flamekeeper loses their seat and, with chance p\_name, names one other member, drawn uniformly; the named member is charged with `speakForbidden`, with no detection draw, if a rule listing it is active. A named member may name another when arrested in turn, so arrests can roll the group up one by one (T10.5), but the chain often stops. Naming each member with chance p\_name wiped out all 15 in the first year of every run (T11.2: about 4 named per arrest).

**Trial.** On the day they reach the court. For each rule in the case the penalty is the first-offence one if they were never convicted under that rule, the repeat one otherwise; every rule in the case counts as convicted from then on. They serve the most severe detention among those penalties (cleaning, then mines, then prison, then none; the longer of the same kind), and every confiscation and family flag among them.

**Penalties** (§6.9):

- *Warning* (no detention): free at once; they walk home.
- *Prison(days):* detained on the court floor until the trial day + days.
- *Mines(days):* a trip to the mines floor, detained there until the trial day + days (the walk counts). If the stairs to the mines are broken, the sentence is served in the court's prison.
- *Cleaning:* carried out the next day (below).
- *Confiscate:* every relic the offender holds leaves the silo for good.
- *FlagFamily:* the offender and their partner, if any, are flagged for life (covert lottery denial, below); their children are not. Flagging the whole household for life, children included, spread the flag down family lines: with σ = 0.1 three quarters of a small silo were flagged within 60 years and births nearly stopped (T11.4).

People in custody or detained have no conversations (§6.11) and can't use their relics; detained people are away (§6.10), so they don't work. They keep their household, department, skill and partner. On the release day they are free and start a trip home; if the stairs are broken they stay where they are, free, and try again each day. Arrests don't take officers away from work: capacity is abstract.

**Places.** Prison and the mines hold at most `prisonPlaces` and `minesPlaces` people at once, counted by where they were sentenced to (a mines sentence served in the court's prison, with the stairs broken, still takes a mines place); 0 means no limit. A detention with no free place *waits*: the convict walks home free and keeps working, talking and holding their relics, and joins the queue for that place in the order of their trials. Each day, after the releases, the first in each queue take the places that freed: their detention starts that day, for the days of their sentence, with a trip to the nearest court they can reach (prison) or to the mines floor (mines; the court's prison if the stairs to it are broken). One who can't start the trip (on the stairs, cut off from every court, or in a zone the rebels hold, §6.14) keeps their place in the queue until a later day. A Flamekeeper seat taken while waiting is lost on admission, as on an arrest. While waiting, a new case against them waits as it would during the sentence. A rebel put out in a fight (§6.14) is taken into custody as usual and leaves the queue; the new trial decides what they serve. Custody before trial and cleaning take no place. The show has a few cells in the Sheriff's and Judicial offices and a labour crew in the mines (lore reference). Unlimited detention held 755 people at once after a cleaning failure, 7.5% of the silo, more than the baseline's 5% food margin, so the food stock ran out and any generator failure became a famine that killed the silo (T12.8, Adam, 2026-11-03).

**Searches.** A search is of one flat: each relic held by someone living there is found with chance p\_find, because relics are hidden well (§6.12). A found relic is confiscated and its holder is charged with `holdRelic`. Flats are searched on every arrest, and in *raids*: each day, with chance R\_raid / 365, Judicial raids one residential floor drawn uniformly and searches every flat on it. Relics hidden in the floors themselves, not yet found by anyone, are never found by the regime.

**Cleaning.** The condemned stays in custody overnight and goes out the next day: they die at that day's deaths step (§3 step 7), handled as any death (§6.1: partner, orphans, job vacancy, and counted for the lottery). At that day's control step, with chance p\_fail the cleaning turns out to have *failed*: the cleaner refused, walked out of sight and never came back, which the silo records as a death too. With v the visibility of the rule they were sentenced under (§6.9):

- *Cleaned:* the silo-wide lift S (§6.12) rises by λ × v, and halves every T\_S years, so everyone's L is a little higher for a while and dissent (G = H × (1 − L), §6.14) falls.
- *Failed:* the cleaning-failure shock. Each living person aged 14 or more who is not dosed watches with chance v, and learns the outside and screen flags as lived. Nobody else learns anything directly, but the watchers tell others (§6.12). There is no lift.

**Fear.** F, 0–1, is how afraid the silo is of being punished. Every trial adds φ × v, with v the visibility of the most visible rule in the case: a warning for talk (v = 0.3) is barely noticed, a public cleaning (v = 1) is seen by everyone. F stays at most 1 and halves every T\_F years, so a crackdown makes people go quiet and a long peace lets them talk again. Fear acts on talk with acquaintances (§6.12, Telling and believing), not on talk in the family, so it slows the spread of knowledge across the silo while doubt still runs in families. It counts trials, not trials per head: a smaller silo has fewer of them and so less fear. Activation reads F too: it raises everyone's felt danger of arrest (§6.14). F starts at 0; the warm-up years let it settle.

**Covert lottery denial** (§6.4). A couple is flagged if either partner is. A flagged couple can be drawn in the lottery as usual and believes it has won, but no implant is removed: the year passes with no child. The draw goes on to the next couple, so a denied win uses up no permit and the silo's births are not cut, only the flagged couple's (Adam, 2026-10-31: when each denied win used up a permit, a small silo with σ = 0.3 died out within 70 years). For 365 days from the draw she is not eligible, as with an open window, but the lottery does not count her in E (the regime knows the window is empty). After that she can enter and win again, and is denied again. Medical puts it down to infertility, so the couple learns nothing. Only the *implant doctors* know: N\_doc Medical workers who do every implant removal, so the regime keeps the secret to as few people as it can. Each denied win teaches the lottery flag, as lived, to every implant doctor not dosed that day, and they can tell others like any knower (§6.12). On day 1 the posts go to the N\_doc most skilled Medical workers; a doctor who dies, retires or is arrested leaves a post, filled at the next control step by the most skilled Medical worker not holding one. The Head of IT knows it too, and never tells. After N\_try denied wins she believes it and stops entering, never eligible again, whoever her partner is later. With nobody flagged the draw stops after N winners, as in the baseline.

**Step.** The control step (§3 step 10.8) runs after the beliefs step, in this order:

1. newcomers get their state;
2. Head of IT: the post filled if empty, the successor named; empty implant doctor posts filled;
3. detection of the acts reported since the last step;
4. releases; then the waiting convicts take the freed places; then the rebels arrested in yesterday's fights (§6.14) taken into custody, with their searches and naming;
5. cleanings of those who died at today's deaths step: their effects; then trials of those who reached the court today, each raising F;
6. a raid, perhaps;
7. arrests, with their searches and naming;
8. each lottery win denied since the last step teaches the implant doctors;
9. the lift and the fear decay.

**Scripted events** (§5): `cleaningFailure`, a failed cleaning that everyone sees (v = 1) with no cleaner: everyone alive aged 14 or more and not dosed learns the outside and screen flags as lived. It is the shock of pattern C2 (§1.2).

**Outputs** (only when the module is on):

- *yearly.csv:* over the year, acts seen, arrests, trials by penalty served (warning, prison, mines, cleaning), failed cleanings, relics confiscated, raids, denied wins and children born to flagged parents (an outcome, T11.4); at the snapshot, people in custody or detained, flagged people alive, open cases, S and F.
- *events.jsonl:* `cleaning` (person, rule, failed or not) and `headOfIT` (person, whether they were the successor).

**Invariants** (§8): the Head of IT, if any, is a living IT worker who knows all five flags and holds no Flamekeeper seat; every implant doctor is a living Medical worker; everyone in custody or detained is alive and holds no seat; every open case is against a living person; relics hidden, held and taken add up to N\_relic; S ≥ 0 unless a playbook lift was negative (§6.15).

**Off and null.** With the module off, nothing of this exists and the lottery is fair. With it on, σ = 0, no raids and no scripted events, nobody is seen, arrested, flagged or cleaned. The only changes are the Head of IT, who knows all five flags, and the IT shadow a Head takes on as successor. That shadow's new master shifts the work module's later draws (§6.5: free masters are drawn), so the run's files differ from a run without the module by chance, though nobody is charged and no rule of the baseline acts differently (T11.1).

**Parameters:** surveillance σ, arrests per officer a, raids R\_raid, find chance p\_find, naming p\_name, the mines floor, prison and mines places, failed cleaning p\_fail, cleaning lift λ and its half-life T\_S, fear per trial φ and its half-life T\_F, implant doctors N\_doc and tries N\_try, in §7 (M11 done, 2026-10-31); the Pact table's weights, penalties and visibilities are in §6.9. In the scenario file they are the `control:` section:

```yaml
control:                          # ODD §6.13; used only when `control` is in modules
  surveillance: 0.05              # σ: chance an act is seen at full IT coverage and detection weight 1
  arrestsPerOfficer: 0.05         # a: arrests a day per Judicial or Sheriff worker
  raidsPerYear: 4                 # R_raid
  findChance: 0.2                 # p_find: per held relic and search
  nameChance: 0.3                 # p_name: an arrested Flamekeeper names one other member
  minesFloor: 144
  prisonPlaces: 50                # prison cells held at once; 0 = no limit
  minesPlaces: 100                # mines places; 0 = no limit
  cleaningFailChance: 0.05        # p_fail
  cleaningLift: 0.05              # λ: lift of L from a cleaning seen by all (× visibility)
  liftHalfLifeYears: 1            # T_S
  fearPerTrial: 0.01              # φ: fear added by a trial seen by all (× the rule's visibility)
  fearHalfLifeYears: 1            # T_F
  implantDoctors: 2               # N_doc: Medical workers who do the implants and know of the denials
  deniedTries: 3                  # N_try: denied lottery wins before a flagged couple stops entering
```

Checked at load: σ, a, p\_find, p\_name, p\_fail and λ are in [0, 1]; R\_raid is in [0, 365]; φ is in [0, 1]; T\_S and T\_F are strictly greater than 0; N\_try is an integer ≥ 1; N\_doc and both place counts are integers ≥ 0; the mines floor is in the silo. That it is a Down Deep floor, and that Judicial has office floors, is checked when the module starts.

### 6.14 Grievance and rebellion (draft)

*Answers Q1 and Q2.* The dissent half of the rebellion model, after Epstein's civil violence model (2002): a person turns against the regime when their grievance outweighs their fear of arrest, comes out with others once enough of the people they know have, and the angriest go on to fight and sabotage. The module needs the control module (§6.13), and through it the beliefs, network and space modules; strikes act through work (§6.5), sabotage through resources (§6.6).

**State** (owned by this module; §2 is unchanged): for each person, their stage (below), their own hardship h\_own and the day of their last loss; for each zone, whether rebels hold it and since when; the water pump, running or down until a day; for each stair segment, the day it last broke; the zone counts of the day before; the open unrest spell and rebellion, if any, with what the log needs (§1.2).

**Hardship.** H, 0–1, is how hard a person's life is. It is recomputed every day for every adult (worker age, 18, §6.5) from four sources, combined like independent chances, so that H stays within 0–1 without a clamp:

```latex
H = 1 - (1 - h_{own})(1 - h_{food})(1 - h_{loss})(1 - h_{DD})
```

- *Own:* h\_own, drawn uniformly on 0–1 at birth (on day 1 for the starting population) and never changed, as Epstein's hardship (2002). It stands for everything about a life the model does not track, from a cramped flat to a cruel supervisor. Like R (§6.12), it adds no parameter.

- *Food:* h\_food = max(0, today's hardship of their home zone) (§6.7): 0 in a fed silo, 0.1 under rationing.
- *Loss:* h\_loss = ℓ × 2^(−t / T\_loss), with t the years since their last loss, and 0 if they never had one. A loss is a family member (partner, household or kin, the first three tie kinds of §6.11) cleaned (§6.13, failed or not) or killed in a fight (below), or, for both partners, a denied lottery win (§6.13). The couple puts the denial down to infertility, but a child that never comes is a loss whatever its cause. A new loss starts the clock again; losses don't add up.
- *Scapegoat:* h\_DD = s\_DD if their home is in the Down Deep, else 0. The regime blames the lowest class when trouble rises, as Mechanical learns in S2 (lore reference). s\_DD is a policy lever: The Order raises it while unrest runs (§6.15).

In a fed silo with no losses H is h\_own Up Top and in the Mids, and 0.2 + 0.8 h\_own Down Deep. Without h\_own, H was 0 for four fifths of the silo and G at most 0.2, so even with fear off and L near 0 at most 1% of the silo was ever active, and the canon silo never rebelled (T12.8; Adam, 2026-11-02).

Health is not a source (Adam, 2026-10-31). H is hardship people blame on the regime, as in Epstein; illness and bad air are physical. With health in H, a strike spiralled: a strike cuts air (§6.6), bad air lowers health, higher H brings out more strikers. At the canon settings an ordinary generator outage, eight years after a cleaning failure, set it off, and every adult in a 1,000-person silo was dead within 100 days (T12.4). Rationing still feeds back, but it adds at most 0.1 and only after the food stock falls below 30 days.

**Grievance.** G = H × (1 − L), with L the person's legitimacy (§6.12), as in Epstein (2002). Hardship alone breeds no revolt among people who believe the regime is right: at L\_edu = 0.8 even H = 1 gives G = 0.2.

**Arrest estimate.** A person judges their chance of arrest, if they turned active, from the zone they are in, by their day location (§6.10):

```latex
P = 1 - \exp\left(-k \times \frac{O_z}{A_z + 1}\right), \qquad O_z = officers \times \frac{a_z}{a}
```

A\_z is the number of adults whose stage was active or beyond yesterday and whose day location was in zone z; the +1 is the person themselves, as in Epstein. O\_z is the zone's *patrol*: the officers on duty (§6.13: Judicial and Sheriff workers not off work) spread over the silo in proportion to adults, where a\_z is the number of adults whose day location yesterday was in z, on a floor connected to a court floor (§6.13), and a the same count over the whole silo. An adult on a floor cut off from every court has O = 0 and so P = 0: the regime can't reach them, and they know it. With k = 2.3, P = 0.9 when the patrol equals the actives plus one (Epstein's choice). As P falls when unrest grows faster than the patrol, a few more actives can tip many others: Epstein's outbursts. The zone rule has one balance point for each number of officers, so a steady fall in officers gives a steep tip, not a jump (T12.2: 1,000 doubters, officers cut from 200 to 0 one a day, a quarter of the rise within 10 days); outbursts that come and go need arrests and releases (§6.13). Patrols are not people on floors: like arrest capacity (§6.13) they are abstract, and no officer moves. The patrol follows the adults; no policy lever moves it (§6.15).

**Stages.** Each day every adult takes one *stage*: calm, active, joined or violent, each needing the one before. Calm are all who are not eligible: officers on duty, the Head of IT, people held by Judicial (§6.13) and people on the stairs (§6.10). Fear F (§6.13) raises the felt danger of arrest the way the sources of hardship combine, so the *net risk* is

```latex
N = R \times \left(1 - (1 - P)(1 - F)\right)
```

with R the person's risk aversion (§6.12). Then:

- *Active* if G − N > T. Active people speak out and agitate: each day they commit the offence `organise` (§6.9).
- *Joined* if active and at least a share R of their ties (§6.11, any kind) were active or beyond yesterday. People come out together (Granovetter 1978), and the risk-averse need more of the people they know beside them. Someone with no ties never joins.
- *Violent* if joined and G − N > T\_v, with T\_v ≥ T: the angriest and least deterred fight. Each day they commit the offence `violence` in place of `organise`.

Every stage is computed from today's G and F and from yesterday's stages and zone counts, so the order people are visited in doesn't matter and the step draws nothing for it. A stage has no memory: a person is active exactly on the days the rule says so. Officers never turn against the regime (no defection), and rebels never march: they act where their day location already puts them.

**Strikes.** From the day after they join, a joined worker or shadow is *on strike*: they stay on their work floor but don't work, as if stranded (§6.10: no output, no skill growth, no work injuries), except that they still count as workers for targets and retraining (§6.5), so the regime doesn't replace them. A Mechanical strike lowers mechanicalCoverage and through the §6.6 loop power, food and air (pattern C3). A department is *on strike* (§1.2) on a day when at least a share q\_strike of its workers are. Nothing ends a strike but the strikers' own grievance.

**Fights** (decision D6). Each day, in each zone (Up Top, Mids, Down Deep, in that order) where violent rebels on floors connected to a court, B\_z of them, meet a patrol O\_z > 0, the two sides fight. By Lanchester's square law each side loses in proportion to the other's size:

```latex
rebels\ out = \min(B_z,\; e_J \times O_z), \qquad officers\ out = \min(O_z,\; e_R \times B_z)
```

A loss that is not a whole number is rounded down, plus one with a chance equal to the fraction. The rebels put out are drawn uniformly from the B\_z; the officers uniformly from all officers on duty, since patrols are abstract. Each one put out is killed with chance p\_kill: they die at the next day's deaths step (§3 step 7) with the cleaners, handled as any death (§6.1), and their family, a rebel's or an officer's, has a loss (above). A rebel not killed is arrested: control takes them into custody at its next step, charged with `violence` with no detection draw and using no arrest capacity, and tries them like anyone arrested (§6.13); with no active rule listing violence there is nothing to charge them with, and they stay free. A rebel put out is calm from then on. An officer not killed is wounded: a work injury (§6.2). Officers are armed and rebels are not, so e\_J > e\_R: at the defaults a patrol beats up to √(e\_J / e\_R) ≈ 2.2 times its number.

**Held zones.** Rebels *hold* a zone on a day when its violent rebels, wherever they are in it, outnumber its patrol. Judicial arrests nobody whose day location is in a held zone (§6.13); their cases wait. Holding Up Top, where the IT floors and the Mayor's office are, for 30 days in a row is a regime change (§1.2). It ends the rebellion that day and is logged, and the count of held days starts again; what the new regime does is the scenario's `regimeChange` playbook (§6.15). Rebels don't march, so only Up Top's own people can hold it: in 19 seeds × 30 years of a 1,000-person silo with a cleaning failure and s\_DD up to 0.6, rebels held the Down Deep 4 times and Up Top never (T12.5).

**Sabotage.** Each violent rebel sabotages with daily chance p\_sab, visited in the tick's shuffled order (§3). The target follows from their day location:

- *On a Mechanical floor* (§5): the generator if it is running, else the water pump if it is running, else the stair segment as below.
- *On a floor below every court floor:* the stair segment between their floor and the next floor up (segment f − 1 for floor f), if it is intact. Breaking it cuts their floor and every floor below it off from the courts, and so from patrols and arrests.
- *Anywhere else* there is nothing to sabotage.

Effects:

- *Generator:* it fails at the next generator step as if the failure draw had come up (§6.6: condition − 0.1, an outage, 30 parts).
- *Water pump:* from the next day water is 0 (§6.6) until it is repaired, which takes the same work as a generator failure, worked out by the resources module at its next life-support step: ⌈3 / (skill × partsShare)⌉ days, at most 10, and 30 parts. Silo 17's raider wrecked the groundwater pump (lore reference). Water harms nobody's health in the baseline; a pump kept down kills the silo through the collapse check below.
- *Stair segment:* it is broken (§6.10).

Each sabotage is the offence `sabotage` (§6.9).

**Stair repairs.** Each day each broken segment, visited from 1 to 143, is repaired with chance mechanicalCoverage / D\_stair, unless a floor at either end lies in a held zone. Segments broken by scripted events are repaired the same way. A segment still broken 90 days after it broke, while the rebellion that saw it break still runs, is a split (§1.2).

**Unrest, rebellion and endings.** At the end of the step the module applies the tests of §1.2 to the day's stages: unrest, the start and end of a rebellion, and its ending. A *collective action* is under way on a day with a critical department on strike, a fight in any zone, or a held zone. On the day an ending is detected, everyone alive aged 14 or more and not dosed learns the history as lived (§6.15, What a rebellion teaches; Adam, 2026-11-03). A split is checked each day while a rebellion runs: each segment broken since its start and broken for `splitDays` days, from 1 to 143, and the first with `splitPeople` on each side ends it. The collapse check runs before these tests, so a collapse ends a rebellion first, including one detected that day. A rebellion's *starting zone* is the home zone of most of the adults active on its start day (on a tie, the first in the order Up Top, Mids, Down Deep).

**Collapse** (§1.2). Checked at the end of the step, and only with this module on, so a baseline run never stops early: air or water below 0.2 for 30 days in a row, or the population below P\_min (4,900, measured in T12.6; 0 switches the test off), its cause air or water if either is below the collapse level that day. The level and the days are §1.2 settings (`collapseLevel`, `collapseDays`). The Safeguard comes with §6.16. The first collapse is logged with its cause and ends a running rebellion with that ending; the run stops after that day, unless `output.runAfterCollapse` is true. Extinction always ends it (§1.2).

**Nobody says "I want to go out" yet.** No question needs it before Silo 1, where walking out is a contact attempt (§6.16), so this module reports no `askToGoOut`, and the `goingOut` rule has no effect for now.

**Step.** The rebellion step (§3 step 10.9) runs after the control step, in this order:

1. newcomers get their state, their h\_own drawn in the order of their IDs; the losses reported today (cleanings, denied wins) are recorded;
2. hardship and grievance;
3. stages, and the acts reported to control, which sees them at its next step;
4. fights, zone by zone;
5. held zones;
6. sabotage;
7. stair repairs;
8. collapse; unrest, rebellion and endings, and what an ending teaches;
9. the zone counts kept for tomorrow.

**Scripted events** (§5):

- `breakStairs` (segment): breaks the segment just before the rebellion step, as in S2's blast at levels 90–92.
- `cull` lives in §6.1, so it works without this module.

**Outputs** (only when the module is on):

- *daily.csv:* adults active, joined and violent; workers on strike; held zones; broken segments; pump down.
- *yearly.csv:* mean H and G of adults by home zone; rebels killed and arrested in fights, officers killed and wounded; sabotage by target; stair repairs; unrest spells ended and rebellions started.
- *events.jsonl:* `unrest` (start and end, logged at its end, once it is sure it never became a rebellion), `rebellionStart` (start) and `rebellionEnd` (start, end and ending), `zoneHeld` and `zoneFreed`, `sabotage` (person, target), `stairRepaired`, `collapse` (cause), `extinction`.
- *rebellions.jsonl:* one line per rebellion that started on a measured day, written at its end, or at the end of the run as unfinished (T12.7). A rebellion's record covers its days from its start day, the run-up before it was detected included, up to its end day, or the day before when it is suppressed, as that end day is the first quiet one; a suppressed rebellion's 29 further quiet days are not part of it. Fields: `start`; `end` (null while unfinished); `days`, end − start (the days so far while unfinished); `trigger`, the type of the latest scripted event on a day d with start − 365 < d ≤ start (the later in the file on a tie), or `none`, and `triggerDay`, its day; `zone`, the starting zone (above); `peakActive`, the highest daily share of adults active or beyond; collective actions: `struck`, the critical departments on strike on any day, `fightDays`, the days with a fight, `held`, the zones held on any day; deaths: `rebelsKilled` and `officersKilled` in fights; arrests: `fightArrests`, rebels arrested in fights, and `arrests`, every arrest Judicial made, for any offence, fight arrests included (§6.13; Adam, 2026-11-01); damage: `segmentsBroken` (by sabotage, not by scripted events), `generatorSabotage`, `pumpSabotage`; and `ending`: `suppressed`, `regimeChange`, `split`, `collapse` or `unfinished`.
- *run.json:* `outcomes`, with `firstRebellionDay` (the start day of the first rebellion that started on a measured day), `rebellions` (started on measured days), `unrestSpells` (ended on measured days), and `collapseDay` and `collapseCause` (a collapse counts on any day, warm-up included). A day or cause is null when there was none: censored at the end of the run.
- *frames.csv.gz:* per floor, the share of adults active and violent.

**Invariants** (§8): everyone not calm is a living eligible adult; violent people are joined and joined people active; everyone on strike is a joined worker or shadow; H, G and P lie in 0–1; in a held zone violent rebels outnumber the patrol; a stopped pump has a repair day.

**Off and null.** With the module off, nothing of this exists and nothing is checked for collapse. With it on and T = 1 nobody is ever active (G ≤ 1 and N ≥ 0), so nobody strikes, fights or sabotages and nothing breaks. The files then differ from a run without the module only by the new columns and outcome fields, unless the silo collapses, which `scenarios/baseline.yaml` never does.

**Parameters:** loss hardship ℓ and its half-life T\_loss, scapegoat hardship s\_DD, activation threshold T and arrest constant k (both Epstein 2002), violence threshold T\_v, officer and rebel effect e\_J and e\_R, killed share p\_kill, sabotage chance p\_sab, stair repair time D\_stair, strike share q\_strike (§1.2), and the point of no return P\_min (§1.2, measured in T12.6); also the definitions of unrest and rebellion (§1.2, decision D1), which are not model parameters but settings of what is counted. They move into §7 when M12 is done. In the scenario file they are the `rebellion:` section:

```yaml
rebellion:                        # ODD §6.14; used only when `rebellion` is in modules
  lossHardship: 0.5               # ℓ: hardship from a loss in the family, at the day of the loss; guess
  lossHalfLifeYears: 2            # T_loss: the loss's hardship halves in this time; guess
  scapegoatHardship: 0.2          # s_DD: hardship of living Down Deep; guess (lore: Mechanical is blamed, S2)
  activeThreshold: 0.1            # T: Epstein (2002)
  arrestConstant: 2.3             # k: P = 0.9 when the patrol equals the actives; Epstein (2002)
  violentThreshold: 0.3           # T_v: guess
  officerEffect: 0.05             # e_J: rebels put out a day per officer in a fight; guess
  rebelEffect: 0.01               # e_R: officers put out a day per violent rebel in a fight; guess
  killedShare: 0.3                # p_kill: share of those put out who die; guess
  sabotageChance: 0.001           # p_sab: per violent rebel and day; guess
  stairRepairDays: 60             # D_stair: mean days to repair a segment at full Mechanical coverage; guess
  strikeShare: 0.1                # q_strike: share of a department's workers on strike that makes it a strike (§1.2); guess
  unrestShare: 0.01               # §1.2, decision D1: unrest is at least this share of adults active ...
  unrestDays: 3                   # ... on this many days in a row
  rebellionShare: 0.05            # a rebellion: at least this share active ...
  rebellionDays: 7                # ... on this many days in a row, with a collective action on one of them
  quietDays: 30                   # both end after this many days in a row below unrestShare
  regimeDays: 30                  # regime change: rebels hold Up Top this many days in a row
  splitDays: 90                   # split: a segment broken in a rebellion still broken this many days later ...
  splitPeople: 1000               # ... with at least this many people living on each side; 0 = no people test
  collapseLevel: 0.2              # collapse: air or water below this level ...
  collapseDays: 30                # ... for this many days in a row
  pointOfNoReturn: 4900           # P_min (§1.2), measured in T12.6 for a silo of 10,000; 0 = off
```

`output.runAfterCollapse` (default false) sits with the other output settings. Checked at load: ℓ, s\_DD, T, T\_v, p\_kill, p\_sab, q\_strike and both shares of §1.2 are in [0, 1], and T\_v ≥ T; the collapse level is in [0, 1]; the six day counts are integers ≥ 1; `splitPeople` is an integer ≥ 0; T\_loss, k and D\_stair are strictly greater than 0; e\_J and e\_R are ≥ 0; P\_min is an integer ≥ 0.

### 6.15 Regime policy and response (draft)

*Answers Q3.* What the regime does during and after unrest and rebellion, and so which response keeps the silo alive. Following decision D7 (Adam, 2026-11-03), the regime does not learn. Each scenario gives it a fixed *playbook*: a list of policy changes, each started by a rebellion event and held for a set time. A response strategy (crackdown, reform, memory wipe, mixed, The Order) is a playbook written in a scenario file. The code holds no strategy logic and no numbers, so strategies are compared by running the same seeds with different files. The module needs the control and rebellion modules (§6.13, §6.14), and through them the beliefs, network and space modules.

**Who decides** (decision D13). The playbook is the Head of IT's (§6.13). While the post is vacant, no new entry starts, and entries already running run on. An entry missed this way is not started later. The `regimeChange` entries are the new regime's, so a vacancy doesn't stop them. The Mayor (§6.8) stays a figurehead. Silo 1's orders reach the silo through the Head in §6.16.

**State** (owned by this module; §2 is unchanged): the running entries, each with the day it started and the day it lapses, if any; whether the regime has changed (§1.2).

**The policy vector.** The levers are settings the model already has. Each one's value in the scenario is its *peacetime* value, and an entry overrides it while it runs:

| Lever | Setting it overrides | § |
| --- | --- | --- |
| surveillance | σ | 6.13 |
| arrestsPerOfficer | a | 6.13 |
| raidsPerYear, findChance | R\_raid, p\_find: relic confiscation. A relic taken leaves the silo for good, so this is also relic destruction | 6.13 |
| prisonPlaces, minesPlaces | detention places | 6.13 |
| taughtLegitimacy | L\_edu, the regime's propaganda in Education and on Freedom Day. It moves the base of people still being raised, so it acts over a generation, not at once (§6.12) | 6.12 |
| rationWeights | the zones' ration weights w\_z | 6.7 |
| scapegoatHardship | s\_DD: how hard the regime blames the Down Deep (The Order, below) | 6.14 |
| denial | covert lottery denial: `false` makes the lottery fair. People are still flagged, but a flagged couple wins and conceives like anyone. Its peacetime value is `regime.denial`, true: the lottery of §6.13 | 6.13 |
| doseConvicts | targeted Vitamin D+: every convict is dosed for this many days from their trial, whatever their penalty; 0 = none. Its peacetime value is `regime.doseConvicts`, 0 | 6.12, 6.13 |
| governance, mayorTerm | how the Mayor is chosen, and the term. A changed mode ends the sitting Mayor's term at the next governance step, so the new way of choosing is used at once | 6.8 |
| emergencyAir | the air line for emergency staffing (below); 0 = off | 6.15 |

A lever's value must pass the same load checks as the setting it overrides. The cleaning rate has no lever of its own: cleanings come only from sentences, so it changes through the Pact's cleaning penalties (a rewrite, below). Patrols follow the adults in every zone (§6.14). No lever moves them, since no strategy needs one.

**Entries.** A playbook entry, and the scripted event `policy` (below), have the same parts:

- `on`: the trigger that starts the entry, one of
  - `unrestStart`: the day the unrest test of §1.2 first holds (the 3rd day in a row at the unrest share), whether or not the spell becomes a rebellion;
  - `rebellionStart`: the day the rebellion test of §1.2 holds (its 7th day);
  - `suppressed`, `split`, `regimeChange`: the day that ending is detected (§1.2, §6.14).
- `hold`: how long it runs. Either a number of days (an integer ≥ 1), `forever`, or `quiet`: until the unrest spell or rebellion that started it ends. A spell that becomes a rebellion is its run-up (§1.2), so an entry started by unrest and held `quiet` runs until the rebellion ends. `quiet` is allowed only on `unrestStart` and `rebellionStart`.
- `set`: levers and their values while it runs.
- `pact`: Pact rows (§6.9) while it runs, each a full row. A row with an existing id replaces that row, and a new id adds a rule. This is a **Pact rewrite**. Convictions are kept by rule id, so a rewritten rule still counts earlier convictions as repeats.
- `lift`: added to the lift S (§6.13) once, on the day the entry starts, and decaying with S. It is the rewrite's legitimacy *benefit* (positive) or *cost* (negative), or a propaganda campaign. S may therefore be negative, and L stays within 0–1 by its own clamp (§6.12).
- `waterDosing`: if true, starts a water dosing of §6.12 from the next day, as the scripted event does (the Flamekeepers' warning included).

A lever, or a Pact row, takes the value of the most recently started running entry that sets it. With none, it takes its peacetime value. When an entry lapses, every lever and row it set falls back the same way. Entries started on the same day count in file order, the later one as more recent. Playbook entries come before the day's `policy` events. A `lift` and a dosing are done once they are applied, so a lapse doesn't undo them.

**Rebellion state as trigger.** At the regime step, the module reads that day's results of the rebellion step (§6.14) and starts the entries for each trigger that fired, in the order of the list above. On the first regime change, every running entry ends, and the old playbook is gone: from then on only its `regimeChange` entries apply, held for good whatever their `hold`. A later regime change starts nothing more. A scenario with none of these runs on at its peacetime settings and makes no response, which is what the new regime does by default.

**What a rebellion teaches** (Adam, 2026-11-03). On the day a rebellion's ending is detected, whatever the ending, everyone alive aged 14 or more and not dosed learns the history as lived, as in `revealHistory` (§6.12). They lived through it. This happens at the rebellion step (§6.14), before the regime step, so a dosing the playbook starts that day can't stop it: the wipe works on what people have already learned. Learning lowers L by δ for most of the silo, and the next rebellion can follow soon after. In the T12.8 smoke run, it started 30 days later. That *echo* is a result of the response, not a defect to remove: a regime that wipes, tells its story (a `lift`) or reforms answers it, and one that does nothing doesn't. Comparing these is Q3.

**Emergency staffing** (Adam, 2026-11-03). Health balances where air = 1 − r × medicalCoverage / d\_air = 0.8 (§6.2). Below that line, lower health lowers Mechanical's output (§6.5), and so power and air, and so health again. A rebellion that removes about a quarter of Mechanical's output can push the silo under the line, and without a response it slides to death (T12.7). Each day at the regime step, the regime declares an *emergency* if today's air is below `emergencyAir`, and lifts it once air is back at or above the line. The emergency acts the next day, through the modules that own the people it moves (T13.1):

1. at the control step, every Mechanical worker or shadow detained in prison or the mines has their sentence end the day before, so the day's releases free them and they walk home (§6.13). Mechanical workers and shadows in the queues for places are passed over while the emergency lasts;
2. at the life-stages step, General workers are retrained into Mechanical by the §6.5 rule, but every day and up to 100% of Mechanical's target, not 80%. General still keeps at least 80% of its own target. Strikers still count as workers (§6.14), so strikes are not replaced. The draws come from the work module's stream, as monthly retraining's do.

At the canon setting `emergencyAir` is the balance line 0.8 itself. The value is derived, not guessed. Retrained workers start at skill 0.2, so the released workers carry most of the effect. Whether it is enough is a result. The mass death on a single day that ends a slide comes from the baseline's health rule: every adult's health follows the same path, and health 0 is certain death (§6.1). The baseline is frozen, so this stays and is reported as part of the result.

**Elections** (Adam, 2026-11-04). With this module on, each Mayor chosen by election (§6.8) adds λ\_elect to the lift S, as a cleaning does (§6.13). Appointment adds nothing. Without this rule the Mayor's office changes nothing, and experiment E5 (appointment vs election) would have no mechanism to compare.

**The strategies.** Each one is a scenario file built on `scenarios/canon.yaml` (`scenarios/strategy-*.yaml`, T13.2), replacing The Order's playbook. Their values are guesses set by one principle, so the comparison is one of equal effort (Adam, 2026-11-04): every lever a strategy pulls moves by the same **factor of 2** from its canon value, sentences included, and every `lift` is one flag's worth, δ = 0.15 (§6.12). The sensitivity analysis (T15.3) varies the factor. Every strategy keeps emergency staffing at the canon line: it answers the life-support spiral, not rebellion, so Q3 compares only the rebellion responses (Adam, 2026-11-04).

| Strategy | Playbook |
| --- | --- |
| None | Empty: the control group |
| **The Order** (canon) | On `unrestStart`, held 365 days: s\_DD raised ("blame Mechanical", S2; lore reference), raids 2×. Held until quiet, the blame kept at least 1% of adults active, so the unrest that started it never ended: it ran from the warm-up's first spell for 200 years (T13.2, Adam, 2026-11-04). On `suppressed`, held 365 days: a `lift` of δ (the Freedom Day story), a water dosing, raids 2×: the wipe that followed the Rebellion (lore reference: lost records, memory drugs \[Likely\]). Its s\_DD is calibrated against C3 (below) |
| Crackdown | On `rebellionStart`, held `quiet`: σ, a, raids and both place counts 2×; the sentences for organising and violence 2× (prison 180 / mines 730; mines 1,460 / cleaning) |
| Reform | On `rebellionStart`, for good: s\_DD halved (0.1), elections, and a Pact rewrite halving the sentences for organising (prison 45 / mines 183) with a `lift` of δ. Rations stay equal: food hardship is only the shortfall, so more food doesn't ease a fed zone, and a 2× weight for the Down Deep left the rest of the silo with 82% of its need (Adam, 2026-11-04) |
| Memory wipe | On `rebellionStart`, held `quiet`: every convict dosed for a year (D\_water). On `suppressed` or `split`, held 365 days: a water dosing and raids 2× |
| Mixed | The crackdown while a rebellion runs; the reform, for good, once it ends (`suppressed` or `split`) |

The Order is the canon regime. It is also meant to carry pattern C3 (Adam, 2026-11-03): with own hardship the same in every zone, the Down Deep's grievance was only 1.2 times the Mids', and every rebellion started in the Mids (T12.8). Blaming Mechanical while unrest builds concentrates grievance where the show puts it. It is a lever, so reform can switch it off. Its value is calibrated against C3, a canon pattern: s\_DD under The Order at 0.4, 0.6 and 0.8 on the canon scenario, and the lowest that passes C3 is kept (`analysis/order-sweep.sh`; Adam, 2026-11-04). Until that sweep runs it is 0.6, provisional.

**Scripted events** (§5): `policy`, an entry run on its day, at the regime step, with a `hold` of days or `forever` (no `on`). It is how experiments impose a policy at a fixed day, such as a Pact rewrite like Season 3's under Camille (§6.9). It applies whether or not the Head of IT's post is filled, and after a regime change too, since the experimenter imposes it, not the regime.

**Step.** The regime step (§3 step 10.95) runs after the rebellion step, in this order:

1. entries whose hold ran out lapse;
2. the triggers that fired today start their entries, then the day's `policy` events; the scenario in force is rebuilt if anything started or lapsed;
3. the emergency is declared or lifted.

The levers set today are read by the steps that run after this one, so in practice from tomorrow. An entry held n days lapses at the regime step n days after its start, so every step reads it on exactly n days. The step draws no random numbers.

**Outputs** (only when the module is on):

- *events.jsonl:* `policy` when an entry starts and `policyEnd` when it lapses, each with its trigger (`scripted` for a policy event), the playbook entry's index (none for a policy event; its levers and rows are in the scenario, in `run.json`) and its start day; `emergency` when an emergency is declared and `emergencyEnd` when it is lifted.
- *yearly.csv:* `emergencyDays`, the days in the year with an emergency declared; `policies`, the entries running at the snapshot.
- Q3's standing-rebellion check, the **share of measured days in a rebellion**, is computed by `silosim report` from `rebellions.jsonl` (`days`), alongside C1's count (§1.2).

**Invariants** (§8): the scenario in force passes the load checks, so every lever is in its setting's range and the Pact in force passes the checks of §6.9; every running entry since a regime change is a `regimeChange` entry or a `policy` event.

**Off and null.** With the module off, nothing of this exists. With it on, an empty playbook, `emergencyAir` 0, no `policy` events and λ\_elect 0, it changes nothing and draws nothing: `daily.csv`, `events.jsonl` and `rebellions.jsonl` are byte-identical to the same scenario without it, and `yearly.csv` differs only by its two columns (T13.1).

**Parameters:** the emergency air line and the election lift λ\_elect, and two peacetime switches with no other home, `denial` and `doseConvicts`, which state the behaviour of §6.13 (denial on, nobody dosed) and are no new assumptions. In the scenario file they are the `regime:` section; the playbook's values are each strategy's own:

```yaml
regime:                           # ODD §6.15; used only when `regime` is in modules
  emergencyAir: 0                 # air line for emergency staffing; 0 = off. Canon: 0.8, the health balance line (§6.2, T12.7)
  electionLift: 0.05              # λ_elect: lift of L from each Mayor chosen by election; guess, the size of a public cleaning's lift (Adam, 2026-11-04)
  denial: true                    # peacetime covert lottery denial (§6.13); false = a fair lottery
  doseConvicts: 0                 # peacetime targeted Vitamin D+ for convicts, days from the trial; 0 = none
  playbook:                       # empty = no response; strategies fill it (T13.2)
    - on: unrestStart             # unrestStart | rebellionStart | suppressed | split | regimeChange
      hold: quiet                 # quiet | forever | days ≥ 1
      set: { scapegoatHardship: 0.5, raidsPerYear: 24 }
      pact:
        - { id: noOrganising, active: true, offences: [organise], detection: 0.6,
            first: { mines: 365 }, repeat: { cleaning: true }, enforcer: judicial, visibility: 0.7 }
      lift: 0.1                   # once, either sign
      waterDosing: true
```

The entry shown is an example of the shape, not a default: `baseline.yaml` has an empty playbook. Checked at load: `emergencyAir` and λ\_elect are in [0, 1]; `doseConvicts` is an integer ≥ 0; each `on` is one of the five triggers; `hold` is `quiet` (only on `unrestStart` and `rebellionStart`), `forever` or an integer ≥ 1; `lift` is in [−1, 1]; every lever passes its setting's checks; the scenario's Pact with any one entry's rows applied passes the checks of §6.9. A combination of running entries that breaks those checks stops the run with an error on the day the later entry would start. A `policy` event takes no `on` and no `quiet`.

## 7. Parameters

Every number the model uses, in one place. Each one becomes a field in the scenario file with this default; "Calibrate" means the value is a starting point to tune in the first calibration task, not something to defend.

The table below is the frozen baseline set. Part-2 parameters are listed in their draft sections (6.9 onward) and move into this table when their milestone is done.

**Ranges checked at load.** No parameter may be negative. These must lie in [0, 1]: every daily chance and percentage; α, ρ, f₀ and the air recovery share (so the updates in 6.5 and 6.6 cannot overshoot); every amount subtracted from, or compared with, a 0–1 variable (injury and illness cost, condition drop, wear, start skill, master skill, sick-leave health); hunger and air damage; recovery; the open-window weight; the shadow output factor; the retraining threshold. Also maternalDeathBase ≤ 0.5 and injuryChance × the largest injury factor ≤ 1, so every derived daily probability stays ≤ 1. These must be strictly greater than 0, because they are divisors or define quantities that must exist: Gompertz A and B (so the day-1 age table ends), floors per zone, the coverage need factor, every department share, adult kcal, the age-gap scale, power demand per person, the capacity margin, kcal per farm unit, target and starting population, window, pregnancy and minimum outage length, and the invariant and frame intervals. invariantEvery and frameEvery are integers ≥ 1. Ages are ordered (shadow < worker < retire, mother min < max, partnering min age ≤ day-1 pairing age), outage min ≤ max, farm and Mechanical floors each fewer than the floors per zone (every zone keeps a residential floor), the home capacity margin is at least 1, and zone splits and department shares each sum to 1. Part 2: both walking hours are strictly greater than 0; every department with office floors lists at least one floor inside the silo, and Farms, Mechanical, General services and Porters list none. That each office floor is a residential floor (not a farm or Mechanical floor) in the department's zone is checked when the space module starts, once the floors exist. The weak ties per kind k is an integer ≥ 0; p\_s and w\_weak are in [0, 1]. L\_edu, ε, δ, π, d and κ are in [0, 1]; both half-lives, T\_find and T\_keep are strictly greater than 0; D\_water is an integer ≥ 1; N\_relic and N\_F are integers ≥ 0. That N\_F is no more than the day-1 adults is checked when the beliefs module starts. σ, a, p\_find, p\_name, p\_fail, λ and φ are in [0, 1]; R\_raid is in [0, 365]; T\_S and T\_F are strictly greater than 0; N\_try is an integer ≥ 1 and N\_doc ≥ 0; the mines floor is in the silo, and the Pact passes the checks of §6.9. That the mines floor is Down Deep and that Judicial has office floors is checked when the control module starts.

| Parameter | Default | Basis | Used in |
| --- | --- | --- | --- |
| Target population | 10,000 | Show | 6.4 |
| Zone split | 32 / 46 / 22% | Follows from department zones | 5, 6.5 |
| Warm-up / measured run | 50 / 200 years | Method | 1, 5 |
| Gompertz A, B | 0.0002, 0.085 per year | Real-world shape; median death about 67 | 6.1 |
| Extra infant hazard | 0.02 per year | Guess | 6.1 |
| Health multiplier k\_h | 2 | Guess | 6.1 |
| Hunger damage d\_hunger | 0.02 per day at zero intake | Calibrate; default kept in T6.1 | 6.2 |
| Air damage d\_air | 0.05 per day at zero air | Calibrate; default kept in T6.1 | 6.2 |
| Injury chance | 0.0002 per day, × department factor | Guess | 6.2 |
| Illness chance | 0.0005 per day | Guess | 6.2 |
| Recovery r | 0.01 per day at full coverage | Calibrate; default kept in T6.1 | 6.2 |
| Coverage need | 0.84 × day-1 target headcount, every department | Calibrate (default kept in T6.1): full staff gives output / need ≈ 1.1 at the measured skill × health 0.92 | 6.2, 6.6 |
| Maternal death | 0.002 × (2 − coverage) | Guess | 6.2 |
| Partnering age | 18+ | Guess | 6.3 |
| Partner search rate | 0.0009 per day (about every 3 years) | Calibrate; default kept in T6.1 | 6.3 |
| Partner weights | Same zone ×4, same department ×2, same sex ×0.05, age scale 5 years | Guess; same sex: calibrate (default kept in T6.1) | 6.3 |
| Max permits per month | 40 | Guess | 6.4 |
| Conception window | 365 days | Show | 6.4 |
| Open windows in E | 0.8 × open windows | About the chance a window ends in pregnancy | 6.4 |
| Daily conception chance | 0.0044 | About 80% within a year, real-world | 6.4 |
| Pregnancy | 270 days | Real-world | 6.4 |
| Girls at birth | 50% | Matches the day-1 split | 6.4 |
| Eligibility | Woman 18–40, fewer than 3 children, youngest at least 2 | Guess | 6.4 |
| Shadow / worker / retire age | 14 / 18 / 65 | Guess from the show | 6.5 |
| Follow a parent's trade | 60% | Guess | 6.5 |
| Cross-zone placement | 5% | Guess | 6.5 |
| Max shadows per master | 2 | Guess | 6.5 |
| Master | Worker with skill above 0.5 | Guess | 6.5 |
| Shadow output | 0.3 × skill × health | Guess | 6.5 |
| Skill growth α | 0.00044 per day (0.8 after about 10 years) | Guess | 6.5 |
| Retraining | Below 80% of target; start skill 0.2 | Guess | 6.5 |
| Sick leave | After 90 days in a row below health 0.3; back to work at 0.3 | Guess | 6.5 |
| All-zone departments by zone | 36 / 40 / 24% of their workers Up Top / Mids / Down Deep | Follows from the zone split | 5, 6.5 |
| Mechanical couples stay Down Deep | While Mechanical is below target; otherwise 50/50 | Guess | 6.3 |
| Generator wear | 0.0004 per day | Calibrate; default kept in T6.1 | 6.6 |
| Repair rate ρ | 0.002 (condition settles at 0.8) | Calibrate; default kept in T6.1 | 6.6 |
| Failure f₀ | 0.002 (about 3 per century at c = 0.8) | Calibrate; default kept in T6.1 | 6.6 |
| Outage length | ⌈3 / (Mechanical skill × parts share)⌉ days, at most 10; condition −0.1 | Guess; the show: the generator can stop for a few days only | 6.6 |
| Backup power | 17.3% of capacity (≈ 26% of day-1 demand) | Guess | 6.6 |
| Power demand | 1 per person, 150 per farm floor (abstract units) | Design choice | 6.6 |
| Air recovery | A third of the gap to powerFactor per day (0.333) | Guess | 6.6 |
| Generator capacity | 1.5 × day-1 demand | Design choice: full power after a failure (1.5 × 0.7 ≥ 1) | 6.6 |
| Parts | Supply 1.2 a day at full coverage; repair 1 a day; failure 30 | Guess | 6.6 |
| Food per unit of farm output | 11,900 kcal | Calibrated to a 5% surplus (4.7% measured; calibration log, 2026-10-29) | 6.6 |
| Food need | 2,200 kcal adult; 60% under 14 | Real-world | 6.7 |
| Rationing | Below 30 days of stock, hand out 90% | Guess | 6.7 |
| Starting stocks | Food 90 days, parts 365 days, condition 0.8 | Guess | 5 |
| Day-1 layout | 48 floors per zone; 20 farm floors, 12 Mechanical; homes 1.15 × households | Show (144 floors), guess | 5 |
| Day-1 couples | 70% of adults aged 20+ | Guess | 5 |
| Mayor term | 5 years, or life (toggle) | Guess | 6.8 |
| Vote for own zone | 80% | Guess | 6.8 |
| Understaffed department (output) | Below 50% of target | Calibration target | 8 |
| *Part 2* | | | |
| Full descent H\_descent | 12 hours of walking, floor 1 → 144 | Guess. Not stated on screen; a reviewer's estimate is about a day top to bottom, fan discussion calls a full climb multi-day (lore reference). With 8 walking hours a day it takes 2 days, which fits both | 6.10 |
| Walking per day H\_walk | 8 hours | Guess | 6.10 |
| Office floors | Administration 1, Sheriff 1, Judicial 2–3, Medical 12, Education 20, IT 33–34, Supply 72 | Guess; Up Top for the first six, Mids for Supply (§6.5 zones). Sheriff and the Mayor's office near the cafeteria at the top \[Likely\] | 6.10 |
| Weak ties per kind k | 5 | Guess: a few close neighbours and workmates out of about 90 people on a floor | 6.11 |
| Strong tie daily chance p\_s | 0.5 | Guess: family and close work ties talk on about every other day; sets how fast news travels, so it is checked against C4 (§1.2) from M13 | 6.11 |
| Weak tie strength w\_weak | 0.3 | Guess: a weak tie is used about a third as often as a strong one (0.15 a day) | 6.11 |
| Taught legitimacy L\_edu | 0.8 | Guess: most people believe the Pact | 6.12 |
| Education weight ε | 0.5 | Guess: school and home count equally | 6.12 |
| Legitimacy cost per flag δ | 0.15 | Guess: all five flags take L from 0.8 to 0.05 | 6.12 |
| Pass chance π | 0.002 per flag and conversation, × (1 − L) of teller and listener | Guess; calibrated against C4 (§1.2) from M13, together with p\_s (§6.11) | 6.12 |
| Half-life of heard knowledge T\_heard | 5 years | Guess | 6.12 |
| Half-life of lived knowledge T\_lived | 20 years | Guess: most witnesses keep it for decades, few for life | 6.12 |
| Hiding while dosed d | 0.01 per flag and day | Guess: a year of water dosing hides 97% of a flag, past the 90% T10.6 asks for | 6.12 |
| Water dosing length D\_water | 365 days | Guess | 6.12 |
| Cue chance κ | 0.05 per cue | Guess: a held relic brings a memory back within two weeks half the time | 6.12 |
| Relics N\_relic | 100 | Guess: rare; banned "down to a Pez dispenser" (lore reference) | 6.12 |
| Relic hiding time T\_find | 50 years on average (1 / 18,250 a day) | Guess: rare finds, about 2 a year silo-wide at the start (Adam, 2026-10-31) | 6.12 |
| Relic keeping time T\_keep | 10 years on average (1 / 3,650 a day) | Guess | 6.12 |
| Flamekeeper seats N\_F | 15 | Guess: "a small secret group" (lore reference) | 6.12 |
| Surveillance σ | 0.05 | Guess. A plausibility check in T12.8: about 0–2 cleanings a year on `canon.yaml` (lore reference, "years can pass between cleanings") | 6.13 |
| Arrests per officer a | 0.05 a day | Guess: about 20 a day with about 400 officers, more than peacetime needs; it binds only in a rebellion | 6.13 |
| Raids R\_raid | 4 a year | Guess | 6.13 |
| Find chance p\_find | 0.2 per held relic and search | Guess: relics are hard to find (Adam, T10.4) | 6.13 |
| Naming p\_name | 0.3 per arrested member | Guess | 6.13 |
| Mines floor | 144 | Lore: the mines lie below Mechanical, at the bottom \[Likely\] | 6.13 |
| Prison places | 50 | Guess; lore: a few cells in the Sheriff's and Judicial offices. With mines places, 1.5% of a 10,000 silo, under the 5% food margin (T12.8) | 6.13 |
| Mines places | 100 | Guess; lore: forced labour in the mines \[Certain\], the crew's size unknown | 6.13 |
| Failed cleaning p\_fail | 0.05 | Guess; experiments script the C2 shock instead | 6.13 |
| Cleaning lift λ | 0.05 × visibility | Guess | 6.13 |
| Lift half-life T\_S | 1 year | Guess | 6.13 |
| Fear per trial φ | 0.01 × visibility | Guess: 100 trials a year at visibility 0.3 hold F near 0.43 (0.3 a year added, ln 2 / T\_F a year lost) | 6.13 |
| Fear half-life T\_F | 1 year | Guess | 6.13 |
| Implant doctors N\_doc | 2 | Show: one or two doctors do the implants | 6.13 |
| Tries before giving up N\_try | 3 denied wins | Guess: "after 3–4 tries they stop" (Adam, 2026-10-31) | 6.13 |

## 8. Outputs, tests and validation

A run is only trusted if its invariants hold and its outputs hit the calibration targets across many seeds. One good-looking run proves nothing.

### Output files per run

| File | Content |
| --- | --- |
| daily.csv | Day; population total and by zone; births, deaths, permits, open windows, winners waiting for Medical (queued), pregnancies; food stock in days; rationing flag; powerFactor; generator condition and outage flag; water; air; average health and hardship by zone; Medical and Mechanical coverage |
| yearly.csv | Median age; age pyramid in 5-year bands; headcount vs target per department; share of Mechanical workers living in Mechanical's zone; people who changed zone; current Mayor and their zone (the snapshot runs before governance, so a row shows the Mayor before that day's change) |
| events.jsonl | Generator failures and repairs, rationing on/off, Mayor changes, any department below 50% of target (and back above); part 2: scripted events (§5) |
| run.json | Scenario file, seed, code version, runtime; part 2: outcomes (§6.14, Outputs) |
| rebellions.jsonl | *Part 2, with the rebellion module on:* one line per rebellion (§6.14, Outputs) |
| frames.csv.gz | *Part 2, optional (`output.frames`, off by default):* one row per day, floor and metric (`day,floor,metric,value`) on measured days that are multiples of `output.frameEvery` (30). Baseline metrics: population, households, and average health per floor (0 where nobody lives); part-2 modules add their own. Values have 4 decimals at most, trailing zeros dropped. Written for replay viewers only; nothing in the model reads it |

Counts in a row (births, deaths, permits, people who changed zone) are what happened since the previous row of that file. Steps that run after daily logging (§3 steps 12–16) therefore appear in the next day's row: a lottery's permits on day 30 show on day 31. The pyramid is women and men in 5-year bands, the last one 100+. A department counts as below 50% when its workers fall under 0.5 × its current target (headcounts are recounted daily, targets monthly). Warm-up is not written unless the scenario asks for it.

### Invariant tests (must always pass)

- [x] Counts agree: people alive = sum over zones = sum over households
- [x] Every living person is in exactly one household; no floor exceeds its home capacity
- [x] Partnerships are symmetric and both partners are alive
- [x] No pregnancy without an open window; windows only for eligible couples
- [x] Food is conserved exactly: tomorrow's stock = today's stock + produced − handed out
- [x] Every 0–1 variable stays within 0–1
- [x] Nobody acts after the tick they died
- [x] Same seed → byte-identical output files; different seeds → different files

### Calibration targets

| Pattern | Metric | Target, in at least 90 of 100 seeds |
| --- | --- | --- |
| Stable population | Population over the 200 measured years | Always within 9,000–11,000 |
| Plausible ages | Median age | 30–40 |
| No demographic shocks | Largest change between consecutive 5-year totals of births | Under 30% after warm-up |
| Balanced resources | Days with rationing | Under 5% of days |
| Rare crises | Generator failures | 1–6 per century |
| Staffed departments | Department below 50% of target | Never for more than a year |
| Persistent zones | Share of Mechanical workers living Down Deep | Above 75% |

`silosim report` measures each target per seed over the measured years only. Population is checked on every day. Median age and the Mechanical share are checked at every yearly snapshot. Births are summed over consecutive 5-year blocks from the first measured day, and an unfinished last block is not compared. Rationing days are counted from the daily rationing flag. Failures per century are the run's failures divided by its length in centuries, an average over the run. A department's spell below 50% runs from its "understaffed" event to its "restaffed" event, or to the end of the run. A pattern holds when at least 90% of the seeds pass it.

Births are summed over 5-year blocks because single years are too noisy to judge: with about 150 births a year, randomness alone gives a largest year-to-year change of 31–46% over 200 years (10th–90th percentile of 1,000 simulated runs of pure Poisson noise), so a yearly target of 30% would fail on noise, not on booms. On 5-year totals the same pure noise gives a median of 12% and never more than 26% in 1,000 runs, so a change above 30% points to a real boom or collapse.

The persistent-zones target was 90% until calibration (2026-10-29). Partners are chosen freely (6.3), so about one Mechanical worker in five lives with a partner from another zone, and the measured share settles near 81%. Every rule that pushed it above 90% did so by growing the Down Deep and overfilling Mechanical at Farms' expense. 75% still means Mechanical clearly stays a Down Deep department, while leaving room for that free choice.

**How to calibrate:** sweep the parameters marked "Calibrate" in section 7, run 100 seeds per setting, keep the setting that hits all targets, and record the result in a calibration log. If no setting works, the rule is wrong, not the number; fix the rule in this spec first, then the code.

## 9. Sources

- Lore and canon decisions: Silo Mechanics Reference (Show Canon, through S3E9)
- Dissent: [Epstein (2002), Modeling civil violence: an agent-based computational approach, PNAS 99(suppl. 3) 7243–7250](https://www.pnas.org/doi/10.1073/pnas.092080199): grievance, risk aversion, arrest estimate and activation threshold (§6.14)
- Joining: Granovetter (1978), Threshold models of collective behavior, American Journal of Sociology 83(6) 1420–1443 (§6.14)
- Fights: Lanchester's square law (Lanchester 1916, Aircraft in Warfare) (§6.14)
- Structure of this spec: [Grimm et al. (2020), The ODD protocol for describing agent-based and other simulation models: a second update, JASSS 23(2) 7](https://www.jasss.org/23/2/7/7.pdf) ([record](https://pubs.usgs.gov/publication/70209554))
- Mortality shape: the Gompertz law, a standard model of adult mortality; the parameters here are tuned by hand, not taken from a life table.
