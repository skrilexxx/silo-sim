// View 2, run dashboard (docs/viewer.md, T16.3): stacked time series on
// one time axis in measured years, cursor and zoom shared; rebellions
// shaded; scripted events, collapses and the Safeguard marked, generator
// failures and policy starts on a toggle. Charts by uPlot (vendored).
const Z = { upTop: "#0072B2", mids: "#009E73", downDeep: "#D55E00" };
const OI = ["#E69F00", "#56B4E9", "#009E73", "#F0E442", "#0072B2", "#D55E00", "#CC79A7"];

const lines = (t) => t.split("\n").filter((l) => l.length);

// csv: a CSV file as {name: Float64Array} by column.
function csv(text) {
  const ls = lines(text), head = ls[0].split(","), cols = head.map(() => new Float64Array(ls.length - 1));
  for (let r = 1; r < ls.length; r++) {
    const v = ls[r].split(",");
    for (let c = 0; c < head.length; c++) cols[c][r - 1] = v[c] === "" ? NaN : +v[c];
  }
  return Object.fromEntries(head.map((h, c) => [h, cols[c]]));
}

// marks: a uPlot plugin drawing rebellion bands and event lines under the series.
function marks(bands, events) {
  return {
    hooks: {
      drawClear: (u) => {
        const { ctx } = u, { top, height } = u.bbox, x = (v) => u.valToPos(v, "x", true);
        ctx.save();
        ctx.fillStyle = "rgba(204,121,167,0.15)";
        for (const [a, b] of bands) ctx.fillRect(x(a), top, Math.max(1, x(b) - x(a)), height);
        for (const e of events()) {
          ctx.strokeStyle = e.color;
          ctx.setLineDash(e.dash ? [4, 3] : []);
          ctx.beginPath();
          ctx.moveTo(x(e.t), top);
          ctx.lineTo(x(e.t), top + height);
          ctx.stroke();
        }
        ctx.restore();
      },
    },
  };
}

