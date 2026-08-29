// Copyright 2014 The go-ethereum Authors
// (original work)
// Copyright 2025 The Viction Authors
// (modifications)
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/viction"
)

const recentBlockSigners = 10500 // Number of recent block sign transactions to keep in memory

// Create a new block with attestor's signature. Only accept non-attested block.
func (s *Ethereum) PosvAttestBlock(
	block *types.Block,
) (*types.Block, error) {
	c := s.engine.(*posv.Posv)
	config := s.blockchain.Config()
	posvConfig := config.Posv

	header := block.Header()
	number := header.Number.Uint64()
	if number <= posvConfig.Epoch {
		return block, nil
	}
	// Only accept non-attested block.
	if len(header.Attestor) == posv.ExtraSeal {
		return nil, nil
	}
	eb, wallet, err := s.getEtherbaseWallet()
	if err != nil || eb.IsZero() {
		return nil, nil
	}
	creator, err := c.Author(header)
	if err != nil {
		return nil, nil
	}
	checkpointHeader := posv.GetCheckpointHeader(posvConfig, header, nil, s.blockchain)
	valAttPairs, _, err := s.PosvGetCreatorAttestorPairs(header, checkpointHeader, config, posvConfig, config.Viction)
	if err != nil {
		return nil, nil
	}
	assigned, ok := valAttPairs[creator]
	if !ok || eb != assigned {
		return nil, nil
	}
	sig, err := wallet.SignData(accounts.Account{Address: eb}, accounts.MimetypePosv, posv.PosvRLP(header))
	if err != nil {
		return nil, err
	}
	attestedHeader := types.CopyHeader(header)
	attestedHeader.Attestor = make([]byte, len(sig))
	copy(attestedHeader.Attestor, sig)
	attestedBlock := block.WithSeal(attestedHeader)
	attestedBlock.ReceivedAt = block.ReceivedAt // preserve for propagation-latency metrics
	return attestedBlock, nil
}

// Get attestors from list of validators.
func (s *Ethereum) PosvGetAttestors(
	header *types.Header, validators []common.Address,
	victionConfig *params.VictionConfig,
) ([]int64, error) {
	state, err := s.BlockChain().State()
	if err != nil {
		return nil, err
	}
	return viction.GetAttestorsFromState(validators, victionConfig, state)
}

// Get creator-attestor pairs from the state.
func (s *Ethereum) PosvGetCreatorAttestorPairs(
	header, checkpointHeader *types.Header,
	config *params.ChainConfig, posvConfig *params.PosvConfig, victionConfig *params.VictionConfig,
) (map[common.Address]common.Address, uint64, error) {
	if config.Viction == nil || checkpointHeader == nil {
		return nil, 0, viction.ErrInvalidAttestorList
	}
	pairs, offset, err := viction.GetCreatorAttestorPairsFromCheckpointHeader(config, config.Posv, header, checkpointHeader)
	if err == nil || err != viction.ErrNoValidator {
		return pairs, offset, err
	}

	number := header.Number.Uint64()
	checkpointNumber := checkpointHeader.Number.Uint64()
	log.Warn("[Backend] Get Creator-Attestor Pairs from checkpoint header failed. Retry with state", "checkpoint", checkpointNumber, "number", number)
	stateAtCheckpoint, err := s.blockchain.StateAt(checkpointHeader.Root)
	if err != nil {
		return nil, 0, err
	}
	return viction.GetCreatorAttestorPairsFromState(config, posvConfig, header, checkpointHeader, stateAtCheckpoint)
}

// Calculate rewards at the end of each epoch.
func (s *Ethereum) PosvGetEpochRewards(
	header *types.Header,
	config *params.ChainConfig, posvConfig *params.PosvConfig, victionConfig *params.VictionConfig,
	chainReader consensus.ChainReader, statedb *state.StateDB, logger log.Logger,
) (*posv.EpochReward, error) {
	number := header.Number.Uint64()
	preCheckpointHeader := chainReader.GetHeader(header.ParentHash, number-1)
	preCheckpointState, err := s.BlockChain().StateAt(preCheckpointHeader.Root)
	if err != nil {
		logger.Warn("[Backend] Get preCheckpoint state failed. Fallback to checkpoint state", "number", number-1, "hash", header.ParentHash, "err", err)
		preCheckpointState = statedb
	}

	return viction.CalcRewardPerEpoch(header, config, posvConfig, victionConfig, chainReader, preCheckpointState, s.blockchain, s.blockSignersCache, logger)
}

// Add balance rewards to the state.
func (s *Ethereum) PosvDistributeEpochRewards(
	header *types.Header, epochReward *posv.EpochReward,
	state *state.StateDB,
) error {
	if state == nil || epochReward == nil {
		return nil
	}

	rewardAmount, stakeholderCount := viction.DistributeStakeholderRewards(state, epochReward)
	log.Info("[Backend] Distributed epoch rewards", "block", header.Number.Uint64(), "stakeholderCount", stakeholderCount, "totalReward", rewardAmount.String())
	return nil
}

