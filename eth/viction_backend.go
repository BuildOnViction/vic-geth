// Copyright 2025 The Viction Authors
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

// Package eth implements the Ethereum protocol.
package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/viction"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/tforce-io/tf-golib/stdx/mathxt/bigxt"
)

// Get attestors from list of validators.
func (s *Ethereum) PosvGetAttestors(vicConfig *params.VictionConfig, header *types.Header, validators []common.Address,
	logger log.Logger,
) ([]int64, error) {
	state, err := s.blockchain.State()
	if err != nil {
		return []int64{}, err
	}
	return viction.GetAttestors(vicConfig, validators, state, logger)
}

// Get block signers from the state.

func (s *Ethereum) PosvGetBlockSignData(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader,
) []*types.Transaction {
	blockNumber := header.Number.Uint64()
	blockNumberBig := header.Number
	block := chain.GetBlock(header.Hash(), blockNumber)
	data := []*types.Transaction{}
	transactions := block.Transactions()
	if config.IsTIPSigning(blockNumberBig) {
		for _, tx := range transactions {
			if IsVicBlockSingingTx(tx, vicConfig) {
				data = append(data, tx)
			}
		}
	} else {
		// TODO: Handle receipts later
		for _, tx := range transactions {
			if IsVicBlockSingingTx(tx, vicConfig) {
				data = append(data, tx)
			}
		}
	}
	return data
}

// Get creator-attestor pairs from the state.

func (s *Ethereum) PosvGetCreatorAttestorPairs(c *posv.Posv, config *params.ChainConfig,
	header, checkpointHeader *types.Header,
) (map[common.Address]common.Address, uint64, error) {
	panic("not implemented")
}

// Calculate and distribute reward at the end of each epoch.

func (s *Ethereum) PosvGetEpochReward(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header, state *state.StateDB,
	chain consensus.ChainReader, logger log.Logger,
) (*posv.EpochReward, error) {
	epochRewards := &posv.EpochReward{}
	blockNumber := header.Number.Uint64()
	blockNumberBig := header.Number

	if bigxt.IsLessThanOrEqualInt(blockNumberBig, new(big.Int).SetUint64(posvConfig.Epoch)) {
		return epochRewards, nil
	}

	// Get initial reward
	totalReward := viction.CalcDefaultRewardPerBlock((*big.Int)(vicConfig.RewardPerEpoch), blockNumber, posvConfig.BlocksPerYear())
	// Get additional reward for Saigon upgrade
	if chain.Config().IsSaigon(blockNumberBig) {
		saigonReward := viction.CalcSaigonRewardPerBlock((*big.Int)(vicConfig.SaigonRewardPerEpoch), chain.Config().SaigonBlock, blockNumber, posvConfig.BlocksPerYear())
		totalReward = new(big.Int).Add(totalReward, saigonReward)
	}

	// Calculate rewards for validators and stakeholders
	validatorRewards, _ := viction.CalcRewardsForValidators(c, config, posvConfig, vicConfig, header, totalReward, chain, logger)
	epochRewards.ValidatorRewards = validatorRewards

	stakeholderRewards, _ := viction.CalcRewardsForStakeholders(c, config, posvConfig, vicConfig, header, validatorRewards, state, logger)
	epochRewards.StakholderRewards = stakeholderRewards

	return epochRewards, nil
}

// Penalize validators for creating bad block or not creating block at all.

func (s *Ethereum) PosvGetPenalties(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader,
) ([]common.Address, error) {
	panic("not implemented")
}

// Get eligble validators from the state.

func (s *Ethereum) PosvGetValidators(vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader,
) ([]common.Address, error) {
	panic("not implemented")
}

// Check a transaction is Viction BlockSign transaction.
func IsVicBlockSingingTx(tx *types.Transaction, vicConfig *params.VictionConfig) bool {
	toAddr := tx.To()
	if toAddr == nil || *toAddr != vicConfig.ValidatorBlockSignContract {
		return false
	}

	data := tx.Data()
	method := common.Bytes2Hex(data[0:4])

	if method != state.SignMethodHex && len(data) >= 68 {
		return false
	}

	return true
}
