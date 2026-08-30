# arcron-keeper-go

Independent Go keeper bot for Arcron TestNet: it lists upkeep boxes on app [`769891898`](https://testnet.explorer.perawallet.app/application/769891898), decodes the 130-byte head, and calls `execute(uint64)` when last-round ≥ next and escrow covers the base fee.

This talks to a **first-party CorvidLabs demo**. The contract is **unaudited** and **frozen=0** (the creator can still replace the programs). TestNet only — any other genesis is refused. There is no MainNet path.

## Live proof

| Check | Result |
|---|---|
| app | `769891898` on TestNet |
| upkeep | box `19` decodes to target `769891902` (`Pulse`) — asserted in CI against live algod |
| execute txid + round | **not done** — a throwaway account was generated, but the public TestNet dispenser (`lora.algokit.io/testnet/fund`) requires a Google login this environment cannot complete. No txid is invented. |
| command | `go run ./cmd/keeper --dry-run` (signs nothing) |

`--dry-run` against live TestNet listed 11 due upkeeps, including pulse upkeep `82`, and skipped `81`.

## How to run

Go 1.24+. A TestNet mnemonic is required only to sign.

```bash
cp .env.example .env          # put KEEPER_MNEMONIC in .env (gitignored)
go test ./...
go run ./cmd/keeper --dry-run
go run ./cmd/keeper --once    # signs; needs funded TestNet account
```

Fund the throwaway from [Lora TestNet Fund](https://lora.algokit.io/testnet/fund) (login required). `--dry-run` prints due upkeeps and signs nothing. Default loops. Upkeep **81** is never executed.

## Listener (no key)

`scripts/listen.py` reads TestNet algod boxes on app `769891898`, decodes the 130-byte head, and writes `docs/due.json` (due = last-round ≥ next and balance ≥ fee; skip 81). It does **not** sign, has **no mnemonic**, and is **not** an execute.

Weekdays at 15:00, 18:00, and 22:00 UTC (9am / 12pm / 4pm America/Denver) `.github/workflows/listen.yml` runs it and commits `docs/due.json` if it changed. No secrets. Pages shows that list.

```bash
python3 scripts/listen.py
```

## Cost

Each `execute` pays **min fee + 2000 µALGO extra** (fee pooling for the inner app call and the keeper payment). Today that is 3000 µALGO out; the contract pays the caller the upkeep's base fee (typically 4000–10000 µALGO) only if the inner call succeeds. A rejected `execute` is discarded by algod and costs nothing.

## What is broken / incomplete

- Fee escalation (`fee_cap`) is ignored; due-ness uses the base fee only.
- No simulate/populate of extra foreign references. Targets that need accounts, assets, or apps beyond the target itself fail and back off (1, 2, 4, 8 intervals, cap 1286 rounds). Pulse does not need them.
- Upkeep 81 is skipped on purpose.
- Real `execute` from this repo is **not done** (see proof table).
- No indexer, no MainNet, not a Python port.

Pages: https://corvid-agent.github.io/arcron-keeper-go/