// Penalize validators for creating bad block or not creating block at all.
func (s *Ethereum) PosvGetPenalties(
	header *types.Header, validators []common.Address,
	config *params.ChainConfig, posvConfig *params.PosvConfig, victionConfig *params.VictionConfig,
	chainReader consensus.ChainReader,
) ([]common.Address, error) {
	if config.IsTIPSigning(header.Number) {
		return viction.PenalizeValidatorsTIPSigning(header, validators, config, posvConfig, victionConfig, chainReader, s.BlockChain(), s.blockSignersCache)
	}
	return viction.PenalizeValidatorsDefault(header, config, posvConfig, victionConfig, chainReader, s.BlockChain())
}

// Get eligble validators from the state.
func (s *Ethereum) PosvGetValidators(
	header *types.Header,
	config *params.ChainConfig, victionConfig *params.VictionConfig,
	chainReader consensus.ChainReader,
) ([]common.Address, error) {
	if header == nil {
		return []common.Address{}, viction.ErrNilHeader
	}
	statedb, err := s.blockchain.StateAt(header.Root)
	if err != nil {
		return nil, err
	}
	return viction.GetValidators(header, config, victionConfig, statedb)
}

// Create new Randomize transaction and submit to TxPool.
func (s *Ethereum) PosvRandomNumber(
	block *types.Block,
) error {
	c := s.engine.(*posv.Posv)
	config := s.blockchain.Config()
	vicConfig := config.Viction
	posvConfig := config.Posv
	number := block.NumberU64()

	if !config.IsTIPRandomize(block.Number()) {
		log.Info("[Backend][RandomNumber] Randomize not enabled", "number", block.NumberU64())
		return nil
	}
	blockOfEpoch := number % posvConfig.Epoch
	commitPhase := vicConfig.IsRandomizerCommitPhase(blockOfEpoch)
	if commitPhase {
		chaindb := s.ChainDb()
		exists, _ := chaindb.Has(viction.RandomizeKeyName)
		if exists {
			// Already committed this epoch.
			return nil
		}

		eb, wallet, err := s.getEtherbaseWallet()
		if err != nil || eb.IsZero() {
			return nil
		}
		if !s.isEligibleValidator(c, eb, block.Header()) {
			log.Debug("[Backend][RandomNumber] Not a validator in this epoch", "etherbase", eb, "number", number)
			return nil
		}

		nonce := s.nextNonce(eb)

		key := viction.GenerateRandomKey()
		secret, err := viction.GenerateRandomNumber(posvConfig.Epoch, key)
		if err != nil {
			log.Error("[Backend][RandomNumber] Failed to generate RandomNumber", "number", number, "err", err)
			return err
		}
		tx := viction.CreateSetRandomizeSecretTransaction(nonce, vicConfig.RandomizerContract, secret)
		if err := s.signAndSubmitTransaction(eb, wallet, tx, config.ChainID, "RandomNumber"); err != nil {
			return err
		}
		if err := chaindb.Put(viction.RandomizeKeyName, key); err != nil {
			log.Error("[Backend][RandomNumber] Failed to store RandomizeKey to database", "number", number, "err", err)
			return err
		}
		return nil
	}
	openingPhase := vicConfig.IsRandomizerOpeningPhase(blockOfEpoch)
	if openingPhase {
		chaindb := s.ChainDb()
		key, err := chaindb.Get(viction.RandomizeKeyName)
		if err != nil || len(key) == 0 {
			// Either already revealed or never committed in this epoch.
			return nil
		}

		eb, wallet, err := s.getEtherbaseWallet()
		if err != nil || eb.IsZero() {
			return nil
		}
		if !s.isEligibleValidator(c, eb, block.Header()) {
			log.Debug("[Backend][RandomNumber] Not a validator in this epoch", "etherbase", eb, "number", number)
			return nil
		}

		nonce := s.nextNonce(eb)

		tx := viction.CreateSetRandomizeOpeningTransaction(nonce, vicConfig.RandomizerContract, key)
		if err := s.signAndSubmitTransaction(eb, wallet, tx, config.ChainID, "RandomNumber"); err != nil {
			return err
		}
		if err := chaindb.Delete(viction.RandomizeKeyName); err != nil {
			log.Error("[Backend][RandomNumber] Failed to delete RandomizeKey in database", "number", number, "err", err)
		}
	}
	return nil
}

