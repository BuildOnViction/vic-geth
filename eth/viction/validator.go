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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Return map of validator-attestor pairs from checkpoint header (Both validators and attestorIndices are from checkpoint header).
func GetCreatorAttestorPairsFromCheckpointHeader(config *params.ChainConfig, posvConfig *params.PosvConfig,
	header, checkpointHeader *types.Header,
) (map[common.Address]common.Address, uint64, error) {
	number := header.Number.Uint64()
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkpointHeader)
	if len(validators) == 0 {
		return nil, 0, ErrNoValidator
	}
	attestorIdxs := posv.ExtractAttestorsFromCheckpointHeader(checkpointHeader)
	return getCreatorAttestorPairs(config, posvConfig, number, validators, attestorIdxs)
}

// Return map of validator-attestor pairs from state (Validators is from checkpoint header, attestorIndices are from state).
func GetCreatorAttestorPairsFromState(config *params.ChainConfig, posvConfig *params.PosvConfig,
	header, checkpointHeader *types.Header,
	statedb *state.StateDB,
) (map[common.Address]common.Address, uint64, error) {
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkpointHeader)
	if len(validators) == 0 {
		return nil, 0, ErrNoValidator
	}
	number := header.Number.Uint64()
	checkpointNumber := checkpointHeader.Number.Uint64()
	attestorIdxs, err := GetAttestors(config.Viction, validators, statedb)
	if err != nil {
		log.Warn("[Backend][GetCreatorAttestorPairsFromState] cannot compute attestors", "checkpoint", checkpointNumber, "err", err)
		return nil, 0, err
	}
	return getCreatorAttestorPairs(config, posvConfig, number, validators, attestorIdxs)
}

func getCreatorAttestorPairs(config *params.ChainConfig, posvConfig *params.PosvConfig,
	number uint64, validators []common.Address, attestorIdxs []int64,
) (map[common.Address]common.Address, uint64, error) {
	results := map[common.Address]common.Address{}
	validatorCount := uint64(len(validators))
	attestorCount := uint64(len(attestorIdxs))
	offset := uint64(0)
	if validatorCount > attestorCount {
		return nil, offset, ErrInvalidAttestorList
	}
	if validatorCount > 0 {
		if config.IsTIPRandomize(new(big.Int).SetUint64(number)) {
			offset = ((number % posvConfig.Epoch) / validatorCount) % validatorCount
		}
		for i, val := range validators {
			attIdx := uint64(attestorIdxs[i]) % validatorCount
			attIdx = (attIdx + offset) % validatorCount
			results[val] = validators[attIdx]
		}
	}
	return results, offset, nil
}
