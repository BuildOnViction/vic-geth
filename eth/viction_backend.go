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
)

// [TO-DO] PosvGetAttestors returns attestors encoded as bytes for the header.
func (s *Ethereum) PosvGetAttestors(vicConfig params.VictionConfig, header *types.Header, validators []common.Address) ([]int64, error) {
	return nil, nil
}

// [TO-DO] PosvGetBlockSignData returns block sign transactions for the given header.
func (s *Ethereum) PosvGetBlockSignData(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader) []types.Transaction {
	return []types.Transaction{}
}

// [TO-DO] PosvGetCreatorAttestorPairs returns creator-attestor pairs for double validation.
func (s *Ethereum) PosvGetCreatorAttestorPairs(c *posv.Posv, config *params.ChainConfig, header, checkpointHeader *types.Header) (map[common.Address]common.Address, uint64, error) {
	return make(map[common.Address]common.Address), 0, nil
}

// PosvGetEpochReward calculates and distributes reward at checkpoint block.
func (s *Ethereum) PosvGetEpochReward(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader, statedb *state.StateDB, logger log.Logger,
) (*posv.EpochReward, error) {
	epochRewards := &posv.EpochReward{}
	blockNumber := header.Number.Uint64()

	// Skip block 900 (1*epoch); first reward at block 1800 (2*epoch)
	if blockNumber <= posvConfig.Epoch {
		return epochRewards, nil
	}

	// Get initial reward
	initialRewardPerEpoch := (*big.Int)(vicConfig.RewardPerEpoch)
	totalReward := viction.CalcDefaultRewardPerBlock(initialRewardPerEpoch, blockNumber, posvConfig.BlocksPerYear())

	// Get additional reward for Saigon upgrade
	if config.IsSaigon(header.Number) && vicConfig.SaigonRewardPerEpoch != nil {
		saigonRewardPerEpoch := (*big.Int)(vicConfig.SaigonRewardPerEpoch)
		saigonReward := viction.CalcSaigonRewardPerBlock(saigonRewardPerEpoch, config.SaigonBlock, blockNumber, posvConfig.BlocksPerYear())
		totalReward = new(big.Int).Add(totalReward, saigonReward)
	}

	// Calculate rewards for validators and stakeholders
	validatorRewards, err := viction.CalcRewardsForValidators(c, config, posvConfig, vicConfig, header, totalReward, chain, logger)
	if err != nil {
		return nil, err
	}
	epochRewards.ValidatorRewards = validatorRewards

	stakeholderRewards, err := viction.CalcRewardsForStakeholders(c, config, posvConfig, vicConfig, header, validatorRewards, statedb, logger)
	if err != nil {
		return nil, err
	}
	epochRewards.StakholderRewards = stakeholderRewards

	return epochRewards, nil
}

// PosvAddBalanceRewards applies epoch rewards to the state by adding balances to all stakeholders.
// It does NOT recalculate; caller should pass the epochReward returned by PosvGetEpochReward.
func (s *Ethereum) PosvDistributeEpochRewards(header *types.Header, state *state.StateDB, epochReward *posv.EpochReward) error {
	blockNumber := header.Number.Uint64()

	if epochReward == nil {
		log.Debug("PosvAddBalanceRewards: no epoch rewards to apply", "block", blockNumber)
		return nil
	}
	if state == nil {
		return nil
	}

	// Apply stakeholder rewards to the state
	totalRewardDistributed := big.NewInt(0)
	rewardCount := 0

	for addr, amount := range epochReward.StakholderRewards {
		if amount == nil || amount.Sign() <= 0 {
			continue
		}
		state.AddBalance(addr, amount)
		totalRewardDistributed.Add(totalRewardDistributed, amount)
		rewardCount++
	}

	log.Info("PosvAddBalanceRewards: applied epoch rewards", "block", blockNumber, "recipientCount", rewardCount, "totalReward", totalRewardDistributed.String())
	return nil
}

// [TO-DO] PosvGetPenalties returns list of penalized validators.
func (s *Ethereum) PosvGetPenalties(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader) ([]common.Address, error) {
	return []common.Address{}, nil
}

// [TO-DO] PosvGetValidators returns list of eligible validators from the state.
func (s *Ethereum) PosvGetValidators(vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader) ([]common.Address, error) {
	return nil, nil
}
