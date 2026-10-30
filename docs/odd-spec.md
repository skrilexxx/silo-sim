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
- **Unrest:** at least 1% of adults active on 3 days in a row, without meeting the rebellion test. Logged; not a rebellion.
- **Rebellion:** starts when, on 7 days in a row, at least 5% of adults are active **and** at least one collective action (§6.14) is under way on one of those days: a strike in a critical department (Mechanical, Farms, Supply, Medical), a clash with Judicial, or rebels holding a floor. Its start day is the first of the 7 days. It ends when fewer than 1% of adults are active for 30 days in a row, or at collapse. Only one rebellion runs at a time.
- **Trigger:** the scripted event (`events:` in the scenario, T7.4) that happened within the 365 days before the start, or "none". With no scripted events, every rebellion is "none", and Q1's trigger question is answered by experiments that script shocks.
- **Endings** (one per rebellion; collapse takes precedence):
  - *Suppressed:* it ends with the regime still in place.
  - *Regime change:* rebels hold the IT floors and the Mayor's office for 30 days in a row. The regime is replaced and the scenario's post-change policy applies (§6.15). The run continues.
  - *Split:* a stair segment broken during the rebellion is still broken 90 days later, and at least 1,000 people live on each side of it.
  - *Collapse:* see below.
- **Collapse** (the silo is dead, and the run stops), on the first day any of these holds; the cause is recorded:
  1. air or water below 0.2 for 30 days in a row;
  2. a Safeguard goes off (§6.16);
  3. population below the **point of no return** P\_min.

  P\_min is measured, not guessed (T12.6): runs start from damaged states (population cut to 9,000, 8,000, 7,000 … and Mechanical cut by 10%, 20% …), and P\_min is the highest population at which fewer than 5% of seeds recover to above 9,000 within 50 years. Until it is measured, test 3 is off. The baseline already shows why the last person is the wrong line: in the T3.2 collapse, health was 0 and the silo still held 643 people in year 100. Canon agrees: Silo 17, with fewer than a hundred survivors and later one, is a dead silo.
- **Extinction:** the last person dies. Logged only, in runs that set `output.runAfterCollapse: true` to see how a dead silo empties.

**Canon patterns** (calibration and validation targets for part 2). They are checked on `scenarios/canon.yaml`, the regime at the show's settings, 300 measured years, 100 seeds. These are the only targets part-2 parameters are tuned to.

| # | Pattern | Source | Check |
| --- | --- | --- | --- |
| C1 | **Long peace:** rebellions are rare, but they do happen | The Rebellion ~140 years before S1; the next in S2 | Mean rate across seeds between 0.3 and 2 rebellions per century |
| C2 | **Visible failures trigger revolts:** a cleaner who walks away matters more than slow hardship | Juliette (S1E10); Silo 17's cleaner | A rebellion within 5 years after a scripted cleaning failure is at least 3× as likely as in a 5-year window without one |
| C3 | **Revolts start Down Deep and use the generator** | The S2 uprising | ≥ 60% of rebellions start with most active rebels living Down Deep; ≥ 50% of those include a Mechanical strike |
| C4 | **Crushed rebellions are forgotten** | Freedom Day, lost records, Vitamin D+ | 40 years after a suppressed rebellion, < 10% of adults hold the true-history flag (§6.12) |
| C5 | **A rebellion can end a silo** | Silo 17 | With Silo 1 on `full`, 5–50% of rebellions end in collapse |

C1 bounds the rebellion rate, which is also what Q1 measures. That is deliberate but limited: C1 is a wide plausibility check (no constant revolts, no eternal peace), never a target to tune toward a point. Q1's answer is what *changes* the rate, read from comparisons between scenarios.

**Outcome metrics**

- *Per run:* time to first rebellion (censored at the end of the run if none); number of rebellions and unrest spells; for each rebellion, its start, trigger, starting zone, peak share of active adults, duration, collective actions, deaths, arrests, damage, and ending; time and cause of collapse (censored if none).
- *Across seeds:* survival curves (Kaplan–Meier) for time to first rebellion and time to collapse; rebellions per century and the share of each ending, with 95% confidence intervals.

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
10.5. **Network** *(part 2, draft)* — weak ties ended and drawn again; the day's conversations (6.11)
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
| Learning | Skill grows with work experience; nothing else is learned |
| Prediction | None |
| Sensing | People know their partner, household, master and department; nothing more |
| Interaction | Partnering, shadowing, shared rations within a zone, competition for lottery permits and homes |
| Stochasticity | Deaths, injuries, illness, conception, partner choice, generator failures, lottery draws; all from one seed, with one random stream per module so adding a module does not shift the others' draws |
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

