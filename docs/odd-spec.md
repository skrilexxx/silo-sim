# Silo Baseline Model — ODD Spec v0.1

Oct 28, 2026 · Adam Kopřiva · exported from the living doc

## 1. Purpose and patterns

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

- [ ] Patterns 1–6 hold in at least 90 of 100 runs with different seeds
- [ ] The same seed gives byte-identical output
- [ ] All invariant tests in section 8 pass
- [ ] One run of 200 years finishes in under a minute on a laptop

**Out of scope for the baseline** (later modules): beliefs and dissent, relics, surveillance, cleaning, covert lottery denial, rebellion, Silo 1, other silos, money.

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
| stage | enum | child, shadow, worker, retired |
| department | enum or none | Set when shadowing starts; kept as a record after retirement and death |
| masterID | int or none | The worker they shadow |
| skill | 0–1 | Grows with experience; drives output |
| health | 0–1 | 1 = healthy; drops with injury, illness, hunger, bad air |
| lowHealth | int | Days in a row a worker's health has been below 0.3 (sick retirement, 6.5) |
| implant | enum | in, removed (window open), pregnant |
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
| output | Effective work this tick: sum of worker skill × health |

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
- **Space:** 144 floors in three zones. Location is an attribute only: nobody walks the stairs in the baseline. Travel time arrives with the rebellion modules, where it starts to matter.
- **Population:** about 10,000 people alive at once, about 40,000–50,000 over a full run.

## 3. Process overview and scheduling

Each tick runs the same fixed sequence: silo-wide systems first, then people. The order is part of the model; changing it changes results, so it is written down here and nowhere else.

### Every day

1. **Generator** — wear, failure check, repair progress → power supply (6.6)
2. **Life support** — water and air from power and Mechanical output (6.6)
3. **Food production** — farm output, scaled by power, added to the stock (6.6)
4. **Rations** — food split by zone, intake per person, stock reduced (6.7)
5. **Work** — department output, skill growth for workers and shadows (6.5)
6. **Health** — hunger, air, work injuries, illness, recovery with Medical (6.2)
7. **Deaths** — mortality draws; widowed partners, orphans and job vacancies handled (6.1)
8. **Reproduction** — conceptions in open windows, births due today, expired windows closed (6.4)
9. **Life stages** — child → shadow at 14, shadow → worker at 18, worker → retired (6.5)
10. **Partnering** — unpartnered adults may pair up (6.3)
11. **Logging** — daily metrics written (8)

### Every 30 days

12. **Lottery** — permits issued and winners drawn (6.4)
13. **Housing** — new couples move to free homes; grown children leave home only by partnering (6.3)
14. **Staffing** — each department's target headcount set for the next shadow placements (6.5)

### Every 365 days

15. **Snapshot** — age pyramid and department table written (8)
16. **Governance** — if the Mayor died or the term ended, a new Mayor is chosen by appointment or election (6.8)

### Update rules

- People are processed one at a time in an order shuffled each tick by the seeded random generator, so no one always goes first.
- Changes take effect immediately (asynchronous updating). A person who dies in step 7 does not reproduce in step 8.
- One goroutine runs one simulation. Parallelism comes from running many seeds at once, never from splitting one tick across threads.

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
8. **Stocks:** food for 90 days of full rations, parts for 365 days of expected use, generator condition 0.8, water and air at 1.

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

## 6. Submodels

Each rule below is what Claude Code implements, one submodel per module. Every number is a starting value from section 7; calibration may change it, the rule shape stays.

### 6.1 Mortality

Each day each person dies with a probability from a Gompertz hazard (risk of death doubling roughly every 8 years of age), raised by poor health and by an extra risk in the first year of life.

```latex
h(a) = A e^{B a} + h_{infant}[a < 1], \quad p_{day} = 1 - \exp\left(-\frac{h(a)\,(1 + k_h (1 - health))}{365}\right)
```

