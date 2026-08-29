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
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
)

// Rerturn attestor indices from state.
func GetAttestorsFromState(vicConfig *params.VictionConfig, validators []common.Address, state *state.StateDB) ([]int64, error) {
	randomizes := []int64{}
	validatorCount := int64(len(validators))
	if validatorCount > 0 {
		for _, validator := range validators {
			random, err := GetSubmittedRandomOfValidator(vicConfig, validator, state)
			if err != nil {
				return nil, err
			}
			randomizes = append(randomizes, random)
		}
		attestors, err := GetAttestorsFromRandomize(randomizes, validatorCount)
		if err != nil {
			return nil, err
		}
		return attestors, nil
	}
	return nil, ErrNoValidator
}

// Get submitted random number of a given validator.
func GetSubmittedRandomOfValidator(vicConfig *params.VictionConfig, validator common.Address, state *state.StateDB) (int64, error) {
	randomizeContract := vicConfig.RandomizerContract
	if randomizeContract == (common.Address{}) {
		return -1, ErrNoContractAddress
	}

	secretsHash := state.VictionGetSecrets(randomizeContract, validator)
	openingHash := state.VictionGetSecretOpening(randomizeContract, validator)

	// Convert []common.Hash to [][32]byte
	secrets := make([][32]byte, len(secretsHash))
	for i, h := range secretsHash {
		secrets[i] = h
	}

	// Convert common.Hash to [32]byte
	opening := [32]byte(openingHash)

	result, err := DecryptRandomize(secrets, opening)
	if err != nil {
		return result, err
	}
	return result, nil
}

// Generate pseudo-random attestor indices based on randomizes.
func GetAttestorsFromRandomize(randomizes []int64, signersLen int64) ([]int64, error) {
	randomSeed := int64(0)
	for _, j := range randomizes {
		randomSeed += j
	}
	rand.Seed(randomSeed)

	randArray := GenerateSequence(0, 1, signersLen)
	attestorIndices := make([]int64, signersLen)
	attestorIndex := int64(0)
	for i := len(randArray) - 1; i >= 0; i-- {
		blockLength := len(randArray) - 1
		if blockLength <= 1 {
			blockLength = 1
		}
		randomIndex := int64(rand.Intn(blockLength))
		attestorIndex = randArray[randomIndex]
		randArray[randomIndex] = randArray[i]
		randArray[i] = attestorIndex
		randArray = append(randArray[:i], randArray[i+1:]...)
		attestorIndices[i] = attestorIndex
	}

	return attestorIndices, nil
}
