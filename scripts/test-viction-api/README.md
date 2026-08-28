# test-viction-api

End-to-end JSON-RPC tests for the Viction/PoSV-specific methods exposed by `vic-geth`.

These methods are registered on the `eth` namespace by
`internal/ethapi/api_viction.go` and are only available when:

- the engine is `*posv.Posv`, and
- `ChainConfig.Posv` is set.

## Covered methods

| Method                            | Notes                                                       |
| --------------------------------- | ----------------------------------------------------------- |
| `eth_getAttestorsPairsByHash`     | `(hash) -> map<creator,attestor>`                           |
| `eth_getAttestorsPairsByNumber`   | `(blockTag|hexNumber) -> map<creator,attestor>`             |
| `eth_getRewardByHash`             | `(checkpointHash) -> { signers, rewards }` (epoch reward)   |
| `eth_getBlockFinalityByHash`      | `(hash) -> uint (0..100)`                                   |
| `eth_getBlockFinalityByNumber`    | `(blockTag|hexNumber) -> uint (0..100)`                     |

## Run

Requires Node.js 20+ (uses the built-in `fetch`).

```bash
cd scripts/test-viction-api
npm install
RPC_URL=http://localhost:8545 npm test
```

### Prerequisite: enable the `eth` namespace on the node

`vic-geth`'s default HTTP modules are only `net` + `web3` (see
`node/defaults.go`). The `eth` namespace (which hosts both the standard
`eth_*` methods *and* the Viction extensions) must be explicitly enabled:

```bash
geth \
  --http \
  --http.addr 0.0.0.0 \
  --http.port 8545 \
  --http.api "eth,net,web3" \
  ...
```

Verify with:

```bash
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"rpc_modules","params":[]}'
# expect: {"jsonrpc":"2.0","id":1,"result":{"eth":"1.0","net":"1.0","web3":"1.0"}}
```

If `eth` is missing, every test will fail with `-32601 method not found` —
the script detects this in preflight and prints an actionable hint.

### Environment variables

| Var          | Default                   | Purpose                                    |
| ------------ | ------------------------- | ------------------------------------------ |
| `RPC_URL`    | `http://localhost:8545`   | JSON-RPC endpoint                          |
| `EPOCH`      | `900`                     | PoSV epoch length (used to pick a checkpoint) |
| `TIMEOUT_MS` | `20000`                   | Per-call HTTP timeout                      |
| `VERBOSE`    | unset                     | When `1`, print stack traces on failure    |

### Examples

```bash
# Hit a remote node, longer timeout
RPC_URL=https://rpc.viction.xyz TIMEOUT_MS=30000 npm test

# Show stack traces when something fails
VERBOSE=1 npm test

# Override epoch for victest / private networks
EPOCH=900 npm test
```

## What the script does

1. **Discovers fixtures** from the node: latest block, chain id, and the latest
   checkpoint block (highest `n * EPOCH` that is at least one epoch below the head
   so its rewards are already materialised).
2. **Runs assertions** per RPC method, including positive cases (well-formed
   addresses/hashes, value ranges) and negative cases (unknown hash, unknown
   future block, non-checkpoint hash).
3. **Cross-checks** that `*ByHash` and `*ByNumber` agree on the same block.

Tests are intentionally **read-only** — they never send transactions or modify
state — so they are safe to run against mainnet, testnet, or a local devnet.

## Exit codes

- `0` — all tests passed (or skipped because preconditions were not met)
- `1` — at least one test failed, or the node was unreachable

## Notes / caveats

- `eth_getRewardByHash` requires the state trie at the checkpoint block. On a
  pruning full-sync node only the most recent ~128 blocks are kept, so the
  reward test may fail with `missing trie node` against older checkpoints.
  Run an archive node (`--gcmode=archive`) to exercise it on arbitrary history.
- `eth_getBlockFinalityBy*` returns `0` (not an error) for unknown blocks —
  this matches the Go implementation in `eth/api_backend_viction.go`.
- The script uses only the Node standard library and `tsx` to run TypeScript;
  no `web3.js`/`ethers.js` dependency is required.
