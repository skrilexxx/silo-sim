# Silo Mechanics Reference (Show Canon, through S3E10)

Oct 30, 2026 · Adam Kopřiva · exported from the living doc on claude.ai

## Purpose and spoiler policy

This is the lore input for the baseline silo model: the mechanics the show establishes, and what each means for the simulation. It is not a plot summary.

- **Canon:** the Apple TV+ series only. When the show is silent, the gap is marked as a parameter to invent, not filled from the books.
- **Spoiler line:** All three seasons, through the Season 3 finale (S3E10). Nothing from the books.
- **Scope:** Silo 18 as a self-contained society. Silo 1 and the outside world stay out of the baseline; part 2 adds Silo 1 as an outside controller.
- **Confidence tags:** \[Certain\] = shown or stated on screen and confirmed by a source; \[Likely\] = strongly implied or sourced from the fan wiki without an episode reference; \[Guessing\] = my inference, to be checked.

## Physical structure

Silo 18 is a 144-level vertical tube joined by one spiral stairwell, split into three zones whose height mirrors social status. For the model, that makes it a line graph of 144 nodes, where distance means time and time means isolation.

| Mechanic | What the show establishes | Confidence |
| --- | --- | --- |
| Levels | 144 levels, connected only by the central spiral stairwell | \[Certain\] |
| Zones | Up Top, the Mids, the Down Deep | \[Certain\] |
| Zone boundaries | Roughly 1–48 Up Top, 49–96 Mids, 97–144 Down Deep (fan explainer, partly book-derived) | \[Likely\] |
| Transport | No elevators or mechanized transport; the design limits movement and contact between floors | \[Likely\] |
| Up Top | Cafeteria with the screen showing the outside, offices, Judicial, Mayor, Sheriff, IT | \[Certain\] for the cafeteria and screen; \[Likely\] for the rest |
| Mids | Most food production (farms, some livestock), markets, apartments | \[Likely\] |
| Down Deep | Mechanical and the generator, which powers the whole silo; at least one water plant | \[Certain\] for the generator; \[Likely\] for water |
| Class gradient | Down Deepers are looked down on; Up Top holds power and privilege | \[Certain\] |
| Travel time | Not stated on screen. A full climb is a multi-day effort in fan discussion | \[Guessing\] → parameter |
| Porters | Goods move by hand on the stairs | \[Guessing\] → check against S1 |

```text
Up Top    (~1-48)   power and information: cafeteria + screen, Mayor, Judicial, Sheriff, IT, Medical
Mids      (~49-96)  food and most homes: farms, markets, Supply, apartments
Down Deep (~97-144) keeps everyone alive: Mechanical, generator (power + air), water
One spiral stairwell links all 144 levels; no elevators.
```

Zone boundaries are approximate. The Down Deep is highlighted because every other zone depends on its systems, which is why Mechanical can use them as leverage.

