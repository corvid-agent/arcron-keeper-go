#!/usr/bin/env python3
"""No-key Arcron listener.

Reads 130-byte upkeep heads from TestNet algod (app 769891898) and writes
docs/due.json. Does not sign. No mnemonic. Not an execute.
"""
from __future__ import annotations

import base64
import json
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

ALGOD = "https://testnet-api.algonode.cloud"
APP = 769891898
HEAD = 130
SKIPPED_ID = 81
TESTNET_GENESIS_ID = "testnet-v1.0"
TESTNET_GENESIS_HASH = "SGO1GKSzyE7IEPItTxCByw9x8FmnrCDexi9/cOUJOiI="
UA = {"User-Agent": "arcron-keeper-go-listener/1.0", "Accept": "application/json"}
ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "docs" / "due.json"


def get(path: str) -> dict:
    url = ALGOD + path
    req = urllib.request.Request(url, headers=UA)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        body = e.read()[:300] if e.fp else b""
        raise RuntimeError(f"GET {path} HTTP {e.code}: {body!r}") from e


def u64(raw: bytes, off: int) -> int:
    return int.from_bytes(raw[off : off + 8], "big")


def parse_box_key(name: bytes) -> int | None:
    if len(name) != 9 or name[0:1] != b"u":
        return None
    return int.from_bytes(name[1:9], "big")


def decode_head(uid: int, raw: bytes) -> dict:
    if len(raw) < HEAD:
        raise ValueError(f"upkeep {uid}: short box ({len(raw)} bytes, need {HEAD})")
    offset = int.from_bytes(raw[40:42], "big")
    if offset != HEAD:
        raise ValueError(f"upkeep {uid}: tail offset {offset}, want {HEAD}")
    return {
        "id": uid,
        "target": u64(raw, 32),
        "interval": u64(raw, 42),
        "next": u64(raw, 50),
        "fee": u64(raw, 58),
        "balance": u64(raw, 66),
        "times": u64(raw, 74),
    }


def list_box_names() -> list[str]:
    names: list[str] = []
    token = ""
    for _ in range(50):
        q = f"?next={urllib.parse.quote(token)}" if token else ""
        page = get(f"/v2/applications/{APP}/boxes{q}")
        for box in page.get("boxes") or []:
            names.append(box["name"])
        token = page.get("next-token") or ""
        if not token:
            return names
    raise RuntimeError("box listing pagination exceeded 50 pages")


def load_upkeeps() -> tuple[list[dict], list[str]]:
    listed: list[dict] = []
    errors: list[str] = []
    for name_b64 in list_box_names():
        name = base64.b64decode(name_b64)
        uid = parse_box_key(name)
        if uid is None:
            continue
        quoted = urllib.parse.quote(name_b64, safe="")
        box = get(f"/v2/applications/{APP}/box?name=b64:{quoted}")
        raw = base64.b64decode(box["value"])
        try:
            listed.append(decode_head(uid, raw))
        except ValueError as e:
            errors.append(str(e))
    listed.sort(key=lambda u: u["id"])
    return listed, errors


def is_due(u: dict, last_round: int) -> bool:
    return last_round >= u["next"] and u["balance"] >= u["fee"]


def main() -> int:
    versions = get("/versions")
    genesis_id = versions.get("genesis_id") or versions.get("genesis-id")
    genesis_hash = versions.get("genesis_hash_b64") or versions.get("genesis-hash-b64")
    if genesis_id != TESTNET_GENESIS_ID or genesis_hash != TESTNET_GENESIS_HASH:
        raise SystemExit(
            f"refusing non-TestNet genesis id={genesis_id!r} hash={genesis_hash!r}"
        )

    status = get("/v2/status")
    last_round = int(status["last-round"])
    listed, errors = load_upkeeps()
    for msg in errors:
        print(f"decode: {msg}", file=sys.stderr)

    skipped = [u["id"] for u in listed if u["id"] == SKIPPED_ID]
    due = [u for u in listed if u["id"] != SKIPPED_ID and is_due(u, last_round)]
    payload = {
        "generated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "algod": ALGOD,
        "app": APP,
        "genesis": genesis_id,
        "last_round": last_round,
        "listed": len(listed),
        "skipped": skipped,
        "due_count": len(due),
        "due": due,
    }
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(payload, indent=2) + "\n")
    print(
        f"wrote {OUT} last_round={last_round} listed={len(listed)} "
        f"due={len(due)} skipped={skipped}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
