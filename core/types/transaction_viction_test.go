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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSignedBlockHash(t *testing.T) {
	blockSignAddr := common.HexToAddress("0x0000000000000000000000000000000000000089")
	otherAddr := common.HexToAddress("0x0000000000000000000000000000000000000090")
	signedHash := common.HexToHash("0x5e1f")
	signData := makeSignData(big.NewInt(900), signedHash)

	tests := []struct {
		name     string
		tx       *Transaction
		wantHash common.Hash
		wantOk   bool
	}{
		{
			name:     "sign transaction to the BlockSigner contract returns the signed hash",
			tx:       NewTransaction(0, blockSignAddr, big.NewInt(0), 200000, big.NewInt(0), signData),
			wantHash: signedHash,
			wantOk:   true,
		},
		{
			name: "sign payload to another contract is not a vote",
			tx:   NewTransaction(0, otherAddr, big.NewInt(0), 200000, big.NewInt(0), signData),
		},
		{
			name: "contract creation is not a vote",
			tx:   NewContractCreation(0, big.NewInt(0), 200000, big.NewInt(0), signData),
		},
		{
			name: "truncated payload is not a vote",
			tx:   NewTransaction(0, blockSignAddr, big.NewInt(0), 200000, big.NewInt(0), signData[:len(signData)-1]),
		},
		{
			name: "payload with trailing bytes is not a vote",
			tx:   NewTransaction(0, blockSignAddr, big.NewInt(0), 200000, big.NewInt(0), append(append([]byte{}, signData...), 0x00)),
		},
		{
			name: "other method selector is not a vote",
			tx:   NewTransaction(0, blockSignAddr, big.NewInt(0), 200000, big.NewInt(0), append([]byte{0xde, 0xad, 0xbe, 0xef}, signData[4:]...)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, ok := tt.tx.SignedBlockHash(blockSignAddr)

			if ok != tt.wantOk {
				t.Fatalf("ok mismatch: have %v, want %v", ok, tt.wantOk)
			}
			if hash != tt.wantHash {
				t.Fatalf("hash mismatch: have %x, want %x", hash, tt.wantHash)
			}
		})
	}
}

// Build the calldata of sign(uint256,bytes32) for the given block.
func makeSignData(number *big.Int, hash common.Hash) []byte {
	data := common.Hex2Bytes("e341eaa4")
	data = append(data, common.LeftPadBytes(number.Bytes(), 32)...)
	return append(data, hash.Bytes()...)
}
