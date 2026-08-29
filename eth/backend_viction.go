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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/viction"
)

const recentBlockSigners = 10500 // Number of recent block sign transactions to keep in memory

// Get attestors from list of validators.
func (s *Ethereum) PosvGetAttestors(
	vicConfig *params.VictionConfig, header *types.Header, validators []common.Address,
) ([]int64, error) {
	state, err := s.blockchain.State()
	if err != nil {
		return nil, err
	}
	return viction.GetAttestors(vicConfig, validators, state)
}

// Get creator-attestor pairs from the state.
func (s *Ethereum) PosvGetCreatorAttestorPairs(
	config *params.ChainConfig, posvConfig *params.PosvConfig, victionConfig *params.VictionConfig, header, checkpointHeader *types.Header,
) (map[common.Address]common.Address, uint64, error) {
	number := header.Number.Uint64()
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkpointHeader)
	attestorIdxs := posv.ExtractAttestorsFromCheckpointHeader(checkpointHeader)
	pairs, offset, err := viction.GetCreatorAttestorPairs(config, posvConfig, number, validators, attestorIdxs)
	if err != viction.ErrInvalidAttestorList {
		// Either success or a non-recoverable error — propagate as-is.
		return pairs, offset, err
	}

	// Try to rebuild create-attestor pairs from state.
	if checkpointHeader == nil {
		return nil, 0, err
	}
	checkpointNumber := checkpointHeader.Number.Uint64()
	stateAtCheckpoint, err2 := s.blockchain.StateAt(checkpointHeader.Root)
	if err2 != nil {
		log.Warn("[Backend][GetCreatorAttestorPairs] fallback: failed to get state at checkpoint", "checkpoint", checkpointNumber, "err", err2)
		return nil, 0, err
	}
	attestorIdxs, err2 = viction.GetAttestors(victionConfig, validators, stateAtCheckpoint)
	if err2 != nil {
		log.Warn("[Backend][GetCreatorAttestorPairs] fallback: cannot compute attestors", "checkpoint", checkpointNumber, "err", err2)
		return nil, 0, err
	}
	log.Warn("[Backend][GetCreatorAttestorPairs] rebuild creator-attestor pairs from state", "checkpoint", checkpointNumber, "number", number)
	return viction.GetCreatorAttestorPairs(config, posvConfig, number, validators, attestorIdxs)
}

// Calculate rewards at the end of each epoch.
func (s *Ethereum) PosvGetEpochRewards(
	c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader, statedb *state.StateDB, logger log.Logger,
) (*posv.EpochReward, error) {
	number := header.Number.Uint64()
	// Use parent state for voter caps
	preCheckpointHeader := chain.GetHeader(header.ParentHash, number-1)
	preCheckpointState, err := s.BlockChain().StateAt(preCheckpointHeader.Root)
	if err != nil {
		logger.Warn("[Backend][GetEpochReward]: failed to get preCheckpoint state", "number", number-1, "hash", header.ParentHash, "err", err)
		return nil, err
	}

	return viction.CalcRewardPerEpoch(config, posvConfig, vicConfig, header, chain, preCheckpointState, s.blockchain, s.blockSignersCache, logger)
}

// Add balance rewards to the state.
func (s *Ethereum) PosvDistributeEpochRewards(
	header *types.Header, state *state.StateDB, epochReward *posv.EpochReward,
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
	c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader, validators []common.Address,
) ([]common.Address, error) {
	if config.IsTIPSigning(header.Number) {
		return viction.PenalizeValidatorsTIPSigning(config, posvConfig, vicConfig, header, validators, chain, s.BlockChain(), s.blockSignersCache)
	}
	return viction.PenalizeValidatorsDefault(config, posvConfig, vicConfig, header, chain, s.BlockChain())
}

// Get eligble validators from the state.
func (s *Ethereum) PosvGetValidators(
	config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader,
) ([]common.Address, error) {
	if header == nil {
		return []common.Address{}, viction.ErrNilHeader
	}
	state, err := s.blockchain.StateAt(header.Root)
	if err != nil {
		return nil, err
	}
	return viction.GetValidators(config, vicConfig, header, state)
}

func (eth *Ethereum) setupPosvBackend(chainConfig *params.ChainConfig) error {
	if chainConfig.Posv == nil {
		return nil
	}

	posvEngine, ok := eth.engine.(*posv.Posv)
	if !ok {
		return fmt.Errorf("posv config present but engine is %T, expected *posv.Posv", eth.engine)
	}

	posvEngine.SetBackend(eth)
	log.Info("[Backend] Set current backend reference to PoSV engine.")

	return nil
}
