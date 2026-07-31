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
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/sortlgc"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Return addresses of eligble validators from the state.
func GetValidators(
	config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader, stateDB *state.StateDB,
) ([]common.Address, error) {
	contracrAddress := vicConfig.ValidatorContract
	if contracrAddress == (common.Address{}) {
		return []common.Address{}, ErrNoContractAddress
	}
	addresses := stateDB.VicGetCandidates(contracrAddress)
	candidates := []*posv.ValidatorInfo{}
	for _, addr := range addresses {
		if addr == (common.Address{}) {
			continue
		}
		_, cap := stateDB.VicGetValidatorInfo(contracrAddress, addr)
		candidates = append(candidates, &posv.ValidatorInfo{Address: addr, Capacity: cap})
	}

	if config.IsAtlas(header.Number) {
		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[i].Capacity.Cmp(candidates[j].Capacity) >= 0
		})
	} else {
		sortlgc.Slice(candidates, func(i, j int) bool {
			return candidates[i].Capacity.Cmp(candidates[j].Capacity) >= 0
		})
	}

	validatorMaxCountInt := int(vicConfig.ValidatorMaxCount)
	if len(candidates) > validatorMaxCountInt {
		candidates = candidates[:validatorMaxCountInt]
	}
	validators := []common.Address{}
	for _, candidate := range candidates {
		validators = append(validators, candidate.Address)
	}
	return validators, nil
}
