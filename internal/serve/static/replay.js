// View 3, run replay (docs/viewer.md, T16.2): the silo as a column of
// floors coloured by one frame metric, a time slider with play and pause,
// a timeline of rebellions and events, and the day's totals.
const NS = "http://www.w3.org/2000/svg";
const ZONES = [["upTop", "Up Top"], ["mids", "Mids"], ["downDeep", "Down Deep"]];
const VIRIDIS = ["#440154", "#482878", "#3e4989", "#31688e", "#26828e", "#1f9e89", "#35b779", "#6ece58", "#b5de2b", "#fde725"];
// Fixed 0–1 scale; the others (population, households, active, violent) run to the run's highest
// floor value: rebels are a few percent of a floor, invisible on 0–1.
const SHARES = new Set(["health", "legitimacy", "knowsHistory"]);
const TOTALS = ["population", "air", "water", "foodStockDays", "active", "joined", "violent", "onStrike", "brokenSegments"];

const el = (tag, attrs = {}, parent) => {
  const e = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs)) e.setAttribute(k, v);
  if (parent) parent.appendChild(e);
  return e;
};

function viridis(x) {
  const t = Math.min(1, Math.max(0, x)) * (VIRIDIS.length - 1);
  const i = Math.min(VIRIDIS.length - 2, Math.floor(t)), f = t - i;
  const rgb = (h) => [1, 3, 5].map((k) => parseInt(h.slice(k, k + 2), 16));
  const a = rgb(VIRIDIS[i]), b = rgb(VIRIDIS[i + 1]);
  return `rgb(${a.map((v, k) => Math.round(v + (b[k] - v) * f)).join(",")})`;
}

const lines = (text) => text.split("\n").filter((l) => l.length);

// parseFrames: day,floor,metric,value rows into per-metric arrays of floors
// by frame index.
function parseFrames(text, floors) {
  const days = [], index = new Map(), data = new Map();
  for (const line of lines(text).slice(1)) {
    const [d, f, m, v] = line.split(",");
    const day = +d;
    if (!index.has(day)) { index.set(day, days.length); days.push(day); }
    if (!data.has(m)) data.set(m, []);
    const frames = data.get(m), i = index.get(day);
    frames[i] ??= new Float32Array(floors);
    frames[i][+f - 1] = +v;
  }
  return { days, data };
}

// layout: the §5 floor plan from the scenario: zones, farm and Mechanical floors.
function layout(init) {
  const pz = init.floorsPerZone, n = 3 * pz, work = new Set();
  for (let i = 0; i < init.farmFloors; i++) work.add(pz + 1 + Math.floor((i * pz) / init.farmFloors));
  for (let f = n - init.mechanicalFloors + 1; f <= n; f++) work.add(f);
  return { n, pz, work, zone: (f) => ZONES[Math.min(2, Math.floor((f - 1) / pz))] };
}

// stairChanges: every segment broken or repaired, in day order (sabotage,
// stair repairs, scripted breakStairs from the scenario).
function stairChanges(events, scripted) {
  const ch = [];
  for (const e of events) {
    if (e.type === "sabotage" && e.target === "stairs") ch.push([e.day, e.segment, true]);
    if (e.type === "stairRepaired") ch.push([e.day, e.segment, false]);
  }
  for (const e of scripted ?? []) if (e.type === "breakStairs") ch.push([e.day, e.segment, true]);
  return ch.sort((a, b) => a[0] - b[0]);
}

