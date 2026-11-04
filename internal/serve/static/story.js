// The Story tab (docs/narrator.md, T17.1): a run's chronicle, rendered
// from the Markdown the server sends.
const esc = (s) => s.replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[c]);

// inline: **bold** and *italics*, after escaping.
const inline = (s) => esc(s).replace(/\*\*(.+?)\*\*/g, "<b>$1</b>").replace(/\*(.+?)\*/g, "<i>$1</i>");

// markdown renders headings, lists and paragraphs, nothing more.
export function markdown(text) {
  const out = [];
  let list = false, para = [];
  const endPara = () => { if (para.length) out.push(`<p>${inline(para.join(" "))}</p>`); para = []; };
  const endList = () => { if (list) out.push("</ul>"); list = false; };
  for (const line of text.split("\n")) {
    const h = line.match(/^(#{1,3}) (.*)/), li = line.match(/^- (.*)/);
    if (h) { endPara(); endList(); out.push(`<h${h[1].length + 1}>${inline(h[2])}</h${h[1].length + 1}>`); }
    else if (li) { endPara(); if (!list) out.push("<ul>"); list = true; out.push(`<li>${inline(li[1])}</li>`); }
    else if (!line.trim()) { endPara(); endList(); }
    else { endList(); para.push(line.trim()); }
  }
  endPara(); endList();
  return out.join("\n");
}

export async function story(view, path) {
  const r = await fetch(`api/chronicle?run=${encodeURIComponent(path)}`);
  if (!r.ok) throw new Error(`chronicle: ${r.status} ${await r.text()}`);
  view.innerHTML = `<h2>Chronicle</h2><article class="doc">${markdown(await r.text())}</article>`;
}
