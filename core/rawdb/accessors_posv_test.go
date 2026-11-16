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

package rawdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestPosvFinalizedBlockHashStorage(t *testing.T) {
	db := NewMemoryDatabase()
	hash := common.HexToHash("0x0f")

	empty := ReadPosvFinalizedBlockHash(db)
	WritePosvFinalizedBlockHash(db, hash)
	written := ReadPosvFinalizedBlockHash(db)
	DeletePosvFinalizedBlockHash(db)
	deleted := ReadPosvFinalizedBlockHash(db)

	if empty != (common.Hash{}) {
		t.Fatalf("empty database returned %x", empty)
	}
	if written != hash {
		t.Fatalf("written hash mismatch: have %x, want %x", written, hash)
	}
	if deleted != (common.Hash{}) {
		t.Fatalf("deleted hash still returned: %x", deleted)
	}
}

func TestPosvSignedBlockHashesBelow(t *testing.T) {
	var (
		db      = NewMemoryDatabase()
		signerA = common.HexToAddress("0x0a")
		signerB = common.HexToAddress("0x0b")
	)
	for _, number := range []uint64{15, 30, 45} {
		WritePosvSignedBlockHash(db, signerA, number, common.BigToHash(common.Big1))
	}
	WritePosvSignedBlockHash(db, signerB, 15, common.BigToHash(common.Big2))

	DeletePosvSignedBlockHashesBelow(db, signerA, 30)

	tests := []struct {
		name   string
		signer common.Address
		number uint64
		want   common.Hash
	}{
		{name: "record below the limit is deleted", signer: signerA, number: 15, want: common.Hash{}},
		{name: "record at the limit is kept", signer: signerA, number: 30, want: common.BigToHash(common.Big1)},
		{name: "record above the limit is kept", signer: signerA, number: 45, want: common.BigToHash(common.Big1)},
		{name: "record of another signer is kept", signer: signerB, number: 15, want: common.BigToHash(common.Big2)},
		{name: "height never signed has no record", signer: signerB, number: 30, want: common.Hash{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if have := ReadPosvSignedBlockHash(db, tt.signer, tt.number); have != tt.want {
				t.Fatalf("have %x, want %x", have, tt.want)
			}
		})
	}
}
