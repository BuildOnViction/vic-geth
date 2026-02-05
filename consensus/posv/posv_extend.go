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

package posv

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// EpochReward stores number of sign made by each validator and rewards for
// all stakeholders (validators and voters) in an epoch.
type EpochReward struct {
	ValidatorRewards  map[common.Address]*ValidatorReward `json:"signers"`
	StakholderRewards map[common.Address]*big.Int         `json:"rewards"`
}

// ValidatorInfo stores basic information about a validator.
type ValidatorInfo struct {
	Address  common.Address `json:"address"`
	Capacity *big.Int       `json:"capacity"`
	Owner    common.Address `json:"owner"`
}

type ValidatorReward struct {
	Sign   uint64   `json:"sign"`
	Reward *big.Int `json:"reward"`
}

type PosvBackend interface {
	// Get attestors from list of validators.
	PosvGetAttestors(vicConfig *params.VictionConfig, header *types.Header, validators []common.Address,
	) ([]int64, error)

	// Get block signers from the state.
	PosvGetBlockSignData(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
		chain consensus.ChainReader,
	) []types.Transaction

	// Get creator-attestor pairs from the state.
	PosvGetCreatorAttestorPairs(c *Posv, config *params.ChainConfig,
		header, checkpointHeader *types.Header,
	) (map[common.Address]common.Address, uint64, error)

	// Calculate and distribute reward at the end of each epoch.
	PosvGetEpochReward(c *Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
		header *types.Header,
		chain consensus.ChainReader, logger log.Logger,
	) (*EpochReward, error)

	// Penalize validators for creating bad block or not creating block at all.
	PosvGetPenalties(c *Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
		header *types.Header,
		chain consensus.ChainReader,
	) ([]common.Address, error)

	// Get eligble validators from the state.
	PosvGetValidators(vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader,
	) ([]common.Address, error)
}
