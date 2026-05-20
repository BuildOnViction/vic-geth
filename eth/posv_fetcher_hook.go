// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// POSV: register block fetcher hooks so the assigned attestor can append M2 on
// propagated blocks, and so validator nodes submit BlockSigner.sign() vote
// transactions for every block they receive from peers (F-3 fix).

package eth

import (
	"fmt" // used for invalid attestor signature error

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/contracts/blocksigner"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// setupPosvFetcherHook wires the attestor-append and block-sign callbacks for
// propagated blocks.
func (s *Ethereum) setupPosvFetcherHook() {
	if _, ok := s.engine.(*posv.Posv); !ok || s.protocolManager == nil {
		return
	}
	s.protocolManager.blockFetcher.SetPOSVAppendAttestorHook(s.posvPropagatedBlockAppendAttestor)
	s.protocolManager.blockFetcher.SetPOSVSignBlockHook(s.posvPropagatedBlockSignHook)
}

// setupPosvMinerHook wires the self-attest callback on the miner so that, when
// this node is both the block creator (M1) and the assigned attestor (M2),
// header.Attestor is set before the block is written to the database.
// Must be called after eth.miner is initialised.
func (s *Ethereum) setupPosvMinerHook() {
	if _, ok := s.engine.(*posv.Posv); !ok || s.miner == nil {
		return
	}
	s.miner.SetPosvSelfAttestHook(s.posvMinedBlockSelfAttest)
	s.miner.SetPosvSignBlockHook(s.posvPropagatedBlockSignHook)
}

// posvAttestBlock is the shared signing core for both the propagated-block
// (M2 fetcher) and locally-mined (M1==M2 self-attest) paths.
//
// It recovers the block creator, looks up the assigned M2 attestor from the
// current epoch checkpoint, and — if this node's etherbase matches that
// attestor — signs the header and returns an updated block with Attestor set.
//
// Returns:
//   - (attestedBlock, true,  nil)  — attestor signature appended successfully.
//   - (block,         false, nil)  — precondition not met; caller keeps original.
//   - (nil,           false, err)  — hard error (signing / pair-lookup failure).
//
// The caller is responsible for any path-specific guards (e.g. "creator must
// equal etherbase" for self-attest) before invoking this function.
func (s *Ethereum) posvAttestBlock(block *types.Block) (*types.Block, bool, error) {
	header := block.Header()
	if header == nil {
		return block, false, nil
	}
	cfg := s.blockchain.Config()
	posvCfg := cfg.Posv
	if posvCfg == nil {
		return block, false, nil
	}
	n := header.Number.Uint64()
	if n <= posvCfg.Epoch {
		return block, false, nil
	}
	if len(header.Attestor) == posv.ExtraSeal {
		return block, false, nil
	}
	posvEngine, ok := s.engine.(*posv.Posv)
	if !ok {
		return block, false, nil
	}
	creator, err := posvEngine.Author(header)
	if err != nil {
		return block, false, nil
	}
	checkpoint := posv.GetCheckpointHeader(posvCfg, header, s.blockchain, nil)
	if checkpoint == nil {
		return block, false, nil
	}
	pairs, _, err := s.PosvGetCreatorAttestorPairs(posvEngine, cfg, header, checkpoint)
	if err != nil {
		log.Warn("[POSV-M2] GetCreatorAttestorPairs failed", "number", header.Number, "err", err)
		return nil, false, err
	}
	log.Debug("[POSV-M2] pairs computed", "number", n, "pairsLen", len(pairs), "creator", creator)
	eb, err := s.Etherbase()
	if err != nil || eb == (common.Address{}) {
		log.Debug("[POSV-M2] skip: no etherbase", "number", n, "err", err)
		return block, false, nil
	}
	assigned, ok := pairs[creator]
	if !ok {
		return block, false, nil
	}
	if assigned != eb {
		return block, false, nil
	}
	wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
	if wallet == nil || err != nil {
		return block, false, nil
	}
	sig, err := wallet.SignData(accounts.Account{Address: eb}, accounts.MimetypePosv, posv.PosvRLP(header))
	if err != nil {
		return nil, false, err
	}
	if len(sig) != posv.ExtraSeal {
		return nil, false, fmt.Errorf("posv: invalid attestor signature length %d", len(sig))
	}
	newH := types.CopyHeader(header)
	newH.Attestor = make([]byte, len(sig))
	copy(newH.Attestor, sig)
	attested := block.WithSeal(newH)
	attested.ReceivedAt = block.ReceivedAt // preserve for propagation-latency metrics
	return attested, true, nil
}

// posvPropagatedBlockAppendAttestor is the M2 fetcher hook for propagated (P2P)
// blocks.  It delegates to posvAttestBlock and adds path-specific log messages.
func (s *Ethereum) posvPropagatedBlockAppendAttestor(block *types.Block) (*types.Block, bool, error) {
	n := block.NumberU64()
	hash := block.Hash()

	attested, appended, err := s.posvAttestBlock(block)
	if err != nil {
		log.Warn("[POSV-M2] attestor signing failed", "number", n, "hash", hash, "err", err)
		return nil, false, err
	}
	if !appended {
		log.Debug("[POSV-M2] skip: precondition not met (see posvAttestBlock)", "number", n, "hash", hash)
		return block, false, nil
	}
	creator, _ := s.engine.(*posv.Posv).Author(block.Header())
	eb, _ := s.Etherbase()
	log.Info("[POSV-M2] appended attestor on propagated block", "number", n, "hash", attested.Hash(), "creator", creator, "attestor", eb)
	return attested, true, nil
}

// posvMinedBlockSelfAttest is the M1==M2 hook for locally mined blocks.
// Locally mined blocks bypass the P2P fetcher, so
// posvPropagatedBlockAppendAttestor never fires for them.
// Returns the attested block, or nil if not applicable (resultLoop keeps the original).
func (s *Ethereum) posvMinedBlockSelfAttest(block *types.Block) *types.Block {
	n := block.NumberU64()

	// Self-attest only applies when this node mined the block (creator == etherbase).
	eb, err := s.Etherbase()
	if err != nil || eb == (common.Address{}) {
		return nil
	}
	posvEngine, ok := s.engine.(*posv.Posv)
	if !ok {
		return nil
	}
	creator, err := posvEngine.Author(block.Header())
	if err != nil {
		log.Warn("[POSV-self-attest] cannot recover creator", "number", n, "err", err)
		return nil
	}
	if creator != eb {
		return nil
	}

	attested, appended, err := s.posvAttestBlock(block)
	if err != nil {
		log.Warn("[POSV-self-attest] signing failed", "number", n, "attestor", eb, "err", err)
		return nil
	}
	if !appended {
		log.Debug("[POSV-self-attest] not assigned as M2 for own block", "number", n, "creator", creator, "etherbase", eb)
		return nil
	}
	log.Info("[POSV-self-attest] creator==M2, attached self-attestation to mined block",
		"number", n, "hash", attested.Hash(), "attestor", eb)
	return attested
}

// [7s62] posvPropagatedBlockSignHook fires after a propagated block is successfully
// imported.  It creates a BlockSigner.sign(blockNumber, blockHash) transaction and
// injects it into the local tx pool so this validator is credited in
// reward/penalty accounting — even when it was not the block creator.
//
// Mirrors victionchain's signHook in eth/backend.go.  Key differences:
//   - Uses IsTIPSigning guard (vic-geth deletes BlockSigner state before that fork)
//   - Uses blockchain.State().GetNonce to avoid race with async pool reset
//   - Uses checkpoint validators for the "am I a validator?" check (no IsSigner callback)
//   - Does NOT port the randomise-key secret/opening tx (legacy TomoX, absent in vic-geth)
func (s *Ethereum) posvPropagatedBlockSignHook(block *types.Block) error {
	cfg := s.blockchain.Config()
	if cfg.Posv == nil || cfg.Viction == nil {
		log.Debug("[POSV sign hook] skipped: no Posv or Viction config")
		return nil
	}

	// Before TIPSigning the BlockSigner contract state is reset at the start of
	// every block, so sign txs have no lasting effect on accounting.
	if !cfg.IsTIPSigning(block.Number()) {
		log.Debug("[POSV sign hook] skipped: not past TIPSigning fork", "block", block.NumberU64(), "tipSigningBlock", cfg.TIPSigningBlock)
		return nil
	}

	// [7s62] After TIP2019: reduce pool spam by submitting a sign tx only on
	// every MergeSignRange-th (15th) block.  Before TIP2019 every block is signed.
	if cfg.IsTIP2019(block.Number()) && block.NumberU64()%blocksigner.MergeSignRange != 0 {
		log.Debug("[POSV sign hook] skipped: not on MergeSignRange boundary", "block", block.NumberU64(), "mergeSignRange", blocksigner.MergeSignRange, "mod", block.NumberU64()%blocksigner.MergeSignRange)
		return nil
	}

	eb, err := s.Etherbase()
	if err != nil || eb == (common.Address{}) {
		log.Debug("[POSV sign hook] skipped: no etherbase", "err", err)
		return nil // not a validator node; skip silently
	}

	// Confirm etherbase is a staked masternode (in the snapshot signers set).
	// This intentionally uses the snapshot — which includes penalized validators —
	// rather than the checkpoint header's active validator list. Penalized nodes
	// must still be able to submit BlockSign txs to prove liveness for comeback.
	c, ok := s.engine.(*posv.Posv)
	if !ok {
		log.Warn("[POSV sign hook] skipped: engine is not *posv.Posv")
		return nil
	}
	snap, err := c.GetSnapshot(s.blockchain, block.Header())
	if err != nil {
		log.Warn("[POSV sign hook] skipped: cannot get snapshot", "block", block.NumberU64(), "err", err)
		return nil
	}
	if _, isSigner := snap.Signers[eb]; !isSigner {
		log.Debug("[POSV sign hook] skipped: etherbase not in snapshot signers", "etherbase", eb, "block", block.NumberU64())
		return nil
	}

	wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
	if wallet == nil || err != nil {
		log.Warn("[POSV sign hook] skipped: no local wallet for etherbase", "etherbase", eb, "err", err)
		return nil
	}

	// Get the nonce from the blockchain state (post-import), NOT from
	// pool.Nonce().  The pool's pendingNonces may still reflect the
	// pre-import state because pool.reset() runs asynchronously.  Using
	// pool.Nonce() can produce a nonce ahead of the state nonce, creating
	// a nonce gap in pending that demoteUnexecutables will evict.
	statedb, err := s.blockchain.State()
	if err != nil {
		log.Error("[POSV sign hook] failed to get state", "err", err)
		return err
	}
	nonce := statedb.GetNonce(eb)
	tx := blocksigner.CreateTxSign(block.Number(), block.Hash(), nonce, cfg.Viction.ValidatorBlockSignContract)
	log.Info("[POSV BlockSigner] creating sign tx", "block", block.NumberU64(), "blockHash", block.Hash(), "from", eb, "nonce", nonce)

	txSigned, err := wallet.SignTx(accounts.Account{Address: eb}, tx, cfg.ChainID)
	if err != nil {
		log.Error("[POSV BlockSigner] failed to sign vote tx", "number", block.NumberU64(), "err", err)
		return err
	}
	if err := s.txPool.AddLocal(txSigned); err != nil {
		// "replacement transaction underpriced" is expected when the sign hook fires
		// multiple times for the same block boundary (timer retry, dual path trigger).
		// The vote was already submitted; this is not an error.
		if err == core.ErrReplaceUnderpriced || err == core.ErrAlreadyKnown {
			log.Info("[POSV BlockSigner] vote tx already in pool (duplicate)", "block", block.NumberU64(), "nonce", nonce)
			return nil
		}
		log.Error("[POSV BlockSigner] failed to add vote tx to pool", "number", block.NumberU64(), "hash", block.Hash(), "from", eb, "nonce", nonce, "txHash", txSigned.Hash(), "err", err)
		return err
	}
	log.Info("[POSV BlockSigner] submitted vote tx", "block", block.NumberU64(), "blockHash", block.Hash(), "from", eb, "nonce", nonce, "txHash", txSigned.Hash())

	// Submit randomize tx (secret or opening) if we are in the right epoch phase.
	// Uses nonce+1 since the sign tx above consumed `nonce`.
	s.posvRandomizeHook(block, eb, wallet, nonce+1)

	return nil
}
