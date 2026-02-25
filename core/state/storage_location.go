package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type StorageLocation []byte

func StorageLocationFromSlot(slot uint64) StorageLocation {
	return StorageLocation(common.BigToHash(new(big.Int).SetUint64(slot)).Bytes())
}

func StorageLocationFromHash(h common.Hash) StorageLocation {
	return StorageLocation(h.Bytes())
}

func (s StorageLocation) Big() *big.Int {
	return new(big.Int).SetBytes(s)
}

func (s StorageLocation) Hash() common.Hash {
	return common.BytesToHash(s)
}

func StorageLocationOfMappingElement(mappingSlot StorageLocation, elementKey []byte) StorageLocation {
	return StorageLocation(crypto.Keccak256(elementKey, mappingSlot))
}

func StorageLocationOfStructElement(structSlot StorageLocation, fieldIndex *big.Int) StorageLocation {
	return StorageLocation(new(big.Int).Add(structSlot.Big(), fieldIndex).Bytes())
}

func StorageLocationOfDynamicArrayElement(arraySlot StorageLocation, elementIndex uint64, elementSize uint64) StorageLocation {
	base := new(big.Int).SetBytes(crypto.Keccak256(arraySlot.Hash().Bytes()))
	slotsPerElement := new(big.Int).Div(
		new(big.Int).Add(new(big.Int).SetUint64(elementSize), big.NewInt(255)),
		common.Big256,
	)
	if slotsPerElement.Cmp(big.NewInt(0)) == 0 {
		slotsPerElement = big.NewInt(1)
	}
	offset := new(big.Int).Mul(new(big.Int).SetUint64(elementIndex), slotsPerElement)
	return StorageLocation(new(big.Int).Add(base, offset).Bytes())
}

func StorageLocationOfFixedArrayElement(arraySlot StorageLocation, elementIndex uint64, elementSize uint64) StorageLocation {
	offset := new(big.Int).Div(
		new(big.Int).SetUint64(elementIndex),
		new(big.Int).Div(common.Big256, new(big.Int).SetUint64(elementSize)),
	)
	return StorageLocation(new(big.Int).Add(arraySlot.Big(), offset).Bytes())
}