With A = 0.0002 and B = 0.085, half of people reach about 67 before health effects. A person whose health reaches 0 dies that day: the health step (§3 step 6) comes just before deaths, so the death chance is 1 then. Deaths are drawn person by person in the tick's shuffled order and handled at once. On a death: the partner becomes unpartnered and may pair again; if an adult (18+) still lives in the household, the children under 18 stay with them (their other parent, or a step-parent); otherwise all the children in that household move together to a random household in the same zone with at least one living adult. If their zone has none (only after a mass death), they go to such a household in any zone. Only when no household in the silo has an adult does the run stop with an error. A household left with no members is dissolved, which frees its home. The job becomes a vacancy (6.5).

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
- **Draw:** winners are picked uniformly at random, without replacement; if fewer couples are eligible than there are permits, all of them win. (The covert-denial rule from the show arrives with the dissent modules; here the draw is fair.)
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
- **Water:** powerFactor × mechanicalCoverage (today's powerFactor, the previous day's coverage).
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

**Penalties:** `warning`, `confiscate` (relics), `prison(days)` and `mines(days)` (no work, no contacts outside the place of detention), `cleaning` (death, public), `flagFamily` (covert lottery denial for the household, §6.13). A `contactAttempt` can also bring Silo 1's response (§6.16), which the table doesn't control.

**The canon Pact** (default for `scenarios/canon.yaml`):

| id | Forbids | Offences | Detection | Penalty: first / repeat | Enforcer | Visibility | Source |
| --- | --- | --- | --- | --- | --- | --- | --- |
| noRelics | Owning or trading objects from before the silo | holdRelic, tradeRelic | 1.0 | confiscate + prison(90) / prison(365) | Judicial | 0.5 | Show: relics banned down to a Pez dispenser (S1) |
| noForbiddenTalk | Speaking about the outside, the time before, or other silos | speakForbidden | 0.5 | warning / prison(180) | Judicial | 0.3 | Show: questioning the rules can lead to cleaning (S1) |
| goingOut | Saying "I want to go out" | askToGoOut | 1.0 | cleaning / — | Sheriff | 1.0 | Show: the request is binding (S1E1) |
| lottery | Conceiving without a permit or removing one's own implant | unpermittedConception | 0.8 | prison(180) + flagFamily / cleaning | Judicial | 0.6 | Show: implants and lottery (S1E1) |
| noContact | Radio contact, digging toward other silos, opening the doors below | contactAttempt | 0.7 | cleaning / — | Judicial | 0.2 | Show: Safeguard and the Directive (S2–S3) |
| noViolence | Attacking officers or damaging the silo's systems | violence, sabotage | 1.0 | mines(730) / cleaning | Judicial | 0.8 | Show: S2 uprising |
| noOrganising | Unauthorised gatherings and strikes | organise | 0.6 | prison(90) / mines(365) | Judicial | 0.7 | Show: Mechanical treated as rebels (S2) [Likely] |

Detection weights, penalties and visibilities are guesses, to be checked by the sensitivity analysis (T15.3). Calibration (C1–C5) may tune them as a group, never row by row toward a wanted outcome.

**Rewriting the Pact** is a change to rows: a rule switched on or off, a penalty raised or lowered, or a new rule added. It happens through a scripted event (`events:` in the scenario, T7.4) or a regime response (§6.15). Season 3's rewrite under Camille is the canon example. Rules the regime may add, off in the canon Pact:

| id | Forbids | Effect | Penalty |
| --- | --- | --- | --- |
| travelPermits | Crossing zones without a permit | Fewer cross-zone contacts (§6.11), slower rebel gathering | warning / prison(30) |
| curfew | Being away from one's home floor at night | Fewer contacts outside the household | warning / prison(30) |

Format in the scenario file (illustrative; the exact keys are fixed when M11 implements it):

```yaml
pact:
  rules:
    - id: noRelics
      active: true
      offences: [holdRelic, tradeRelic]
      detection: 1.0
      penalty: { first: [confiscate, prison90], repeat: [prison365] }
      enforcer: judicial
      visibility: 0.5
```

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
- After arriving, the person is *away*: their day location is the floor they reached, and they don't work, until a module sends them on another trip. A trip back to the home floor ends being away.
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
- a coworker tie, when the two no longer share a department and work floor: one retired, changed department or got another work floor (moving home, or a change in the stairs, can do that). Sick leave and being away or on the stairs (§6.10) keep the work floor, so they don't end it; a stranded worker has no work floor, so being stranded does;
- a porter tie, when neither of the two is a porter any more (the porter retired or left Porters). Moving zone doesn't end it.

Each rule looks only at the two people's state today, not at what changed.

The lost ties are drawn again at the next network step.

