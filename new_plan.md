# Liquidator — Architecture Plan

## Overview

Liquidator is a Go-based on-chain liquidation bot structured around a central orchestrator that coordinates several specialized components. Each component owns a clear domain; communication flows through channels and shared state rather than direct coupling.

---

## Package Structure

```
internal/
  connector/     — RPC, WS, fallback, rate limit
  market/        — cache + snapshots + MarketReader interface (fusion of cache + state)
  liquidate/     — simulation, dryRun, tx building + signing
  runner/        — orchestration, lifecycle
  utils/         — helpers, formatting, async logger
  testutil/      — Anvil helpers, mock clients (test only, never imported by prod)

pkg/
  config/        — Config per chain, Signer
  morpho/        — Morpho types, math, ABI
  api/           — Morpho GraphQL client
  cex/           — Coinbase WebSocket price feed
  swap/          — Uniswap helpers
```

---

## Internal Packages

### `runner` — Orchestrator

The top-level struct that holds the entire application together.

**Responsibilities**
- Owns and initializes all components
- Launches and manages goroutines for each subsystem
- Handles graceful shutdown and lifecycle signals

---

### `connector` — RPC Connection Manager

Maintains three concurrent connections to Ethereum RPC endpoints: a primary
HTTP client, an HTTP fallback, and a WebSocket client for log subscriptions.

**Responsibilities**
- Dials and holds primary HTTP (`mainRPC`), fallback HTTP (`quoteRPC`), and
  WebSocket (`wsRPC`) connections at startup — hard-panics if any dial fails,
  since connectivity is a non-negotiable requirement for the bot
- Dispatches batched `eth_call` requests through the primary client; on any
  transient error (`timeout`, `EOF`, `context deadline exceeded`) automatically
  retries the same batch on the fallback
- Enforces a rate limit of 300 req/min with a burst of 10; calls that exceed
  the budget return a typed `ErrRateLimited` sentinel wrappable with `errors.Is`
- Subscribes to Morpho position events over WebSocket (`Borrow`, `Repay`,
  `SupplyCollateral`, `Liquidate`, `AccrueInterest`) and fans log events out
  through an internal buffered channel exposed read-only as `LogsCh()`
- Detects subscription drops and dead WebSocket connections, re-dials
  automatically with a 2 s back-off loop, and increments `Reconnects` on each
  successful recovery — without blocking the primary HTTP path
- Maintains lock-free atomic counters (`EthCalls`, `Reconnects`, `Fallbacks`)
  snapshotted and emitted to a log channel every 10 minutes via `LogMetrics`

**Key types**
- `RPCClient` — minimal interface (`CallCtx`, `Subscribe`) over `w3.Client`;
  the seam used for all unit tests
- `ConnectorMetrics` — three `atomic.Uint64` counters; `Snapshot()` resets and
  returns all three atomically for periodic reporting
- `Connector` — holds the three clients behind a `sync.RWMutex`; only the WS
  pointer is ever swapped post-construction (during reconnect)

**Testing**
- Unit tests via `mockRPCClient` injected through `newWithClients`
- `Metrics.Fallbacks == 1` when primary returns retryable error
- `ErrRateLimited` returned when context is cancelled before `limiter.Wait`
- `Metrics.Reconnects == 1` after WS subscription drop
- `watchLogs` exits cleanly on context cancellation
- Integration tests in `connector_anvil_test.go` with build tag `integration`

---

### `market` — Market State Store _(fusion of `cache` + `state`)_

Central in-memory store for all tracked market data, plus the shared
`MarketReader` interface consumed by `liquidate`, `runner`, and others.

**Responsibilities**
- Declares the `MarketReader` interface — the seam between producers (onchain)
  and consumers (liquidate, runner)
- Stores per-market state: open positions, oracle prices, total borrow
  assets/shares, maximum swappable collateral amounts
- Provides atomic snapshots to avoid lock contention during reads
- Inserts and removes positions as events arrive
- Sorts positions by health factor proximity

**Key types**
- `MarketReader` interface — `GetSnapshot(id [32]byte) *MarketSnapshot`
- `MarketSnapshot` — point-in-time view of a market
- `MarketStats`, `OracleData` — sub-structs of the snapshot

**Testing**
- Insert then check ordering
- Remove then check ordering
- Insert duplicate key — assert no duplicates
- Snapshot immutability — mutate cache after snapshot, assert snapshot unchanged
- Concurrent insert/remove under `-race`

---

### `liquidate` — Transaction Builder & Sender

Executes liquidation transactions.

**Responsibilities**
- Exposes `EthCaller` interface — the seam between liquidate and connector,
  enabling full mock injection in unit tests
- `SimulateAndPreComputeTx` orchestrates three isolated steps:
  1. `computeLiquidationParams` — pure math, zero RPC, unit testable
  2. `encodeLiquidateCalldata` — ABI encode, testable in isolation
  3. `dryRun` — batched `eth_call` + `EstimateGas`, mockable via `EthCaller`
- Gas estimate from `dryRun` is forwarded directly into `TxParams.GasEstimate`
  — no redundant RPC call in `SendSignedTx`