export async function dashboard(view, path, file) {
  const [info, daily, yearly, rebellions, events] = await Promise.all([
    file(path, "run.json").then(JSON.parse), file(path, "daily.csv"), file(path, "yearly.csv", ""),
    file(path, "rebellions.jsonl", ""), file(path, "events.jsonl", ""),
  ]);
  const warmup = info.scenario.warmupYears * 365, yr = (d) => (d - warmup) / 365;
  const d = csv(daily), y = yearly ? csv(yearly) : null;
  const xd = d.day.map(yr), last = xd[xd.length - 1];
  const bands = lines(rebellions).map(JSON.parse).map((r) => [yr(r.start), r.end == null ? last : yr(r.end)]);
  const evs = lines(events).map(JSON.parse);
  const show = { failure: false, policy: false };
  const always = evs.filter((e) => ["scripted", "collapse", "extinction", "safeguard"].includes(e.type))
    .map((e) => ({ t: yr(e.day), color: e.type === "scripted" ? "#555" : "#000", dash: e.type === "scripted" }));
  const optional = evs.filter((e) => e.type === "failure" || e.type === "policy")
    .map((e) => ({ type: e.type, t: yr(e.day), color: e.type === "failure" ? "#E69F00" : "#56B4E9", dash: true }));
  const shown = () => always.concat(optional.filter((e) => show[e.type]));

  const out = info.outcomes ?? {};
  view.innerHTML = `
    <p>${info.scenario.name}, seed ${info.seed}: ${(xd.length / 365).toFixed(0)} measured years;
      rebellions ${out.rebellions ?? "—"}; collapse ${out.collapseCause ? `${out.collapseCause} in measured year ${yr(out.collapseDay).toFixed(1)}` : "none"}.
      Drag to zoom, double-click to reset.</p>
    <div class="controls"><label><input type="checkbox" id="failure"> generator failures</label>
      <label><input type="checkbox" id="policy"> policy starts</label>
      <span class="key"><i style="background:rgba(204,121,167,0.35)"></i> rebellion <i class="dash"></i> scripted event <i class="solid"></i> collapse</span></div>
    <div id="panels"></div>`;
  const host = view.querySelector("#panels");
  const width = () => Math.max(320, host.clientWidth);
  const plots = [];
  let syncing = false;

  function panel(title, x, series, axes = {}) {
    series = series.filter((s) => s.data);
    if (!series.length) return;
    const box = document.createElement("div");
    box.className = "panel";
    box.innerHTML = `<h3>${title}</h3>`;
    host.appendChild(box);
    const u = new uPlot({
      width: width(), height: 180,
      cursor: { sync: { key: "silo" }, drag: { x: true, y: false } },
      scales: { x: { time: false }, ...axes.scales },
      // Every panel has a right axis of the same size, blank where unused, so the years line up.
      axes: [{ label: "measured year" }, { size: 56 }, axes.right ? { scale: axes.right, side: 1, grid: { show: false }, size: 56 }
        : { side: 1, size: 56, grid: { show: false }, ticks: { show: false }, values: () => [] }],
      series: [{ label: "year", value: (u, v) => v == null ? "—" : v.toFixed(2) },
        ...series.map((s) => ({ label: s.label, stroke: s.color, width: s.width ?? 1, dash: s.dash, scale: s.scale ?? "y",
          value: (u, v) => v == null || isNaN(v) ? "—" : (+v).toFixed(s.digits ?? 3) }))],
      plugins: [marks(bands, shown)],
      hooks: { setScale: [(u, key) => {
        if (key !== "x" || syncing) return;
        syncing = true;
        for (const p of plots) if (p !== u) p.setScale("x", { min: u.scales.x.min, max: u.scales.x.max });
        syncing = false;
      }] },
    }, [x, ...series.map((s) => s.data)], box);
    plots.push(u);
  }

  panel("Population", xd, [
    { label: "total", data: d.population, color: "#000", width: 1.5, digits: 0 },
    { label: "Up Top", data: d.popUpTop, color: Z.upTop, digits: 0 },
    { label: "Mids", data: d.popMids, color: Z.mids, digits: 0 },
    { label: "Down Deep", data: d.popDownDeep, color: Z.downDeep, digits: 0 }]);
  panel("Life support", xd, [
    { label: "power factor", data: d.powerFactor, color: OI[0] },
    { label: "water", data: d.water, color: OI[1] },
    { label: "air", data: d.air, color: OI[2] },
    { label: "food stock (days)", data: d.foodStockDays, color: OI[6], scale: "days", digits: 1 }],
  { scales: { y: { range: [0, 1.05] } }, right: "days" });
  panel("Health (solid) and hardship (dashed) by zone", xd, [
    { label: "health Up Top", data: d.healthUpTop, color: Z.upTop },
    { label: "health Mids", data: d.healthMids, color: Z.mids },
    { label: "health Down Deep", data: d.healthDownDeep, color: Z.downDeep },
    { label: "hardship Up Top", data: d.hardshipUpTop, color: Z.upTop, dash: [4, 3] },
    { label: "hardship Mids", data: d.hardshipMids, color: Z.mids, dash: [4, 3] },
    { label: "hardship Down Deep", data: d.hardshipDownDeep, color: Z.downDeep, dash: [4, 3] }]);
  panel("Dissent (adults)", xd, [
    { label: "active", data: d.active, color: OI[0], digits: 0 },
    { label: "joined", data: d.joined, color: OI[5], digits: 0 },
    { label: "violent", data: d.violent, color: "#000", digits: 0 },
    { label: "on strike", data: d.onStrike, color: OI[4], digits: 0 }]);
  if (y) {
    const xy = y.day.map(yr);
    panel("Legitimacy by zone and knowledge (yearly)", xy, [
      { label: "mean L Up Top", data: y.meanLUpTop, color: Z.upTop, width: 1.5 },
      { label: "mean L Mids", data: y.meanLMids, color: Z.mids, width: 1.5 },
      { label: "mean L Down Deep", data: y.meanLDownDeep, color: Z.downDeep, width: 1.5 },
      ...["Outside", "Screen", "History", "Silos", "Lottery"].map((f, i) => ({ label: `know ${f.toLowerCase()}`, data: y["know" + f], color: OI[[0, 1, 6, 3, 4][i]], dash: [4, 3] }))],
    { scales: { y: { range: [0, 1] } } });
  }

  for (const t of ["failure", "policy"]) view.querySelector("#" + t).onchange = (e) => { show[t] = e.target.checked; plots.forEach((p) => p.redraw()); };
  const resize = () => plots.forEach((p) => p.setSize({ width: width(), height: 180 }));
  addEventListener("resize", resize);
  addEventListener("hashchange", () => removeEventListener("resize", resize), { once: true });
}
