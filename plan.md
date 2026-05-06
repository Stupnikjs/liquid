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

Maintains a stable connection to Ethereum RPC endpoints.

**Responsibilities**
- Establishes connections to RPC providers
- Detects and recovers from dropped connections
- Exposes a stable client handle to the rest of the system


**Testing**
 - Force deconnection and swich on ClientHTTPFallback
 - Timeout

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

---

### `liquidate` — Transaction Builder & Sender

Executes liquidation transactions.

**Responsibilities**
- Precomputes contract call parameters based on a selected position
- Signs and submits the liquidation transaction via the configured signer
- Handles gas estimation and submission retries

**Testing**

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