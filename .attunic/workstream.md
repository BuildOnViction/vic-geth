# Workstream

State of the current audit or cleanup workstream; kept current during the work.

**Workstream**: PoSV finality enforcement (phases 1-3): track the finalized block from BlockSigner votes, refuse chains conflicting with it from the Prometheus fork, stop validators from signing conflicting blocks
**Started**: 2026-11-16   **Last updated**: 2026-11-16
**Status**: in progress - all batches done, awaiting user review and close

Decisions (user, 2026-11-16):

- Threshold 75% of the signed block's epoch validators (creator + attestor + BlockSigner tx senders within `consensusLimitTimeFinality` blocks on the same chain).
- Enforcement active once the finalized block is at/after `prometheusBlock`; tracking runs on every PoSV chain from now (no fork-choice effect before Prometheus).
- Blocks conflicting with the enforced finalized block are rejected before execution (`ErrPosvFinalityConflict`), not stored as side blocks.
- No CLI flag, no chainspec threshold field.

## Baseline

Recorded before the first change, verbatim:

- **Verify command**: `GOTOOLCHAIN=go1.18.10 go test ./core/... ./consensus/... ./eth/... ./miner/... ./internal/victionapi/... ./params/...`
- **Result**: fail - 24 packages `ok`, 1 `FAIL` (`eth/downloader`: `--- FAIL: TestFakedSyncProgress66Full ... final progress mismatch: have {... CurrentBlock:1004 HighestBlock:1003 ...} want {... CurrentBlock:1003 HighestBlock:1003 ...}`); rerun `go test ./eth/downloader/ -run TestFakedSyncProgress -count=5` passed
- **Known-failing before we started**: `eth/downloader` `TestFakedSyncProgress66Full` (flaky, 1 failure in 6 runs before any change; awaiting user confirmation as known failure)

## Batches

One batch = one intent; never mix a mechanical sweep with structural edits in the same diff.

- [x] 1. core/types, internal/victionapi, eth - extract shared BlockSigner vote decoding (`Transaction.SignedBlockHash`, `Receipt.IsSuccessful`); `GetBlockSignData`, reward, penalty and RPC use it, behavior unchanged - verified: build, vet, gofmt, `go test ./core/types/ ./internal/victionapi/ ./eth/` ok (victionapi has no tests; equivalence by inspection) - commit: bda56ec11 (by user)
- [x] 2. consensus/posv, core, core/rawdb - finality tracker: tally votes on each canonical block (hook in `writeHeadBlock`), advance and persist the finalized block, reload on startup and SetHead (hook in `loadLastState`) - verified: build, vet, gofmt (go1.18.10), new tests pass, full verify command 25/25 ok - staged by user
- [x] 3. core - enforcement: reject batches conflicting with the enforced finalized block in `insertChain` (before execution), guard `reorg()`, from Prometheus - verified: new tests pass and fail with the guards disabled, full verify command 25/25 ok
- [x] 4. eth/downloader - raise the common-ancestor floor to the enforced finalized block (`BlockChain.PosvEnforcedFinalizedHeader`) - verified: new tests pass, build/vet/gofmt clean, full verify command 25/25 ok
- [x] 5. internal/victionapi, eth, core/rawdb - validator signing safety: sign only canonical blocks, persist signed hash per validator and height (written before submitting, kept 2 epochs) and never sign a different one - verified: new tests pass, build/vet/gofmt clean, full verify command 26/26 ok

## Found But Not Fixed

Bugs and risks discovered while cleaning up: logged, not fixed, because a cleanup batch must not change behavior.

| What | Where | Severity | Note |
|------|-------|----------|------|
| Signing interval fork check differs: TIP2019 in `SubmitBlockSignTransaction`/`getGoverningValidators`, TIPSigning in `findNearestSignedBlock` | `internal/victionapi/blocksign.go`, `eth/api_posv.go` | low | RPC-only effect, mainnet blocks 1,050,000-3,000,000 |
| RPC finality of a signing block uses the votes of the next signing block | `eth/api_posv.go` `findNearestSignedBlock` | info | legacy semantics; tracker uses each block's own votes |
| `.attunic/workstream.md` not gitignored, `.gitignore` has no `# Project-specific` marker | `.gitignore` | info | protocol says workstream.md is gitignored; template-owned file, not edited |
| `TestFakedSyncProgress66Full` flaky | `eth/downloader/downloader_test.go` | low | failed 1 of 6 runs at baseline |
| A future block queued before the finalized block advanced past its fork is retried by `procFutureBlocks` every 5s, ignoring `ErrPosvFinalityConflict`, until the LRU evicts it | `core/blockchain.go` `procFutureBlocks` | low | wasted retries only; the block is still refused |
| "Skipped. BlockSign transaction already exists in pool" log passes `number` without a key, misaligning its key/value pairs | `internal/victionapi/blocksign.go` `SubmitBlockSignTransaction` | info | log output only |
| `core/mkalloc.go` not gofmt-clean with the go1.18.10 gofmt (the PATH `gofmt` is a newer release and flags more files) | `core/mkalloc.go` | info | pre-existing, untouched; check formatting with the toolchain's gofmt |

## Deferred

Deliberately out of scope, with the reason.

| What | Why deferred | Revisit when |
|------|--------------|--------------|
| Phase 4: RPC answers 100 for blocks at/below the finalized block, `posv_getFinalizedBlock` | user chose phases 1-3 | after enforcement ships |
| Phase 5: protocol-level finality (source/target votes, slashing) | hard fork, large protocol change | cross-height race becomes a practical concern |
| Enforcement in header-only/light sync (`HeaderChain.writeHeaders`, `les`) | finality needs receipts; full sync only | light clients are supported for Viction |
| Signing blocks that become canonical after a reorg | conflicts with the no-double-sign rule; minor reward impact | penalties observed from missed signatures |
