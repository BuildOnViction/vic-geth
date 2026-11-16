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

package victionapi

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestCheckBlockSignable(t *testing.T) {
	var (
		etherbase = common.HexToAddress("0x0a")
		other     = common.HexToAddress("0x0b")
		block     = makeBlockSignTestBlock(30, 1)
		competing = makeBlockSignTestBlock(30, 2)
	)
	tests := []struct {
		name      string
		canonical common.Hash
		record    func(db ethdb.Database)
		want      error
	}{
		{
			name:      "canonical block never signed at its height may be signed",
			canonical: block.Hash(),
		},
		{
			name:      "block off the canonical chain is not signed",
			canonical: competing.Hash(),
			want:      errSignNonCanonical,
		},
		{
			name:      "block already signed may be signed again",
			canonical: block.Hash(),
			record:    func(db ethdb.Database) { recordBlockSign(db, etherbase, block, 1800) },
		},
		{
			name:      "block competing with a signed one is not signed",
			canonical: block.Hash(),
			record:    func(db ethdb.Database) { recordBlockSign(db, etherbase, competing, 1800) },
			want:      errSignConflict,
		},
		{
			name:      "signature of another validator at the height does not block signing",
			canonical: block.Hash(),
			record:    func(db ethdb.Database) { recordBlockSign(db, other, competing, 1800) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := rawdb.NewMemoryDatabase()
			if tt.record != nil {
				tt.record(db)
			}
			chain := canonicalTestChain{block.NumberU64(): tt.canonical}

			err := checkBlockSignable(db, chain, etherbase, block)

			if !errors.Is(err, tt.want) {
				t.Fatalf("error mismatch: have %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRecordBlockSignDropsRecordsBeyondRetention(t *testing.T) {
	var (
		db        = rawdb.NewMemoryDatabase()
		etherbase = common.HexToAddress("0x0a")
		old       = makeBlockSignTestBlock(15, 1)
		kept      = makeBlockSignTestBlock(45, 1)
		latest    = makeBlockSignTestBlock(1845, 1)
	)
	recordBlockSign(db, etherbase, old, 1800)
	recordBlockSign(db, etherbase, kept, 1800)

	recordBlockSign(db, etherbase, latest, 1800)

	if have := rawdb.ReadPosvSignedBlockHash(db, etherbase, old.NumberU64()); have != (common.Hash{}) {
		t.Fatalf("record beyond retention kept: %x", have)
	}
	if have := rawdb.ReadPosvSignedBlockHash(db, etherbase, kept.NumberU64()); have != kept.Hash() {
		t.Fatalf("record within retention dropped: have %x, want %x", have, kept.Hash())
	}
	if have := rawdb.ReadPosvSignedBlockHash(db, etherbase, latest.NumberU64()); have != latest.Hash() {
		t.Fatalf("latest record missing: have %x, want %x", have, latest.Hash())
	}
}

// canonicalTestChain maps heights to canonical block hashes.
type canonicalTestChain map[uint64]common.Hash

func (c canonicalTestChain) GetCanonicalHash(number uint64) common.Hash { return c[number] }

// Build a block at number whose hash differs per variant.
func makeBlockSignTestBlock(number uint64, variant int64) *types.Block {
	return types.NewBlockWithHeader(&types.Header{Number: new(big.Int).SetUint64(number), Difficulty: big.NewInt(variant)})
}
