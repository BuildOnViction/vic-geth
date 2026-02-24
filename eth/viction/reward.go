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

// [TO-DO]
// Return amount of reward per block of PoSV based on current block number
func CalcDefaultRewardPerBlock(rewardPerEpoch *big.Int, number uint64, blockPerYear uint64) *big.Int {
	return nil
}

// [TO-DO]
// Return amount of reward per block of Saigon hard fork based on current block number
func CalcSaigonRewardPerBlock(rewardPerEpoch *big.Int, saigonBlock *big.Int, number uint64, blockPerYear uint64) *big.Int {
	return nil
}

// [TO-DO]
// Calculate rewards for all validators in the epoch.
func CalcRewardsForValidators(
	c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header, rewardPerEpoch *big.Int, chain consensus.ChainReader, logger log.Logger,
) (map[common.Address]*posv.ValidatorReward, error) {

	return nil, nil
}

// [TO-DO]
// Calculate rewards for all stakeholders (owner, voters, foundation) in the epoch.
func CalcRewardsForStakeholders(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header, validatorRewards map[common.Address]*posv.ValidatorReward, state *state.StateDB, logger log.Logger,
) (map[common.Address]*big.Int, error) {

	return nil, nil
}