- `SendSignedTx` fetches only nonce + gasPrice (two calls), builds a
  `DynamicFeeTx`, signs via `Config.Signer`, and submits
- `log()` helper wraps the logger channel with a non-blocking `select/default`
  — never blocks the liquidation path
- `Config config.Config` passed by value — read-only, no pointer needed
- `NewConsumer` accepts `EthCaller` (not `*connector.Connector`) — consistent
  with the interface already used by the struct

**Known issues to fix before Anvil tests**
- `seizeAssets` capped at `MaxUniSwappable` but `repayShares` not rescaled
  proportionally — will cause Morpho revert
- `ComputeMinOut` called with same price for both collateral and loan —
  two distinct oracle prices needed in the snapshot
- `GasFeeCap` computed from `eth_gasPrice` (legacy) instead of base fee —
  may cause tx rejection under congestion

**Testing**
- Unit: inject `mockEthCaller`, assert `dryRun` error → `IsLiquidable == false`
- Unit: `ComputeMinOut` table tests (pure math)
- Unit: `encodeLiquidateCalldata` round-trip
- Integration (Anvil fork): create undercollateralized position via
  `anvil_setStorageAt`, run full liquidate flow, assert position cleared

---

### `utils` — Math, Formatting & Async Logger

Shared utility functions with no external dependencies.

**Responsibilities**
- Fixed-point math helpers (WAD arithmetic, rounding)
- `WAD` and other unit constants
- Type conversion utilities (`*big.Int` ↔ `float64`, hex encoding)
- `FormatDecimals`, `FormatWAD` — Morpho-specific formatting
- `RunTicker` — context-aware periodic ticker (absorbs `logging` package)

---

### `testutil` — Test Infrastructure _(test only)_

Never imported by production code. Shared across all `_test.go` files.

**Responsibilities**
- `AnvilInstance` — wraps an Anvil process; obtained only via `StartAnvil` or
  `StartAnvilFork`, never constructed directly
  - Fields unexported; exposed via `RPCURL()`, `WSURL()`, `Port()` accessors
  - `waitReady` polls HTTP until the node responds — no fixed sleep
  - `stop` kills the process and reaps the zombie via `cmd.Wait`
  - Registered via `t.Cleanup` — no manual teardown needed
- `StartAnvil(t)` — blank chain, auto-mine 1s blocks
- `StartAnvilFork(t, forkURL, forkBlock)` — forked chain for integration tests
- `FundedAccounts` — the 10 deterministic Anvil pre-funded addresses
- `WaitForBlock(ctx, client, n)` — polls until n blocks are mined
- `mockRPCClient` — configurable in-memory RPC client for connector unit tests

**Build tag:** `//go:build integration` on all files in this package

---

## Architectural Decisions

| Decision | Rationale |
|---|---|
| `Config` passed by value | Read-only at runtime; pointer adds nil risk with no benefit |
| `EthCaller` interface in `liquidate` | Decouples from `*connector.Connector`; enables mock injection |
| `dryRun` gas reused in `SendSignedTx` | Avoids double RPC; gas validated by simulation is the gas sent |
| `log()` non-blocking | Logger channel full must never stall a liquidation |
| `market` = fusion of `cache` + `state` | `MarketReader` interface belongs with the types it describes |
| `testutil` independent of `internal/*` | Prevents import cycles; mocks depend on interfaces, not implementations |
| Build tag `integration` | Anvil tests excluded from `go test ./...`; run explicitly with `-tags integration` |
| `NewConsumer` accepts `EthCaller` | Constructor signature consistent with struct field type |

---

## Component Dependency Map

```
runner
├── connector     → RPC client (primary + fallback HTTP, WS)
├── market        → market state (snapshots, positions, MarketReader)
├── onchain       → eth_call batches + event parsing → updates market
├── liquidate     → simulation + tx build + sign + send
└── utils         → math helpers + logger (used by all)

pkg/
├── config        → Config per chain, Signer (consumed by liquidate, connector, runner)
├── morpho        → types, math, ABI (consumed by liquidate, market, onchain)
├── api           → GraphQL → Morpho API → bootstraps market positions
├── cex           → Coinbase WS price feed → runner volatility signal
└── swap          → Uniswap helpers (consumed by liquidate)

testutil          → Anvil + mocks (test only, never in prod graph)
```

---

## Open Questions

- [ ] `seizeAssets` cap — rescale `repayShares` proportionally or use a different capping strategy?
- [ ] `ComputeMinOut` — add `CollateralPrice` and `LoanPrice` as distinct fields in `MarketSnapshot`?
- [ ] `GasFeeCap` — switch to `eth_getBlockByNumber` base fee + tip multiplier for EIP-1559 correctness?
- [ ] Health factor calculation — `market` package or delegated to `morpho`?
- [ ] Cache snapshots — copy-on-write structs or explicit snapshot methods?
- [ ] Position sorting — trigger-based (on insert) or periodic re-sort?
- [ ] Retry/backoff strategy in `liquidate` for nonce-too-low and gas-too-low failures?