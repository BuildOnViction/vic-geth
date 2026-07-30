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

package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const SignMethodHex = "e341eaa4"

var vicBlockSignerStorageMap = map[string]uint64{
	"blockSigners": 0,
	"blocks":       1,
}

var vicRandomizeStorageMap = map[string]uint64{
	"randomSecret":  0,
	"randomOpening": 1,
}

var vicValidatorStorageMap = map[string]uint64{
	"withdrawsState":         0,
	"validatorsState":        1,
	"voters":                 2,
	"candidates":             3,
	"candidateCount":         4,
	"minCandidateCap":        5,
	"minVoterCap":            6,
	"maxValidatorNumber":     7,
	"candidateWithdrawDelay": 8,
	"voterWithdrawDelay":     9,
}

// ----------------------------- BlockSign contract ----------------------------

// Return address of singers for a given block.
func (statedb *StateDB) GetSigners(contractAddress common.Address, block *types.Block) []common.Address {
	signerslot := StorageLocationFromSlot(vicBlockSignerStorageMap["blockSigners"])
	signerArrSlot := StorageLocationOfMappingElement(signerslot, block.Hash().Bytes())
	arrLength := statedb.GetState(contractAddress, signerArrSlot.Hash()).Big().Uint64()
	signers := make([]common.Address, 0, arrLength)
	for i := uint64(0); i < arrLength; i++ {
		signerSlot := StorageLocationOfDynamicArrayElement(signerArrSlot, i, 160)
		signer := common.BytesToAddress(statedb.GetState(contractAddress, signerSlot.Hash()).Bytes())
		signers = append(signers, signer)
	}
	return signers
}

// ----------------------------- Randomize contract ----------------------------

// Return first part of secret submitted by an address. This value will be used in Commit phase.
func (statedb *StateDB) VictionGetSecrets(contractAddress common.Address, address common.Address) []common.Hash {
	secretsMappingSlot := StorageLocationFromSlot(vicRandomizeStorageMap["randomSecret"])
	secretsArrSlot := StorageLocationOfMappingElement(secretsMappingSlot, address.Hash().Bytes())

	// Get array length
	secretsStateData := statedb.GetState(contractAddress, secretsArrSlot.Hash())
	arrayLength := secretsStateData.Big().Uint64()

	secrets := make([]common.Hash, 0, arrayLength)
	for i := uint64(0); i < arrayLength; i++ {
		secretSlot := StorageLocationOfDynamicArrayElement(secretsArrSlot, i, 256)
		secretStateData := statedb.GetState(contractAddress, secretSlot.Hash())
		secret := common.BytesToHash(secretStateData.Bytes())
		secrets = append(secrets, secret)
	}
	return secrets
}

// Return second part of secret submitted by an address. This value will be used in Reveal phase.
func (statedb *StateDB) VictionGetSecretOpening(contractAddress common.Address, address common.Address) common.Hash {
	openingMappingSlot := StorageLocationFromSlot(vicRandomizeStorageMap["randomOpening"])
	openingElemSlot := StorageLocationOfMappingElement(openingMappingSlot, address.Hash().Bytes())
	openingStateData := statedb.GetState(contractAddress, openingElemSlot.Hash())
	opening := common.BytesToHash(openingStateData.Bytes())
	return opening
}

// ----------------------------- Validator contract ----------------------------

// Return owner for a given validator.
func (statedb *StateDB) VicGetValidatorOwner(contractAddress common.Address, validator common.Address) common.Address {
	validatorMappingSlot := StorageLocationFromSlot(vicValidatorStorageMap["validatorsState"])
	validatorStructSlot := StorageLocationOfMappingElement(validatorMappingSlot, validator.Hash().Bytes())

	ownerData := statedb.GetState(contractAddress, validatorStructSlot.Hash())
	owner := common.BytesToAddress(ownerData.Bytes())
	if owner == (common.Address{}) {
		return common.Address{}
	}

	return owner
}

// Return owner and staking capacity for a given validator.
func (statedb *StateDB) VicGetValidatorInfo(contractAddress common.Address, validator common.Address) (common.Address, *big.Int) {
	validatorMappingSlot := StorageLocationFromSlot(vicValidatorStorageMap["validatorsState"])
	validatorStructSlot := StorageLocationOfMappingElement(validatorMappingSlot, validator.Hash().Bytes())

	ownerData := statedb.GetState(contractAddress, validatorStructSlot.Hash())
	owner := common.BytesToAddress(ownerData.Bytes())
	if owner == (common.Address{}) {
		return common.Address{}, common.Big0
	}

	capSlot := StorageLocationOfStructElement(validatorStructSlot, common.Big1)
	capData := statedb.GetState(contractAddress, capSlot.Hash())
	return owner, new(big.Int).SetBytes(capData.Bytes())
}

// Return all addresses voted for a given validator.
func (statedb *StateDB) VicGetValidatorVoters(contractAddress common.Address, validator common.Address) []common.Address {
	votersMappingSlot := StorageLocationFromSlot(vicValidatorStorageMap["voters"])
	votersArrSlot := StorageLocationOfMappingElement(votersMappingSlot, validator.Hash().Bytes())

	arrLength := statedb.GetState(contractAddress, votersArrSlot.Hash()).Big().Uint64()
	voters := make([]common.Address, 0, arrLength)
	for i := uint64(0); i < arrLength; i++ {
		elemSlot := StorageLocationOfDynamicArrayElement(votersArrSlot, i, 160)
		voter := common.BytesToAddress(statedb.GetState(contractAddress, elemSlot.Hash()).Bytes())
		voters = append(voters, voter)
	}
	return voters
}

// Return staking capacity by a given address of a validator.
func (statedb *StateDB) VicGetValidatorVoterCap(contractAddress common.Address, validator, voter common.Address) *big.Int {
	validatorMappingSlot := StorageLocationFromSlot(vicValidatorStorageMap["validatorsState"])
	validatorStructSlot := StorageLocationOfMappingElement(validatorMappingSlot, validator.Hash().Bytes())

	votersMappingSlot := StorageLocationOfStructElement(validatorStructSlot, common.Big2)
	voterElemSlot := StorageLocationOfMappingElement(votersMappingSlot, voter.Hash().Bytes())

	return new(big.Int).SetBytes(statedb.GetState(contractAddress, voterElemSlot.Hash()).Bytes())
}

// Return all addresses of candidates applied to be validator.
func (statedb *StateDB) VicGetCandidates(contractAddress common.Address) []common.Address {
	candidatesSlot := StorageLocationFromSlot(vicValidatorStorageMap["candidates"])
	candidatesStateData := statedb.GetState(contractAddress, candidatesSlot.Hash())
	arrayLength := candidatesStateData.Big().Uint64()
	candidates := make([]common.Address, 0, arrayLength)
	for i := uint64(0); i < arrayLength; i++ {
		candidateSlot := StorageLocationOfDynamicArrayElement(candidatesSlot, i, 160)
		candidateStateData := statedb.GetState(contractAddress, candidateSlot.Hash())
		candidates = append(candidates, common.BytesToAddress(candidateStateData.Bytes()))
	}
	return candidates
}
