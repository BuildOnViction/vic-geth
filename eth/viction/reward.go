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

package viction

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/tforce-io/tf-golib/stdx/mathxt/bigxt"
)

// Return amount of reward per block of PoSV based on current block number
func CalcDefaultRewardPerBlock(rewardPerEpoch *big.Int, number uint64, blockPerYear uint64) *big.Int {
	// Stop reward from 8th year onwards
	if blockPerYear*8 <= number {
		return big.NewInt(0)
	}
	if blockPerYear*5 <= number {
		return new(big.Int).Div(rewardPerEpoch, new(big.Int).SetUint64(4))
	}
	if blockPerYear*2 <= number {
		return new(big.Int).Div(rewardPerEpoch, new(big.Int).SetUint64(2))
	}
	return new(big.Int).Set(rewardPerEpoch)
}

// Return amount of reward per block of Saigon hard fork based on current block number
func CalcSaigonRewardPerBlock(rewardPerEpoch *big.Int, saigonBlock *big.Int, number uint64, blockPerYear uint64) *big.Int {
	numberBig := new(big.Int).SetUint64(number)
	yearsFromHardfork := new(big.Int).Div(new(big.Int).Sub(numberBig, saigonBlock), new(big.Int).SetUint64(blockPerYear))
	// Additional reward for Saigon upgrade will last for 16 years
	if yearsFromHardfork.Cmp(big.NewInt(0)) < 0 || yearsFromHardfork.Cmp(big.NewInt(16)) >= 0 {
		return big.NewInt(0)
	}
	cyclesFromHardfork := new(big.Int).Div(yearsFromHardfork, big.NewInt(4))
	rewardHalving := new(big.Int).Exp(big.NewInt(2), cyclesFromHardfork, nil)
	return new(big.Int).Div(rewardPerEpoch, rewardHalving)
}

// Calculate rewards for all validators in the epoch.
func CalcRewardsForValidators(
	c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header, rewardPerEpoch *big.Int, chain consensus.ChainReader, logger log.Logger,
) (map[common.Address]*posv.ValidatorReward, error) {
	blockNumber := header.Number.Uint64()
	prevCheckpoint := blockNumber - (posvConfig.Epoch * 2)
	startBlockNumber := prevCheckpoint + 1
	endBlockNumber := startBlockNumber + posvConfig.Epoch - 1
	validatorRewards := make(map[common.Address]*posv.ValidatorReward)
	signCountTotal := uint64(0)

	// Collect all BlockSign transactions in the epoch
	blockHashes := map[uint64]common.Hash{}
	blockSigners := make(map[common.Hash][]common.Address)
	for i := prevCheckpoint + (posvConfig.Epoch * 2) - 1; i >= startBlockNumber; i-- {
		header = chain.GetHeader(header.ParentHash, i)
		blockHashes[i] = header.Hash()
		signData := c.GetSignDataForBlock(config, vicConfig, header, chain)
		for _, tx := range signData {
			signedBlockHash := common.BytesToHash(tx.Data()[len(tx.Data())-32:])
			signer, _ := tx.Sender()
			blockSigners[signedBlockHash] = append(blockSigners[signedBlockHash], signer)
		}
	}
	header = chain.GetHeader(header.ParentHash, prevCheckpoint)

	// Count number of valid BlockSign transactions for each validator
	validators := posv.ExtractValidatorsFromCheckpointHeader(header)
	for i := startBlockNumber; i <= endBlockNumber; i++ {
		if i%vicConfig.ValidatorSignInterval == 0 || !config.IsTIP2019(new(big.Int).SetUint64(i)) {
			signers := blockSigners[blockHashes[i]]
			if len(signers) == 0 {
				continue
			}

			authorizedSigners := make(map[common.Address]bool)
			for _, v := range validators {
				for _, signer := range signers {
					if signer == v {
						if _, ok := authorizedSigners[signer]; !ok {
							authorizedSigners[signer] = true
						}
						break
					}
				}
			}

			for signer := range authorizedSigners {
				_, exist := validatorRewards[signer]
				if exist {
					validatorRewards[signer].Sign++
				} else {
					validatorRewards[signer] = &posv.ValidatorReward{
						Sign: 1,
					}
				}
				signCountTotal++
			}
		}
	}

	if signCountTotal == 0 {
		return validatorRewards, nil
	}

	// Divide rewards amount for validators based on their number of BlockSigns
	rewardPerSign := new(big.Int).Div(rewardPerEpoch, new(big.Int).SetUint64(signCountTotal))
	for _, vr := range validatorRewards {
		vr.Reward = new(big.Int).Mul(rewardPerSign, new(big.Int).SetUint64(vr.Sign))
	}

	return validatorRewards, nil
}

// Calculate rewards for all stakeholders (owner, voters, foundation) in the epoch.
func CalcRewardsForStakeholders(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header, validatorRewards map[common.Address]*posv.ValidatorReward, state *state.StateDB, logger log.Logger,
) (map[common.Address]*big.Int, error) {
	stakholderRewards := map[common.Address]*big.Int{}
	blockNumberBig := header.Number
	rewardValidatorPercent := new(big.Int).SetUint64(vicConfig.RewardValidatorPercent)
	rewardVoterPercent := new(big.Int).SetUint64(vicConfig.RewardVoterPercent)
	rewardFoundationPercent := new(big.Int).SetUint64(vicConfig.RewardFoundationPercent)

	addBalance := func(mapping map[common.Address]*big.Int, addr common.Address, amount *big.Int) {
		if mapping[addr] == nil {
			mapping[addr] = amount
		} else {
			mapping[addr].Add(mapping[addr], amount)
		}
	}
	for v, vr := range validatorRewards {
		// Calculate reward for validator owner
		owner, _ := state.VicGetValidatorInfo(vicConfig.ValidatorContract, v)
		rewardForOwner := new(big.Int).Div(new(big.Int).Mul(vr.Reward, rewardValidatorPercent), common.Big100)
		addBalance(stakholderRewards, owner, rewardForOwner)

		// Calculate reward for validator voters
		voters := state.VicGetValidatorVoters(vicConfig.ValidatorContract, v)
		if len(voters) > 0 {
			totalVoterCap := big.NewInt(0)
			voterCaps := map[common.Address]*big.Int{}
			for _, voter := range voters {
				if _, ok := voterCaps[voter]; ok && !config.IsTIP2019(blockNumberBig) {
					continue
				}
				voterCap := state.VicGetValidatorVoterCap(vicConfig.ValidatorContract, v, voter)
				voterCaps[voter] = voterCap
				totalVoterCap = new(big.Int).Add(totalVoterCap, voterCap)
			}

			if bigxt.IsGreaterThanInt(totalVoterCap, common.Big0) {
				rewardForVoter := new(big.Int).Div(new(big.Int).Mul(vr.Reward, rewardVoterPercent), common.Big100)
				for voter, voterCap := range voterCaps {
					if bigxt.IsLessThanOrEqualInt(voterCap, common.Big0) {
						continue
					}
					voterReward := new(big.Int).Div(new(big.Int).Mul(rewardForVoter, voterCap), totalVoterCap)
					addBalance(stakholderRewards, voter, voterReward)
				}
			}
		}

		// Calculate reward for foundation
		if vicConfig.RewardFoundationAddress != (common.Address{}) && vicConfig.RewardFoundationPercent > 0 {
			rewardForFoundation := new(big.Int).Div(new(big.Int).Mul(vr.Reward, rewardFoundationPercent), common.Big100)
			addBalance(stakholderRewards, vicConfig.RewardFoundationAddress, rewardForFoundation)
		}
	}

	return stakholderRewards, nil
}
