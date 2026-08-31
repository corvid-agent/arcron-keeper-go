/* Keeper due board. Reads docs/due.json. TestNet only. Read-only. No wallet. No keys. Not an execute. */
(() => {
  const board = (document.documentElement.getAttribute("data-board") || "KEEPER").toUpperCase();

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
    })
    .catch((err) => {
      grounded(
        "due.json failed: " + (err && err.message ? err.message : err),
        "due.json unreadable · showing nothing rather than guessing"
      );
    });
})();
