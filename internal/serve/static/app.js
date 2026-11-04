// The viewer's page (docs/viewer.md). It only reads what the server's API
// returns; it computes no statistic of its own.
import { replay } from "./replay.js";
import { dashboard } from "./dashboard.js";
import { experiment } from "./experiment.js";
import { story } from "./story.js";

const view = document.getElementById("view");
const crumb = document.getElementById("crumb");
const sub = document.getElementById("sub");

const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);
const when = (t) => new Date(t * 1000).toLocaleString();

// file fetches one of a run's files as text; with a fallback, a missing
// file gives the fallback instead of an error.
async function file(run, name, fallback) {
  const r = await fetch(`api/file?run=${encodeURIComponent(run)}&name=${name}`);
  if (r.ok) return r.text();
  if (fallback !== undefined && r.status === 404) return fallback;
  throw new Error(`${name}: ${r.status}`);
}

async function api(path) {
  const r = await fetch(path);
  if (!r.ok) throw new Error(`${path}: ${r.status} ${await r.text()}`);
  return r.json();
}

const BANNER = `  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░╔═╗░░░░░  <span>the outside</span>
  ███████╗██╗██╗      ██████╗      ║▒║  <span>Up Top      floors 1–48</span>
  ██╔════╝██║██║     ██╔═══██╗     ║▒║
  ███████╗██║██║     ██║   ██║     ║▒║  <span>Mids        floors 49–96</span>
  ╚════██║██║██║     ██║   ██║     ║▒║
  ███████║██║███████╗╚██████╔╝     ║▒║  <span>Down Deep   floors 97–144</span>
  ╚══════╝╚═╝╚══════╝ ╚═════╝      ╚═╝

<span>           18  ·  10,000 people  ·  one Pact</span>`;

// View 1, Runs (home): the banner and the runs outside batches, newest first.
async function home() {
  const { runs } = await api("api/runs");
  const loose = runs.filter((r) => !r.batch);
  view.innerHTML = `<pre class="banner">${BANNER}</pre>
    <h2>Runs</h2>${loose.length ? runTable(loose) : `<p class="empty">No single runs outside batches.</p>`}`;
}

function batchTable(batches, tick) {
  return `<div class="wrap"><table>
    <tr>${tick ? "<th></th>" : ""}<th>batch</th><th>scenario</th><th class="num">runs</th><th class="num">collapses</th><th>modified</th></tr>
    ${batches.map((b) => `<tr>${tick ? `<td><input type="checkbox" value="${esc(b.path)}"></td>` : ""}
      <td><a href="#/batch/${encodeURIComponent(b.path)}">${esc(b.path)}</a></td>
      <td>${esc(b.scenario)}</td><td class="num">${b.runs}</td><td class="num">${b.collapses}</td><td>${when(b.modified)}</td></tr>`).join("")}
  </table></div>`;
}

// Batches: every batch; a click opens its experiment view.
async function batches() {
  const { batches } = await api("api/runs");
  view.innerHTML = `<h2>Batches</h2>${batches.length ? batchTable(batches, false) : `<p class="empty">No batches.</p>`}`;
}

// Compare: tick batches, then open the experiment view for them.
async function picker() {
  const { batches } = await api("api/runs");
  if (!batches.length) { view.innerHTML = `<h2>Compare</h2><p class="empty">No batches.</p>`; return; }
  view.innerHTML = `<h2>Compare batches</h2>
    <div class="controls"><button class="go" id="go" disabled>Compare</button><span class="empty" id="n">Tick two or more batches.</span></div>
    ${batchTable(batches, true)}`;
  const go = view.querySelector("#go");
  const ticked = () => [...view.querySelectorAll("input:checked")].map((c) => c.value);
  view.onchange = () => {
    const n = ticked().length;
    go.disabled = n === 0;
    go.textContent = n > 1 ? `Compare ${n} batches` : "Compare";
  };
  go.onclick = () => { location.hash = `#/compare/${ticked().map(encodeURIComponent).join("+")}`; };
}

function runTable(runs) {
  return `<div class="wrap"><table>
    <tr><th>run</th><th>scenario</th><th class="num">seed</th><th class="num">years</th><th>collapse</th><th>finished</th><th>replay</th></tr>
    ${runs.map((r) => `<tr><td><a href="#/run/${encodeURIComponent(r.path)}">${esc(r.path)}</a></td><td>${esc(r.scenario)}</td>
      <td class="num">${r.seed}</td><td class="num">${r.years}</td><td>${esc(r.collapse ?? "—")}</td><td>${r.finished ? "yes" : "no"}</td>
      <td>${r.frames ? `<a href="#/replay/${encodeURIComponent(r.path)}">replay</a>` : "—"}</td></tr>`).join("")}
  </table></div>`;
}

// View for a batch: its experiment view, then its runs.
async function batch(path) {
  const [{ runs }] = await Promise.all([api("api/runs"), experiment(view, [path], api)]);
  view.insertAdjacentHTML("beforeend", `<h2>Runs in ${esc(path)}</h2>${runTable(runs.filter((r) => r.batch === path))}`);
}

// runTabs: Dashboard and Replay for one run; Replay only with frames.
async function runTabs(path, on) {
  const { runs } = await api("api/runs");
  const frames = runs.find((r) => r.path === path)?.frames;
  const p = encodeURIComponent(path);
  sub.innerHTML = `<div class="tabs"><a href="#/run/${p}" class="${on === "run" ? "on" : ""}">Dashboard</a>${frames
    ? `<a href="#/replay/${p}" class="${on === "replay" ? "on" : ""}">Replay</a>`
    : `<span title="This run has no frames: run it with scenarios/replay.yaml">Replay</span>`}<a href="#/story/${p}" class="${on === "story" ? "on" : ""}">Story</a></div>`;
}

const TAB = { "": "runs", run: "runs", replay: "runs", story: "runs", batches: "batches", batch: "batches", compare: "compare" };

async function route() {
  const [, kind = "", ...rest] = location.hash.replace(/^#\/?/, "#/").split("/");
  const path = decodeURIComponent(rest.join("/"));
  document.querySelectorAll("nav a").forEach((a) => a.classList.toggle("on", a.dataset.tab === TAB[kind]));
  sub.innerHTML = "";
  view.onchange = null;
  crumb.textContent = path ? path : "";
  try {
    if (kind === "batches") await batches();
    else if (kind === "batch") await batch(path);
    else if (kind === "compare" && path) { const ps = rest.join("/").split("+").map(decodeURIComponent); crumb.textContent = `${ps.length} batches`; await experiment(view, ps, api); }
    else if (kind === "compare") await picker();
    else if (kind === "run") { view.innerHTML = "<p>Loading…</p>"; await runTabs(path, kind); await dashboard(view, path, file); }
    else if (kind === "story") { view.innerHTML = "<p>Loading…</p>"; await runTabs(path, kind); await story(view, path); }
    else if (kind === "replay") { view.innerHTML = "<p>Loading frames…</p>"; await runTabs(path, kind); await replay(view, path, file); }
    else await home();
  } catch (e) {
    view.innerHTML = `<p class="empty">${esc(e.message)}</p>`;
  }
}

addEventListener("hashchange", route);
route();
