// The viewer's page (docs/viewer.md). It only reads what the server's API
// returns; it computes no statistic of its own.
import { replay } from "./replay.js";

const view = document.getElementById("view");
const crumb = document.getElementById("crumb");

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
  if (!r.ok) throw new Error(`${path}: ${r.status}`);
  return r.json();
}

// View 1: runs and batches, newest first.
async function home() {
  crumb.textContent = "";
  const { batches, runs } = await api("api/runs");
  const loose = runs.filter((r) => !r.batch);
  view.innerHTML = `
    <h2>Batches</h2>${batches.length ? `<div class="wrap"><table>
      <tr><th>batch</th><th>scenario</th><th class="num">runs</th><th class="num">collapses</th><th>modified</th></tr>
      ${batches.map((b) => `<tr><td><a href="#/batch/${encodeURIComponent(b.path)}">${esc(b.path)}</a></td>
        <td>${esc(b.scenario)}</td><td class="num">${b.runs}</td><td class="num">${b.collapses}</td><td>${when(b.modified)}</td></tr>`).join("")}
    </table></div>` : `<p class="empty">No batches.</p>`}
    <h2>Runs</h2>${loose.length ? runTable(loose) : `<p class="empty">No single runs outside batches.</p>`}`;
}

function runTable(runs) {
  return `<div class="wrap"><table>
    <tr><th>run</th><th>scenario</th><th class="num">seed</th><th class="num">years</th><th>collapse</th><th>finished</th><th>replay</th></tr>
    ${runs.map((r) => `<tr><td><a href="#/run/${encodeURIComponent(r.path)}">${esc(r.path)}</a></td><td>${esc(r.scenario)}</td>
      <td class="num">${r.seed}</td><td class="num">${r.years}</td><td>${esc(r.collapse ?? "—")}</td><td>${r.finished ? "yes" : "no"}</td>
      <td>${r.frames ? `<a href="#/replay/${encodeURIComponent(r.path)}">replay</a>` : "—"}</td></tr>`).join("")}
  </table></div>`;
}

// View for a batch: its runs; the experiment view comes in T16.4.
async function batch(path) {
  crumb.textContent = `/ ${path}`;
  const { runs } = await api("api/runs");
  view.innerHTML = `<h2>Runs in ${esc(path)}</h2>${runTable(runs.filter((r) => r.batch === path))}`;
}

function soon(what, path) {
  crumb.textContent = `/ ${path}`;
  view.innerHTML = `<p class="empty">The ${what} of ${esc(path)} comes in a later task.</p>`;
}

async function route() {
  const [, kind, ...rest] = location.hash.replace(/^#\/?/, "#/").split("/");
  const path = decodeURIComponent(rest.join("/"));
  try {
    if (kind === "batch") await batch(path);
    else if (kind === "run") soon("dashboard", path);
    else if (kind === "replay") { crumb.textContent = `/ ${path} / replay`; view.innerHTML = "<p>Loading frames…</p>"; await replay(view, path, file); }
    else await home();
  } catch (e) {
    view.innerHTML = `<p class="empty">${esc(e.message)}</p>`;
  }
}

addEventListener("hashchange", route);
route();
