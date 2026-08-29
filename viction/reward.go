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

package viction

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru"
)

// Return reward amount per block based on current block number, including default and Saigon rules.
func CalcRewardPerEpoch(
	config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader, preCheckpointState *state.StateDB, bc *core.BlockChain, blkSigCache *lru.ARCCache, logger log.Logger,
) (*posv.EpochReward, error) {
	epochRewards := &posv.EpochReward{}
	number := header.Number.Uint64()

	// First epoch won't include any rewards
	if !posvConfig.IsCheckpointBlock(number) || number <= posvConfig.Epoch {
		return epochRewards, nil
	}

	// Get initial reward
	initialRewardPerEpoch := (*big.Int)(vicConfig.RewardPerEpoch)
	totalReward := CalcDefaultRewardPerBlock(initialRewardPerEpoch, number, posvConfig.BlocksPerYear())

	// Get additional reward for Saigon upgrade
	if config.IsSaigon(header.Number) && vicConfig.SaigonRewardPerEpoch != nil {
		saigonRewardPerEpoch := (*big.Int)(vicConfig.SaigonRewardPerEpoch)
		saigonReward := CalcSaigonRewardPerBlock(saigonRewardPerEpoch, config.SaigonBlock, number, posvConfig.BlocksPerYear())
		totalReward = new(big.Int).Add(totalReward, saigonReward)
	}

	// Calculate rewards for validators and stakeholders
	validatorRewards, err := CalcRewardsForValidators(config, posvConfig, vicConfig, header, totalReward, chain, bc, blkSigCache, logger)
	if err != nil {
		return nil, err
	}
	epochRewards.ValidatorRewards = validatorRewards

	stakeholderRewards, nestedRewards, err := CalcRewardsForStakeholders(config, posvConfig, vicConfig, header, validatorRewards, preCheckpointState, logger)
	if err != nil {
		return nil, err
	}
	epochRewards.StakeholderRewards = stakeholderRewards
	epochRewards.Rewards = nestedRewards

	return epochRewards, nil
}

