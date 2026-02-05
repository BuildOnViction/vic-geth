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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Get attestors from list of validators.

func (s *Ethereum) PosvGetAttestors(vicConfig *params.VictionConfig, header *types.Header, validators []common.Address,
) ([]int64, error) {
	panic("not implemented")
}

// Get block signers from the state.

func (s *Ethereum) PosvGetBlockSignData(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader,
) []types.Transaction {
	panic("not implemented")
}

// Get creator-attestor pairs from the state.

func (s *Ethereum) PosvGetCreatorAttestorPairs(c *posv.Posv, config *params.ChainConfig,
	header, checkpointHeader *types.Header,
) (map[common.Address]common.Address, uint64, error) {
	panic("not implemented")
}

// Calculate and distribute reward at the end of each epoch.

func (s *Ethereum) PosvGetEpochReward(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader, logger log.Logger,
) (*posv.EpochReward, error) {
	panic("not implemented")
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
