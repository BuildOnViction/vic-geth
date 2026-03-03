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

package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const SignMethodHex = "e341eaa4"

var vicBlockSignerStorageMap = map[string]*big.Int{
	"blockSigners": common.Big0,
	"blocks":       common.Big1,
}

// Return address of all signers that signed for a particular blockHash.
func (s *StateDB) VictionGetSigners(contractAddress common.Address, blockHash common.Hash) []common.Address {
	blockSignersMappingSlot := StorageLocation(vicBlockSignerStorageMap["blockSigners"].Bytes())
	blockSignersElemSlot := StorageLocationOfMappingElement(blockSignersMappingSlot, blockHash.Bytes())
	blockSignersStateData, _ := s.GetState2(contractAddress, blockSignersElemSlot.Hash())
	signers := []common.Address{}
	for i := uint64(0); i <= blockSignersStateData.Big().Uint64(); i++ {
		signerSlot := StorageLocationOfDynamicArrayElement(blockSignersElemSlot, i, 160)
		signerStateData, _ := s.GetState2(contractAddress, signerSlot.Hash())
		signer := common.BytesToAddress(signerStateData.Bytes())
		signers = append(signers, signer)
	}
	return signers
}

var vicRandomizeStorageMap = map[string]*big.Int{
	"randomSecret":  common.Big0,
	"randomOpening": common.Big1,
}

// Return first part of secret submitted by an address. This value will be used in Commit phase.
func (s *StateDB) VictionGetSecrets(contractAddress common.Address, address common.Address) []common.Hash {
	secretsMappingSlot := StorageLocation(vicRandomizeStorageMap["randomSecret"].Bytes())
	secretsElemSlot := StorageLocationOfMappingElement(secretsMappingSlot, address.Bytes())
	secretsStateData, _ := s.GetState2(contractAddress, secretsElemSlot.Hash())
	secrets := []common.Hash{}
	for i := uint64(0); i <= secretsStateData.Big().Uint64(); i++ {
		secretSlot := StorageLocationOfDynamicArrayElement(secretsElemSlot, i, 256)
		secretStateData, _ := s.GetState2(contractAddress, secretSlot.Hash())
		secret := common.BytesToHash(secretStateData.Bytes())
		secrets = append(secrets, secret)
	}
	return secrets
}

// Return second part of secret submitted by an address. This value will be used in Reveal phase.
func (s *StateDB) VictionGetSecretOpening(contractAddress common.Address, address common.Address) common.Hash {
	openingMappingSlot := StorageLocation(vicRandomizeStorageMap["randomOpening"].Bytes())
	openingElemSlot := StorageLocationOfMappingElement(openingMappingSlot, address.Bytes())
	openingStateData, _ := s.GetState2(contractAddress, openingElemSlot.Hash())
	opening := common.BytesToHash(openingStateData.Bytes())
	return opening
}

var vicValidatorStorageMap = map[string]*big.Int{
	"withdrawsState":         common.Big0,
	"validatorsState":        common.Big1,
	"voters":                 common.Big2,
	"candidates":             common.Big3,
	"candidateCount":         common.Big4,
	"minCandidateCap":        common.Big5,
	"minVoterCap":            common.Big6,
	"maxValidatorNumber":     common.Big7,
	"candidateWithdrawDelay": common.Big8,
	"voterWithdrawDelay":     common.Big9,
}

// Return all addressed that are proposed as validators.
func (s *StateDB) VicGetCandidates(contractAddress common.Address) []common.Address {
	candidatesSlot := StorageLocation(vicValidatorStorageMap["candidates"].Bytes())
	candidatesStateData, _ := s.GetState2(contractAddress, candidatesSlot.Hash())
	candidates := []common.Address{}
	for i := uint64(0); i <= candidatesStateData.Big().Uint64(); i++ {
		candidateSlot := StorageLocationOfDynamicArrayElement(candidatesSlot, i, 160)
		candidateStateData, _ := s.GetState2(contractAddress, candidateSlot.Hash())
		candidate := common.BytesToAddress(candidateStateData.Bytes())
		candidates = append(candidates, candidate)
	}
	return candidates
}

// Return owner address and their capacity of a particular validator.
func (s *StateDB) VicGetValidatorInfo(contractAddress common.Address, validator common.Address) (common.Address, *big.Int) {
	validatorMappingSlot := StorageLocation(vicValidatorStorageMap["validatorsState"].Bytes())
	validatorStructSlot := StorageLocationOfMappingElement(validatorMappingSlot, validator.Bytes())
	validatorOwnerStateData, _ := s.GetState2(contractAddress, validatorStructSlot.Hash())
	validatorOwner := common.BytesToAddress(validatorOwnerStateData.Bytes())
	if validatorOwner == (common.Address{}) {
		return common.Address{}, common.Big0
	}
	validatorCapSlot := StorageLocationOfStructElement(validatorStructSlot, common.Big1)
	validatorCapStateData, _ := s.GetState2(contractAddress, validatorCapSlot.Hash())
	validatorCap := new(big.Int).SetBytes(validatorCapStateData.Bytes())
	return validatorOwner, validatorCap
}

// Return all addresses that voted for a particular validator.
func (s *StateDB) VicGetValidatorVoters(contractAddress common.Address, validator common.Address) []common.Address {
	votersMappingSlot := StorageLocation(vicValidatorStorageMap["voters"].Bytes())
	votersElemSlot := StorageLocationOfMappingElement(votersMappingSlot, validator.Bytes())
	votersStateData, _ := s.GetState2(contractAddress, votersElemSlot.Hash())
	voters := []common.Address{}
	for i := uint64(0); i <= votersStateData.Big().Uint64(); i++ {
		signerSlot := StorageLocationOfDynamicArrayElement(votersMappingSlot, i, 160)
		signerStateData, _ := s.GetState2(contractAddress, signerSlot.Hash())
		voter := common.BytesToAddress(signerStateData.Bytes())
		voters = append(voters, voter)
	}
	return voters
}

// Return amount of tokens a voted has committed to a particular validator.
func (s *StateDB) VicGetValidatorVoterCap(contractAddress common.Address, validator, voter common.Address) *big.Int {
	validatorMappingSlot := StorageLocation(vicValidatorStorageMap["validatorsState"].Bytes())
	validatorStructSlot := StorageLocationOfMappingElement(validatorMappingSlot, validator.Bytes())
	voterCapMappingSlot := StorageLocationOfStructElement(validatorStructSlot, common.Big2)
	voterCapElemSlot := StorageLocationOfMappingElement(voterCapMappingSlot, voter.Bytes())
	voterCapStateData, _ := s.GetState2(contractAddress, voterCapElemSlot.Hash())
	voterCap := new(big.Int).SetBytes(voterCapStateData.Bytes())
	return voterCap
}

// Alternative version of GetState that returns common.Hash and error as result.
func (s *StateDB) GetState2(contractAddress common.Address, storLoc common.Hash) (common.Hash, error) {
	stateData := s.GetState(contractAddress, storLoc)
	return stateData, nil
}
