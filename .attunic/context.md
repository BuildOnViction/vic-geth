# Project Context

Part of the project memory layer: human-readable memory of what this project is and what has been confirmed.

The reading and writing rules for this file and the rest of `.attunic/` are defined in [attunic-protocol.md](../.agents/attunic-protocol.md).

## Abstract

A newcomer should be able to tell what the software does without knowing what it is built with. One paragraph that describes the domain, not the technology, in simple terms:
<!-- Project-specific / Abstract -->

This is fork of go-ethereum for Viction Network. It includes the PoSV (proof of state voting) consensus engine and other support libraries to make it compatible with Viction. This implementation will be used for running full node, archive node and validator in the network.

## Technical Specifications

A brief overview of the ingredients used for developing this project. These are things developers should know when developing the project.

Typical ingredients include, but are not limited to: primary language, frameworks, architecture styles, environment prerequisites, libraries, test runners, tools, databases...
<!-- Project-specific / Technical Specifications -->

- Primary language: Go, plus C for the ECDSA/signature packages.
- Base: fork of go-ethereum.
- Build system: go.
- Test runner: go.
- Linter: gofmt.
- Consensus engine: Proof-of-Stake-Voting (PoSV).
- Storage: LevelDB.
- API: JSON-RPC over HTTP/WebSocket/IPC (`rpc`), `ethclient`, `graphql`.

## Layout

Layout of this project; [project-structure.md](../.agents/project-structure.md) holds the language's standard map, kept for reference.

List of important folders, files and their roles in the project:

- They visualize the actual file system where source files live.
- If a top-level file/folder is missing from the list, prompt the user to update it.

```
├── .agents/            # project AI agent instructions (attunic protocol, coding-conventions, development-guideline, pull-request-guideline)
├── .attunic/           # project AI agent memory (context, memory, wip)
├── .github/            # workflows, issue/PR templates
├── accounts/           # key management and account abstractions (keystore, HD wallets)
├── build/              # CI/build driver (`ci.go` behind the Makefile), lint tooling, `build/bin` for output binaries
├── cmd/                # main packages, one subdir per binary
│   ├── abidump/        # ABI dump utility
│   ├── abigen/         # contract binding generator
│   ├── bootnode/       # discovery-protocol-only node
│   ├── checkpoint-admin/ # checkpoint management CLI
│   ├── clef/           # standalone signer tool
│   ├── devp2p/         # devp2p debugging utilities
│   ├── ethkey/         # keystore key management CLI
│   ├── evm/            # isolated EVM bytecode runner/debugger
│   ├── faucet/         # testnet faucet
│   ├── geth/           # the Viction full node client
│   ├── p2psim/         # peer simulation client
│   ├── puppeth/        # private-network builder wizard
│   ├── rlpdump/        # RLP dump formatting tool
│   └── utils/          # shared flag/config helpers for the other commands
├── common/             # shared types (hash, address, big ints)
├── consensus/          # consensus engines, one per protocol family
│   ├── clique/         # ethereum clique engine
│   ├── ethash/         # ethereum PoW engine
│   ├── misc/           # shared engine helpers
│   └── posv/           # viction PoSV engine
├── console/            # interactive JS console
├── contracts/          # system-contract Go bindings consumed by consensus/state code
│   ├── blocksigner/    # block-signer bindings
│   ├── checkpointoracle/ # checkpoint-oracle bindings
│   ├── multisig/       # multisig-wallet bindings
│   ├── randomize/      # creator/attestor randomize bindings
│   ├── token/          # token/gas-token bindings
│   ├── trading/        # trading-engine bindings (listing, relayer registration)
│   ├── validator/      # validator-contract bindings
│   └── zerogas/        # zero-gas contract bindings
├── core/               # chain/block processing, EVM, state
│   ├── state/          # state layer with Viction extension
│   │   └── state_viction.go        # Viction state layer
│   ├── blockchain_viction.go       # Viction block importer / VictionProcessor wiring
│   ├── state_processor_viction.go  # VictionProcessor: block processing incl. lending/trading roots
│   └── state_transition_viction.go # Viction-specific state-transition rules
├── crypto/             # signature primitives (secp256k1 C code → C compiler required)
├── docs/               # documentation
├── eth/                # protocol service wiring; Viction backend in backend_viction.go
├── ethclient/          # Go client for Ethereum-style JSON-RPC
├── ethdb/              # database abstractions (LevelDB, memory)
├── ethstats/           # reporting to ethstats dashboards
├── event/              # event feed/subscription helpers
├── graphql/            # GraphQL API service
├── internal/           # non-importable internals
│   ├── ethapi/         # common RPC facades shared by all transports
│   └── victionapi/     # Viction system service APIs (validators, attestors, penalties, rewards)
├── legacy/             # leagcy lending/trading engines (kept for compatibility)
├── les/                # light-server protocol
├── light/              # light-client helpers
├── log/                # structured logging
├── metrics/            # metrics collection/export
├── miner/              # block production/miner control
├── mobile/             # Android/iOS wrappers (gomobile: .aar / .framework)
├── node/               # node container/service registry
├── p2p/                # devp2p networking: discovery, transport
├── params/             # protocol constants and chain configurations
│   └── chainspecs/     # genesis chainspecs for Viction Mainnet (viction) and Testnet (victest)
├── rlp/                # RLP encoding/decoding
├── rpc/                # JSON-RPC server implementation
├── signer/             # clef signer core
├── swarm/              # distributed-storage service integration
├── tests/              # cross-implementation state/genesis/common tests
├── trie/               # Merkle Patricia trie
├── .copier-answers.yml # project Copier answers
├── .dockerignore       # docker-build exclude patterns
├── .gitattributes      # git attributes
├── .gitignore          # git ignore patterns
├── .gitmodules         # git submodule definitions
├── .golangci.yml       # golangci-lint config used by `make lint`
├── .mailmap            # git author mapping
├── .travis.yml         # Travis CI config (retained upstream)
├── AGENTS.md           # AI assistant entry point
├── AUTHORS             # contributor list
├── CLAUDE.md           # Claude-specific instructions
├── COPYING             # GPL license
├── COPYING.LESSER      # LGPL license
├── Dockerfile          # container build definition
├── Dockerfile.alltools # container build for the full tool suite
├── Makefile            # build/test/lint targets
├── README.md           # project overview
└── SECURITY.md         # security policy
├── appveyor.yml        # AppVeyor CI config (retained upstream)
├── circle.yml          # CircleCI config (retained upstream)
├── go.mod              # Go module definition
├── go.sum              # module checksums
├── interfaces.go       # root-level API interfaces kept from geth
├── oss-fuzz.sh         # OSS-Fuzz integration script
```
<!-- Project-specific / Layout -->