// Return reward amount per block following default rule based on current block number.
func CalcDefaultRewardPerBlock(rewardPerEpoch *big.Int, number uint64, blockPerYear uint64) *big.Int {
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

// Return reward amount per block following Saigon rule based on current block number.
func CalcSaigonRewardPerBlock(rewardPerEpoch *big.Int, saigonBlock *big.Int, number uint64, blockPerYear uint64) *big.Int {
	numberBig := new(big.Int).SetUint64(number)
	yearsFromHardfork := new(big.Int).Div(new(big.Int).Sub(numberBig, saigonBlock), new(big.Int).SetUint64(blockPerYear))
	// Additional reward for Saigon will last for 16 years
	if yearsFromHardfork.Cmp(common.Big0) < 0 || yearsFromHardfork.Cmp(big.NewInt(16)) >= 0 {
		return common.Big0
	}
	cyclesFromHardfork := new(big.Int).Div(yearsFromHardfork, big.NewInt(4))
	rewardHalving := new(big.Int).Exp(big.NewInt(2), cyclesFromHardfork, nil)
	return new(big.Int).Div(rewardPerEpoch, rewardHalving)
}

// Return reward amount for all validators in a given epoch.
func CalcRewardsForValidators(
	config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header, rewardPerEpoch *big.Int,
	chain consensus.ChainReader, bc *core.BlockChain, blkSigCache *lru.ARCCache, logger log.Logger,
) (map[common.Address]*posv.ValidatorReward, error) {
	blockNumber := header.Number.Uint64()
	prevCheckpoint := blockNumber - (posvConfig.Epoch * 2)
	startBlockNumber := prevCheckpoint + 1
	endBlockNumber := startBlockNumber + posvConfig.Epoch - 1
	validatorRewards := make(map[common.Address]*posv.ValidatorReward)
	signCountTotal := uint64(0)

	blockHashes := map[uint64]common.Hash{}
	signersByBlockHash := make(map[common.Hash][]common.Address)
	h := header
	for i := prevCheckpoint + (posvConfig.Epoch * 2) - 1; i >= startBlockNumber; i-- {
		h = chain.GetHeader(h.ParentHash, i)
		if h == nil {
			break
		}
		blockHashes[i] = h.Hash()

		// Use GetBlockSignData so that pre-TIPSigning blocks are filtered by receipt status
		txs, err := GetBlockSignData(config, vicConfig, h, chain, bc, blkSigCache)
		if err != nil {
			return nil, err
		}
		signer := types.MakeSigner(config, h.Number)
		for _, tx := range txs {
			txData := tx.Data()
			if len(txData) < common.HashLength {
				continue
			}
			signedBlockHash := common.BytesToHash(txData[len(txData)-common.HashLength:])
			msg, err := tx.AsMessage(signer, nil)
			if err != nil {
				continue
			}
			signersByBlockHash[signedBlockHash] = append(signersByBlockHash[signedBlockHash], msg.From())
		}
	}

	prevHeader := chain.GetHeader(h.ParentHash, prevCheckpoint)
	if prevHeader == nil {
		return validatorRewards, nil
	}
	validators := posv.ExtractValidatorsFromCheckpointHeader(prevHeader)

	for i := startBlockNumber; i <= endBlockNumber; i++ {
		if i%vicConfig.ValidatorSignInterval == 0 || !config.IsTIP2019(new(big.Int).SetUint64(i)) {
			signers := signersByBlockHash[blockHashes[i]]
			if len(signers) == 0 {
				continue
			}

			authorizedSigners := make(map[common.Address]bool)
			for _, v := range validators {
				for _, signer := range signers {
					if signer == v {
						authorizedSigners[signer] = true
						break
					}
				}
			}

			for signer := range authorizedSigners {
				if vr, exist := validatorRewards[signer]; exist {
					vr.Sign++
				} else {
					validatorRewards[signer] = &posv.ValidatorReward{
						Sign:   1,
						Reward: new(big.Int),
					}
				}
				signCountTotal++
			}
		}
	}

	if signCountTotal == 0 {
		return validatorRewards, nil
	}

	rewardPerSign := new(big.Int).Div(rewardPerEpoch, new(big.Int).SetUint64(signCountTotal))
	for _, vr := range validatorRewards {
		vr.Reward = new(big.Int).Mul(rewardPerSign, new(big.Int).SetUint64(vr.Sign))
	}

	return validatorRewards, nil
}

// Return reward amount for all stakeholders in a given epoch.
func CalcRewardsForStakeholders(
	config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header, validatorRewards map[common.Address]*posv.ValidatorReward,
	statedb *state.StateDB, logger log.Logger,
) (map[common.Address]*big.Int, map[common.Address]map[common.Address]*big.Int, error) {
	stakeholderRewards := make(map[common.Address]*big.Int)
	nestedRewards := make(map[common.Address]map[common.Address]*big.Int)
	blockNumber := header.Number.Uint64()
	rewardValidatorPercent := vicConfig.RewardValidatorPercent
	rewardVoterPercent := vicConfig.RewardVoterPercent
	rewardFoundationPercent := vicConfig.RewardFoundationPercent

	addBalance := func(mapping map[common.Address]*big.Int, addr common.Address, amount *big.Int) {
		if mapping[addr] == nil {
			mapping[addr] = amount
		} else {
			mapping[addr].Add(mapping[addr], amount)
		}
	}

	for validator, vr := range validatorRewards {
		if vr == nil || vr.Reward == nil || vr.Reward.Sign() <= 0 {
			continue
		}

		distributedTotal := new(big.Int)
		validatorNested := make(map[common.Address]*big.Int)

		owner, _ := statedb.VicGetValidatorInfo(vicConfig.ValidatorContract, validator)
		rewardForOwner := new(big.Int).Mul(vr.Reward, new(big.Int).SetUint64(rewardValidatorPercent))
		rewardForOwner.Div(rewardForOwner, common.Big100)
		addBalance(stakeholderRewards, owner, rewardForOwner)
		addBalance(validatorNested, owner, new(big.Int).Set(rewardForOwner))
		distributedTotal.Add(distributedTotal, rewardForOwner)

		voters := statedb.VicGetValidatorVoters(vicConfig.ValidatorContract, validator)
		voterRewardDistributed := new(big.Int)
		if len(voters) > 0 {
			totalVoterReward := new(big.Int).Mul(vr.Reward, new(big.Int).SetUint64(rewardVoterPercent))
			totalVoterReward.Div(totalVoterReward, common.Big100)
			totalCap := new(big.Int)
			voterCaps := make(map[common.Address]*big.Int)

			tip2019Block := config.TIP2019Block
			for _, voteAddr := range voters {
				if _, ok := voterCaps[voteAddr]; ok && tip2019Block != nil && tip2019Block.Uint64() <= blockNumber {
					continue
				}
				voterCap := statedb.VicGetValidatorVoterCap(vicConfig.ValidatorContract, validator, voteAddr)
				totalCap.Add(totalCap, voterCap)
				voterCaps[voteAddr] = voterCap
			}

			if totalCap.Cmp(common.Big0) > 0 {
				for addr, voteCap := range voterCaps {
					if voteCap == nil || voteCap.Sign() <= 0 {
						continue
					}
					rcap := new(big.Int).Mul(totalVoterReward, voteCap)
					rcap.Div(rcap, totalCap)
					addBalance(stakeholderRewards, addr, rcap)
					addBalance(validatorNested, addr, new(big.Int).Set(rcap))
					voterRewardDistributed.Add(voterRewardDistributed, rcap)
				}
			}
		}
		distributedTotal.Add(distributedTotal, voterRewardDistributed)

		if vicConfig.RewardFoundationAddress != (common.Address{}) && rewardFoundationPercent > 0 {
			rewardForFoundation := new(big.Int).Mul(vr.Reward, new(big.Int).SetUint64(rewardFoundationPercent))
			rewardForFoundation.Div(rewardForFoundation, common.Big100)
			addBalance(stakeholderRewards, vicConfig.RewardFoundationAddress, rewardForFoundation)
			addBalance(validatorNested, vicConfig.RewardFoundationAddress, new(big.Int).Set(rewardForFoundation))
			distributedTotal.Add(distributedTotal, rewardForFoundation)
		}

		nestedRewards[validator] = validatorNested
	}

	return stakeholderRewards, nestedRewards, nil
}

// Add balance rewards to the state.
func DistributeStakeholderRewards(state *state.StateDB, epochReward *posv.EpochReward) (*big.Int, int) {
	rewardAmount := big.NewInt(0)
	stakeholderCount := 0
	for addr, amount := range epochReward.StakeholderRewards {
		if amount == nil || amount.Sign() <= 0 {
			continue
		}
		state.AddBalance(addr, amount)
		rewardAmount.Add(rewardAmount, amount)
		stakeholderCount++
	}
	return rewardAmount, stakeholderCount
}
