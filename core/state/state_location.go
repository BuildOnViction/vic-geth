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
	"github.com/ethereum/go-ethereum/crypto"
)

type StorageLocation []byte

// Return StorageLocation from hash.
func StorageLocationFromHash(h common.Hash) StorageLocation {
	return StorageLocation(h.Bytes())
}

// Return StorageLocation from slot number.
func StorageLocationFromSlot(slot uint64) StorageLocation {
	return StorageLocation(common.BigToHash(new(big.Int).SetUint64(slot)).Bytes())
}

// Return StorageLocation in big.Int format.
func (s StorageLocation) Big() *big.Int {
	return new(big.Int).SetBytes(s)
}

// Return StorageLocation in bytes format.
func (s StorageLocation) Hash() common.Hash {
	return common.BytesToHash(s)
}

// Return StorageLocation of a struct field.
func StorageLocationOfStructElement(structSlot StorageLocation, fieldIndex *big.Int) StorageLocation {
	slotNum := new(big.Int).Add(structSlot.Big(), fieldIndex)
	slotHash := slotNum.Bytes()
	return StorageLocation(slotHash)
}

// Return StorageLocation of an element in fixed-length array.
func StorageLocationOfFixedArrayElement(arraySlot StorageLocation, elementIndex uint64, elementSize uint64) StorageLocation {
	offset := new(big.Int).Div(
		new(big.Int).SetUint64(elementIndex),
		new(big.Int).Div(common.Big256, new(big.Int).SetUint64(elementSize)),
	)
	slotNum := new(big.Int).Add(arraySlot.Big(), offset)
	slotHash := slotNum.Bytes()
	return StorageLocation(slotHash)
}

// Return StorageLocation of an element in dynamic-length array.
func StorageLocationOfDynamicArrayElement(arraySlot StorageLocation, elementIndex uint64, elementSize uint64) StorageLocation {
	slotZero := new(big.Int).SetBytes(crypto.Keccak256(arraySlot.Hash().Bytes()))
	slotsPerElement := new(big.Int).Div(
		new(big.Int).Add(new(big.Int).SetUint64(elementSize), big.NewInt(255)),
		common.Big256,
	)
	if slotsPerElement.Cmp(big.NewInt(0)) == 0 {
		slotsPerElement = big.NewInt(1)
	}
	offset := new(big.Int).Mul(new(big.Int).SetUint64(elementIndex), slotsPerElement)
	slotNum := new(big.Int).Add(slotZero, offset)
	slotHash := slotNum.Bytes()
	return StorageLocation(slotHash)
}

// Return StorageLocation of an element in mapping.
func StorageLocationOfMappingElement(mappingSlot StorageLocation, elementKey []byte) StorageLocation {
	slotHash := crypto.Keccak256(elementKey, mappingSlot)
	return StorageLocation(slotHash)
}