## Layer Dependencies

List of layers and their relationships:

- Declare the innermost layer first.
- A layer may depend on itself and on any layer declared before it, and on nothing declared after it.
- If a layer appears in the table below, the dependency graph must be enforced.

| Layer | Path | Depends on | Must never contain |
|-------|------|------------|--------------------|
<!-- Project-specific / Layer Dependencies -->

### Deliberate Exceptions

List of documented exceptions that cannot follow the architecture rules above:

- Any place the rules above are knowingly broken, with the reason and the condition for fixing it.
- An undocumented exception is a defect; a documented one is a decision.

| What | Why | Revisit-if |
|------|-----|------------|
<!-- Project-specific / Deliberate Exceptions -->

## Supporting Tools

All the supporting tools declared here are opinionated for the sake of consistent results across development environments.

List each required tool/library under its own heading, followed by the commands for each use case.
Common use cases include: Build, Test, Lint, Format, Install, ...

<!-- Project-specific / Supporting Tools -->

### Go toolchain

- This project supports Go version upto go1.18. When building or testing, set the environment variable `GOTOOLCHAIN=go1.18.10` to ensure a consistent result across development environments.

### Build

- `make geth` — build only the geth binary (via `build/ci.go install ./cmd/geth`), output in `./build/bin`.
- `make all` — build the full suite of tools from `cmd/`.
- Plain go alternative: `go build ./...` — on modern toolchains it requires `GOFLAGS=-ldflags=-checklinkname=0` (see Confirmed Facts).
- Cross compile: `make geth-cross` (per-OS targets like `geth-linux-amd64` use `build/ci.go xgo`).

### Test

- `go test ./...` — run the whole test suite with the go test runner.
- `make test` — depends on `all` (rebuilds everything first), then `build/ci.go test`.
- Narrower: `go test ./core/...`, `go test ./consensus/posv/...`, etc.

### Lint

- `make lint` — runs golangci-lint through `build/ci.go lint`, configured by `.golangci.yml`.

### Install

- `make devtools` — installs `go generate` helper tools (stringer, go-bindata, abigen, ...).
- `go install ./cmd/geth` — install geth into `GOPATH`/`GOBIN`.

## Confirmed Facts

Facts about the project which have been confirmed; they outrank anything derived or guessed.

| Fact | Reason |
|------|--------|
<!-- Project-specific / Confirmed Facts -->

## Project-specific

<!-- Project-specific -->
