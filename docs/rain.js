/* Rain hub board. Reads docs/rain.json. TestNet only. Read-only. No wallet. No keys. Not a send. */
(() => {
  function flaps(el, text) {
    el.replaceChildren();
    for (const ch of String(text)) {
      const d = document.createElement("span");
      d.className = "flap" + (ch === " " ? " blank" : "");
      d.textContent = ch === " " ? "\u00a0" : ch;
      el.appendChild(d);
    }
  }

  function renderRain(rains) {
    const host = document.getElementById("rain-rows");
    if (!host) return;
    host.replaceChildren();
    if (!rains.length) {
      const p = document.createElement("p");
      p.className = "empty";
      p.textContent = "no rains";
      host.appendChild(p);
      return;
    }
    const fields = [
      ["id", "id"],
      ["label", "label"],
      ["mode", "mode"],
      ["status", "status"],
      ["locked", "prize_locked"],
    ];
    for (const r of rains) {
      const flight = document.createElement("article");
      flight.className = "flight";
      if (r.status === "abandonable") flight.classList.add("late");
      for (const [label, key] of fields) {
        const cell = document.createElement("div");
        cell.className = "cell";
        const lab = document.createElement("div");
        lab.className = "label";
        lab.textContent = label;
        const row = document.createElement("div");
        row.className = "flaps compact";
        const val = r[key];
        flaps(row, val == null ? "—" : String(val));
        cell.append(lab, row);
        flight.appendChild(cell);
      }
      host.appendChild(flight);
    }
  }

  const host = document.getElementById("rain-rows");
  if (!host) return;
  fetch("./rain.json", { cache: "no-store" })
    .then((res) => {
      if (!res.ok) throw new Error("rain.json " + res.status);
      return res.json();
    })
    .then((data) => {
      const rains = Array.isArray(data.rains) ? data.rains : [];
      renderRain(rains);
    })
    .catch((err) => {
      const p = document.createElement("p");
      p.className = "empty";
      p.textContent = "rain.json unreadable · showing nothing rather than guessing";
      host.replaceChildren(p);
    });
})();