With A = 0.0002 and B = 0.085, half of people reach about 67 before health effects. Deaths are drawn person by person in the tick's shuffled order and handled at once. On a death: the partner becomes unpartnered and may pair again; if an adult (18+) still lives in the household, the children under 18 stay with them (their other parent, or a step-parent); otherwise all the children in that household move together to a random household in the same zone with at least one living adult, and if there is none the run stops with an error. A household left with no members is dissolved, which frees its home. The job becomes a vacancy (6.5).

### 6.2 Health and Medical

Health runs from 0 to 1 and changes every day: damage from hunger, bad air, injuries and illness, and recovery that depends on how well Medical is staffed.

- **Hunger:** −d\_hunger × (1 − intake/need)
- **Bad air:** −d\_air × (1 − airQuality)
- **Work injury:** daily chance by department (Mechanical ×2, Farms ×1, others ×0.5); an injury costs 0.3 health
- **Illness:** small daily chance; costs 0.15 health
- **Recovery:** +r × medicalCoverage, capped at 1

```latex
medicalCoverage = \min\left(1, \frac{MedicalOutput}{0.6 \times MedicalTargetHeadcount}\right)
```

Medical also performs implant removals for lottery winners. If coverage is below 1, winners wait in a queue, which delays births when the silo is short of doctors. Mothers face a small death risk at birth, higher when coverage is low.

### 6.3 Partnering and housing

Partners are chosen freely. Each day, each unpartnered adult (18+), in the tick's shuffled order, starts looking with a small chance (on average about every 3 years) and at once pairs with someone picked from all other unpartnered adults in the silo, with weights:

- same zone ×4, same department ×2, same sex ×0.05
- an age-gap penalty: weight × e^(−|age gap| / 5)
- close relatives are never picked: parent, child, or sibling (sharing a mother or a father)

Any couple can form, so about 5% of new couples are same-sex; only a woman–man couple can enter the birth lottery. New couples join a housing queue. Each month the queue is handled oldest first. The first time a couple is handled it gets a target zone: the zone of a partner whose department has a zone (every department but General services and Porters); if both partners have one and they differ, one of the two at random; if neither, one of the two partners' current zones at random. The couple then moves into a random free home in that zone (a floor's homes minus the households living there), or keeps waiting, living apart, if none is free; a couple waiting for one zone never blocks couples waiting for another. A couple broken by a death leaves the queue. Moving starts a new household: from each household a partner leaves, children under 18 come along if they are a partner's own children, or if the partners were the only adults there, so no child is left without an adult. Homes left empty are free at once for the rest of the queue. Young adults stay in their parents' household until they partner. Partnering and job placement are the only ways a person changes zone, which makes zone mobility measurable.

### 6.4 Lottery and reproduction

Every 30 days the silo issues permits to replace recent deaths and close the gap to the target population, spread over a year so the population does not swing.

```latex
N = \mathrm{clamp}\left(\mathrm{round}\left(D_{30} + \frac{T - P - E}{12}\right),\ 0,\ N_{max}\right), \quad E = pregnant + 0.8 \times openWindows
```

D₃₀ = deaths in the last 30 days, T = target (10,000), P = people alive.

D₃₀ counts deaths since the last lottery; round is half away from zero. Unused permits are not carried over.

- **Eligible couples:** woman and man, woman aged 18–40 in whole years (up to her 41st birthday), fewer than 3 children born (alive or not), her last birth at least 2 years ago, no open window or pregnancy. Couples still waiting for a home are eligible.
- **Draw:** winners are picked uniformly at random, without replacement; if fewer couples are eligible than there are permits, all of them win. (The covert-denial rule from the show arrives with the dissent modules; here the draw is fair.)
- **Window:** Medical removes the implant (6.2); the couple has the 365 days after the draw. If her partner dies while the window is open, it closes and the implant goes back in.
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
- **Worker at 18** in the same department; the master link ends. **Retired** at 65 or after 90 days in a row with health below 0.3; the department stays on record.
- **Skill:** grows each day by α × (1 − skill) for workers and shadows, so reaching 0.8 takes about 10 years.
- **Trade inheritance (outcome, not a rule):** about 80% of workers end up in a parent's department, more than the 60% follow chance, because the other placements often land in a parent's department too.
- **Output:** department output = sum of skill × health over its workers, plus 0.3 × that for each shadow.
- **Retraining:** each month, a department below 80% of target may take workers from General services in its zone; they start at skill 0.2. This keeps critical departments staffed after deaths.

