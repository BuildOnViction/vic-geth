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
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/victionapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const recentBlockSigners = 10500 // Number of recent block sign transactions to keep in memory

// Create a new block with attestor's signature. Only accept non-attested block.
func (s *Ethereum) PosvAttestBlock(
	block *types.Block,
) (*types.Block, error) {
	config := s.blockchain.Config()
	posvConfig := config.Posv
	header := block.Header()

	eb, wallet, err := s.getEtherbaseWallet()
	if err != nil || eb.IsZero() {
		return nil, nil
	}
	checkpointHeader := posv.GetCheckpointHeader(posvConfig, header, nil, s.blockchain)
	return victionapi.AttestBlock(block, checkpointHeader, config, posvConfig, eb, wallet)
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
	return victionapi.GetAttestorsFromState(validators, victionConfig, state)
}

// Get creator-attestor pairs from the state.
func (s *Ethereum) PosvGetCreatorAttestorPairs(
	header, checkpointHeader *types.Header,
	config *params.ChainConfig, posvConfig *params.PosvConfig, victionConfig *params.VictionConfig,
) (map[common.Address]common.Address, uint64, error) {
	if config.Viction == nil || checkpointHeader == nil {
		return nil, 0, victionapi.ErrInvalidAttestorList
	}
	pairs, offset, err := victionapi.GetCreatorAttestorPairsFromCheckpointHeader(config, config.Posv, header, checkpointHeader)
	if err == nil || err != victionapi.ErrNoValidator {
		return pairs, offset, err
	}

	number := header.Number.Uint64()
	checkpointNumber := checkpointHeader.Number.Uint64()
	log.Warn("[Backend] Get Creator-Attestor Pairs from checkpoint header failed. Retry with state", "checkpoint", checkpointNumber, "number", number)
	stateAtCheckpoint, err := s.blockchain.StateAt(checkpointHeader.Root)
	if err != nil {
		return nil, 0, err
	}
	return victionapi.GetCreatorAttestorPairsFromState(config, posvConfig, header, checkpointHeader, stateAtCheckpoint)
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

	return victionapi.CalcRewardPerEpoch(header, config, posvConfig, victionConfig, chainReader, preCheckpointState, s.blockchain, s.blockSignersCache, logger)
}

// Add balance rewards to the state.
func (s *Ethereum) PosvDistributeEpochRewards(
	header *types.Header, epochReward *posv.EpochReward,
	state *state.StateDB,
) error {
	if state == nil || epochReward == nil {
		return nil
	}

	rewardAmount, stakeholderCount := victionapi.DistributeStakeholderRewards(state, epochReward)
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
		return victionapi.PenalizeValidatorsTIPSigning(header, validators, config, posvConfig, victionConfig, chainReader, s.BlockChain(), s.blockSignersCache)
	}
	return victionapi.PenalizeValidatorsDefault(header, config, posvConfig, victionConfig, chainReader, s.BlockChain())
}

// Get eligble validators from the state.
func (s *Ethereum) PosvGetValidators(
	header *types.Header,
	config *params.ChainConfig, victionConfig *params.VictionConfig,
	chainReader consensus.ChainReader,
) ([]common.Address, error) {
	if header == nil {
		return []common.Address{}, victionapi.ErrNilHeader
	}
	statedb, err := s.blockchain.StateAt(header.Root)
	if err != nil {
		return nil, err
	}
	return victionapi.GetValidators(header, config, victionConfig, statedb)
}

// Create new Randomize transaction and submit to TxPool.
func (s *Ethereum) PosvRandomNumber(
	block *types.Block,
) error {
	c := s.engine.(*posv.Posv)
	eb, wallet, err := s.getEtherbaseWallet()
	if err != nil || eb.IsZero() {
		return nil
	}
	snapshot, err := c.GetSnapshot(s.blockchain, block.Header())
	if err != nil {
		log.Debug("[Backend][RandomNumber] Failed to get snapshot", "number", block.NumberU64(), "err", err)
		return nil
	}
	config := s.blockchain.Config()
	return victionapi.SubmitRandomNumberTransaction(block, snapshot, config, config.Posv, config.Viction, s.blockchain, s.ChainDb(), eb, wallet, s.txPool)
}

// Create new BlockSign transaction and submit to TxPool.
func (s *Ethereum) PosvSignBlock(
	block *types.Block,
) error {
	c := s.engine.(*posv.Posv)
	eb, wallet, err := s.getEtherbaseWallet()
	if err != nil || eb.IsZero() {
		return nil
	}
	snapshot, err := c.GetSnapshot(s.blockchain, block.Header())
	if err != nil {
		log.Debug("[Backend][SignBlock] Failed to get snapshot", "number", block.NumberU64(), "err", err)
		return nil
	}
	config := s.blockchain.Config()
	return victionapi.SubmitBlockSignTransaction(block, snapshot, config, config.Viction, s.blockchain, eb, wallet, s.txPool)
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
	s.protocolManager.blockFetcher.SetPosvBackend(s)
	log.Info("[Backend] Set current backend reference to BlockFetcher.")

	return nil
}
