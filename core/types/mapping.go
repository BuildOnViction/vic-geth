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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// BalanceMap holds snapshots of account balances.
type BalanceMap map[common.Address]*big.Int

// Return a deep copy of BalanceMap.
func (m BalanceMap) Copy() BalanceMap {
	if m == nil {
		return nil
	}
	cp := make(BalanceMap, len(m))
	for k, v := range m {
		if v != nil {
			cp[k] = new(big.Int).Set(v)
		}
	}
	return cp
}
