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
const TransferMethodHex = "a9059cbb"
const TransferFromMethodHex = "23b872dd"

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

var vicVRC25StorageMap = map[string]uint64{
	"balances": 0,
	"minFee":   1,
	"issuer":   2,
}

var vicZeroGasStorageMap = map[string]uint64{
	"minCap":      0,
	"tokens":      1,
	"tokensState": 2,
}

// ----------------------------- BlockSign contract ----------------------------

// Return address of singers for a given block.
func (statedb *StateDB) GetSigners(contractAddress common.Address, block *types.Block) []common.Address {
	signerMappingSlot := StorageLocationFromSlot(vicBlockSignerStorageMap["blockSigners"])
	signerArraySlot := StorageLocationOfMappingElement(signerMappingSlot, block.Hash().Bytes())
	signerArrayLength := statedb.GetState(contractAddress, signerArraySlot.Hash()).Big().Uint64()
	signers := make([]common.Address, 0, signerArrayLength)
	for i := uint64(0); i < signerArrayLength; i++ {
		signerSlot := StorageLocationOfDynamicArrayElement(signerArraySlot, i, 160)
		signerData := statedb.GetState(contractAddress, signerSlot.Hash())
		signer := common.BytesToAddress(signerData.Bytes())
		signers = append(signers, signer)
	}
	return signers
}

// ----------------------------- Randomize contract ----------------------------

// Return first part of secret submitted by an address. This value will be used in Commit phase.
func (statedb *StateDB) VictionGetSecrets(contractAddress common.Address, address common.Address) []common.Hash {
	secretMappingSlot := StorageLocationFromSlot(vicRandomizeStorageMap["randomSecret"])
	secretArraySlot := StorageLocationOfMappingElement(secretMappingSlot, address.Hash().Bytes())
	secretArrayLength := statedb.GetState(contractAddress, secretArraySlot.Hash()).Big().Uint64()
	secrets := make([]common.Hash, 0, secretArrayLength)
	for i := uint64(0); i < secretArrayLength; i++ {
		secretSlot := StorageLocationOfDynamicArrayElement(secretArraySlot, i, 256)
		secretData := statedb.GetState(contractAddress, secretSlot.Hash())
		secret := common.BytesToHash(secretData.Bytes())
		secrets = append(secrets, secret)
	}
	return secrets
}

// Return second part of secret submitted by an address. This value will be used in Reveal phase.
func (statedb *StateDB) VictionGetSecretOpening(contractAddress common.Address, address common.Address) common.Hash {
	openingMappingSlot := StorageLocationFromSlot(vicRandomizeStorageMap["randomOpening"])
	openingSlot := StorageLocationOfMappingElement(openingMappingSlot, address.Hash().Bytes())
	openingData := statedb.GetState(contractAddress, openingSlot.Hash())
	opening := common.BytesToHash(openingData.Bytes())
	return opening
}

// ----------------------------- VRC25 contract --------------------------------

// Return token balance for a given address.
func (statedb *StateDB) VicGetVrc25Balance(contractAddress common.Address, address common.Address) *big.Int {
	balanceMappingSlot := StorageLocationFromSlot(vicVRC25StorageMap["balances"])
	balanceSlot := StorageLocationOfMappingElement(balanceMappingSlot, address.Hash().Bytes())
	balanceData := statedb.GetState(contractAddress, balanceSlot.Hash())
	return new(big.Int).SetBytes(balanceData.Bytes())
}

// Return minimum fee of the contract.
func (statedb *StateDB) VicGetVrc25MinFee(contractAddress common.Address) *big.Int {
	minFeeSlot := StorageLocationFromSlot(vicVRC25StorageMap["minFee"])
	minFeeData := statedb.GetState(contractAddress, minFeeSlot.Hash())
	return new(big.Int).SetBytes(minFeeData.Bytes())
}

// Return issuer of the contract.
func (statedb *StateDB) VicGetVrc25Issuer(contractAddress common.Address) common.Address {
	issuerSlot := StorageLocationFromSlot(vicVRC25StorageMap["issuer"])
	issuerData := statedb.GetState(contractAddress, issuerSlot.Hash())
	issuer := common.BytesToAddress(issuerData.Bytes())
	return issuer
}

// Set token balance for a given address.
func (statedb *StateDB) VicSetVrc25Balance(contractAddress common.Address, address common.Address, value *big.Int) {
	balanceMappingSlot := StorageLocationFromSlot(vicVRC25StorageMap["balances"])
	balanceSlot := StorageLocationOfMappingElement(balanceMappingSlot, address.Hash().Bytes())
	statedb.SetState(contractAddress, balanceSlot.Hash(), common.BigToHash(value))
}

// ----------------------------- Validator contract ----------------------------

// Return owner for a given validator.
func (statedb *StateDB) VicGetValidatorOwner(contractAddress common.Address, validator common.Address) common.Address {
	validatorMappingSlot := StorageLocationFromSlot(vicValidatorStorageMap["validatorsState"])
	validatorStructSlot := StorageLocationOfMappingElement(validatorMappingSlot, validator.Hash().Bytes())
	ownerData := statedb.GetState(contractAddress, validatorStructSlot.Hash())
	owner := common.BytesToAddress(ownerData.Bytes())
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
	capacitySlot := StorageLocationOfStructElement(validatorStructSlot, common.Big1)
	capacityData := statedb.GetState(contractAddress, capacitySlot.Hash())
	return owner, new(big.Int).SetBytes(capacityData.Bytes())
}

