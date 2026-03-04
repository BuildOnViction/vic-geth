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

package viction

import (
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/tforce-io/tf-golib/stdx/mathxt/bigxt"
)

// Get eligble validators from the state.
//
// *NOTE: The injected state must be at the checkpoint block.
func GetValidators(vicConfig *params.VictionConfig,
	state *state.StateDB, logger log.Logger,
) ([]common.Address, error) {
	addresses := state.VicGetCandidates(vicConfig.ValidatorContract)

	candidates := []*posv.ValidatorInfo{}
	for _, addr := range addresses {
		if addr == (common.Address{}) {
			continue
		}
		_, cap := state.VicGetValidatorInfo(vicConfig.ValidatorContract, addr)
		candidates = append(candidates, &posv.ValidatorInfo{Address: addr, Capacity: cap})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return bigxt.IsGreaterThanOrEqualInt(candidates[i].Capacity, candidates[j].Capacity)
	})
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
