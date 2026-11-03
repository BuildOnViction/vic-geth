// Copyright 2015 The go-ethereum Authors
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

package common

import "math/big"

var (
	ZeroAddress = Address{}
	ZeroHash    = Hash{}
)

// Returns true if the address is zero.
func (a Address) IsZero() bool {
	return a == Address{}
}

// Returns true if the hash is zero.
func (h Hash) IsZero() bool {
	return h == Hash{}
}

// Return string representation of given address.
func AddressToString(addr Address) string {
	return addr.Hex()
}

// Return strings representation of given addresses.
func AddressesToStrings(addrs []Address) []string {
	results := []string{}
	for _, addr := range addrs {
		results = append(results, addr.Hex())
	}
	return results
}

// Return given uint64 as *Hash*.
func Uint64ToHash(n uint64) Hash {
	return BigToHash(new(big.Int).SetUint64(n))
}