Sources: [Fiction Horizon on levels](https://fictionhorizon.com/how-many-levels-does-silo-have-explained/), [Fiction Horizon on population](https://fictionhorizon.com/how-many-people-live-in-the-silo-18-numbers-revealed/), [Flicks on the vertical class divide](https://www.flicks.us/features/inside-silos-central-setting-another-great-vertical-dystopia/), [Sportskeeda on why there are no elevators](https://sportskeeda.com/us/shows/why-elevators-silo-explained).

## Population and family

About 10,000 people live in the silo, and births are gated by a lottery plus mandatory contraception. The lottery is also a control tool: people who ask questions can be quietly denied children.

| Mechanic | What the show establishes | Confidence |
| --- | --- | --- |
| Population | About 10,000, held roughly constant | \[Certain\] for the size; \[Likely\] for "constant" |
| Contraception | Every woman carries a birth control implant by default | \[Certain\] |
| Permission to conceive | A couple wins the lottery; the doctor removes the implant for a 365-day window | \[Certain\] |
| Covert denial | Doctors are told to fake the removal for people seen as troublemakers (Allison, Phoebe) | \[Certain\] |
| Fertility counseling | A fertility counselor (Gloria) supports couples through the process | \[Certain\] |
| What triggers a lottery | Not explained on screen; likely tied to deaths or a population cap | \[Guessing\] → parameter |
| Partnering | Relationships need official sanction | \[Likely\] |
| Children and schooling | Shown in passing; no numbers on ages, school or nurseries | \[Guessing\] → parameter |

**Model note:** the covert-denial rule is a direct link between dissent and demography. A family marked as disloyal stops reproducing, which shrinks dissenting bloodlines over generations. That is an emergent effect worth measuring.

Sources: [Wikipedia: Silo (TV series)](<https://Www.wikipedia.org/wiki/Silo_(TV_series)>), [TV Tropes: Silo](https://tvtropes.org/pmwiki/pmwiki.php/Series/Silo), [Looper on forced birth control](https://www.looper.com/1309452/silo-forced-birth-control-prevent-rebellions), [AnthEa on the fertility counselor role](https://anthearepresents.com/news/sophie-thompson-plays-a-mysterious-and-enigmatic-fertility-specialist-in-apple-tv-s-new-dystopian-drama-series-silo).

## Work and departments

Everyone belongs to a department, and trades pass down through shadowing: a young person apprentices to a master. Departments double as social classes, because each one lives in its own zone.

| Department | Role in the silo | Zone | Confidence |
| --- | --- | --- | --- |
| Mayor's office | Top of the civil hierarchy; all departments answer to the Mayor | Up Top | \[Certain\] |
| IT | Runs the computers, controls information and hidden knowledge; its head holds real power | Up Top | \[Certain\] |
| Judicial | Enforces the Pact through a strict legal system; keeps records of relics | Up Top | \[Certain\] |
| Sheriff's Department | Everyday law enforcement, with deputies in different zones | Up Top, deputies spread out | \[Certain\] |
| Medical | Doctors; also carries out implant removal for the lottery | Up Top | \[Certain\] |
| Education, Maintenance | Named as departments; little shown | Unclear | \[Likely\] |
| Supply | Stores and hands out materials and parts | Mids | \[Certain\] for existence; \[Likely\] for zone |
| Farms | Grow food and raise livestock | Mostly Mids | \[Likely\] |
| Mechanical | Keeps the generator and machines running; the poorest group | Down Deep | \[Certain\] |

**Shadowing:** a teenager becomes a master's shadow to learn the trade (Juliette shadowed Martha Walker; Cooper shadowed Juliette). Juliette forged a note to get in, which suggests a parent or guardian has to approve the placement. \[Certain\] for shadowing; \[Likely\] for the approval rule.

**Other labor:** scavenging and recycling scrap is treated as work for offenders, and forced work in the mines appears as a punishment. \[Likely\]

**Model note:** jobs are mostly inherited by zone and by shadowing, so social mobility is low by design. That is one of the main sources of grievance to measure later.

Sources: [Otakukart Season 1 explainer](https://otakukart.com/silo-season-1-ending-explained-the-mystery-conspiracy-and-shocking-revelations-that-set-the-stage-for-an-epic-season-2/), [Wikipedia cast list](<https://wiki.riteme.site/wiki/Silo_(TV_series)>), [Cult of Mac S1 recap](https://www.cultofmac.com/reviews/silo-apple-tv-recap-truth), [The Review Geek S2E7](https://www.thereviewgeek.com/silo-s2e7review/).

## Governance and law

Power sits Up Top in three hands: the Mayor's office, Judicial with its raiders, and IT, which controls information. The Pact is the law, and the show makes clear it can be rewritten by those in power.

| Mechanic | What the show establishes | Confidence |
| --- | --- | --- |
| The Pact | The silo's constitution and rulebook; presented as "the only history we have" | \[Certain\] |
| What the Pact contains | Ordinary protections (against illegal search and seizure) plus rules on how the silo is divided; its subtext favors Judicial | \[Certain\] |
| The Pact can change | In Season 3, parts of it are rewritten on the orders of the Head of IT | \[Certain\] |
| Judicial | Enforces the Pact, investigates, holds the relic records; commands armed raiders | \[Certain\] |
| Security / enforcer | Robert Sims acts as the regime's enforcer and runs surveillance | \[Certain\] |
| IT surveillance | IT watches residents; Mechanical uses hand signals to avoid being overheard | \[Certain\] |
| Relics | Any object from before the silo is banned, down to a PEZ dispenser; a black market of relic dealers exists | \[Certain\] |
| Punishments | Cleaning (capital), forced labor in the mines, prison; questioning the rules can trigger them | \[Certain\] for cleaning and mines; \[Likely\] for the scale of sentences |
| "The syndrome" | People with this condition are barred from public office and positions of power | \[Certain\] |
| How leaders are chosen | Holston named his own successor; how a Mayor is chosen is unclear | \[Certain\] for succession by pick; \[Guessing\] for elections |

**Model note:** the levers the regime controls are exactly the ones named in the plan: surveillance, punishment, information, reproduction. The fact that the Pact can be rewritten supports modeling the regime as adaptive policy, not fixed rules.

Sources: [Signal Horizon S1E5 on the Pact](https://signalhorizon.com/?p=16221), [TVLine S1E1 recap](https://tvline.com/2023/05/05/silo-recap-season-1-episode-1-rashida-jones-david-oyelowo/), [/Film on S3E9 and the Pact](https://slashfilm.com/2243825/silo-season-3-episode-9-the-pact-ai/), [Reason on relics](https://reason.com/2023/09/29/silo/), [Looper on the syndrome](https://www.looper.com/1315037/silo-deputy-billings-the-syndrome-explains-why-helping-juliette), [Soapcentral S2 ending](https://www.soapcentral.com/shows/silo-season-2-ending-explained-the-rebellion-brings-silo-18-edge).

## Economy and resources

The silo is a closed loop: one aging generator, farms, a water supply and recycled parts. The show gives the systems but almost no numbers, so nearly every rate here becomes a model parameter.

| Resource | What the show establishes | Confidence | For the model |
| --- | --- | --- | --- |
| Power | One main generator (turbine) in the Down Deep powers the whole silo; it is old and needs constant care; a near-failure is a major crisis (S1E3) | \[Certain\] | Stock with wear and failure probability |
| Air | The generator also keeps the air breathable | \[Likely\] | Tie air to power |
| Backup power | A limited backup exists for emergencies | \[Likely\] | Short buffer after a failure |
| Power as leverage | Mechanical cuts power to the whole silo to force a meeting (S2E4) | \[Certain\] | Mechanical can use the generator as a weapon |
| Food | Grown on farm levels, with some livestock | \[Likely\] | Production per farm worker, consumption per person |
| Water | Groundwater sits at the base; a groundwater pump on the lowest floor matters for survival | \[Likely\] | Critical system that can be sabotaged |
| Parts and materials | Supply hands out parts; quality varies (heat tape); scrap is scavenged and recycled | \[Certain\] | Parts stock, wear, scavenging rate |
| Currency | A physical pewter currency exists; its name and how wages work are not clear on screen | \[Certain\] for existence; \[Guessing\] for how it works | Skip in baseline; add later if needed |
| Rations | Not shown | \[Guessing\] | Parameter |

**Model note:** the baseline does not need money. Food, power, water and parts are enough to create scarcity, and scarcity feeds hardship later.

Sources: [Show Snob S1E3 recap](https://showsnob.com/2023/05/23/silo-season-1-episode-3-recap-machines/), [Santa Maria Sun review](https://www.santamariasun.com/?p=35119), [DMTalkies S2E1 on backup power](https://dmtalkies.com/silo-season-2-episode-1-ending-explained-2024/), [Show Snob S2E4 recap](https://showsnob.com/silo-season-2-episode-4-recap-the-harmonium), [DMTalkies S2E5 on the groundwater pump](https://dmtalkies.com/silo-season-2-episode-5-recap-2024/), [Cult of Mac S1 finale recap](https://www.cultofmac.com/?p=813285), [IndieTalk on the props](https://indietalk.com/threads/that-duct-tape-in-the-silo-finale-took-30-designs-to-get-right.68801).

## Cleaning

Cleaning is the silo's death penalty and its main deterrent: the condemned walk outside, wipe the camera that feeds the cafeteria screen, and die in public view. It removes dissenters and warns everyone else at the same time.

| Mechanic | What the show establishes | Confidence |
| --- | --- | --- |
| Two ways in | Sentenced for breaking the law, or by asking to go out (saying "I want to go out") | \[Certain\] |
| Saying it is final | Once said, the request is treated as binding | \[Likely\] |
| The act | The cleaner wipes the exterior camera lens that feeds the screen everyone watches | \[Certain\] |
| Why cleaners comply | The helmet shows a fake green world, so they believe they are helping others see the truth | \[Certain\] |
| Spectacle | Residents watch; the ritual is meant to keep people in line | \[Certain\] |
| Who it targets | People who question the system, break the Pact, or want to leave | \[Certain\] |
| When it fails | Juliette refused to clean and walked out of sight (S1 finale); that single event triggered unrest across Silo 18 in Season 2 | \[Certain\] |
| Frequency | Never stated; years can pass between cleanings | \[Guessing\] → parameter |

**Model note:** cleaning has two opposite effects. It removes dissenters and raises fear, but a visible failure can raise grievance across the whole population at once. The model should treat "a cleaning that goes wrong" as a shock event with its own probability.

Sources: [Reason review](https://reason.com/2023/09/29/silo/), [Fandomwire on cleaning](https://fandomwire.com/meaning-of-cleaning-in-the-tv-show-silo-explained/), [Hello Rayo S1E2 recap](https://www.hellorayo.co.uk/mfr/entertainment/television/silo-season-one-episode-two), [All the Fanfare Q&A](https://newsletter.allthefanfare.com/p/all-your-silo-questions-answered), [Space.com S2E1](https://space.com/entertainment/silo-season-2-episode-1-what-happened-to-silo-17s-tragic-rebel-uprising).

## History

The show gives three rebellion cases: Silo 18's old Rebellion, Silo 17's fatal one, and Silo 18's Mechanical uprising. These are the closest thing to validation data the model will have.

| When | Event | What it tells the model | Confidence |
| --- | --- | --- | --- |
| End of Season 3 (E10) | Silo 18 caps its Safeguard pipe before Silo 1 can gas it. The voice of the Algorithm offers a deal: no contact with other silos, no Safeguard. Juliette accepts but plans to attack Silo 1 | An outside controller can kill a silo; Mechanical know-how can disarm it; peace with Silo 1 depends on isolation | \[Certain\] |
| Season 3 (E9) | Flashbacks: the Pact was written by AI for Victor Crnkovich, creator of the memory drug, and edited with Anna Thurman | The rules were designed, not inherited; changing them is a legitimate model lever | \[Certain\] |
| Season 3 | Vitamin D+, a memory suppressant, is given to Juliette and later put into Silo 18's water on the Algorithm's orders | Memory is a regime lever: the regime can erase what people know | \[Certain\] for Juliette; \[Likely\] for the water |
| Season 3 | The Pact itself is rewritten by IT | The regime adapts its rules under pressure | \[Certain\] |
| Season 2 | Mechanical uprising in Silo 18: riots at the stairwell barricades, Mechanical cuts the power, raiders sent down; at the end rebels take the upper levels and the stairs are blown around levels 90–92, splitting the silo in two | A rebellion can start in one zone, use a critical system as leverage, and physically fragment the silo | \[Certain\] |
| Season 2 | Mechanical learns it has always been made the scapegoat when rebellions rise | Blame falls on the lowest class, which feeds grievance | \[Certain\] |
| End of Season 1 | Juliette refuses to clean and walks out of sight; unrest follows | One visible failure of the system can spark dissent | \[Certain\] |
| Over 25 years before S2 | Silo 17: a cleaner refused to clean and walked off; the fight for control plus exposure to the outside killed nearly everyone; a raider wrecked the groundwater pump to flood Mechanical. Season 3 adds that thousands of escapees reached the surface and were killed by Silo 1's drones and gas, about 9,800 dead | Rebellion can kill a silo through damage to life-support, and through Silo 1 destroying a silo that breaks isolation | \[Certain\] |
| About 140 years before S1 | "The Rebellion" in Silo 18, crushed by the authorities, with Head of IT Salvador Quinn at its centre; celebrated as Freedom Day; records from before it are lost; many current rules trace back to it | Long stable stretches between revolts; the regime tightened rules after winning | \[Certain\] for the date and holiday; \[Likely\] for lost records, Quinn's role and memory drugs used afterwards |
| About 300 years before S1 | Rough age of the silo, as suggested by the production team | Total run length to simulate | \[Likely\] |

The Flamekeepers, a secret group that keeps the true history alive, and Gloria's account of past rebellions and forced birth control (S1E7) show that knowledge of the past survives underground despite the ban. \[Certain\]

Sources: [Serienjunkies on Freedom Day](https://www.serienjunkies.de/news/redaktion/silo-freedom-day-review-der-pilotfolge-92588547.html), [TV Tropes](https://tvtropes.org/pmwiki/pmwiki.php/Series/Silo), [TV Obsessive S2E3](https://tvobsessive.com/2024/11/29/silo-s2e3-recap-solo-begins-bringing-everyone-together/), [TV Obsessive podcast on S2E4](https://tvobsessive.com/2024/12/06/the-tv-obsessive-podcast-episode-78-news-notes-silo-s2e4/), [A.V. Club S2 finale](https://avclub.com/silo-season-2-finale-recap), [/Film S2 ending](https://www.slashfilm.com/1761055/silo-season-2-ending-explained), [Looper on S1E7](https://www.looper.com/1309452/silo-forced-birth-control-prevent-rebellions), [IndieTalk on the props](https://indietalk.com/threads/that-duct-tape-in-the-silo-finale-took-30-designs-to-get-right.68801).

## Season 3: the system behind the silos

Season 3 shows that Silo 18 is not self-governing: Silo 1 watches it through the Head of IT, can gas it, and kills anyone who leaves. It also shows that memory itself is a tool of control, through a drug disguised as vitamins.

| Mechanic | What the show establishes | Confidence |
| --- | --- | --- |
| Silo 1 | The control silo. Its people sleep in cryosleep and wake for shifts; a director (Thurman) and specialists run the other silos from there | \[Certain\] |
| The Algorithm | Not a machine: people in Silo 1 speak through it to each silo's Head of IT, watch the silo, and give orders | \[Certain\] |
| Head of IT | The only link to the Algorithm; in Season 3 Camille Sims holds the post and carries out its orders | \[Certain\] |
| The Directive | Silos must stay isolated. Contact between silos is treated as a threat, and Silo 1 destroys a silo that threatens the system | \[Certain\] |
| Safeguard | Poison gas sent through a pipe into a silo, triggered from Silo 1. Capping the pipe defeats it: Silo 17's survivors did, and Silo 18 does in E10 | \[Certain\] |
| Outside the silo | Armed drones and gas kill anyone who leaves; a drone now watches Silo 18 | \[Certain\] |
| How cleaners die | Poison in the suit and Silo 1's drones; the outside air did not kill Silo 17's escapees at once | \[Likely\] |
| Vitamin D+ | A memory suppressant disguised as vitamins. Juliette's amnesia came from it, and the Algorithm ordered her dose doubled | \[Certain\] |
| Memory comes back | After her pills were swapped for sugar pills, objects from her past (a Pez dispenser, George's watch) brought her memories back | \[Certain\] |
| Mass memory wipe | The Algorithm ordered Vitamin D+ into Silo 18's water; similar drugs are suggested after the Rebellion 140 years ago | \[Likely\] |
| The Pact's origin | Written by AI for Victor Crnkovich and edited with Anna Thurman; rewritten in Silo 18 during Season 3 (Billings, under Camille) | \[Certain\] |
| Contact routes | Radio (Walker, taught by Bernard); tunnels from the mines and the Digger Void toward other silos; a door below Silo 18 where the Algorithm answers | \[Certain\] |
| Number and place | About 50 silos near Atlanta, built against a nanotechnology ("gray goo") catastrophe | \[Likely\] |
| Power in Silo 18 | Juliette is made Mayor while amnesiac, as a figurehead; real power sits with the Head of IT and the Algorithm | \[Likely\] |

**Model note:** Season 3 turns three things into levers. *Memory:* the regime can erase knowledge, and objects can restore it, so relics and memory belong in one module. *Silo 1:* an outside actor whose trigger is contact between silos or escape, not unrest as such. *Contact:* radio and digging are the rebel actions Silo 1 punishes. "Vitamin D" in this model means the drug, not nutrition.

Sources: [/Film on Vitamin D+](https://slashfilm.com/2215921/silo-season-3-vitamis-d-plus-explaned/), [Inverse on Vitamin D+](https://www.inverse.com/entertainment/vitamin-d-silos-memory-erasing-drug-explained), [/Film on the Algorithm](https://slashfilm.com/2249477/silo-season-3-true-face-daniel-memory-algorithm/), [/Film on S3E9 and the Pact](https://slashfilm.com/2243825/silo-season-3-episode-9-the-pact-ai/), [Seat42F S3E10 recap](https://seat42f.com/silo-season-3-episode-10-recap/), [The Astromech: Season 3 explained](https://www.theastromech.com/2026/09/silo-events-of-season-3-explained.html), [Ready Steady Cut: Season 3 ending](https://readysteadycut.com/2026/09/04/silo-season-3-ending-explained/), [Ready Steady Cut: the Algorithm and Camille](https://readysteadycut.com/2026/07/03/silo-season-3-algorithm-camille/), [/Film: questions after the finale](https://www.slashfilm.com/2248232/silo-season-3-finale-unanswered-questions/), [Soapcentral on Salvador Quinn](https://www.soapcentral.com/shows/who-salvador-quinn-silo-details-140-year-old-rebellion-explored), [Memoria: Season 3 recap](https://memoria.film/silo-season-3-ending-explained-plot-recap/), [/Film on the Safeguard](https://slashfilm.com/1804875/silo-season-2-safeguard-explained).

## Model implications

The show supplies the rules and structure; almost every rate has to be invented. The baseline needs five modules, and the parameters below are starting guesses to calibrate until the silo stays stable for 200 years.

**Status (Oct 30, 2026):** the baseline is built and calibrated (tag `baseline-v1`). Part 2 adds the modules marked below; the plan is in `docs/tasks-part2.md`.

### What goes into the baseline

| Module | Built from | In baseline? |
| --- | --- | --- |
| Structure | 144 levels, three zones, one stairwell, travel time | Yes |
| Demography | 10,000 people, implants, lottery, 365-day window, aging, deaths | Yes |
| Work | Departments by zone, shadowing, inherited trades | Yes |
| Resources | Generator, air, food, water, parts, Supply | Yes |
| Institutions | Mayor, Judicial, Sheriff, IT as roles; Pact as a rule set | Yes, as passive roles |
| Beliefs and dissent | Relics, Flamekeepers, surveillance, covert lottery denial | Part 2 |
| Cleaning and rebellion | Cleaning as punishment and shock, uprisings, sabotage | Part 2 |
| Memory | Vitamin D+ wipes knowledge; relics bring it back; history erased after rebellions | Part 2 |
| Silo 1 | Algorithm surveillance, the Directive, Safeguard gas, drones, contact by radio and tunnels | Part 2 |

### Parameters to invent

All starting values are \[Guessing\] unless the show column gives the number.

| Parameter | Show says | Starting value | Anchor |
| --- | --- | --- | --- |
| Population | About 10,000 | 10,000 | Show |
| Time step | — | 1 day | Rebellions unfold over days |
| Full stair descent | About a day top to bottom (reviewer's estimate) | 12 hours of walking | Show, roughly |
| Population split by zone | Most people Up Top and in the Mids | 35% / 40% / 25% | Guess |
| Lottery trigger | Not explained | One permit per death, capped at the target size | Keeps population flat |
| Conception window | 365 days | 365 days | Show |
| Chance of birth within the window | — | About 80% | Real-world fertility for couples trying for a year |
| Life expectancy | — | About 70 years | Basic medicine, no outside world |
| Shadowing age | Teenagers | 14 | Guess from Juliette's story |
| Food need | — | About 2,200 kcal per person per day | Adult average |
| Generator failure | Old, needs constant care | Daily failure chance rising with neglect | Calibrate to rare crises |
| Cleanings per year | Years can pass between them | 0–2 | Calibrate |

### Open gaps to settle in the spec

- [x] Partnering: chosen freely. Children still need a lottery win, so partnering shapes the social network, not population size.
- [x] Mayor: appointed by Up Top by default, with elections as a scenario toggle. Elections give grievance a peaceful outlet, so comparing the two answers part of the rebellion question.
- [x] Rations and money: model rations as each zone's share of the food supply, a policy lever the regime controls. Skip money in the baseline.
- [x] Departments: Medical gets a small module (deaths, injuries, implant removals for the lottery). Education stays job slots for now and becomes the source of loyalty in the beliefs phase. Maintenance folds into Resources as wear and repair.

## Sources

Each section lists its own sources. The pages used most:

- [Wikipedia: Silo (TV series)](<https://Www.wikipedia.org/wiki/Silo_(TV_series)>)
- [TV Tropes: Silo](https://tvtropes.org/pmwiki/pmwiki.php/Series/Silo)
- [Fiction Horizon: how many levels the silo has](https://fictionhorizon.com/how-many-levels-does-silo-have-explained/)
- [Fiction Horizon: how many people live in Silo 18](https://fictionhorizon.com/how-many-people-live-in-the-silo-18-numbers-revealed/)
- [Otakukart: Season 1 explainer](https://otakukart.com/silo-season-1-ending-explained-the-mystery-conspiracy-and-shocking-revelations-that-set-the-stage-for-an-epic-season-2/)
- [Soapcentral: Season 2 ending](https://www.soapcentral.com/shows/silo-season-2-ending-explained-the-rebellion-brings-silo-18-edge)
- [TV Obsessive: S2E3 recap](https://tvobsessive.com/2024/11/29/silo-s2e3-recap-solo-begins-bringing-everyone-together/)
- [/Film: S3E9 and the Pact](https://slashfilm.com/2243825/silo-season-3-episode-9-the-pact-ai/)

Fan explainers sometimes mix in book details. Anything that looked book-derived is tagged \[Likely\] at most, and nothing from the books was used.