### 6.6 Resources

The generator is the root of the loop: it powers the farms, the water pumps and the air, so a weak Mechanical department eventually shows up as hunger and illness everywhere.

- **Generator condition c:** loses 0.0004 per day (from new to broken in about 7 years without care). Repair adds ρ × mechanicalCoverage × (1 − c) and uses 1 part a day; with no parts there is no repair. Because repair slows as c nears 1, condition settles where wear and repair balance: c\* = 1 − wear / (ρ × coverage), which is 0.8 at full coverage with ρ = 0.002.
- **Failure:** daily chance f₀ × (1 − c)², with f₀ = 0.002, which gives about 3 failures per century at c = 0.8. A failure takes 3–14 days to fix, consumes extra parts and drops c by 0.1. While it is down, backup power gives 20% of normal output.
- **Power:** supply = capacity × c × mechanicalCoverage (or backup while down); demand is fixed per person and per farm floor. Capacity is set on day 1 to 1.3 × day-1 demand, so at condition 0.8 and full coverage the silo has just enough power. powerFactor = min(1, supply / demand).
- **Water:** powerFactor × mechanicalCoverage.
- **Air:** moves toward powerFactor by a third of the gap each day, so short outages are survivable and long ones are not.
- **Food:** production = farm output × kcal per unit of farm output × powerFactor, added to the stock.
- **Parts:** measured in days of normal use. Supply makes 1.2 units a day × supplyCoverage (scavenging and recycling); each repair day uses 1 unit; each failure uses 30 more.

mechanicalCoverage = min(1, Mechanical output / Mechanical need), the same shape as medicalCoverage. Every coverage uses the same need: 0.6 × the department's target headcount, so a fully staffed department at average skill and health gives coverage of about 1.1, and shortages start to bite at about 90% staffing.

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

