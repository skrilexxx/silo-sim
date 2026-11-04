// View 4, experiment (docs/viewer.md, T16.4): batches side by side, every
// number from /api/outcomes, the one silosim report --outcomes prints.
// Kaplan–Meier curves by uPlot; intervals and endings in plain SVG.
const OI = ["#E69F00", "#56B4E9", "#009E73", "#F0E442", "#0072B2", "#D55E00", "#CC79A7"];
const END = { suppressed: "#0072B2", regimeChange: "#D55E00", split: "#CC79A7", collapse: "#000000", unfinished: "#BBBBBB" };
const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);
const rgba = (hex, a) => `rgba(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(",")},${a})`;
const pct = (x, d = 0) => (100 * x).toFixed(d);

export async function experiment(view, paths, api) {
  view.innerHTML = `<p>Loading outcomes of ${paths.length} batch${paths.length > 1 ? "es" : ""}…</p>`;
  const got = await Promise.allSettled(paths.map((p) => api(`api/outcomes?batch=${encodeURIComponent(p)}`)));
  const bs = [], failed = [];
  got.forEach((g, i) => g.status === "fulfilled" ? bs.push({ path: paths[i], o: g.value, color: OI[bs.length % OI.length] })
    : failed.push(`${esc(paths[i])}: ${esc(g.reason.message)}`));
  view.innerHTML = (failed.length ? `<p class="empty">Left out: ${failed.join("; ")}</p>` : "") + (bs.length ? `
    <p class="key">${bs.map((b) => `<i style="background:${b.color}"></i>${esc(b.path)}`).join("")}</p>
    <div class="panel"><h3>Survival to the first rebellion (Kaplan–Meier, Greenwood 95% shaded)</h3><div id="km0"></div></div>
    <div class="panel"><h3>Survival to collapse (Kaplan–Meier, Greenwood 95% shaded)</h3><div id="km1"></div></div>
    <div class="panel"><h3>Rebellions per century, 95%</h3>${forest(bs, (o) => o.perCentury, (v) => v.toFixed(2))}</div>
    <div class="panel"><h3>Share of measured time in a rebellion, 95%</h3>${forest(bs, (o) => o.inRebellion, (v) => pct(v, 1) + "%")}</div>
    <div class="panel"><h3>Endings of rebellions</h3>${endings(bs)}</div>
    <h2>Numbers</h2>${table(bs)}` : "");
  if (!bs.length) return;

  const plots = [0, 1].map((i) => km(view.querySelector("#km" + i), bs, i));
  const resize = () => plots.forEach((p) => p.setSize({ width: Math.max(320, p.root.parentNode.clientWidth), height: 220 }));
  addEventListener("resize", resize);
  addEventListener("hashchange", () => removeEventListener("resize", resize), { once: true });
}

// km draws curve i of every batch on one time axis: each curve is read as
// the step function it is, at the union of all batches' step times.
function km(host, bs, i) {
  const last = Math.max(...bs.map((b) => b.o.last));
  const xs = [...new Set([0, last, ...bs.flatMap((b) => b.o.curves[i].steps.map((s) => s.t))])].sort((a, b) => a - b);
  const data = [xs], series = [{ label: "year" }], bands = [];
  const stepped = uPlot.paths.stepped({ align: 1 });
  for (const b of bs) {
    const steps = b.o.curves[i].steps;
    const read = (k) => xs.map((x) => { let j = 0; while (j + 1 < steps.length && steps[j + 1].t <= x) j++; return steps[j][k]; });
    const n = data.length;
    data.push(read("s"), read("hi"), read("lo"));
    series.push({ label: b.path, stroke: b.color, width: 2, paths: stepped, points: { show: false } },
      { label: "hi", paths: stepped, points: { show: false } }, { label: "lo", paths: stepped, points: { show: false } });
    bands.push({ series: [n + 1, n + 2], fill: rgba(b.color, 0.15) });
  }
  return new uPlot({
    width: Math.max(320, host.clientWidth), height: 220, legend: { show: false }, bands,
    scales: { x: { time: false }, y: { range: [0, 1.02] } },
    axes: [{ label: "measured year" }, { label: "S(t)", size: 56 }],
    series,
  }, data, host);
}

