#!/usr/bin/env python3
"""Append one TestNet due sample into docs/history.json when last_round is new.

Reads docs/due.json. No key. Not an execute. Dedupes by round.
"""
from __future__ import annotations

import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DUE = ROOT / "docs" / "due.json"
HIST = ROOT / "docs" / "history.json"
KEEPER = 769891898


def main() -> int:
    if not DUE.is_file():
        print("missing docs/due.json", file=sys.stderr)
        return 1
    data = json.loads(DUE.read_text())
    due = data.get("due") or []
    skipped = data.get("skipped") or []
    rnd = data.get("last_round")
    if rnd is None:
        print("due.json missing last_round", file=sys.stderr)
        return 1
    sample = {
        "t": data.get("generated_at") or "",
        "network": "testnet",
        "keeper_app": int(data.get("app") or KEEPER),
        "round": int(rnd),
        "listed": int(data.get("listed") or 0),
        "due": int(data.get("due_count") if data.get("due_count") is not None else len(due)),
        "skipped": len(skipped),
        "escrow_due_micro": sum(int(u.get("balance") or 0) for u in due),
        "fee_due_micro": sum(int(u.get("fee") or 0) for u in due),
        "source": "due.json@listen",
    }
    history = []
    if HIST.is_file():
        try:
            history = json.loads(HIST.read_text())
        except json.JSONDecodeError:
            history = []
    if not isinstance(history, list):
        history = []
    if any(int(r.get("round") or -1) == sample["round"] for r in history if isinstance(r, dict)):
        print(f"history unchanged (round {sample['round']} already present)")
        return 0
    history.append(sample)
    history.sort(key=lambda r: int(r.get("round") or 0) if isinstance(r, dict) else 0)
    HIST.write_text(json.dumps(history, indent=2) + "\n")
    print(f"appended round {sample['round']} → {HIST} ({len(history)} samples)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