Each day the silo hands out food from the stock. If the stock falls below 30 days of need, it hands out only 90%, which is rationing. Food is split between zones by policy weights, then shared within each zone in proportion to need (children under 14 need 60% of an adult's 2,200 kcal).

```latex
share_z = \frac{w_z N_z}{\sum_{z'} w_{z'} N_{z'}}, \qquad \frac{intake_i}{need_i} = \frac{share_z \times distributed}{need_z}
```

In the baseline every weight is 1.0, so everyone gets the same share of their need. The regime can change the weights later, for example to favor Up Top. Each day the model logs hardship per zone = 1 − average intake/need; the dissent modules will use it as a source of grievance.

### 6.8 Governance

How a Mayor is chosen and how long they serve are two separate settings. In the baseline the office changes no rules; it is logged so the dissent modules have it ready.

- **Appointment (default):** the heads of IT, Judicial and Administration (the most skilled worker in each) choose the most skilled senior worker from those three departments.
- **Election (scenario toggle):** each zone puts up its most skilled senior worker; every adult votes for their own zone's candidate with an 80% chance, otherwise at random. Without beliefs, this simply measures how zone size decides an election.
- **Term (scenario toggle):** a 5-year term (default), or for life, so a Mayor serves until death. A life term works with either way of choosing; it removes any regular, peaceful change of leader, which matters once dissent is modeled.

## 7. Parameters

Every number the model uses, in one place. Each one becomes a field in the scenario file with this default; "Calibrate" means the value is a starting point to tune in the first calibration task, not something to defend.

**Ranges checked at load.** No parameter may be negative. These must lie in [0, 1]: every daily chance and percentage; α, ρ, f₀ and the air recovery share (so the updates in 6.5 and 6.6 cannot overshoot); every amount subtracted from, or compared with, a 0–1 variable (injury and illness cost, condition drop, wear, start skill, master skill, sick-retire health); hunger and air damage; recovery; the open-window weight; the shadow output factor; the retraining threshold. Also maternalDeathBase ≤ 0.5 and injuryChance × the largest injury factor ≤ 1, so every derived daily probability stays ≤ 1. These must be strictly greater than 0, because they are divisors or define quantities that must exist: Gompertz A and B (so the day-1 age table ends), floors per zone, the coverage need factor, every department share, adult kcal, the age-gap scale, power demand per person, the capacity margin, kcal per farm unit, target and starting population, window, pregnancy and minimum outage length, and the invariant interval. invariantEvery is an integer ≥ 1. Ages are ordered (shadow < worker < retire, mother min < max, partnering min age ≤ day-1 pairing age), outage min ≤ max, farm and Mechanical floors each fewer than the floors per zone (every zone keeps a residential floor), the home capacity margin is at least 1, and zone splits and department shares each sum to 1.

| Parameter | Default | Basis | Used in |
| --- | --- | --- | --- |
| Target population | 10,000 | Show | 6.4 |
| Zone split | 32 / 46 / 22% | Follows from department zones | 5, 6.5 |
| Warm-up / measured run | 50 / 200 years | Method | 1, 5 |
| Gompertz A, B | 0.0002, 0.085 per year | Real-world shape; median death about 67 | 6.1 |
| Extra infant hazard | 0.02 per year | Guess | 6.1 |
| Health multiplier k\_h | 2 | Guess | 6.1 |
| Hunger damage d\_hunger | 0.02 per day at zero intake | Calibrate | 6.2 |
| Air damage d\_air | 0.05 per day at zero air | Calibrate | 6.2 |
| Injury chance | 0.0002 per day, × department factor | Guess | 6.2 |
| Illness chance | 0.0005 per day | Guess | 6.2 |
| Recovery r | 0.01 per day at full coverage | Calibrate | 6.2 |
| Coverage need | 0.6 × target headcount, every department | Full staff gives coverage ≈ 1.1 | 6.2 |
| Maternal death | 0.002 × (2 − coverage) | Guess | 6.2 |
| Partner search rate | 0.0009 per day (about every 3 years) | Calibrate | 6.3 |
| Partner weights | Same zone ×4, same department ×2, same sex ×0.05, age scale 5 years | Guess; same sex: calibrate | 6.3 |
| Max permits per month | 40 | Guess | 6.4 |
| Conception window | 365 days | Show | 6.4 |
| Daily conception chance | 0.0044 | About 80% within a year, real-world | 6.4 |
| Pregnancy | 270 days | Real-world | 6.4 |
| Girls at birth | 50% | Matches the day-1 split | 6.4 |
| Eligibility | Woman 18–40, fewer than 3 children, youngest at least 2 | Guess | 6.4 |
| Shadow / worker / retire age | 14 / 18 / 65 | Guess from the show | 6.5 |
| Follow a parent's trade | 60% | Guess | 6.5 |
| Cross-zone placement | 5% | Guess | 6.5 |
| Max shadows per master | 2 | Guess | 6.5 |
| Skill growth α | 0.00044 per day (0.8 after about 10 years) | Guess | 6.5 |
| Retraining | Below 80% of target; start skill 0.2 | Guess | 6.5 |
| Generator wear | 0.0004 per day | Calibrate | 6.6 |
| Repair rate ρ | 0.002 (condition settles at 0.8) | Calibrate | 6.6 |
| Failure f₀ | 0.002 (about 3 per century at c = 0.8) | Calibrate | 6.6 |
| Outage length | 3–14 days; condition −0.1 | Guess | 6.6 |
| Backup power | 20% | Guess | 6.6 |
| Generator capacity | 1.3 × day-1 demand | Design choice | 6.6 |
| Parts | Supply 1.2 a day at full coverage; repair 1 a day; failure 30 | Guess | 6.6 |
| Food per unit of farm output | About 16,000 kcal | Calibrate to a 5% surplus | 6.6 |
| Food need | 2,200 kcal adult; 60% under 14 | Real-world | 6.7 |
| Rationing | Below 30 days of stock, hand out 90% | Guess | 6.7 |
| Starting stocks | Food 90 days, parts 365 days, condition 0.8 | Guess | 5 |
| Day-1 layout | 48 floors per zone; 20 farm floors, 12 Mechanical; homes 1.15 × households | Show (144 floors), guess | 5 |
| Day-1 couples | 70% of adults aged 20+ | Guess | 5 |
| Mayor term | 5 years, or life (toggle) | Guess | 6.8 |
| Vote for own zone | 80% | Guess | 6.8 |

## 8. Outputs, tests and validation

A run is only trusted if its invariants hold and its outputs hit the calibration targets across many seeds. One good-looking run proves nothing.

### Output files per run

| File | Content |
| --- | --- |
| daily.csv | Day; population total and by zone; births, deaths, permits, open windows, pregnancies; food stock in days; powerFactor; generator condition and outage flag; water; air; average health and hardship by zone; Medical and Mechanical coverage |
| yearly.csv | Age pyramid in 5-year bands; headcount vs target per department; people who changed zone; current Mayor and their zone |
| events.jsonl | Generator failures and repairs, rationing on/off, Mayor changes, any department below 50% of target |
| run.json | Scenario file, seed, code version, runtime |

### Invariant tests (must always pass)

- [ ] Counts agree: people alive = sum over zones = sum over households
- [ ] Every living person is in exactly one household; no floor exceeds its home capacity
- [ ] Partnerships are symmetric and both partners are alive
- [ ] No pregnancy without an open window; windows only for eligible couples
- [ ] Food is conserved exactly: tomorrow's stock = today's stock + produced − handed out
- [ ] Every 0–1 variable stays within 0–1
- [ ] Nobody acts after the tick they died
- [ ] Same seed → byte-identical output files; different seeds → different files

### Calibration targets

| Pattern | Metric | Target, in at least 90 of 100 seeds |
| --- | --- | --- |
| Stable population | Population over the 200 measured years | Always within 9,000–11,000 |
| Plausible ages | Median age | 30–40 |
| No demographic shocks | Largest change between consecutive 5-year totals of births | Under 30% after warm-up |
| Balanced resources | Days with rationing | Under 5% of days |
| Rare crises | Generator failures | 1–6 per century |
| Staffed departments | Department below 50% of target | Never for more than a year |
| Persistent zones | Share of Mechanical workers living Down Deep | Above 90% |

Births are summed over 5-year blocks because single years are too noisy to judge: with about 150 births a year, randomness alone gives a largest year-to-year change of 31–46% over 200 years (10th–90th percentile of 1,000 simulated runs of pure Poisson noise), so a yearly target of 30% would fail on noise, not on booms. On 5-year totals the same pure noise gives a median of 12% and never more than 26% in 1,000 runs, so a change above 30% points to a real boom or collapse.

**How to calibrate:** sweep the parameters marked "Calibrate" in section 7, run 100 seeds per setting, keep the setting that hits all targets, and record the result in a calibration log. If no setting works, the rule is wrong, not the number; fix the rule in this spec first, then the code.

## 9. Sources

- Lore and canon decisions: Silo Mechanics Reference (Show Canon, through S3E9)
- Structure of this spec: [Grimm et al. (2020), The ODD protocol for describing agent-based and other simulation models: a second update, JASSS 23(2) 7](https://www.jasss.org/23/2/7/7.pdf) ([record](https://pubs.usgs.gov/publication/70209554))
- Mortality shape: the Gompertz law, a standard model of adult mortality; the parameters here are tuned by hand, not taken from a life table.
