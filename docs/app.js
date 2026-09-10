/* Keeper due board. Reads docs/due.json + history.json. TestNet only. Read-only. No wallet. No keys. Not an execute. */
(() => {
  const board = (document.documentElement.getAttribute("data-board") || "KEEPER").toUpperCase();
  const KEEPER = 769891898;
  const PHOS = "#7cff6b";
  const AMBER = "#e6c15a";
  const HOT = "#ff5a4a";
  const DIM = "#3a8a32";

  function flaps(el, text) {
    el.replaceChildren();
    for (const ch of String(text)) {
      const d = document.createElement("span");
      d.className = "flap" + (ch === " " ? " blank" : "");
      d.textContent = ch === " " ? "\u00a0" : ch;
      el.appendChild(d);
    }
  }

  function setStatus(word, cls, sub) {
    const el = document.getElementById("status");
    el.className = "flaps big " + cls;
    flaps(el, String(word).toUpperCase());
    document.getElementById("subhead").textContent = sub;
  }

  function skippedLabel(data) {
    const ids = data.skipped || data.skip_upkeep_ids || [81];
    if (!ids.length) return "—";
    return ids.join(" ");
  }

  function fillStats(lastRound, listed, dueCount, skipped) {
    flaps(document.getElementById("stat-round"), lastRound);
    flaps(document.getElementById("stat-listed"), listed);
    flaps(document.getElementById("stat-due"), dueCount);
    flaps(document.getElementById("stat-skip"), skipped);
  }

  function renderDue(due) {
    const host = document.getElementById("due-rows");
    host.replaceChildren();
    if (!due.length) {
      const p = document.createElement("p");
      p.className = "empty";
      p.textContent = "none due";
      host.appendChild(p);
      return;
    }
    const fields = [
      ["id", "id"],
      ["target", "target"],
      ["next", "next"],
      ["fee", "fee"],
      ["balance", "balance"],
    ];
    for (const u of due) {
      const flight = document.createElement("article");
      flight.className = "flight";
      for (const [label, key] of fields) {
        const cell = document.createElement("div");
        cell.className = "cell";
        const lab = document.createElement("div");
        lab.className = "label";
        lab.textContent = label;
        const row = document.createElement("div");
        row.className = "flaps compact";
        const val = u[key];
        flaps(row, val == null ? "—" : String(val));
        cell.append(lab, row);
        flight.appendChild(cell);
      }
      host.appendChild(flight);
    }
  }

  function grounded(msg, sub) {
    setStatus("FEED DOWN", "down", sub);
    document.title = board + " — FEED DOWN";
    fillStats("—", "—", "—", "—");
    const err = document.getElementById("err");
    err.hidden = false;
    err.textContent = msg;
    renderDue([]);
  }

  function sizeCanvas(c) {
    const ratio = window.devicePixelRatio || 1;
    const w = c.width;
    const h = c.height;
    c.style.width = w + "px";
    c.style.height = h + "px";
    c.width = Math.floor(w * ratio);
    c.height = Math.floor(h * ratio);
    const ctx = c.getContext("2d");
    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    return { ctx, w, h };
  }

  function drawSeries(ctx, w, h, series, color, max) {
    if (!series.length) return;
    const pad = 8;
    const innerW = w - pad * 2;
    const innerH = h - pad * 2;
    ctx.beginPath();
    series.forEach((v, i) => {
      const x = pad + (series.length === 1 ? innerW / 2 : (i / (series.length - 1)) * innerW);
      const y = pad + innerH - (Math.max(0, Number(v) || 0) / max) * innerH;
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.5;
    ctx.shadowColor = color;
    ctx.shadowBlur = 6;
    ctx.stroke();
    ctx.shadowBlur = 0;
  }

  function drawDue(rows) {
    const c = document.getElementById("due-canvas");
    const meta = document.getElementById("due-meta");
    if (!c) return;
    const { ctx, w, h } = sizeCanvas(c);
    ctx.clearRect(0, 0, w, h);
    if (meta) meta.textContent = "sqlite " + rows.length + " samples · TestNet";
    if (!rows.length) {
      ctx.fillStyle = DIM;
      ctx.font = "12px IBM Plex Mono, monospace";
      ctx.fillText("no history yet", 12, 28);
      return;
    }
    const due = rows.map((r) => Number(r.due || 0));
    const max = Math.max(...due, 1);
    drawSeries(ctx, w, h, due, PHOS, max);
    const last = rows[rows.length - 1];
    if (meta) {
      meta.textContent =
        "due " + last.due + " · round " + last.round + " · " + rows.length + " samples";
    }
  }

  function drawEscrow(rows) {
    const c = document.getElementById("escrow-canvas");
    const meta = document.getElementById("escrow-meta");
    if (!c) return;
    const { ctx, w, h } = sizeCanvas(c);
    ctx.clearRect(0, 0, w, h);
    if (!rows.length) {
      ctx.fillStyle = DIM;
      ctx.font = "12px IBM Plex Mono, monospace";
      ctx.fillText("no history yet", 12, 28);
      return;
    }
    const esc = rows.map((r) => Number(r.escrow_due_micro || 0));
    const fee = rows.map((r) => Number(r.fee_due_micro || 0));
    const max = Math.max(...esc, ...fee, 1);
    drawSeries(ctx, w, h, esc, PHOS, max);
    drawSeries(ctx, w, h, fee, AMBER, max);
    const last = rows[rows.length - 1];
    if (meta) {
      meta.textContent =
        "escrow " +
        Number(last.escrow_due_micro || 0).toLocaleString() +
        " µ · fee " +
        Number(last.fee_due_micro || 0).toLocaleString() +
        " µ";
    }
  }

  function drawListed(rows) {
    const c = document.getElementById("listed-canvas");
    const meta = document.getElementById("listed-meta");
    if (!c) return;
    const { ctx, w, h } = sizeCanvas(c);
    ctx.clearRect(0, 0, w, h);
    if (!rows.length) {
      ctx.fillStyle = DIM;
      ctx.font = "12px IBM Plex Mono, monospace";
      ctx.fillText("no history yet", 12, 28);
      return;
    }
    const listed = rows.map((r) => Number(r.listed || 0));
    const skipped = rows.map((r) => Number(r.skipped || 0));
    const max = Math.max(...listed, ...skipped, 1);
    drawSeries(ctx, w, h, listed, PHOS, max);
    drawSeries(ctx, w, h, skipped, HOT, max);
    const last = rows[rows.length - 1];
    if (meta) {
      meta.textContent =
        "listed " + last.listed + " · skipped " + last.skipped + " · skip 81";
    }
  }

  let sqlDb = null;

  async function bootSql(rows) {
    if (typeof initSqlJs !== "function") return rows;
    const SQL = await initSqlJs({
      locateFile: (f) => "https://cdnjs.cloudflare.com/ajax/libs/sql.js/1.11.0/" + f,
    });
    sqlDb = new SQL.Database();
    sqlDb.run(
      "CREATE TABLE samples (t TEXT, network TEXT, keeper_app INTEGER, round INTEGER, listed INTEGER, due INTEGER, skipped INTEGER, escrow_due_micro INTEGER, fee_due_micro INTEGER, source TEXT)"
    );
    const ins = sqlDb.prepare("INSERT INTO samples VALUES (?,?,?,?,?,?,?,?,?,?)");
    rows.forEach((r) => {
      ins.run([
        r.t || "",
        r.network || "testnet",
        Number(r.keeper_app || KEEPER),
        Number(r.round || 0),
        Number(r.listed || 0),
        Number(r.due || 0),
        Number(r.skipped || 0),
        Number(r.escrow_due_micro || 0),
        Number(r.fee_due_micro || 0),
        r.source || "",
      ]);
    });
    ins.free();
    const res = sqlDb.exec(
      "SELECT t, network, keeper_app, round, listed, due, skipped, escrow_due_micro, fee_due_micro, source FROM samples WHERE network='testnet' AND keeper_app=" +
        KEEPER +
        " ORDER BY round ASC"
    );
    if (!res.length) return rows;
    const cols = res[0].columns;
    return res[0].values.map((v) => {
      const o = {};
      cols.forEach((c, i) => {
        o[c] = v[i];
      });
      return o;
    });
  }

  function sampleFromDue(data) {
    const due = Array.isArray(data.due) ? data.due : [];
    const skipped = data.skipped || [];
    return {
      t: data.generated_at || new Date().toISOString(),
      network: "testnet",
      keeper_app: Number(data.app || KEEPER),
      round: Number(data.last_round || 0),
      listed: Number(data.listed || 0),
      due: data.due_count == null ? due.length : Number(data.due_count),
      skipped: skipped.length,
      escrow_due_micro: due.reduce((a, u) => a + Number(u.balance || 0), 0),
      fee_due_micro: due.reduce((a, u) => a + Number(u.fee || 0), 0),
      source: "due.json@live",
    };
  }

  async function loadHistoryGraphs(liveSample) {
    let history = [];
    try {
      const res = await fetch("./history.json", { cache: "no-store" });
      if (res.ok) history = await res.json();
    } catch (_) {
      history = [];
    }
    if (!Array.isArray(history)) history = [];
    history = history.filter(
      (r) =>
        r &&
        r.network === "testnet" &&
        Number(r.keeper_app) === KEEPER
    );
    if (liveSample && liveSample.round) {
      const exists = history.some((r) => Number(r.round) === Number(liveSample.round));
      if (!exists) history = history.concat([liveSample]);
    }
    let rows = history;
    try {
      rows = await bootSql(history);
    } catch (_) {
      rows = history;
    }
    drawDue(rows);
    drawEscrow(rows);
    drawListed(rows);
  }

  fillStats("—", "—", "—", "81");
  flaps(document.getElementById("status"), "DUE");
  document.title = board + " — DUE, skip 81";

  fetch("./due.json", { cache: "no-store" })
    .then((res) => {
      if (!res.ok) throw new Error("due.json " + res.status);
      return res.json();
    })
    .then((data) => {
      const due = Array.isArray(data.due) ? data.due : [];
      const n = data.due_count == null ? due.length : Number(data.due_count);
      const listed = data.listed == null ? "—" : String(data.listed);
      const last = data.last_round == null ? "—" : String(data.last_round);
      const skipped = skippedLabel(data);
      fillStats(last, listed, String(n), skipped);
      renderDue(due);
      const cls = n > 0 ? "" : "grounded";
      setStatus("DUE " + n, cls, "skip 81 · listed " + listed + " · not an execute");
      document.title = board + " — DUE " + n + ", skip 81";
      const live = sampleFromDue(data);
      loadHistoryGraphs(live).catch((err) => {
        console.warn("history graphs failed", err);
      });
    })
    .catch((err) => {
      grounded(
        "due.json failed: " + (err && err.message ? err.message : err),
        "due.json unreadable · showing nothing rather than guessing"
      );
      loadHistoryGraphs(null).catch(() => {});
    });
})();
