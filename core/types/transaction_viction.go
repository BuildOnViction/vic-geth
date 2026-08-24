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

package types

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
)

var signMethodSelector = common.Hex2Bytes("e341eaa4")

func (tx *Transaction) IsTradingTransaction(tradingContract common.Address) bool {
	if tx.To() == nil {
		return false
	}
	return *tx.To() == tradingContract
}

func (tx *Transaction) IsLendingTransaction(lendingContract common.Address) bool {
	if tx.To() == nil {
		return false
	}
	return *tx.To() == lendingContract
}

func (tx *Transaction) IsLendingFinalizedTradeTransaction(lendingFinalizedContract common.Address) bool {
	if tx.To() == nil {
		return false
	}
	return *tx.To() == lendingFinalizedContract
}

func (tx *Transaction) IsSigningTransaction(blockSignAddr common.Address) bool {
	if tx == nil || tx.To() == nil {
		return false
	}
	if *tx.To() != blockSignAddr {
		return false
	}
	data := tx.Data()
	// sign(uint256 blockNumber, bytes32 blockHash) = 4 + 32 + 32 = 68 bytes
	if len(data) != 68 {
		return false
	}
	return bytes.Equal(data[0:4], signMethodSelector)
}

func (tx *Transaction) IsPosvTransaction() bool {
	if tx.To() == nil {
		return false
	}
	addr := *tx.To()
	return addr == common.HexToAddress("0x0000000000000000000000000000000000000090") ||
		addr == common.HexToAddress("0x0000000000000000000000000000000000000089")
}
