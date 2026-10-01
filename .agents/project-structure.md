# Project Structure

## Guideline

- Keep `main()` functions thin: parse flags/config, wire dependencies, then delegate to `internal/` packages.
- Code that should never be imported by other projects belongs under `internal/`.
- Only put code under `pkg/` if it's genuinely meant to be a public, importable API - don't use it as a dumping ground.
- Group files within a package by responsibility, not by type (avoid generic buckets like `utils.go` or `helpers.go` when a more specific name fits).
- Keep test files (`_test.go`) alongside the code they test, in the same package or a `_test` package for black-box tests.

<!-- Project-specific / Guideline -->

## Layout

```
├── .agents/        # shared AI agent instructions (coding conventions, PR/task checklists, project structure)
├── .github/        # GitHub-specific config (workflows, issue/PR templates)
├── .vscode/        # VS Code editor/workspace settings
├── api/            # OpenAPI/Swagger specs, JSON schemas, protocol definitions
├── assets/         # repository assets (images, logos, etc)
├── cmd/            # main packages (one subdirectory per binary)
├── common/         # shared types, helper methods for whole project (project-level stdlib)
├── config/         # global application configuration
├── deployments/    # deployment configs and templates (docker-compose, kubernetes/helm, terraform)
├── docs/           # design and user documents (beyond godoc)
├── examples/       # examples for applications and/or public libraries
├── init/           # system init (systemd, upstart, sysv) and supervisor (runit, supervisord) configs
├── internal/       # private application/library code, not importable by other modules
├── pkg/            # public library code intended for external use (optional)
├── scripts/        # build/install/analysis scripts (keeps root Makefile small)
├── test/           # additional external test apps and test data
├── tui/            # terminal UI components
├── vendor/         # application dependencies (created by `go mod vendor`)
├── web/            # web app components: static assets, server-side templates, SPAs
├── .editorconfig   # editor formatting rules
├── .gitignore      # git ignore patterns
├── AGENTS.md       # instructions for AI coding agents
├── CLAUDE.md       # instructions specific for Claude coding agents
├── CONTRIBUTING.md # contribution guidelines
├── Dockerfile      # container build definition
├── Makefile        # build/test/lint task automation
├── go.mod          # Go module definition
├── go.sum          # Go module checksums
├── main.go         # default application entrypoint
```

<!-- Project-specific / Layout -->

This repository is a go-ethereum (geth) fork for Viction — the actual layout differs from the generic template above. Agents MUST inspect the actual top-level directories (core protocol packages at root, `cmd/`, `internal/`, `build/`, `tests/`, `docs/`, `common/`, `contracts/`, `legacy/`) before assuming the generic layout applies. The generic template is a fallback only; project-specific layout wins on conflict.

Important packages:

| Package | Description |
|---------|-------------|
| `accounts` | wallet/account management |
| `accounts/abi` | contract ABI encoding/decoding |
| `accounts/keystore` | encrypted keystore wallet |
| `build` | build/CI scripts and packaging (`ci.go`, nsis, deb, maven) |
| `cmd/geth` | main client binary (others under `cmd/`: `clef`, `bootnode`, `evm`, `abigen`, `ethkey`, `faucet`, `puppeth`, `devp2p`, `rlpdump`, `p2psim`, `abidump`, `checkpoint-admin`, `utils`) |
| `common` | shared types (`Address`, `Hash`, ...) — project-level stdlib |
| `consensus/clique` | clique PoA consensus engine |
| `consensus/ethash` | ethash PoW consensus engine |
| `consensus/posv` | PoSV consensus engine (Viction's Proof-of-Stake Voting); masternode election, snapshot, block sign verification |
| `console` | JavaScript interactive console |
| `contracts` | Solidity contract sources for binding generation |
| `core` | state transition, transaction pool, block processing, genesis handling |
| `core/rawdb` | low-level DB access/serialization for chain data |
| `core/state` | state trie / account state management |
| `core/types` | block, tx, tx signature types |
| `core/vm` | EVM implementation |
| `crypto` | secp256k1, keccak256, signatures |
| `docs` | documentation beyond godoc |
| `eth` | full-node protocol implementation (handler, peers) |
| `eth/downloader` | full+light chain sync |
| `eth/fetcher` | announce-based block/tx propagation |
| `eth/filters` | event/log & block filters |
| `eth/gasprice` | gas price oracle |
| `eth/tracers` | tx tracer framework |
| `ethclient` | read-only client binding to the RPC API |
| `ethdb` | key-value database backends |
| `ethdb/leveldb` | LevelDB backend |
| `ethdb/memorydb` | in-memory DB (tests) |
| `ethstats` | stats reporting daemon |
| `event` | event subscription/dispatch helpers |
| `graphql` | GraphQL interface to the node API |
| `internal/ethapi` | JSON-RPC API layer (private) |
| `internal/jsre` | JS runtime abstraction (console/clef) |
| `internal/victionapi` | Viction-specific RPC APIs: validator, reward, penalty, blocksign, attest, randomize |
| `internal/web3ext` | web3.js RPC extensions for console |
| `legacy/lending` | legacy lending: order processor, lending state |
| `legacy/trading` | legacy trading: order processor, trading state, token |
| `les` | light Ethereum subprotocol (LES) server |
| `light` | light-client trie helpers |
| `log` | logging API |
| `metrics` | stats/metrics collection |
| `miner` | block mining/transaction packing loop |
| `mobile` | gomobile bindings for Android/iOS |
| `node` | node service wiring (P2P, RPC, databases, lifecycle) |
| `p2p` | devp2p networking protocol |
| `p2p/discover` | node discovery v4 protocol |
| `p2p/discv5` | node discovery v5 protocol |
| `p2p/enode` | node URI/enode identity |
| `p2p/rlpx` | rlpx secure transport |
| `params` | chain/network parameters and fork scheduling |
| `rlp` | RLP encoding/serialization |
| `rpc` | RPC client/server framework (JSON, IPC, WebSocket) |
| `signer` | transaction signing/attestation service core (clef) |
| `swarm` | legacy Swarm distributed storage service |
| `tests` | cross-implementation test data, fuzzers, solidity tests |
| `trie` | Merkle Patricia trie |

## Project-specific

<!-- Project-specific -->
