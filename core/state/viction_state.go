package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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

func (statedb *StateDB) VicGetValidatorVoterCap(contractAddress common.Address, validator, voter common.Address) *big.Int {
	validatorMappingSlot := StorageLocationFromSlot(vicValidatorStorageMap["validatorsState"])
	validatorStructSlot := StorageLocationOfMappingElement(validatorMappingSlot, validator.Hash().Bytes())

	votersMappingSlot := StorageLocationOfStructElement(validatorStructSlot, common.Big2)
	voterElemSlot := StorageLocationOfMappingElement(votersMappingSlot, voter.Hash().Bytes())

	return new(big.Int).SetBytes(statedb.GetState(contractAddress, voterElemSlot.Hash()).Bytes())
}

func (statedb *StateDB) VictionGetSigners(contractAddress common.Address, blockHash common.Hash) []common.Address {
	blockSignersMappingSlot := StorageLocationFromSlot(vicBlockSignerStorageMap["blockSigners"])
	blockSignersArrSlot := StorageLocationOfMappingElement(blockSignersMappingSlot, blockHash.Bytes())

	arrLength := statedb.GetState(contractAddress, blockSignersArrSlot.Hash()).Big().Uint64()
	signers := make([]common.Address, 0, arrLength)
	for i := uint64(0); i < arrLength; i++ {
		elemSlot := StorageLocationOfDynamicArrayElement(blockSignersArrSlot, i, 160)
		signer := common.BytesToAddress(statedb.GetState(contractAddress, elemSlot.Hash()).Bytes())
		signers = append(signers, signer)
	}
	return signers
}
