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

package victionapi

import (
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/sortlgc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Return map of validator-attestor pairs from checkpoint header (Both validators and attestorIndices are from checkpoint header).
func GetCreatorAttestorPairsFromCheckpointHeader(
	config *params.ChainConfig, posvConfig *params.PosvConfig,
	header, checkpointHeader *types.Header,
) (map[common.Address]common.Address, uint64, error) {
	number := header.Number.Uint64()
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkpointHeader)
	if len(validators) == 0 {
		return nil, 0, ErrNoValidator
	}
	attestorIndices := posv.ExtractAttestorsFromCheckpointHeader(checkpointHeader)
	return getCreatorAttestorPairs(config, posvConfig, number, validators, attestorIndices)
}

// Return map of validator-attestor pairs from state (Validators is from checkpoint header, attestorIndices are from state).
func GetCreatorAttestorPairsFromState(
	config *params.ChainConfig, posvConfig *params.PosvConfig,
	header, checkpointHeader *types.Header,
	statedb *state.StateDB,
) (map[common.Address]common.Address, uint64, error) {
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkpointHeader)
	if len(validators) == 0 {
		return nil, 0, ErrNoValidator
	}
	number := header.Number.Uint64()
	attestorIndices, err := GetAttestorsFromState(validators, config.Viction, statedb)
	if err != nil {
		return nil, 0, err
	}
	return getCreatorAttestorPairs(config, posvConfig, number, validators, attestorIndices)
}

// Return addresses of eligible validators from the state.
func GetValidators(
	header *types.Header,
	config *params.ChainConfig, victionConfig *params.VictionConfig,
	statedb *state.StateDB,
) ([]common.Address, error) {
	contractAddress := victionConfig.ValidatorContract
	if contractAddress == (common.Address{}) {
		return nil, ErrNoContractAddress
	}
	addresses := statedb.VicGetCandidates(contractAddress)
	candidates := []*posv.ValidatorInfo{}
	for _, addr := range addresses {
		if addr == (common.Address{}) {
			continue
		}
		_, capacity := statedb.VicGetValidatorInfo(contractAddress, addr)
		candidates = append(candidates, &posv.ValidatorInfo{Address: addr, Capacity: capacity})
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

	validatorMaxCountInt := int(victionConfig.ValidatorMaxCount)
	if len(candidates) > validatorMaxCountInt {
		candidates = candidates[:validatorMaxCountInt]
	}
	validators := []common.Address{}
	for _, candidate := range candidates {
		validators = append(validators, candidate.Address)
	}
	return validators, nil
}

// Return validator-attestor pairs from validator list and attestor indices.
func getCreatorAttestorPairs(
	config *params.ChainConfig, posvConfig *params.PosvConfig,
	number uint64, validators []common.Address, attestorIndices []int64,
) (map[common.Address]common.Address, uint64, error) {
	results := map[common.Address]common.Address{}
	validatorCount := uint64(len(validators))
	attestorCount := uint64(len(attestorIndices))
	offset := uint64(0)
	if validatorCount > attestorCount {
		return nil, offset, ErrInvalidAttestorList
	}
	if validatorCount > 0 {
		if config.IsTIPRandomize(new(big.Int).SetUint64(number)) {
			offset = ((number % posvConfig.Epoch) / validatorCount) % validatorCount
		}
		for i, val := range validators {
			attIdx := uint64(attestorIndices[i]) % validatorCount
			attIdx = (attIdx + offset) % validatorCount
			results[val] = validators[attIdx]
		}
	}
	return results, offset, nil
}
