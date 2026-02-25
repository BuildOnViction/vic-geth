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
)

func CalcDefaultRewardPerBlock(rewardPerEpoch *big.Int, number uint64, blockPerYear uint64) *big.Int {
	// vicConfig is passed for future extensibility, currently using hard-coded values
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
	if yearsFromHardfork.Cmp(common.Big0) < 0 || yearsFromHardfork.Cmp(big.NewInt(16)) >= 0 {
		return common.Big0
	}
	cyclesFromHardfork := new(big.Int).Div(yearsFromHardfork, big.NewInt(4))
	rewardHalving := new(big.Int).Exp(big.NewInt(2), cyclesFromHardfork, nil)
	return new(big.Int).Div(rewardPerEpoch, rewardHalving)
}

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

	blockHashes := map[uint64]common.Hash{}
	blockSigners := make(map[common.Hash][]common.Address)
	h := header
	for i := prevCheckpoint + (posvConfig.Epoch * 2) - 1; i >= startBlockNumber; i-- {
		h = chain.GetHeader(h.ParentHash, i)
		if h == nil {
			break
		}
		blockHashes[i] = h.Hash()

		block := chain.GetBlock(h.Hash(), i)
		if block == nil {
			continue
		}
		blockSignAddr := vicConfig.ValidatorBlockSignContract
		for _, tx := range block.Transactions() {
			if !IsSigningTransaction(tx, blockSignAddr) {
				continue
			}
			txData := tx.Data()
			if len(txData) < common.HashLength {
				continue
			}
			signedBlockHash := common.BytesToHash(txData[len(txData)-common.HashLength:])
			signer := types.MakeSigner(config, h.Number)
			msg, err := tx.AsMessage(signer)
			if err != nil {
				logger.Debug("CalcRewardsForValidators: failed to get sender", "txHash", tx.Hash().Hex(), "err", err)
				continue
			}
			blockSigners[signedBlockHash] = append(blockSigners[signedBlockHash], msg.From())
		}
	}

	prevHeader := chain.GetHeader(h.ParentHash, prevCheckpoint)
	if prevHeader == nil {
		return validatorRewards, nil
	}
	validators := posv.ExtractValidatorsFromCheckpointHeader(prevHeader)

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

func CalcRewardsForStakeholders(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header, validatorRewards map[common.Address]*posv.ValidatorReward, statedb *state.StateDB, logger log.Logger,
) (map[common.Address]*big.Int, error) {
	stakeholderRewards := make(map[common.Address]*big.Int)
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

		// 		validatorRewardTotal := new(big.Int).Set(vr.Reward)
		distributedTotal := new(big.Int)

		owner, _ := statedb.VicGetValidatorInfo(vicConfig.ValidatorContract, validator)
		rewardForOwner := new(big.Int).Mul(vr.Reward, new(big.Int).SetUint64(rewardValidatorPercent))
		rewardForOwner.Div(rewardForOwner, common.Big100)
		addBalance(stakeholderRewards, owner, rewardForOwner)
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
					voterRewardDistributed.Add(voterRewardDistributed, rcap)
				}
			}
		}
		distributedTotal.Add(distributedTotal, voterRewardDistributed)

		if vicConfig.RewardFoundationAddress != (common.Address{}) && rewardFoundationPercent > 0 {
			rewardForFoundation := new(big.Int).Mul(vr.Reward, new(big.Int).SetUint64(rewardFoundationPercent))
			rewardForFoundation.Div(rewardForFoundation, common.Big100)
			addBalance(stakeholderRewards, vicConfig.RewardFoundationAddress, rewardForFoundation)
			distributedTotal.Add(distributedTotal, rewardForFoundation)
		}

		// if distributedTotal.Cmp(validatorRewardTotal) != 0 {
		// 	missing := new(big.Int).Sub(validatorRewardTotal, distributedTotal)
		// 	if missing.Cmp(big.NewInt(100)) > 0 {
		// 		logger.Warn("CalcRewardsForStakeholders: significant reward distribution mismatch", "validator", validator.Hex(), "totalReward", validatorRewardTotal.String(), "distributed", distributedTotal.String(), "missing", missing.String())
		// 	}
		// }
	}

	return stakeholderRewards, nil
}
