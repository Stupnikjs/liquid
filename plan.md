# Liquidator — Architecture Plan

## Overview

Liquidator is a Go-based on-chain liquidation bot structured around a central orchestrator that coordinates several specialized components. Each component owns a clear domain; communication flows through channels and shared state rather than direct coupling.

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
- `Connector` — holds the three clients behind an `sync.RWMutex`; only the WS
  pointer is ever swapped post-construction (during reconnect)

**Testing**
- Inject two `mockRPCClient` implementations via `newWithClients`; make the
  primary return a retryable error and assert the fallback is called and
  `Metrics.Fallbacks == 1`
- Inject a context timeout shorter than the rate-limiter wait; assert the call
  returns `ErrRateLimited` and no goroutines leak (verify with `goleak`)
- Drop the WS connection mid-flight by closing the mock subscription error
  channel; assert `reconnectWS` is called, `dialWS` is invoked, and
  `Metrics.Reconnects == 1` within a bounded timeout
- Cancel the context while `watchLogs` is blocked on a live subscription;
  assert the loop exits and `Unsubscribe` is called exactly once

---

### `cache` — Market State Store

Central in-memory store for all tracked market data.

**Responsibilities**
- Stores per-market state:
  - Open positions
  - Oracle prices
  - Total borrow assets and shares
  - Maximum swappable collateral amounts
- Provides atomic snapshots to avoid lock contention during reads
- Inserts and removes positions as events arrive
- Sorts positions (e.g. by health factor proximity)

> **Note:** Health factor calculation may be delegated to `state` — to be decided.

**Testing**
 - Inserting pos then checking good ordering 
 - Removing pos then checking good ordering 
 - Insert a position that updates an existing one (same key), assert no duplicates
 - Snapshot test: mutate cache after snapshot, assert snapshot is not affected
 - Concurrent goroutines inserting/removing — run with -race flag
---

### `state` — Interfaces & Shared Types

Defines the contracts between components.

**Responsibilities**
- Declares the `MarketReader` interface
- Defines shared data types used across packages
- Provides structured logging helpers

**Testing**

---

### `logging` — Async Log Writer

Decoupled, non-blocking logger.

**Responsibilities**
- Receives log messages from goroutines and appends them to an in-memory slice
- Launches a background goroutine for periodic flushing
- Writes accumulated logs asynchronously to a log file at a configurable interval

**Testing**

---

### `onchain` — Chain Data Fetcher

Handles all read interactions with the blockchain.

**Responsibilities**
- Constructs and dispatches `eth_call` batches for targeted market queries
- Applies fetched data to update market state in `cache`
- Processes on-chain events and translates them into cache mutations

**Testing**
- Spin up anvil --fork-url $RPC_URL in TestMain
- Fire real eth_call batches, assert the parsed result matches known on-chain values for a real market (hardcode a known block + expected oracle price)
- Feed a real event log (captured from mainnet) into the event parser, assert it produces the correct cache mutation
- Test the "stale block" edge case: what happens when the fetcher gets a response from a block behind the cache's current head
---

### `liquidate` — Transaction Builder & Sender

Executes liquidation transactions.

**Responsibilities**
- Precomputes contract call parameters based on a selected position
- Signs and submits the liquidation transaction via the configured signer
- Handles gas estimation and submission retries

**Testing**
- Given a position struct with known values, assert the precomputed call params match expected calldata byte-for-byte
- Signer unit test: mock signer returns a known signature, assert it's embedded correctly in the tx envelope
- Gas estimation: mock the estimator, assert the multiplier/cap logic is applied correctly
- Deploy or use a forked Morpho deployment
- Create an artificially undercollateralized position by manipulating storage slots (anvil_setStorageAt)
- Call the full liquidate flow end-to-end, assert the tx lands and the position is cleared
- Test the retry logic: simulate a first submission failure (nonce too low, gas too low) and assert the second attempt succeeds
---

### `utils` — Math & Conversion Helpers

Shared utility functions with no external dependencies.

**Responsibilities**
- Provides fixed-point math helpers (WAD arithmetic, rounding)
- Exposes `WAD` and other unit constants as package-level variables
- Offers type conversion utilities (e.g. `*big.Int` ↔ `float64`, hex encoding)

**Testing**

---

## External Packages

### `api` — Morpho GraphQL Client

Fetches off-chain market and position data from the Morpho API.

**Responsibilities**
- Constructs and sends GraphQL queries to the Morpho API endpoint
- Parses and deserializes JSON responses into typed Go structs
- Provides a clean interface consumed by `cache` or `runner` at startup and refresh intervals

**Testing**

---

### `cex` — Coinbase Websocket

Listening to coinbase ws (for now) to get price insight without wasting rpc calls .

**Responsibilities**
- Launching and holding the websocket 
- Holding the Cexcache struct with price info about main coins 
- Calculating volatility params for runner to increase rpc load 

---

### `config` — Config

Listening to coinbase ws (for now) to get price insight without wasting rpc calls .

**Responsibilities**
- Launching and holding the websocket 
- Holding the Cexcache struct with price info about main coins 
- Calculating volatility params for runner to increase rpc load 

---

### `morpho` — morpho 

 .

**Responsibilities**
- 
-  
- 

---


### `swap` — Swap (Uniswap)

Listening to coinbase ws (for now) to get price insight without wasting rpc calls .

**Responsibilities**
- 
-  
- 

---

## Component Dependency Map

```
runner
├── connector   →  RPC client
├── cache       →  market state (reads via onchain, mutated by events)
├── state       →  interfaces + types (used by cache, onchain, liquidate)
├── logging     →  async sink (used by all)
├── onchain     →  eth_call + event parsing → updates cache
├── liquidate   →  tx build + sign + send
└── utils       →  math helpers (used by onchain, liquidate, cache)

api  (external)
└── GraphQL → Morpho API → bootstraps cache positions
```

---

## Open Questions

- [ ] Should health factor calculation live in `cache` or `state`?
- [ ] Should `cache` snapshots be copy-on-write structs or explicit snapshot methods?
- [ ] Define the retry/backoff strategy in `liquidate` for failed transactions.
- [ ] Clarify ownership of position sorting: trigger-based vs. periodic re-sort.