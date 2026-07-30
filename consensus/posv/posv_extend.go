// Copyright 2017 The go-ethereum Authors
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

package posv

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// EpochReward stores number of sign made by each validator and rewards for
// all stakeholders (validators and voters) in an epoch.
type EpochReward struct {
	Rewards            map[common.Address]map[common.Address]*big.Int `json:"rewards"`
	ValidatorRewards   map[common.Address]*ValidatorReward            `json:"signers"`
	StakeholderRewards map[common.Address]*big.Int                    `json:"-"`
}

// EpochReward stores the rewards of a validator.
type ValidatorReward struct {
	Sign   uint64   `json:"sign"`
	Reward *big.Int `json:"reward"`
}

// ValidatorInfo stores basic information about a validator.
type ValidatorInfo struct {
	Address  common.Address `json:"address"`
	Capacity *big.Int       `json:"capacity"`
	Owner    common.Address `json:"owner"`
}

type PosvBackend interface {
	// Get attestors from list of validators.
	PosvGetAttestors(
		vicConfig *params.VictionConfig, header *types.Header, validators []common.Address,
	) ([]int64, error)

	// Get block signers from the state.
	PosvGetBlockSignData(
		config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
		chain consensus.ChainReader,
	) ([]types.Transaction, error)

	// Get creator-attestor pairs from the state.
	PosvGetCreatorAttestorPairs(
		c *Posv, config *params.ChainConfig, header, checkpointHeader *types.Header,
	) (map[common.Address]common.Address, uint64, error)

	// Calculate and distribute reward at the end of each epoch.
	PosvGetEpochReward(
		c *Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header,
		chain consensus.ChainReader, state *state.StateDB, logger log.Logger,
	) (*EpochReward, error)

	// Add balance rewards to the state (apply the rewards returned by PosvGetEpochReward).
	PosvDistributeEpochRewards(
		header *types.Header, state *state.StateDB, epochReward *EpochReward,
	) error

	// Penalize validators for creating bad block or not creating block at all.
	PosvGetPenalties(
		c *Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header,
		chain consensus.ChainReader, validators []common.Address,
	) ([]common.Address, error)

	// Get eligble validators from the state.
	PosvGetValidators(
		vicConfig *params.VictionConfig, header *types.Header,
		chain consensus.ChainReader,
	) ([]common.Address, error)
}

// Process block header Extra field of a checkpoint block to return the list of new validators.
func ExtractValidatorsFromCheckpointHeader(header *types.Header) []common.Address {
	if header == nil {
		return []common.Address{}
	}

	validators := make([]common.Address, (len(header.Extra)-extraVanity-extraSeal)/int(common.AddressLength))
	for i := 0; i < len(validators); i++ {
		copy(validators[i][:], header.Extra[extraVanity+i*int(common.AddressLength):])
	}

	return validators
}

// Get all BlockSign transactions for a given block. If it's not cached yet, get it from the state.
func (c *Posv) GetSignDataForBlock(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader) ([]types.Transaction, error) {
	if header == nil {
		return nil, fmt.Errorf("GetSignDataForBlock: header is nil")
	}
	blockHash := header.Hash()
	if signers, ok := c.BlockSigners.Get(blockHash); ok {
		if signers, ok := signers.([]types.Transaction); ok && signers != nil {
			return signers, nil
		}
	}
	signers, err := c.backend.PosvGetBlockSignData(config, vicConfig, header, chain)
	if err != nil {
		return nil, err
	}
	c.BlockSigners.Add(blockHash, signers)
	return signers, nil
}