**Conversations.** After the ties are updated, every living person, in the tick's shuffled order, starts c conversations. Each picks one of their ties with weight equal to its strength, among the ties they can reach today: neither is on the stairs, and their day locations are connected (§6.10). Each tie is picked at most once per person per day; with fewer reachable ties than c, they start fewer. A conversation is between two people, and either may pass something to the other (§6.12), so on average a person takes part in about 2c a day. Today's conversations, with the kind of tie each used, are kept until the next network step for the steps that use them later in the day.

There is no distance weight. Distance acts through who a person's ties are (neighbours share a floor, coworkers a work floor) and through one hard limit, the stairs: when they break, kin and porter ties across the break go silent until they are repaired.

**Outputs** (only when the module is on, as extra columns of yearly.csv): the mean number of ties per person and its 10th, 50th and 90th percentiles at the snapshot, and the number of conversations since the previous row by tie kind (partner, household, kin, master–shadow, coworker, neighbour, porter).

**Off and null.** With the module off, nothing of this exists. With it on, the network changes no baseline state and draws only from its own stream, so `daily.csv` and `events.jsonl` are byte-identical to a run without it; `yearly.csv` keeps its baseline columns and gains the network's.

**Parameters** (move to §7 when M9 is done):

| Parameter | Default | Basis |
| --- | --- | --- |
| Weak ties per kind k | 5 | Guess: a few close neighbours and workmates out of about 90 people on a floor |
| Conversations started per day c | 3 | Guess; sets how fast news travels, so it is checked against C4 (§1.2) in T12.8 |
| Weak tie strength w\_weak | 0.3 | Guess: a weak tie is picked about a third as often as a strong one |

Ranges checked at load: k and c are integers ≥ 0; w\_weak is in [0, 1].

## 7. Parameters

Every number the model uses, in one place. Each one becomes a field in the scenario file with this default; "Calibrate" means the value is a starting point to tune in the first calibration task, not something to defend.

The table below is the frozen baseline set. Part-2 parameters are listed in their draft sections (6.9 onward) and move into this table when their milestone is done.

**Ranges checked at load.** No parameter may be negative. These must lie in [0, 1]: every daily chance and percentage; α, ρ, f₀ and the air recovery share (so the updates in 6.5 and 6.6 cannot overshoot); every amount subtracted from, or compared with, a 0–1 variable (injury and illness cost, condition drop, wear, start skill, master skill, sick-leave health); hunger and air damage; recovery; the open-window weight; the shadow output factor; the retraining threshold. Also maternalDeathBase ≤ 0.5 and injuryChance × the largest injury factor ≤ 1, so every derived daily probability stays ≤ 1. These must be strictly greater than 0, because they are divisors or define quantities that must exist: Gompertz A and B (so the day-1 age table ends), floors per zone, the coverage need factor, every department share, adult kcal, the age-gap scale, power demand per person, the capacity margin, kcal per farm unit, target and starting population, window, pregnancy and minimum outage length, and the invariant and frame intervals. invariantEvery and frameEvery are integers ≥ 1. Ages are ordered (shadow < worker < retire, mother min < max, partnering min age ≤ day-1 pairing age), outage min ≤ max, farm and Mechanical floors each fewer than the floors per zone (every zone keeps a residential floor), the home capacity margin is at least 1, and zone splits and department shares each sum to 1. Part 2: both walking hours are strictly greater than 0; every department with office floors lists at least one floor inside the silo, and Farms, Mechanical, General services and Porters list none. That each office floor is a residential floor (not a farm or Mechanical floor) in the department's zone is checked when the space module starts, once the floors exist.

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

## 8. Outputs, tests and validation

A run is only trusted if its invariants hold and its outputs hit the calibration targets across many seeds. One good-looking run proves nothing.

### Output files per run

| File | Content |
| --- | --- |
| daily.csv | Day; population total and by zone; births, deaths, permits, open windows, winners waiting for Medical (queued), pregnancies; food stock in days; rationing flag; powerFactor; generator condition and outage flag; water; air; average health and hardship by zone; Medical and Mechanical coverage |
| yearly.csv | Median age; age pyramid in 5-year bands; headcount vs target per department; share of Mechanical workers living in Mechanical's zone; people who changed zone; current Mayor and their zone (the snapshot runs before governance, so a row shows the Mayor before that day's change) |
| events.jsonl | Generator failures and repairs, rationing on/off, Mayor changes, any department below 50% of target (and back above); part 2: scripted events (§5) |
| run.json | Scenario file, seed, code version, runtime |
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
- Structure of this spec: [Grimm et al. (2020), The ODD protocol for describing agent-based and other simulation models: a second update, JASSS 23(2) 7](https://www.jasss.org/23/2/7/7.pdf) ([record](https://pubs.usgs.gov/publication/70209554))
- Mortality shape: the Gompertz law, a standard model of adult mortality; the parameters here are tuned by hand, not taken from a life table.