// forest is a point per batch with its 95% interval, on an axis from 0.
function forest(bs, pick, fmt) {
  const W = 760, L = 200, R = 190, row = 24, top = 8, h = top + row * bs.length + 24;
  const hi = Math.max(...bs.map((b) => pick(b.o).hi)) || 1, x = (v) => L + (W - L - R) * v / hi;
  const ticks = [0, 0.25, 0.5, 0.75, 1].map((f) => f * hi);
  return `<svg class="chart" viewBox="0 0 ${W} ${h}" width="100%" style="max-width:${W}px">
    ${ticks.map((t) => `<line x1="${x(t)}" x2="${x(t)}" y1="${top}" y2="${h - 20}" stroke="#ddd"/>
      <text x="${x(t)}" y="${h - 6}" text-anchor="middle" class="zl">${fmt(t)}</text>`).join("")}
    ${bs.map((b, i) => {
      const v = pick(b.o), y = top + row * i + row / 2;
      return `<text x="${L - 8}" y="${y + 4}" text-anchor="end">${esc(b.path)}</text>
        <line x1="${x(v.lo)}" x2="${x(v.hi)}" y1="${y}" y2="${y}" stroke="${b.color}" stroke-width="2"/>
        <circle cx="${x(v.value)}" cy="${y}" r="4" fill="${b.color}"/>
        <text x="${W - R + 8}" y="${y + 4}" class="zl">${fmt(v.value)} (${fmt(v.lo)}–${fmt(v.hi)})</text>`;
    }).join("")}</svg>`;
}

// endings stacks each batch's shares of rebellion endings to 100%.
function endings(bs) {
  const W = 760, L = 200, R = 190, row = 24, top = 8, h = top + row * bs.length + 8;
  return `<p class="key">${Object.entries(END).map(([k, c]) => `<i style="background:${c}"></i>${k}`).join("")}</p>
    <svg class="chart" viewBox="0 0 ${W} ${h}" width="100%" style="max-width:${W}px">${bs.map((b, i) => {
      const y = top + row * i, n = b.o.rebellions;
      let at = L;
      const segs = n ? b.o.endings.map((e) => {
        const w = (W - L - R) * e.share, s = `<rect x="${at}" y="${y + 3}" width="${w}" height="${row - 6}" fill="${END[e.name]}"><title>${e.name}: ${e.count} (${pct(e.share)}%, ${pct(e.lo)}–${pct(e.hi)}%)</title></rect>`;
        at += w;
        return s;
      }).join("") : "";
      return `<text x="${L - 8}" y="${y + row / 2 + 4}" text-anchor="end">${esc(b.path)}</text>${segs}
        <text x="${W - R + 8}" y="${y + row / 2 + 4}" class="zl">${n ? `${n} rebellions` : "no rebellions"}</text>`;
    }).join("")}</svg>`;
}

// table gives every number, formatted as silosim report --outcomes prints it.
function table(bs) {
  const med = (c) => c.median == null ? "not reached" : `${c.median.toFixed(1)} years`;
  return `<div class="wrap"><table>
    <tr><th>batch</th><th class="num">runs</th><th class="num">rebellions</th><th>per century</th><th>in a rebellion</th>
      <th>collapses</th><th>median to first rebellion</th><th>median to collapse</th></tr>
    ${bs.map(({ path, o }) => `<tr><td>${esc(path)}</td>
      <td class="num">${o.runs}${o.leftOut?.length ? ` (+${o.leftOut.length} left out)` : ""}</td><td class="num">${o.rebellions}</td>
      <td>${o.perCentury.value.toFixed(2)} (${o.perCentury.lo.toFixed(2)}–${o.perCentury.hi.toFixed(2)})</td>
      <td>${pct(o.inRebellion.value, 1)}% (${pct(o.inRebellion.lo, 1)}–${pct(o.inRebellion.hi, 1)}%)</td>
      <td>${o.collapses.count} of ${o.runs} (${pct(o.collapses.lo)}–${pct(o.collapses.hi)}%): ${o.causes.map((c) => `${c.name} ${c.count}`).join(", ")}</td>
      <td>${med(o.curves[0])}</td><td>${med(o.curves[1])}</td></tr>`).join("")}
  </table></div>
  <div class="wrap"><table>
    <tr><th>batch</th>${bs[0].o.endings.map((e) => `<th>${e.name}</th>`).join("")}</tr>
    ${bs.map(({ path, o }) => `<tr><td>${esc(path)}</td>${o.endings.map((e) =>
      `<td>${e.count} (${pct(e.share)}%, ${pct(e.lo)}–${pct(e.hi)}%)</td>`).join("")}</tr>`).join("")}
  </table></div>
  ${bs.filter((b) => b.o.leftOut?.length).map(({ path, o }) => `<p class="empty">${esc(path)} left out: ${o.leftOut.map((l) => `seed ${l.seed} (${esc(l.why)})`).join("; ")}</p>`).join("")}`;
}
