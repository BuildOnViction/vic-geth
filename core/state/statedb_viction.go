// Copyright 2026 The Vic-geth Authors
// Viction-specific state DB helpers for reading contract storage slots.
package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Storage slot mappings for the BlockSigner contract (0x89).
var slotBlockSignerMapping = map[string]uint64{
	"blockSigners": 0,
	"blocks":       1,
}

// Storage slot mappings for the Validator contract (0x88).
var slotValidatorMapping = map[string]uint64{
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

// GetLocMappingAtKey computes the storage slot for a mapping: keccak256(key, slot).
func GetLocMappingAtKey(key common.Hash, slot uint64) *big.Int {
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	retByte := crypto.Keccak256(key.Bytes(), slotHash.Bytes())
	ret := new(big.Int)
	ret.SetBytes(retByte)
	return ret
}

// GetLocDynamicArrAtElement computes the storage slot for element [index] of a dynamic array.
func GetLocDynamicArrAtElement(slotHash common.Hash, index uint64, elementSize uint64) common.Hash {
	slotKecBig := crypto.Keccak256Hash(slotHash.Bytes()).Big()
	arrBig := slotKecBig.Add(slotKecBig, new(big.Int).SetUint64(index*elementSize))
	return common.BigToHash(arrBig)
}

// GetSigners returns the list of addresses that signed the given block
// by reading the BlockSigner contract storage slots directly.
func GetSigners(statedb *StateDB, blockSignerAddr common.Address, block *types.Block) []common.Address {
	slot := slotBlockSignerMapping["blockSigners"]
	keyArrSlot := GetLocMappingAtKey(block.Hash(), slot)
	arrSlot := statedb.GetState(blockSignerAddr, common.BigToHash(keyArrSlot))
	arrLength := arrSlot.Big().Uint64()

	keys := make([]common.Hash, arrLength)
	for i := uint64(0); i < arrLength; i++ {
		keys[i] = GetLocDynamicArrAtElement(common.BigToHash(keyArrSlot), i, 1)
	}

	rets := make([]common.Address, 0, arrLength)
	for _, key := range keys {
		ret := statedb.GetState(blockSignerAddr, key)
		rets = append(rets, common.HexToAddress(ret.Hex()))
	}
	return rets
}

// GetCandidateOwner returns the owner of a masternode candidate by reading
// validatorsState[candidate].owner from the Validator contract.
func GetCandidateOwner(statedb *StateDB, validatorAddr common.Address, candidate common.Address) common.Address {
	slot := slotValidatorMapping["validatorsState"]
	locValidatorsState := GetLocMappingAtKey(candidate.Hash(), slot)
	locCandidateOwner := new(big.Int).Add(locValidatorsState, new(big.Int).SetUint64(0))
	ret := statedb.GetState(validatorAddr, common.BigToHash(locCandidateOwner))
	return common.HexToAddress(ret.Hex())
}

// GetVoters returns the list of voters for a candidate from the Validator contract.
func GetVoters(statedb *StateDB, validatorAddr common.Address, candidate common.Address) []common.Address {
	slot := slotValidatorMapping["voters"]
	locVoters := GetLocMappingAtKey(candidate.Hash(), slot)
	arrLength := statedb.GetState(validatorAddr, common.BigToHash(locVoters))

	keys := make([]common.Hash, 0, arrLength.Big().Uint64())
	for i := uint64(0); i < arrLength.Big().Uint64(); i++ {
		key := GetLocDynamicArrAtElement(common.BigToHash(locVoters), i, 1)
		keys = append(keys, key)
	}

	rets := make([]common.Address, 0, len(keys))
	for _, key := range keys {
		ret := statedb.GetState(validatorAddr, key)
		rets = append(rets, common.HexToAddress(ret.Hex()))
	}
	return rets
}

// GetVoterCap returns the staked amount of a voter for a specific candidate.
func GetVoterCap(statedb *StateDB, validatorAddr common.Address, candidate, voter common.Address) *big.Int {
	slot := slotValidatorMapping["validatorsState"]
	locValidatorsState := GetLocMappingAtKey(candidate.Hash(), slot)
	locCandidateVoters := new(big.Int).Add(locValidatorsState, new(big.Int).SetUint64(2))
	retByte := crypto.Keccak256(voter.Hash().Bytes(), common.BigToHash(locCandidateVoters).Bytes())
	ret := statedb.GetState(validatorAddr, common.BytesToHash(retByte))
	return ret.Big()
}
