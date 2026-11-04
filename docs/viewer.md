# Results viewer — spec (T16.0)

Nov 4, 2026 · decision D12, approved by Adam · Milestone 16

The viewer is how a run is *seen*: a rebellion spreading through the floors, a silo sliding to collapse, strategies side by side. It answers no research question by itself (`docs/tasks-part2.md`, M16), so it adds no model code and changes no run.

## Rules

- **Read-only, files only** (the viewer rule, T7.2): `silosim serve` reads files under its `--runs` folder and nothing else. It never imports the engine or a module, never writes, and serves only GET requests. A run viewed is byte-identical to a run not viewed.
- **One binary, no build step** (D12): Go's `net/http`, the page embedded with `embed`. The page is plain HTML, CSS and JavaScript modules, no framework, no bundler.
- **One vendored chart library** (the agreed exception to the dependency rule): **uPlot 1.6.31** (MIT, about 50 KB), copied into `internal/serve/static/` with its licence. It draws every time series and curve; the floor cross-section is plain SVG.
- **Same numbers as the CLI:** the experiment view gets its numbers from the `report` package, the code behind `silosim report --outcomes`, through the server. The page computes no statistic of its own.
- **Local by default:** it listens on `127.0.0.1:8080` (`--addr` to change).

## Command

```text
silosim serve --runs runs/ [--addr 127.0.0.1:8080]
```

## Data contract

What the viewer reads, all written by the model already, except the two frame metrics marked new.

| File | Read by | Used for |
| --- | --- | --- |
| `run.json` | list, every view | scenario name, seed, finished, outcomes; `scenario.init` for the floor layout (§5); `scenario.events` for scripted `breakStairs` segments |
| `daily.csv` | dashboard | time series, by column name |
| `yearly.csv` | dashboard | legitimacy and knowledge by zone, yearly |
| `events.jsonl` | dashboard, replay | markers; stair segments broken (`sabotage` with `segment`, scripted `breakStairs`) and repaired (`stairRepaired`) |
| `rebellions.jsonl` | dashboard, replay | rebellion spans, shaded |
| `frames.csv.gz` | replay | per floor and frame day: `population`, `households`, `health`, `legitimacy`, `knowsHistory`, and **new** `active` and `violent` |

**New frame metrics (T16.2).** ODD §6.14, Outputs, promises "per floor, the share of adults active and violent" in `frames.csv.gz`, but the rebellion module never registered them. T16.2 adds them: per floor, the share of the adults *living* there (home floor, as the beliefs metrics) who are active or beyond, and who are violent. Frames are off by default, so the golden hash and every existing output stay as they are.

**Replay needs frames.** A run has frames only if `output.frames: true`; the M15 batches don't. The list marks which runs can be replayed; a replay run is made with one scenario override, for example `scenarios/replay.yaml` (`extends: canon.yaml`, `output: { frames: true }`), added in T16.2.

**Endpoints** (JSON unless said):

| Endpoint | Returns |
| --- | --- |
| `GET /` | the page |
| `GET /api/runs` | every run folder (has `run.json`) under `--runs`, and every batch folder (holds run folders): path, scenario name, seed, finished, measured years, collapse, whether it has frames |
| `GET /api/file?run=PATH&name=NAME` | the raw file; NAME is one of the six above; `frames.csv.gz` sent with `Content-Encoding: gzip`, so the browser unpacks it |
| `GET /api/outcomes?batch=PATH` | the outcome metrics of `silosim report --outcomes` for one batch, as JSON (rates, intervals, endings, Kaplan–Meier steps) |

PATH is relative to `--runs`; one that is absolute, contains `..`, or resolves outside it is refused (403).

## Views

**1. Runs (home).** Batches and loose runs in two tables, newest first: name, seeds, years, collapses, frames yes/no. A click opens a run's dashboard, a batch's experiment view; a "replay" link on runs with frames.

**2. Run dashboard (T16.3).** Stacked time-series panels sharing one time axis (measured years), zoom by dragging, a cursor read-out of every series:

1. population: total and by zone;
2. life support: powerFactor, water, air, food stock in days (own axis);
3. health and hardship by zone;
4. dissent: adults active, joined, violent; workers on strike;
5. yearly: mean L and the share who know each flag, by zone where the file has it.

Rebellions are shaded bands across every panel; scripted events, collapses and the Safeguard are vertical lines with a label; a toggle shows generator failures and policy starts.

**3. Replay (T16.2).** The silo as a column of 144 floors, top to bottom, each a horizontal bar coloured by the chosen metric for the current frame; zone boundaries labelled; farm and Mechanical floors (the §5 layout from `run.json`) hatched; broken stair segments drawn as gaps between floors. A time slider over the frame days with play and pause (1, 5 or 20 frames a second) and a timeline under it with rebellion spans and event marks; the current values of the floor under the pointer. Metrics and colour scales:

| Metric | Scale |
| --- | --- |
| population, households | sequential, 0 to the run's highest value on any floor |
| health, legitimacy, knowsHistory, active, violent | sequential, fixed 0–1 |

Dashboard panels 1–4 for the current day sit beside the column, so the replay and the totals move together.

**4. Experiment view (T16.4).** Pick two or more batches (for example E3's six). Side by side, same seeds:

- Kaplan–Meier curves to the first rebellion and to collapse, one line per batch, Greenwood bands shaded;
- rebellions per century and share of time in a rebellion: a point per batch with its 95% interval;
- endings: stacked shares per batch.

Every number is the one `silosim report --outcomes` prints for that batch.

## Colours

Colour-blind safe, the same in every view (Okabe–Ito): Up Top `#0072B2` (blue), Mids `#009E73` (green), Down Deep `#D55E00` (vermillion); total `#000000`. Rebellion bands `#CC79A7` at 15% opacity; events dark grey dashed, collapse and Safeguard solid black. Batches in the experiment view: Okabe–Ito in order (`#E69F00`, `#56B4E9`, `#009E73`, `#F0E442`, `#0072B2`, `#D55E00`, `#CC79A7`). Floor scales: viridis. Light background; no dark mode.

## Code

```text
cmd/silosim/serve.go          # the serve command: flags, then serve.Handler
internal/serve/serve.go       # handlers, path checks, run listing
internal/serve/static/        # index.html, app.js, style.css, uPlot (vendored, with LICENSE)
internal/report/outcomes.go   # the outcome numbers as a struct, rendered as text (CLI) and JSON (viewer)
```

## Tests

- Go (`httptest`): the listing finds runs and batches in a hand-made folder tree; files are served with the right type and encoding; `..`, absolute paths and unknown names get 403; `/api/outcomes` gives the same numbers as `WriteOutcomes` for the same batch; the new frame metrics on a tiny world.
- Browser: each view checked by hand in Chrome on a canon replay run and an M15 batch (CLAUDE.md, Visual changes), with screenshots in the task's summary.

## Tasks

- **T16.1** `silosim serve`, the runs list and the file endpoint.
- **T16.2** the two new frame metrics, `scenarios/replay.yaml`, the replay view.
- **T16.3** the run dashboard.
- **T16.4** the outcomes as JSON and the experiment view.
- *Milestone done when:* a canon run with frames replays from start to end; the experiment view shows the same numbers as `silosim report --outcomes`.

## Left out

Live viewing of a running simulation; editing or starting runs from the page; dark mode; mobile layout beyond a readable single column; any chart library besides uPlot.
