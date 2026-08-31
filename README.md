# arcron-keeper-go

Independent Go keeper bot for Arcron TestNet: it lists upkeep boxes on app [`769891898`](https://testnet.explorer.perawallet.app/application/769891898), decodes the 130-byte head, and calls `execute(uint64)` when last-round ≥ next and escrow covers the base fee.

This talks to a **first-party CorvidLabs demo**. The contract is **unaudited** and **frozen=0** (the creator can still replace the programs). TestNet only — any other genesis is refused. There is no MainNet path.

## Live proof

| Check | Result |
|---|---|
| app | `769891898` on TestNet |
| upkeep | box `19` decodes to target `769891902` (`Pulse`) — asserted in CI against live algod |
| plod | live upkeep `110` → target `770734249` (`Plod`) — asserted in CI against live algod; next `67054248` — **not due yet** |
| rain | hub `770130162` rain `3` is ONE, 224-byte RainRec with prize_locked — live CI. Pre-#213 (no enter-while-locked assert), immutable, not product rain ([#232](https://github.com/CorvidLabs/arcron/issues/232)) |
| execute txid + round | **not done** — a throwaway account was generated, but the public TestNet dispenser (`lora.algokit.io/testnet/fund`) requires a Google login this environment cannot complete. No txid is invented. |
| command | `go run ./cmd/decode --id 19` (Pulse `769891902`); `--id 110` (Plod `770734249`) |

`go run ./cmd/listen` (unsigned, no mnemonic) listed 29 boxes, due_count 0, skipped `81`. Did not poke `87`. `go run ./cmd/register` and `go run ./cmd/simulate` remain no-key dry-runs (register has no `--send`; simulate sends nothing).

## How to run

Go 1.24+. Decode and listen need no key. A TestNet mnemonic is required only to sign `execute`.

```bash
go test ./...
go run ./cmd/decode --id 19   # JSON of the 130-byte head; live box 19 → target 769891902
go run ./cmd/decode --id 110  # live plod box; target 770734249
go run ./cmd/listen           # writes docs/due.json; skip 81; no key
go run ./cmd/rain             # writes docs/rain.json from hub 770130162; no key
go run ./cmd/register         # prints unsigned register group; signs nothing; no --send
go run ./cmd/simulate         # no-key algod simulate of execute() on due boxes; skip 81; nothing sent
go run ./cmd/keeper --dry-run # lists due upkeeps; signs nothing
```

Fund a throwaway from [Lora TestNet Fund](https://lora.algokit.io/testnet/fund) (login required) only if you intend to sign. `--dry-run` prints due upkeeps and signs nothing. Default loops. Upkeep **81** is never executed.

```bash
cp .env.example .env          # gitignored; only needed to sign
go run ./cmd/keeper --once    # signs; needs funded TestNet account
```

## Decode (no key)

`go run ./cmd/decode --id 19` fetches one box by id from TestNet algod (default app `769891898`), refuses any non-TestNet genesis, and prints the 130-byte head as JSON. Live box 19 has target `769891902` (Pulse). Live box 110 has target `770734249` (Plod). Flags: `--algod`, `--app-id`.

## Listener (no key)

`go run ./cmd/listen` lists every upkeep box on app `769891898`, skips 81, and writes `docs/due.json` (due = last-round ≥ next and balance ≥ fee). Shape: `last_round`, `due_count`, `due[]`, `skipped`. It does **not** sign, has **no mnemonic**, and is **not** an execute.

Weekdays at 15:00, 18:00, and 22:00 UTC (9am / 12pm / 4pm America/Denver) `.github/workflows/listen.yml` runs `go run ./cmd/listen` (Go 1.24) and commits `docs/due.json` if it changed. No secrets. Pages shows that list.

```bash
go run ./cmd/listen
```

## Register dry-run (no key, never sends)

`go run ./cmd/register` builds the live TestNet register group and prints it as JSON. It does **not** sign, has **no mnemonic**, and has **no `--send`**.

The on-chain method is `register(pay,pay,uint64,byte[][],uint64,uint64,uint64,uint64,uint64,uint64)uint64`, so the group is **three** transactions:

1. payment — box MBR to the app account (`note=arcron:mbr`). Formula: `2500 + 400*(9+130) + 400*len(ARC-4 byte[][])`.
2. payment — escrow funding to the app account (`note=arcron:funding`). Distinct note so the two pays never share a txid.
3. app call — ABI `register`, box `u||next_upkeep_id` (9 bytes), default target Pulse `769891902` with `call_args = [tick()uint64 selector]`.

`next_upkeep_id` is read from keeper global state. Sender defaults to an ephemeral address whose secret is discarded (never printed). Pass `--sender <addr>` to fill a real address; still unsigned.

```bash
go run ./cmd/register
go run ./cmd/register --interval 100 --fee 10000 --funding 40000 --target 769891902
```

## Simulate execute (no key, never sends)

`go run ./cmd/simulate` lists due upkeeps, skips **81** (Vigil), and asks TestNet algod to evaluate `execute(uint64)` (`selector 5b49cc5c`) with empty signatures and unnamed-resource access. It does **not** broadcast. Sender is the upkeep creator (already on chain); no mnemonic is loaded. Live `Call` still refuses 87 (Rot).

```bash
go run ./cmd/simulate
go run ./cmd/simulate --upkeep 19
```

## Cost

Each `execute` pays **min fee + 2000 µALGO extra** (fee pooling for the inner app call and the keeper payment). Today that is 3000 µALGO out; the contract pays the caller the upkeep's base fee (typically 4000–10000 µALGO) only if the inner call succeeds. A rejected `execute` is discarded by algod and costs nothing.

## What is broken / incomplete

- Fee escalation (`fee_cap`) is ignored; due-ness uses the base fee only.
- Keeper `--once` still does not populate extra foreign references from simulate. `cmd/simulate` is a no-key probe only. Targets that need extra accounts/assets/apps fail and back off (1, 2, 4, 8 intervals, cap 1286 rounds). Pulse does not need them.
- Upkeep 81 is skipped on purpose. Live execute also refuses 87.
- Real `execute` from this repo is **not done** (see proof table).
- `cmd/register` is dry-run only. A live register still needs a funded TestNet account.
- Execute of upkeep 87 (Rot) is refused the same way as 81.
- No indexer, no MainNet, not a Python port.

Pages: https://corvid-agent.github.io/arcron-keeper-go/
