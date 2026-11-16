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

package downloader

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestPosvAncestorFloor(t *testing.T) {
	tests := []struct {
		name  string
		chain BlockChain
		floor int64
		want  int64
	}{
		{name: "light sync without a block chain keeps the floor", chain: nil, floor: 10, want: 10},
		{name: "chain without PoSV finality keeps the floor", chain: plainFloorTestChain{}, floor: 10, want: 10},
		{name: "no enforced finalized block keeps the floor", chain: posvFloorTestChain{}, floor: 10, want: 10},
		{name: "finalized block above the floor raises it", chain: posvFloorTestChain{finalized: 100}, floor: 10, want: 99},
		{name: "finalized block below the floor keeps it", chain: posvFloorTestChain{finalized: 5}, floor: 10, want: 10},
		{name: "finalized block sets a missing floor", chain: posvFloorTestChain{finalized: 1}, floor: -1, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Downloader{blockchain: tt.chain}

			have := d.posvAncestorFloor(tt.floor)

			if have != tt.want {
				t.Fatalf("floor mismatch: have %d, want %d", have, tt.want)
			}
		})
	}
}

// plainFloorTestChain is a block chain without PoSV finality; only its type matters.
type plainFloorTestChain struct{ BlockChain }

// posvFloorTestChain enforces a finalized block at number finalized, none when zero.
type posvFloorTestChain struct {
	BlockChain
	finalized uint64
}

func (c posvFloorTestChain) PosvEnforcedFinalizedHeader() *types.Header {
	if c.finalized == 0 {
		return nil
	}
	return &types.Header{Number: new(big.Int).SetUint64(c.finalized)}
}
