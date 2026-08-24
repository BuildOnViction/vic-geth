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
	"github.com/ethereum/go-ethereum/core/types"
)

func CreateBlockSignData(blockNumber *big.Int, blockHash common.Hash) []byte {
	data := common.Hex2Bytes("e341eaa4") // sign(uint256,bytes32)
	data = append(data, common.LeftPadBytes(blockNumber.Bytes(), 32)...)
	data = append(data, blockHash.Bytes()...)
	return data
}

func CreateBlockSignTransaction(nonce uint64, contractAddr common.Address, blockNumber *big.Int, blockHash common.Hash) *types.Transaction {
	data := CreateBlockSignData(blockNumber, blockHash)
	return types.NewTransaction(nonce, contractAddr, big.NewInt(0), 200000, big.NewInt(0), data)
}
