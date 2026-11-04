# Chronicle and narrator — spec (T17.0)

Nov 4, 2026 · decision D14, approved by Adam · Milestone 17 (optional work, started before M15 at Adam's request)

A run's files say *what* happened. This milestone turns them into something a person reads from top to bottom: a dated **chronicle** of the facts, written by Go code, and, on request, a **story** written from that chronicle by Claude. Like the viewer, it answers no research question (`docs/tasks-part2.md`, Optional) and adds no model code.

## Rules

- **Files in, nothing changed.** Both read only a run folder's files (the viewer rule, T7.2). The model's six files are never touched; `narrate` adds one file, `story.md`, next to them. A run narrated is the same run as one not narrated.
- **The chronicle is plain, deterministic Go** (`internal/chronicle`, standard library only): the same run always gives the same chronicle, byte for byte. Every line in it comes from a line of a run file, by the rules below; it computes no statistic and draws no random number.
- **Only `narrate` talks to the network**, and only to the Claude API. It sends the chronicle and nothing else. The server (`silosim serve`) never calls out; it shows a `story.md` that is already there.
- **One new dependency, used in one package** (D14): the official Anthropic Go SDK, `github.com/anthropics/anthropic-sdk-go`, imported only by `internal/narrate`. The simulation, the report and the server never import it, so the golden hash and every run stay as they are.
- **Roles, never names.** The model has no named people (`docs/tasks-part2.md`, Left out on purpose), so neither does the chronicle or the story: "a cleaner from the Down Deep", "the new Head of IT", never a name, and never a character, place or event from the series or the books that the run did not produce.

## Commands

```text
silosim chronicle --run DIR                    # the chronicle as Markdown, on stdout
silosim narrate   --run DIR [--model ID]       # writes DIR/story.md; default model claude-opus-5-5
```

`narrate` needs Claude API credentials: `ANTHROPIC_API_KEY`, or a profile from `ant auth login`, which the SDK finds by itself. Without them it stops with a message saying so, and writes nothing. A story costs about $0.10–0.20 (a chronicle of a few thousand tokens in, up to about 3,000 words out, at Opus 5.5's $4 / $20 per million tokens).

## The chronicle

Time is measured years from the first measured day, one decimal, as in the dashboard: year = (day − warm-up days) / 365. Entries are in day order; entries on the same day keep the order of the files they come from (`events.jsonl`, then `rebellions.jsonl`). A span (a rebellion, a spell of unrest, a policy) is placed at its start day, though the files log it at its end; one that began in the warm-up is dated "Warm-up" (T17.1).

**Header.** Scenario name, seed, measured years run, the modules on, and the outcome: collapse (cause, year) or the silo still standing at the end; the number of rebellions and unrest spells.

**Entries**, one line each, from `events.jsonl` unless marked:

| Source | Entry |
| --- | --- |
| `scripted` | the scripted event and its day, e.g. "a cleaning failure" (`cleaningFailure`), "stairs broken at segment *s*" (`breakStairs`) |
| `failure` + `repair`, `emergency` + `emergencyEnd` | the generator failed; days without power until the repair; an emergency declared and lifted in that window is part of the same entry. An emergency outside any outage is an entry of its own (T17.1) |
| `unrest` | a spell of unrest, its start and end years |
| `rebellions.jsonl` | a rebellion: start year, trigger, starting zone, peak share of adults active (%), duration, departments struck, zones held, fight days, rebels and officers killed, arrests, stair segments broken, generator and pump sabotage, ending |
| `zoneHeld`, `zoneFreed` | rebels take or lose a zone |
| `policy`, `policyEnd` | the regime answers: the trigger and the entry's levers (`set`, `lift`, `waterDosing`, Pact rows), and until when it holds. The entry is read from `run.json`: the playbook entry the event names, Silo 1's `orders`, or the scenario's scripted `policy` event on that day |
| `headOfIT` | a new Head of IT, and whether their shadow succeeded |
| `mayor` | a new Mayor: why the post fell vacant (`reason`), appointment or election, the zone they come from |
| `cleaning` with `failed: true` | a cleaner walked out of sight, with the Pact rule they were sent out under |
| `safeguard`, `collapse`, `extinction` | Silo 1's Safeguard; the collapse and its cause; the last death |

**Cleanings that do not fail** are too many to list one by one (hundreds in a long run). They are counted instead, per decade, by the Pact rule they were sent out under, in the decade lines.

**Decade lines**, closing each ten measured years (a run cut short gets a last, shorter section), from the last yearly snapshot on or before the section's end (`yearly.csv`) and the population on that snapshot's day (`daily.csv`), so the line is of one moment: population, mean legitimacy, the share of adults who know the true history, Flamekeepers, the people held by Judicial (`held`; the `prison` column counts sentences, not people), and that decade's cleanings. A column the run does not have (its module was off) is left out of the line. Arrests are left out (T17.1): the first measured year's yearly counts of the control module also hold the whole warm-up's, so a decade's sum would mislead; a rebellion's arrests are in its entry.

Nothing else from the files goes in. In particular: individual sabotage acts and arrests (they are counted in the rebellion entry and the decade lines), rationing switches, departments falling below half their target, and the frames.

## The story

`narrate` sends one request to the Messages API: a fixed system prompt (below), and the chronicle as the user message. Model `claude-opus-5-5` unless `--model` names another; adaptive thinking (the model's default); the server-side refusal fallback on; streamed, since the story is long. The reply's text is written to `DIR/story.md`, after a header line naming the model and the date. A refusal or an error writes nothing and says why.

**System prompt** (fixed in `internal/narrate`, quoted here so it can be reviewed):

> You are the archivist of Silo 18, an underground silo of 144 floors under the Pact, writing the record of its years from the chronicle you are given. Write it as an account a reader in the silo could keep: in eras, under headings, in plain prose.
>
> Use only what the chronicle says. Every event, year and number in your account must be in it; when you round a number, say "about". Do not add events, causes, motives or people it does not give. Where it is silent, say so or say nothing. People have no names: call them by their role and floor or zone ("a cleaner from the Down Deep", "the new Head of IT"). Do not use characters, places or events from the television series *Silo* or the *Wool* books.
>
> Aim for 1,000 to 2,500 words, in Markdown, with a heading per era and a short closing paragraph on how the silo stands at the end.

## Viewer

A run's pages get a third tab, **Story** (`docs/viewer.md`, Menu): `story.md` if the run has one, rendered (headings, paragraphs, lists, bold and italics only, by a few lines in the page; no library), then the chronicle under it. Without `story.md` it shows the chronicle and the `narrate` command for that run.

New endpoint: `GET /api/chronicle?run=PATH` returns the chronicle as text (the same bytes `silosim chronicle` prints), with the path checks of `/api/file`. `story.md` joins the files `/api/file` may send.

## Tests

- **Chronicle:** a hand-made run folder with one of each event type gives the expected lines in the expected order; a run with only the baseline modules gives a chronicle without the part-2 entries and columns; two calls give identical bytes.
- **Narrate:** against a fake API server (`httptest`, through the SDK's base-URL option): the request carries the model, the system prompt and the chronicle; the reply is written to `story.md` with its header; an error writes nothing. No test calls the real API.
- **Viewer:** `/api/chronicle` matches `silosim chronicle` for the same run; checked by hand in Chrome with a story generated once.

## Tasks

- **T17.0** this spec, D14, the milestone in `docs/tasks-part2.md`.
- **T17.1** `internal/chronicle`, `silosim chronicle`, `/api/chronicle` and the Story tab (chronicle only).
- **T17.2** the SDK, `internal/narrate`, `silosim narrate`, `story.md` in the Story tab.
- *Milestone done when:* a canon run's chronicle and story read correctly from start to end, and every number in the story can be found in the chronicle.

## Left out

Generated names for people; a story per batch or comparing scenarios; streaming the story into the page; narrating a run while it runs; any other model provider.
