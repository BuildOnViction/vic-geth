// Copyright (c) 2026 Viction
// POSV: Randomize transaction submission hook.
//
// After each block is imported, this hook checks whether the validator should
// submit a setSecret or setOpening transaction to the Randomize contract (0x89)
// based on the current position within the epoch.

package eth

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/viction"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// posvRandomizeHook is called after each block import to submit randomize
// transactions when the validator is in the commit or reveal phase of the epoch.
//
// Phase 1 (Commit/Secret): block % epoch in [CommitNthBlock, RevealNthBlock)
//   - Generate a 32-byte random key
//   - Encrypt a random number with the key
//   - Submit setSecret(bytes32[]) to the Randomize contract
//   - Persist the key in chainDb for the reveal phase
//
// Phase 2 (Reveal/Opening): block % epoch in [RevealNthBlock, FinaleNthBlock]
//   - Retrieve the persisted key from chainDb
//   - Submit setOpening(bytes32) to the Randomize contract
//   - Delete the key from chainDb
func (s *Ethereum) posvRandomizeHook(block *types.Block, eb common.Address, wallet accounts.Wallet, nonce uint64) {
	cfg := s.blockchain.Config()
	vicConfig := cfg.Viction
	if vicConfig == nil {
		log.Warn("[POSV randomize] skip: viction config is nil")
		return
	}
	if vicConfig.RandomizerContract == (common.Address{}) {
		log.Warn("[POSV randomize] skip: randomizer contract is zero address")
		return
	}
	if !cfg.IsTIPRandomize(block.Number()) {
		log.Warn("[POSV randomize] skip: not past TIPRandomize fork", "block", block.NumberU64(), "tipRandomizeBlock", cfg.TIPRandomizeBlock)
		return
	}

	blockNumber := block.NumberU64()
	blockInEpoch := blockNumber % cfg.Posv.Epoch
	log.Info("[POSV randomize] hook entered", "block", blockNumber, "epochPos", blockInEpoch,
		"commitAt", vicConfig.RandomizerCommitNthBlock, "revealAt", vicConfig.RandomizerRevealNthBlock,
		"finaleAt", vicConfig.RandomizerFinaleNthBlock)

	s.submitRandomizeIfNeeded(cfg, vicConfig, eb, wallet, nonce, blockNumber, blockInEpoch)
}

func (s *Ethereum) submitRandomizeIfNeeded(
	cfg *params.ChainConfig,
	vicConfig *params.VictionConfig,
	eb common.Address,
	wallet accounts.Wallet,
	baseNonce uint64,
	blockNumber uint64,
	blockInEpoch uint64,
) {
	chainDb := s.ChainDb()
	randomizeContract := vicConfig.RandomizerContract

	// Phase 1: Commit secret
	if viction.ShouldSendSecret(vicConfig, blockInEpoch) {
		exists, _ := chainDb.Has(viction.RandomizeKeyName)
		if exists {
			// Already committed this epoch; skip.
			return
		}

		key := viction.GenerateRandomizeKey()
		tx, err := viction.BuildSecretTx(baseNonce, randomizeContract, cfg.Posv.Epoch, key)
		if err != nil {
			log.Error("[POSV randomize] failed to build secret tx", "block", blockNumber, "err", err)
			return
		}

		signedTx, err := s.signAndSubmitRandomizeTx(wallet, eb, cfg, tx, blockNumber, "secret")
		if err != nil {
			return
		}

		// Persist key for the reveal phase.
		if err := chainDb.Put(viction.RandomizeKeyName, key); err != nil {
			log.Error("[POSV randomize] failed to persist key", "block", blockNumber, "err", err)
			return
		}
		log.Info("[POSV randomize] submitted secret tx",
			"block", blockNumber, "epochPos", blockInEpoch, "txHash", signedTx.Hash())
		return
	}

	// Phase 2: Reveal opening
	if viction.ShouldSendOpening(vicConfig, blockInEpoch) {
		key, err := chainDb.Get(viction.RandomizeKeyName)
		if err != nil || len(key) == 0 {
			// No key stored — either already revealed or never committed.
			return
		}

		tx := viction.BuildOpeningTx(baseNonce, randomizeContract, key)
		signedTx, err := s.signAndSubmitRandomizeTx(wallet, eb, cfg, tx, blockNumber, "opening")
		if err != nil {
			return
		}

		// Clear the key — reveal is done for this epoch.
		if err := chainDb.Delete(viction.RandomizeKeyName); err != nil {
			log.Warn("[POSV randomize] failed to delete key from db", "block", blockNumber, "err", err)
		}
		log.Info("[POSV randomize] submitted opening tx",
			"block", blockNumber, "epochPos", blockInEpoch, "txHash", signedTx.Hash())
	}
}

// signAndSubmitRandomizeTx signs and submits a randomize transaction to the local pool.
func (s *Ethereum) signAndSubmitRandomizeTx(
	wallet accounts.Wallet,
	eb common.Address,
	cfg *params.ChainConfig,
	tx *types.Transaction,
	blockNumber uint64,
	phase string,
) (*types.Transaction, error) {
	signed, err := wallet.SignTx(accounts.Account{Address: eb}, tx, cfg.ChainID)
	if err != nil {
		log.Error("[POSV randomize] failed to sign "+phase+" tx", "block", blockNumber, "err", err)
		return nil, err
	}
	if err := s.txPool.AddLocal(signed); err != nil {
		if err == core.ErrReplaceUnderpriced || err == core.ErrAlreadyKnown {
			log.Debug("[POSV randomize] "+phase+" tx already in pool", "block", blockNumber, "nonce", tx.Nonce())
			return signed, nil
		}
		log.Error("[POSV randomize] failed to add "+phase+" tx to pool",
			"block", blockNumber, "from", eb, "nonce", tx.Nonce(), "err", err)
		return nil, err
	}
	return signed, nil
}
