# arcron-keeper-go

Independent Go keeper bot for Arcron TestNet: it lists upkeep boxes on app [`769891898`](https://testnet.explorer.perawallet.app/application/769891898), decodes the 130-byte head, and calls `execute(uint64)` when last-round ≥ next and escrow covers the base fee.

This talks to a **first-party CorvidLabs demo**. The contract is **unaudited** and **frozen=0** (the creator can still replace the programs). TestNet only — any other genesis is refused. There is no MainNet path.

## Live proof

| Check | Result |
|---|---|
| app | `769891898` on TestNet |
| upkeep | box `19` decodes to target `769891902` (`Pulse`) — asserted in CI against live algod |
| execute txid + round | **not done** — a throwaway account was generated, but the public TestNet dispenser (`lora.algokit.io/testnet/fund`) requires a Google login this environment cannot complete. No txid is invented. |
| command | `go run ./cmd/decode --id 19` (JSON head; target `769891902`) |

`--dry-run` against live TestNet listed 11 due upkeeps, including pulse upkeep `82`, and skipped `81`.

## How to run

Go 1.24+. Decode and listen need no key. A TestNet mnemonic is required only to sign `execute`.

```bash
go test ./...
go run ./cmd/decode --id 19   # JSON of the 130-byte head; live box 19 → target 769891902
go run ./cmd/listen           # writes docs/due.json; skip 81; no key
go run ./cmd/keeper --dry-run # lists due upkeeps; signs nothing
```

Fund a throwaway from [Lora TestNet Fund](https://lora.algokit.io/testnet/fund) (login required) only if you intend to sign. `--dry-run` prints due upkeeps and signs nothing. Default loops. Upkeep **81** is never executed.

```bash
cp .env.example .env          # gitignored; only needed to sign
go run ./cmd/keeper --once    # signs; needs funded TestNet account
```

## Decode (no key)

`go run ./cmd/decode --id 19` fetches one box by id from TestNet algod (default app `769891898`), refuses any non-TestNet genesis, and prints the 130-byte head as JSON. Live box 19 has target `769891902` (Pulse). Flags: `--algod`, `--app-id`.

## Listener (no key)

`go run ./cmd/listen` lists every upkeep box on app `769891898`, skips 81, and writes `docs/due.json` (due = last-round ≥ next and balance ≥ fee). Shape: `last_round`, `due_count`, `due[]`, `skipped`. It does **not** sign, has **no mnemonic**, and is **not** an execute.

Weekdays at 15:00, 18:00, and 22:00 UTC (9am / 12pm / 4pm America/Denver) `.github/workflows/listen.yml` runs `go run ./cmd/listen` (Go 1.24) and commits `docs/due.json` if it changed. No secrets. Pages shows that list.

```bash
go run ./cmd/listen
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
