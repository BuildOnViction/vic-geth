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

const (
	yearsUntilHalving1  = 2
	yearsUntilHalving2  = 5
	yearsUntilHalving3  = 8
	halvingDivisor1     = 2
	halvingDivisor2     = 4
	saigonRewardYears   = 16
	saigonCycleYears    = 4
	minTxDataLength     = 32
	percentDivisor      = 100
	significantMismatch = 100
)

func CalcDefaultRewardPerBlock(rewardPerEpoch *big.Int, number uint64, blockPerYear uint64) *big.Int {
	if blockPerYear*yearsUntilHalving3 <= number {
		return big.NewInt(0)
	}
	if blockPerYear*yearsUntilHalving2 <= number {
		return new(big.Int).Div(rewardPerEpoch, big.NewInt(halvingDivisor2))
	}
	if blockPerYear*yearsUntilHalving1 <= number {
		return new(big.Int).Div(rewardPerEpoch, big.NewInt(halvingDivisor1))
	}
	return new(big.Int).Set(rewardPerEpoch)
}

func CalcSaigonRewardPerBlock(rewardPerEpoch *big.Int, saigonBlock *big.Int, number uint64, blockPerYear uint64) *big.Int {
	if saigonBlock == nil {
		return big.NewInt(0)
	}
	numberBig := new(big.Int).SetUint64(number)
	yearsFromHardfork := new(big.Int).Div(
		new(big.Int).Sub(numberBig, saigonBlock),
		new(big.Int).SetUint64(blockPerYear),
	)
	if yearsFromHardfork.Sign() < 0 || yearsFromHardfork.Cmp(big.NewInt(saigonRewardYears)) >= 0 {
		return big.NewInt(0)
	}
	cyclesFromHardfork := new(big.Int).Div(yearsFromHardfork, big.NewInt(saigonCycleYears))
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
			if !tx.IsSigningTransaction(blockSignAddr) {
				continue
			}
			txData := tx.Data()
			if len(txData) < minTxDataLength {
				continue
			}
			signedBlockHash := common.BytesToHash(txData[len(txData)-minTxDataLength:])
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

	big100 := big.NewInt(percentDivisor)
	big0 := big.NewInt(0)

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

		validatorRewardTotal := new(big.Int).Set(vr.Reward)
		distributedTotal := big.NewInt(0)

		owner, _ := statedb.VicGetValidatorInfo(vicConfig.ValidatorContract, validator)
		rewardForOwner := new(big.Int).Mul(vr.Reward, new(big.Int).SetUint64(rewardValidatorPercent))
		rewardForOwner.Div(rewardForOwner, big100)
		addBalance(stakeholderRewards, owner, rewardForOwner)
		distributedTotal.Add(distributedTotal, rewardForOwner)

		voters := statedb.VicGetValidatorVoters(vicConfig.ValidatorContract, validator)
		voterRewardDistributed := big.NewInt(0)
		if len(voters) > 0 {
			totalVoterReward := new(big.Int).Mul(vr.Reward, new(big.Int).SetUint64(rewardVoterPercent))
			totalVoterReward.Div(totalVoterReward, big100)
			totalCap := big.NewInt(0)
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

			if totalCap.Cmp(big0) > 0 {
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
			rewardForFoundation.Div(rewardForFoundation, big100)
			addBalance(stakeholderRewards, vicConfig.RewardFoundationAddress, rewardForFoundation)
			distributedTotal.Add(distributedTotal, rewardForFoundation)
		}

		if distributedTotal.Cmp(validatorRewardTotal) != 0 {
			missing := new(big.Int).Sub(validatorRewardTotal, distributedTotal)
			if missing.Cmp(big.NewInt(significantMismatch)) > 0 {
				logger.Warn("CalcRewardsForStakeholders: significant reward distribution mismatch", "validator", validator.Hex(), "totalReward", validatorRewardTotal.String(), "distributed", distributedTotal.String(), "missing", missing.String())
			}
		}
	}

	return stakeholderRewards, nil
}
