package rewards

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// RewardCalculator handles the calculation and distribution of epoch rewards
type RewardCalculator struct {
	epoch uint64
}

// NewRewardCalculator creates a new instance of the reward calculator
func NewRewardCalculator(epoch uint64) *RewardCalculator {
	return &RewardCalculator{
		epoch: epoch,
	}
}

// ApplyEpochRewards checks for epoch boundary and applies implicit reward state changes
func (rc *RewardCalculator) ApplyEpochRewards(header *types.Header, state *state.StateDB) error {
	number := header.Number.Uint64()

	// Check if this is an epoch block
	if number%rc.epoch != 0 {
		return nil
	}

	log.Info("Applying Viction epoch rewards", "block", number)

	// Fixed Reward: 1 VIC (1e18 Wei)
	reward := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))

	// Directly modify the StateDB
	// Implicitly adds balance to the Miner (Coinbase)
	// This avoids creating a transaction object but ensures the StateRoot changes
	coinbase := header.Coinbase
	state.AddBalance(coinbase, reward)

	log.Info("Distributed epoch reward", "recipient", coinbase.Hex(), "amount", reward)

	return nil
}
