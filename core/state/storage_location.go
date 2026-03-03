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
	"github.com/ethereum/go-ethereum/crypto"
)

// Struct StorageLocation represents a slot in Solidity storage layout.
type StorageLocation []byte

func (s StorageLocation) Big() *big.Int {
	return new(big.Int).SetBytes(s)
}

func (s StorageLocation) Hash() common.Hash {
	return common.BytesToHash(s)
}

// Calculate storage slot of a element in a dynamic size array. elementSize is their data type's size in bits.
func StorageLocationOfDynamicArrayElement(arraySlot StorageLocation, elementIndex uint64, elementSize uint64) StorageLocation {
	baseSlot := crypto.Keccak256(arraySlot.Hash().Bytes())
	elementOffset := new(big.Int).Div(new(big.Int).SetUint64(elementIndex), new(big.Int).Div(common.Big256, new(big.Int).SetUint64(elementSize)))
	slotNum := new(big.Int).Add(new(big.Int).SetBytes(baseSlot), elementOffset)
	return StorageLocation(slotNum.Bytes())
}

// Calculate storage slot of a element in a fixed size array. elementSize is their data type's size in bits.
func StorageLocationOfFixedArrayElement(arraySlot StorageLocation, elementIndex uint64, elementSize uint64) StorageLocation {
	elementOffset := new(big.Int).Div(new(big.Int).SetUint64(elementIndex), new(big.Int).Div(common.Big256, new(big.Int).SetUint64(elementSize)))
	slotNum := new(big.Int).Add(arraySlot.Big(), elementOffset)
	return StorageLocation(slotNum.Bytes())
}

// Calculate storage slot of a field in a struct.
func StorageLocationOfStructElement(arraySlot StorageLocation, elementOffset *big.Int) StorageLocation {
	slotNum := new(big.Int).Add(arraySlot.Big(), elementOffset)
	return StorageLocation(slotNum.Bytes())
}

// Calculate storage slot of a element in a mapping.
func StorageLocationOfMappingElement(mappingSlot StorageLocation, elementKey []byte) StorageLocation {
	slotHash := crypto.Keccak256(elementKey, mappingSlot)
	return StorageLocation(slotHash)
}