// Create new BlockSign transaction and submit to TxPool.
func (s *Ethereum) PosvSignBlock(
	block *types.Block,
) error {
	c := s.engine.(*posv.Posv)
	config := s.blockchain.Config()
	vicConfig := config.Viction
	number := block.NumberU64()

	// Pre-TIP2019: Emit SignBlock transaction every block.
	// TIP2019: Emit SignBlock transaction every *vicConfig.ValidatorSignInterval* blocks.
	if config.IsTIP2019(block.Number()) && number%vicConfig.ValidatorSignInterval != 0 {
		log.Info("[Backend][SignBlock] Skipped sign block: not interval block", "number", block.NumberU64(), "inveral", vicConfig.ValidatorSignInterval, "blockOfInterval", number%vicConfig.ValidatorSignInterval)
		return nil
	}

	eb, wallet, err := s.getEtherbaseWallet()
	if err != nil || eb.IsZero() {
		return nil
	}
	if !s.isEligibleValidator(c, eb, block.Header()) {
		log.Debug("[Backend][SignBlock] Not a validator in this epoch", "etherbase", eb, "number", block.NumberU64())
		return nil
	}

	nonce := s.nextNonce(eb)
	pendingTxs, _ := s.txPool.Pending(false)

	expectedData := viction.CreateBlockSignData(block.Number(), block.Hash())
	contractAddr := config.Viction.ValidatorBlockSignContract
	for _, tx := range pendingTxs[eb] {
		if tx.To() != nil && *tx.To() == contractAddr && bytes.Equal(tx.Data(), expectedData) {
			log.Info("[Backend][SignBlock] Skipped. BlockSign transaction already exists in pool", number, "blockHash", block.Hash(), "etherbase", eb, "nonce", tx.Nonce())
			return nil
		}
	}

	tx := viction.CreateBlockSignTransaction(nonce, config.Viction.ValidatorBlockSignContract, block.Number(), block.Hash())
	if err := s.signAndSubmitTransaction(eb, wallet, tx, config.ChainID, "SignBlock"); err != nil {
		return err
	}
	return nil
}

// Get current etherbase altogether with it signing wallet.
func (s *Ethereum) getEtherbaseWallet() (common.Address, accounts.Wallet, error) {
	eb, err := s.Etherbase()
	if err != nil || eb.IsZero() {
		return common.Address{}, nil, err
	}
	wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
	if wallet == nil || err != nil {
		return common.Address{}, nil, err
	}
	return eb, wallet, nil
}

// Check if given etherbase is eligble signer for given header.
func (s *Ethereum) isEligibleValidator(c *posv.Posv, eb common.Address, header *types.Header) bool {
	snap, err := c.GetSnapshot(s.blockchain, header)
	if err != nil {
		return false
	}
	_, ok := snap.Signers[eb]
	return ok
}

// Return available nonce for the given address, with txPool awareness.
func (s *Ethereum) nextNonce(addr common.Address) uint64 {
	statedb, err := s.blockchain.State()
	if err != nil {
		return 0
	}
	nonce := statedb.GetNonce(addr)
	pendingTxs, _ := s.txPool.Pending(false)
	nonce += uint64(len(pendingTxs[addr]))
	return nonce
}

// Sign and submit given transaction to local txPool.
func (s *Ethereum) signAndSubmitTransaction(eb common.Address, wallet accounts.Wallet, tx *types.Transaction, chainID *big.Int, txLabel string) error {
	signedTx, err := wallet.SignTx(accounts.Account{Address: eb}, tx, chainID)
	if err != nil {
		return err
	}
	if err := s.txPool.AddLocal(signedTx); err != nil {
		if err == core.ErrReplaceUnderpriced || err == core.ErrAlreadyKnown {
			log.Info(fmt.Sprintf("[Backend][%s] Transaction is duplicated", txLabel),
				"txHash", signedTx.Hash(), "etherbase", eb, "nonce", signedTx.Nonce())
			return nil
		}
		log.Warn(fmt.Sprintf("[Backend][%s] Failed to submit transaction", txLabel),
			"txHash", signedTx.Hash(), "etherbase", eb, "nonce", signedTx.Nonce(), "err", err)
		return err
	}
	log.Info(fmt.Sprintf("[Backend][%s] Submitted transaction", txLabel),
		"txHash", signedTx.Hash(), "etherbase", eb, "nonce", signedTx.Nonce())
	return nil
}

// Attach services required by Viction blockchain.
func (s *Ethereum) setupPosvBackend(chainConfig *params.ChainConfig) error {
	if chainConfig.Posv == nil {
		return nil
	}

	posvEngine, ok := s.engine.(*posv.Posv)
	if !ok {
		return fmt.Errorf("posv config present but engine is %T, expected *posv.Posv", s.engine)
	}

	posvEngine.SetBackend(s)
	log.Info("[Backend] Set current backend reference to PoSV engine.")
	s.handler.blockFetcher.SetPosvBackend(s)
	log.Info("[Backend] Set current backend reference to BlockFetcher.")

	return nil
}
