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
| D4 | Beliefs: one number or several | T10.0 | **Decided 2026-10-31:** legitimacy L (0–1) = an upbringing base minus a cost per known flag; five knowledge flags; no free-text beliefs; hardship enters grievance, not L (ODD §6.12) |
| D5 | Vitamin D+ in the water: can memories come back? | T10.0 | **Decided 2026-10-31:** dosing hides flags; after it stops, a hidden flag comes back with a chance per cue (a held relic, or a conversation with someone who knows it, the latter × the person's doubt 1 − L since T13.2); hidden flags never fade on their own (ODD §6.12) |
| D6 | How rebels and Judicial fight | T12.0 | **Decided 2026-10-31:** per zone, no marching: violent rebels fight the zone's abstract patrol by Lanchester's square law; those put out are killed or arrested; rebels hold a zone they outnumber its patrol in (ODD §6.14) |
| D7 | Regime behaviour | T13.0 | **Decided 2026-11-03:** a fixed playbook per scenario (an experimental factor), not a learning regime: policy changes on rebellion triggers, held for a set time, written in YAML; strategies are scenario files (ODD §6.15) |
| D8 | Silo 1 as agents or as a rule set | T14.0 | **Scope decided 2026-11-04:** a rule set with two settings, `off` and `full`; no second population, no radio, digging, capping, drones or deal (the lean plan below) |
| D9 | Several silos at once | — | [no; only as optional work after M15] |
| D10 | The Pact: prose or rules | T7.1 | **Decided 2026-10-30:** a rule table in the scenario file, no prose (ODD §6.9) |
| D11 | Run length and seeds for part 2 | T7.1 | **Decided 2026-10-30:** 50 + 300 years; 100 seeds for calibration, 1,000 for the final experiments. **Revised 2026-11-04 (lean plan):** experiments 50 + 150 years, 30 seeds per scenario with common random numbers (ODD §1.2) |
| D12 | Viewer technology | T16.0 | [`silosim serve`: Go's standard HTTP server with the page embedded in the binary, plus one vendored JavaScript chart library; read-only] |
| D13 | The Head of IT as a role | T11.0 (role, succession), T13.0 (decisions), T14.0 (Silo 1's link) | **Role and succession decided 2026-10-31:** one IT worker, for life (only death ends it); knows all five flags, never dosed, never tells, can't be arrested; their shadow succeeds them, else the most skilled IT worker (ODD §6.13). **Decisions settled 2026-11-03:** the playbook is the Head's, and a vacant post starts no response (ODD §6.15). Silo 1's link (T14.0) still open |

**D13, the Head of IT (raised by Adam, 2026-10-31).** The Head of IT is the most powerful person in the silo, not the Mayor: they know almost everything about it, and Silo 1 speaks to them, and only them, through the Algorithm in the vault on the IT floors, which nobody else can enter (lore reference, Season 3). The model has no such role yet: the baseline has a Mayor (§6.8) and an IT department, nothing more. Before it is built, the specs must settle:

- **Who holds it, and who follows.** One IT worker, with a successor (in the show, a shadow the Head trains). How they are chosen and replaced (death, removal, regime change).
- **What they know.** All five flags, and Vitamin D+ (§6.12); whether they keep it, or can be dosed.
- **What they decide.** Surveillance, Pact rewrites, cleanings and dosing are the Head of IT's levers (M11, M13), with the Mayor as a figurehead.
- **Silo 1's link.** In `watch` and `full`, Silo 1's orders reach the silo only through them, and they comply with some probability (T14.0, T14.1); the vault is where that happens, and only they can use it.
- **The vault and IT as targets.** Rebels who take the IT floors already end the regime (§1.2, regime change); whether they also gain the vault, and with it the link to Silo 1.

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

- [x] **T9.0 Spec: social network (ODD §6.11).** (2026-10-30, approved) Ties: partner, household, close kin, master–shadow, coworkers (same department and work floor, up to k), neighbours (same home floor, up to k), and weak cross-zone ties for Porters. Each day a person talks with a few contacts, picked from their ties by tie strength and floor distance.
- [x] **T9.1 Build and maintain the network.** (2026-10-31) Built on day 1, updated on birth, death, partnering, moving and job change.
  - *Done when:* invariants: ties are symmetric and only between living people; degree distribution logged yearly.
- [x] **T9.2 Daily contacts.** (2026-10-31; lazy per-tie conversations, ODD §6.11) k contacts per person per day, sampled by tie weight and distance.
  - *Done when:* the full run stays inside the T7.2 performance budget; contact counts by tie type logged.

## Milestone 10: Beliefs, knowledge and memory

*Answers Q1 and Q3.* Legitimacy is the core of the rebellion model. What people know lowers it; what they forget lets it recover. This is where relics, the Flamekeepers and **Vitamin D+** live.

> **About vitamin D.** In the show, "Vitamin D+" isn't nutrition. It's Silo 1's memory suppressant, disguised as vitamins (S3). Modelling vitamin D as a vitamin would add a health parameter that changes none of the four answers. Modelling it as the memory drug gives the regime its strongest lever. This plan does the second.

- [x] **T10.0 Spec: beliefs and memory (ODD §6.12).** (2026-10-31, approved) Decisions D4–D5. Per person:
  - **legitimacy L** (0–1): how much they believe the Pact and the leaders are right
  - **knowledge flags:** the outside may be survivable; the cleaning view is a lie; the true history; other silos exist; the lottery is rigged
  - **risk aversion R** (a fixed trait)

  Rules: L moves down with knowledge and hardship and up with propaganda (Freedom Day, Education); children inherit L partly from their parents and partly from Education; knowledge spreads in contacts, faster between kin; knowledge fades, and dies with the people who held it.
- [x] **T10.1 Belief state and legitimacy.** (2026-10-31) Day-1 beliefs, the legitimacy rule, socialisation of children and shadows.
  - *Done when:* with nothing to learn, L stays steady in every zone for 200 years (no drift from the rule itself).
- [x] **T10.2 Information spread.** (2026-10-31) Telling and believing in daily contacts; the chance to believe falls with the listener's L.
  - *Done when:* one seeded "knower" gives an S-shaped spread curve; spread through Porters crosses zones faster than without them (test).
- [x] **T10.3 Memory and forgetting.** (2026-10-31; C4 fails without a regime: 80% still know after 40 years, calibration log) A forgetting rate per flag (lived experience fades slower than hearsay); generational turnover.
  - *Done when:* knowledge half-life matches the parameter (statistical test); pattern C4 measured on a scripted crushed rebellion.
- [x] **T10.4 Relics.** (2026-10-31; also handed over in conversations and kept hidden at home, Adam's choice) Relics hidden in floors, found by digging and clearing flats, held and shared, confiscated later by Judicial (M11). Holding one raises the chance of learning the true history. A relic also **cues memory** after Vitamin D+, as the Pez dispenser and the watch did for Juliette.
- [x] **T10.5 Flamekeepers.** (2026-10-31; dying out tested by removing members, as arrests will) A small secret group that keeps the true history alive. It recruits trusted, low-L contacts and loses members to arrests and deaths.
  - *Done when:* without any regime, the group survives generations; with arrests on (M11), it can die out.
- [x] **T10.6 Vitamin D+.** (2026-10-31; M10 done, parameters in ODD §7; Flamekeepers warned by a doubting insider in IT or Medical shield their households, Adam's choice) Two forms: a **targeted dose** for named people, and **water dosing** for everyone, for a set period. While dosed, people's knowledge flags are hidden at a daily rate, so their L climbs back to their upbringing base. After dosing stops, flags can return when cued by a relic or a contact who still knows. Triggered only by scripted events for now; the regime uses it in M13 and Silo 1 orders it in M14.
  - *Done when:* a scripted year of water dosing removes ≥ 90% of a flag; memory comes back after dosing only with cues (test with and without relics).

## Milestone 11: Regime control

*Answers Q1 and Q3.* The deterrence half of the rebellion model: how likely being caught is, and what follows.

- [x] **T11.0 Spec: surveillance, Judicial, punishment, cleaning (ODD §6.13).** (2026-10-31; Pact YAML keys fixed; D13 role and succession decided; denied couples blame infertility, only the implant doctors know) Offences and penalties come from the Pact table (ODD §6.9); this task fixes its exact YAML keys. Decision D13: the Head of IT as a role (who holds it, succession, what they know).
  - **Offences:** holding a relic, spreading forbidden knowledge, organising, violence, contact attempts.
  - **Detection:** a chance per offence that rises with IT coverage and the surveillance level (a policy lever).
  - **Judicial and Sheriff:** capacity from coverage; arrests are trips (M8), so a Down Deep arrest takes days.
  - **Sentences:** warning, prison, the mines (forced labour), cleaning.
  - **Covert lottery denial:** flagged people "win" the lottery but their implant stays in.
- [x] **T11.1 Pact table, Head of IT, surveillance and detection.** (2026-10-31; the Head taking a shadow shifts the work module's draws, so the null setting is not byte-identical, ODD §6.13) The `pact:` and `control:` sections; the Head of IT and successor; acts reported by beliefs (tellings, handovers) and seen at σ × c\_IT × w.
- [x] **T11.2 Arrests, trials and sentences.** (2026-10-31; a cleaning sentence stops the run until T11.3) Prison and the mines take people out of work and out of contact. Arrest capacity, trips to the court, searches and raids, confiscation, Flamekeepers naming each other.
  - *Done when:* with arrests on, the Flamekeepers can die out (T10.5); with σ = 0 and no raids nobody is arrested.
- [x] **T11.3 Cleaning and fear.** (2026-10-31; fear F from trials silences talk over weak ties, Adam's choice; cleaners die at the next deaths step) Sentenced cleanings ("I want to go out" is said under M12's rules). A cleaning slightly raises everyone's L for a while (the lift S); the fear it causes is read by activation in M12. A cleaner who refuses and walks out of sight is the **cleaning-failure shock**: everyone watching learns that the outside may be survivable and the screen lies, and L drops as they tell others. It is also a scripted event type for experiments (pattern C2).
  - *Done when:* the shock's effect on L and knowledge matches the spec; a normal cleaning raises L, and so lowers dissent, for a while (test).
- [x] **T11.4 Covert lottery denial.** (2026-10-31; flags the couple only, a denied win uses up no permit, Adam's choice; calibration log, M11) Flagged couples never conceive; their family lines shrink.
  - *Done when:* the share of births to flagged families falls as the spec predicts; logged as an outcome, not tuned.

## Milestone 12: Grievance and rebellion

*Answers Q1 and Q2. This is the core of the thesis.*

- [x] **T12.0 Spec: grievance, activation and collective action (ODD §6.14).** (2026-10-31, approved; D6 decided: zone fights by Lanchester's square law, no marching; fear in the net risk; `askToGoOut` deferred to §6.16) Decision D6. Based on Epstein's civil violence model (2002), extended with networks and thresholds:
  - **hardship H** from zone hardship (rations), health, losses in the family (denied lottery, cleaned relatives) and being in the Down Deep (scapegoating, S2)
  - **grievance G** = H × (1 − L)
  - **active** when G − R × P(arrest) > threshold, where P(arrest) is estimated from officers and active rebels among the person's contacts and floor
  - **joining** (Granovetter): an active person joins a collective action once enough of their ties have joined
  - **escalation:** unrest → protest or strike → uprising
  - **actions:** strikes in a department (a Mechanical strike cuts power), occupying floors, barricading or breaking stair segments, sabotaging the generator or water pump, clashes with Judicial (force ratio decides casualties)
  - **endings:** suppressed, regime change, split silo, silo death (D2)
- [x] **T12.1 Module, hardship and grievance.** (2026-10-31; H measured near 0 Up Top, 0.2 Down Deep; a cleaning failure more than doubles mean G) The `rebellion:` section; H from food, health, losses and the Down Deep; G = H × (1 − L); losses reported by control (cleanings, denied wins). Mean H and G by zone in `yearly.csv`.
- [x] **T12.2 Arrest estimate and activation.** (2026-10-31; a steep tip, not a jump: a quarter of the rise within 10 of 200 days as officers fall; daily `active` column) Patrols per zone, P, net risk with fear, the active stage and its `organise` acts.
  - *Done when:* reproduces the qualitative result of Epstein's model in a test world: with low legitimacy, outbursts come in bursts when the officer ratio drops.
- [x] **T12.3 Joining and escalation.** (2026-10-31; unrest ends like a rebellion and a spell that becomes one is its run-up, Adam's choice; §1.2's shares and day counts in `rebellion:`) The joined stage (share R of ties), the violent stage (T\_v) and its `violence` acts; unrest and the rebellion test of §1.2.
- [x] **T12.4 Strikes and leverage.** (2026-10-31; with health in H a strike spiralled, bad air → lower health → more hardship → more strikers, and killed a canon silo; health dropped from H, Adam's choice) Joined workers and shadows stop working (still counted for targets and retraining); a Mechanical strike lowers power through the existing loop (§6.6).
- [x] **T12.5 Fights and held zones.** (2026-10-31; fight arrests go to control's next step; no regime change seen, as rebels don't march) Lanchester's square law per zone; deaths, wounds and arrests; held zones block arrests; regime change.
- [x] **T12.6 Sabotage and collapse.** (2026-11-01; P\_min = 4,900, recovery test kept, Adam; `cull` in §6.1; extinction always ends a run) Stair segments, generator, water pump; stair repairs; the `breakStairs` and `cull` events; the collapse checks of ODD §1.2; the run records the cause and stops (or keeps going to extinction with `output.runAfterCollapse`). The deaths step must not stop the run with an error when no adult is left: T12.4's strike spiral (since broken) killed every adult before air stayed below 0.2 for 30 days. **Measure P_min:** start runs from damaged states (population cut to 9,000, 8,000, 7,000 …; Mechanical cut by 10%, 20% …), 100 seeds each; P_min is the highest population at which fewer than 5% of seeds recover above 9,000 within 50 years. Write it into ODD §1.2 and the calibration log.
- [x] **T12.7 Rebellion log and outcomes.** (2026-11-01; arrests both in fights and by Judicial overall, Adam's choice; split ending added: a segment broken 90 days while the rebellion still runs, 1,000 people on each side, Adam) `rebellions.jsonl`: start, trigger, zone, peak size, actions, casualties, damage, ending.
- [x] **T12.8 First-results checkpoint.** (2026-11-03; two batches and four rule changes: own hardship, doubting tellers, C4 from M13, detention places; C1 0.92 per century and C2 182× pass, C3 fails (all start in the Mids), and with places rebellions stop ending; Adam carried C3 and the standing rebellion into M13 without a third batch; calibration log) `scenarios/canon.yaml` with modules 8–12 on, 100 seeds. Check patterns C1–C4. **Stop here if they fail.** Fix rules in the spec, not numbers, then continue. This is the first moment the thesis has real results; write them into the calibration log.

## Milestone 13: Regime response and adaptation

*Answers Q3.* What the regime does during and after a rebellion, and which choice keeps the silo alive.

- [x] **T13.0 Spec: policy and response (ODD §6.15).** (2026-11-03; D7 decided: YAML playbooks; everyone learns the history at a rebellion's end; The Order raises s\_DD for C3; emergency staffing for the spiral; travelPermits and curfew left out; λ\_elect proposed) Decision D7.
  - **Policy vector:** surveillance, punishment severity, cleaning rate, ration weights (§6.7), relic confiscation, Vitamin D+ (targeted or water), governance mode and term (§6.8), lottery fairness.
  - **The Order** (S2) as the default playbook: blame Mechanical, raids, propaganda.
  - **Response strategies:** crackdown, reform (raise L, lower hardship, elections), memory wipe (water dosing plus relic destruction), mixed.
  - **Pact rewrite** = a policy change with a legitimacy cost or benefit.
  - **What a rebellion teaches (moved here from T12.8, Adam, 2026-11-02).** Approved in T12.8 as: when a rebellion ends, everyone aged 14+ and not dosed learns the history as lived (like `revealHistory`), so C4 tests the forgetting of a rebellion everyone saw. In the smoke run it caused an echo: L fell 0.15 for everyone at the end, and a second rebellion started 30 days later. Settle it here, with the wipe that competes with it (who learns: everyone or participants; when: start or end; does the regime's story raise L).
  - **The standing rebellion (found in T12.8, Adam, 2026-11-03).** With detention places (§6.13), Judicial can no longer take the chronic few percent of actives out, so rebellions last decades (47–76% of measured time in seeds 1 and 53) and never fall below the 1% an ending needs. The response strategies must be what ends them; check the share of time in a rebellion as well as C1's count.
  - **C3 still fails (T12.8).** All 223 rebellions of the second batch started in the Mids: own hardship is the same in every zone, so the Down Deep's grievance is only 1.2× the Mids', against twice the adults. The Order's "blame Mechanical" (S2) is the regime's lever here; decide whether it, s\_DD, or the starting-zone rule (heads vs share of a zone's adults) carries C3.
  - **The life-support spiral (found in T12.7, left for M13 by Adam, 2026-11-01).** Health balances at air = 1 − recoveryRate × medicalCoverage / airDamage = 0.8; below it, lower health lowers Mechanical's output (skill × health), so power and air, so health again. A rebellion that takes away about a quarter of Mechanical's output (arrests, deaths) plus one sabotaged generator puts the silo under the line, and nothing reacts: in a harsh test (s\_DD 0.9, T\_v 0.15) the silo slid for 100 days after the rebellion was suppressed, then about 9,000 people died on one day, as everyone's health is the same and health 0 is certain death. Decide whether the regime responds to it (moving workers to Mechanical, rations), and whether the one-day cliff is acceptable.
- [x] **T13.1 Policy vector and scheduled policy changes.** (2026-11-04; levers reach modules through `World.Policy`, the scenario in force; the emergency acts the next day through control and work; `denial` and `doseConvicts` given peacetime keys in `regime:`) The `regime` module and its RNG stream, the levers, entries and their holding, the `policy` event, emergency staffing, the election lift, and the teaching rule in the rebellion step.
  - *Done when:* empty playbook = byte-identical to the same scenario without the module; a `policy` event sets and reverts a lever on the right days; emergency staffing releases detained Mechanical workers when air is below the line (test).
- [x] **T13.2 Response strategies,** (2026-11-04: The Order in canon, five strategy files at a factor of 2, time in a rebellion in the report, C4 judged; after smoke runs, Adam: The Order's blame held a year, recall × (1 − L), a rebellion starts where the share of a zone's adults active is highest; C4 left until the standing rebellion is understood, as recurrence drives it; the canon batch moved to T15.3 with the lean plan; calibration log) triggered by rebellion state. Playbook triggers; the strategy scenario files (none, The Order, crackdown, reform, memory wipe, mixed) with their values; The Order in `scenarios/canon.yaml`; the share of time in a rebellion in `silosim report`; C3 and C4 judged on the canon batch.
- [x] **T13.3 Memory-wipe strategy,** using T10.4 and T10.6. (2026-11-04; `scenarios/strategy-wipe.yaml` from T13.2: convicts dosed during, water dosing and raids 2× after; a 1,000-person run hides a flag from 99% of adults, `TestMemoryWipe`)
- [x] **T13.4 Pact rewrites** (ODD §6.9, §6.15; 2026-11-04, built in T13.1; a rule switched off stops its trials and a lift of −0.1 lowers S, `TestPactRewrite`): rows switched, penalties changed or added through `pact` in an entry, and their legitimacy cost or benefit through `lift` (S may go negative). `travelPermits` and `curfew` are left out (T13.0).
- ~~**T13.5 Performance**~~ dropped with the lean plan (Adam, 2026-11-04): 30 seeds × 150 years fit in minutes per scenario as is. Kept for the record: A canon run takes about 470 s against the 4-minute budget of `docs/architecture.md`, and the T13.2 profile shows no single hot spot: the rebellion step's several daily passes over everyone (hardship, stages, counts, active share, strikes; 26%), the network (18%), health, life stages, beliefs and location (about 8% each). Merge passes and cache per-person lookups.
  - *Done when:* a canon run fits the budget; golden hash unchanged and a canon seed's files byte-identical before and after.
  - *Milestone done when:* moved to experiment E3 (T15.5), with the lean plan's 30 seeds.

## Milestone 14: Silo 1 (lean)

*Answers Q2 and Q4.* Silo 1 is an outside controller. Its trigger is **escape**, not unrest as such (S3). The lean plan (Adam, 2026-11-04) keeps only what Q4 and C5 need: settings `off` and `full`, escapes as the only contact, Silo 1's orders through the Head of IT, and the Safeguard. Radio, digging, capping, drones and the deal are left out.

- [ ] **T14.0 Spec: Silo 1 (ODD §6.16).** Decision D8: what counts as an escape, Silo 1's threat and its orders, the Safeguard's trigger and delay.
- [ ] **T14.1 Silo 1 module.** Threat from escapes, orders through the Head of IT, the Safeguard as a collapse with cause `safeguard` (D2).
  - *Milestone done when:* `off` vs `full` runs in E4 (T15.5); C5 measured there.

## Milestone 15: Experiments and thesis (lean)

All experiments: 50 + 150 years, 30 seeds per scenario, the same seeds in every scenario (common random numbers), so differences come from the factor, not noise (D11, revised 2026-11-04).

- [ ] **T15.1 Outcome report.** `silosim report --outcomes`: per seed and across seeds, with confidence intervals: time to first rebellion, rebellions per century, endings, causes of death, survival curves (Kaplan–Meier), share of time in a rebellion.
- [ ] **T15.2 Sensitivity.** One at a time, about 6 key part-2 parameters at ±50% (e.g. L\_edu, δ, π, T, s\_DD, p\_fail) on the canon scenario, 30 seeds each. Which move the outcomes? Replaces Morris screening and `silosim sweep`.
- [ ] **T15.3 Canon validation.** C1–C5 on the canon scenario's 30 seeds, reported pass or fail; C3's strike half and C4 reported as they are.
- [ ] **T15.4 Experiments.** Write each one down before running it: question, scenarios, seeds, the metric that answers it.
  - **E1:** canon without shocks. How long does peace last? (Q1)
  - **E2:** cleaning-failure shock at a fixed day. Trigger effect (Q1, C2)
  - **E3:** none vs The Order vs crackdown vs reform vs memory wipe vs mixed (Q3; M13's milestone check)
  - **E4:** Silo 1 `off` vs `full` (Q2, Q4, C5)
  - **E5:** appointment vs election, 5-year vs life term (Q3)
- [ ] **T15.5 Freeze model v2.** ODD v2 complete, `scenarios/canon.yaml` frozen, golden hashes regenerated, tag `model-v2`. Thesis figures generated from `runs/` by scripts in `analysis/`.

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