export async function replay(view, path, file) {
  const [info, frames, rebellions, events, daily] = await Promise.all([
    file(path, "run.json").then(JSON.parse), file(path, "frames.csv.gz"),
    file(path, "rebellions.jsonl", ""), file(path, "events.jsonl", ""), file(path, "daily.csv", ""),
  ]);
  const sc = info.scenario, plan = layout(sc.init), warmup = sc.warmupYears * 365;
  const { days, data } = parseFrames(frames, plan.n);
  if (!days.length) throw new Error("this run has no frames");
  const metrics = [...data.keys()];
  const top = new Map(metrics.map((m) => [m, SHARES.has(m) ? 1 : Math.max(1e-9, ...data.get(m).map((a) => Math.max(...a)))]));
  const rebs = lines(rebellions).map(JSON.parse), evs = lines(events).map(JSON.parse);
  const changes = stairChanges(evs, sc.events);
  const marks = evs.filter((e) => ["scripted", "collapse", "safeguard", "extinction"].includes(e.type));
  const dl = lines(daily), head = dl.length ? dl[0].split(",") : [], first = dl.length > 1 ? +dl[1].split(",")[0] : 0;
  const year = (d) => ((d - warmup) / 365).toFixed(1);

  view.innerHTML = `
    <div class="controls">
      <label>metric <select id="metric">${metrics.map((m) => `<option>${m}</option>`).join("")}</select></label>
      <button id="play">play</button>
      <label>speed <select id="fps"><option value="1">1</option><option value="5" selected>5</option><option value="20">20</option></select> frames/s</label>
      <span id="when"></span>
    </div>
    <input id="slider" type="range" min="0" max="${days.length - 1}" value="0">
    <svg id="timeline" class="timeline" height="28"></svg>
    <div class="replay"><svg id="column" width="260" height="${plan.n * 4 + 50}"></svg>
      <div><p id="hover" class="empty">Point at a floor.</p><table id="totals"></table></div></div>`;
  const $ = (id) => view.querySelector("#" + id);
  const col = $("column"), tl = $("timeline");

  // Column: zone strips and labels, one rect per floor, hatching, stairs.
  const defs = el("defs", {}, col), pat = el("pattern", { id: "hatch", width: 6, height: 6, patternUnits: "userSpaceOnUse", patternTransform: "rotate(45)" }, defs);
  el("line", { x1: 0, y1: 0, x2: 0, y2: 6, stroke: "#fff", "stroke-width": 2, "stroke-opacity": 0.6 }, pat);
  const y = (f) => 10 + (f - 1) * 4;
  ZONES.forEach(([k, label], z) => {
    el("rect", { x: 70, y: y(z * plan.pz + 1), width: 6, height: plan.pz * 4, fill: `var(--${k})` }, col);
    const t = el("text", { x: 64, y: y(z * plan.pz + 1) + plan.pz * 2, "text-anchor": "end", class: "zl" }, col);
    t.textContent = label;
  });
  const rects = [];
  for (let f = 1; f <= plan.n; f++) {
    rects.push(el("rect", { x: 80, y: y(f), width: 160, height: 4 }, col));
    if (plan.work.has(f)) el("rect", { x: 80, y: y(f), width: 160, height: 4, fill: "url(#hatch)", "pointer-events": "none" }, col);
  }
  const breaks = el("g", {}, col);
  const legend = el("g", {}, col), ly = y(plan.n) + 16;
  for (let i = 0; i < 20; i++) el("rect", { x: 80 + i * 8, y: ly, width: 8, height: 8, fill: viridis(i / 19) }, legend);
  const lo = el("text", { x: 80, y: ly + 20, class: "zl" }, legend), hi = el("text", { x: 240, y: ly + 20, "text-anchor": "end", class: "zl" }, legend);

  // Timeline: rebellions shaded, events marked, the cursor.
  const W = () => tl.clientWidth || 800, d0 = days[0], d1 = days[days.length - 1];
  const tx = (d) => ((d - d0) / Math.max(1, d1 - d0)) * W();
  function drawTimeline(day) {
    tl.innerHTML = "";
    for (const r of rebs) el("rect", { x: tx(r.start), y: 4, width: Math.max(1, tx(r.end ?? d1) - tx(r.start)), height: 20, fill: "var(--rebellion)", "fill-opacity": 0.35 }, tl);
    for (const e of marks) el("line", { x1: tx(e.day), x2: tx(e.day), y1: 2, y2: 26, stroke: e.type === "scripted" ? "#555" : "#000", "stroke-dasharray": e.type === "scripted" ? "3 2" : "" }, tl);
    el("line", { x1: tx(day), x2: tx(day), y1: 0, y2: 28, stroke: "#d00", "stroke-width": 2 }, tl);
  }

  let i = 0, timer = null;
  function draw() {
    const m = $("metric").value, day = days[i], v = data.get(m)[i], max = top.get(m);
    rects.forEach((r, k) => r.setAttribute("fill", viridis(v[k] / max)));
    lo.textContent = "0";
    hi.textContent = SHARES.has(m) ? "1" : max >= 10 ? max.toFixed(0) : max.toFixed(2);
    breaks.innerHTML = "";
    const broken = new Set();
    for (const [d, s, b] of changes) if (d <= day) b ? broken.add(s) : broken.delete(s);
    for (const s of broken) el("rect", { x: 78, y: y(s + 1) - 1.5, width: 164, height: 3, fill: "#d00" }, breaks);
    $("when").textContent = `day ${day} · measured year ${year(day)} · ${broken.size} segments broken`;
    $("slider").value = i;
    drawTimeline(day);
    const row = dl[day - first + 1]?.split(",");
    $("totals").innerHTML = row ? TOTALS.filter((c) => head.includes(c)).map((c) => `<tr><td>${c}</td><td class="num">${row[head.indexOf(c)]}</td></tr>`).join("") : "";
  }

  $("metric").onchange = draw;
  $("slider").oninput = (e) => { i = +e.target.value; draw(); };
  tl.onclick = (e) => { const d = d0 + (e.offsetX / W()) * (d1 - d0); i = days.findIndex((x) => x >= d); if (i < 0) i = days.length - 1; draw(); };
  $("play").onclick = () => {
    if (timer) { clearInterval(timer); timer = null; $("play").textContent = "play"; return; }
    if (i >= days.length - 1) i = 0;
    $("play").textContent = "pause";
    timer = setInterval(() => {
      if (i >= days.length - 1 || !view.isConnected) { clearInterval(timer); timer = null; $("play").textContent = "play"; return; }
      i++; draw();
    }, 1000 / +$("fps").value);
  };
  $("fps").onchange = () => { if (timer) { $("play").click(); $("play").click(); } };
  col.onmousemove = (e) => {
    const f = Math.floor((e.offsetY - 10) / 4) + 1;
    if (f < 1 || f > plan.n) return;
    const m = $("metric").value;
    $("hover").textContent = `floor ${f} · ${plan.zone(f)[1]}${plan.work.has(f) ? " · work floor" : ""} · ${m} ${data.get(m)[i][f - 1].toFixed(3)}`;
  };
  addEventListener("hashchange", () => timer && clearInterval(timer), { once: true });
  draw();
}