// Return all addresses voted for a given validator.
func (statedb *StateDB) VicGetValidatorVoters(contractAddress common.Address, validator common.Address) []common.Address {
	voterMappingSlot := StorageLocationFromSlot(vicValidatorStorageMap["voters"])
	voterArraySlot := StorageLocationOfMappingElement(voterMappingSlot, validator.Hash().Bytes())
	voterArrayLength := statedb.GetState(contractAddress, voterArraySlot.Hash()).Big().Uint64()
	voters := make([]common.Address, 0, voterArrayLength)
	for i := uint64(0); i < voterArrayLength; i++ {
		voterSlot := StorageLocationOfDynamicArrayElement(voterArraySlot, i, 160)
		voterData := statedb.GetState(contractAddress, voterSlot.Hash())
		voter := common.BytesToAddress(voterData.Bytes())
		voters = append(voters, voter)
	}
	return voters
}

// Return staking capacity by a given address of a validator.
func (statedb *StateDB) VicGetValidatorVoterCap(contractAddress common.Address, validator, voter common.Address) *big.Int {
	validatorMappingSlot := StorageLocationFromSlot(vicValidatorStorageMap["validatorsState"])
	validatorStructSlot := StorageLocationOfMappingElement(validatorMappingSlot, validator.Hash().Bytes())
	voterMappingSlot := StorageLocationOfStructElement(validatorStructSlot, common.Big2)
	voterSlot := StorageLocationOfMappingElement(voterMappingSlot, voter.Hash().Bytes())
	voterCapacityData := statedb.GetState(contractAddress, voterSlot.Hash())
	return new(big.Int).SetBytes(voterCapacityData.Bytes())
}

// Return all addresses of candidates applied to be validator.
func (statedb *StateDB) VicGetCandidates(contractAddress common.Address) []common.Address {
	candidateArraySlot := StorageLocationFromSlot(vicValidatorStorageMap["candidates"])
	candiateArrayLength := statedb.GetState(contractAddress, candidateArraySlot.Hash()).Big().Uint64()
	candidates := make([]common.Address, 0, candiateArrayLength)
	for i := uint64(0); i < candiateArrayLength; i++ {
		candidateSlot := StorageLocationOfDynamicArrayElement(candidateArraySlot, i, 160)
		candidateData := statedb.GetState(contractAddress, candidateSlot.Hash())
		candidate := common.BytesToAddress(candidateData.Bytes())
		candidates = append(candidates, candidate)
	}
	return candidates
}

// ----------------------------- ZeroGas contract ------------------------------

// Return remaining sponsoring capacity for a given token.
func (statedb *StateDB) VicGetZeroGasCapacity(contractAddress common.Address, token *common.Address) *big.Int {
	if token == nil {
		return new(big.Int)
	}
	tokenCapacityMappingSlot := StorageLocationFromSlot(vicZeroGasStorageMap["tokensState"])
	tokenCapacitySlot := StorageLocationOfMappingElement(tokenCapacityMappingSlot, token.Hash().Bytes())
	tokenCapacityData := statedb.GetState(contractAddress, tokenCapacitySlot.Hash())
	return new(big.Int).SetBytes(tokenCapacityData.Bytes())
}

// Return remaining sponsoring capacities for all tokens.
func (statedb *StateDB) VicGetZeroGasCapacities(contractAddress common.Address) map[common.Address]*big.Int {
	tokenCapacityMappingSlot := StorageLocationFromSlot(vicZeroGasStorageMap["tokensState"])
	tokenArraySlot := StorageLocationFromSlot(vicZeroGasStorageMap["tokens"])
	tokenArrayLength := statedb.GetState(contractAddress, tokenArraySlot.Hash()).Big().Uint64()
	capacities := map[common.Address]*big.Int{}
	for i := uint64(0); i < tokenArrayLength; i++ {
		tokenSlot := StorageLocationOfDynamicArrayElement(tokenArraySlot, i, 1)
		tokenData := statedb.GetState(contractAddress, tokenSlot.Hash())
		token := common.BytesToAddress(tokenData.Bytes())
		tokenCapacitySlot := StorageLocationOfMappingElement(tokenCapacityMappingSlot, token.Hash().Bytes())
		tokenCapacityData := statedb.GetState(contractAddress, tokenCapacitySlot.Hash())
		capacities[token] = new(big.Int).SetBytes(tokenCapacityData.Bytes())
	}
	return capacities
}

// Set sponsoring capacity for a given token.
func (statedb *StateDB) VicSetZeroGasCapacity(contractAddress common.Address, token common.Address, value *big.Int) {
	tokenCapacityMappingSlot := StorageLocationFromSlot(vicZeroGasStorageMap["tokensState"])
	tokenCapacitySlot := StorageLocationOfMappingElement(tokenCapacityMappingSlot, token.Hash().Bytes())
	statedb.SetState(contractAddress, tokenCapacitySlot.Hash(), common.BigToHash(value))
}

// Set sponsoring capacities for all tokens.
func (statedb *StateDB) VicSetZeroGasCapacities(contract common.Address, newBalances map[common.Address]*big.Int, totalFeeUsed *big.Int) {
	for token, balance := range newBalances {
		statedb.VicSetZeroGasCapacity(contract, token, balance)
	}
	statedb.SubBalance(contract, totalFeeUsed)
}

// Set mininum capacity required for sponsoring enrollment.
func (statedb *StateDB) VicSetZeroGasMinCap(contractAddress common.Address, capacity *big.Int) {
	minCapSlot := StorageLocationFromSlot((vicZeroGasStorageMap["minCap"]))
	statedb.SetState(contractAddress, minCapSlot.Hash(), common.BigToHash(capacity))
}
