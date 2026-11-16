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

import "github.com/ethereum/go-ethereum/core/types"

// posvFinalityChain is a local chain whose fork choice keeps a PoSV-finalized block.
type posvFinalityChain interface {
	PosvEnforcedFinalizedHeader() *types.Header
}

// Raise floor, the highest block number a common ancestor must be above, to just below the enforced PoSV-finalized block.
// The chain refuses a fork from a lower ancestor, so the sync fails fast and drops the peer instead of downloading it.
func (d *Downloader) posvAncestorFloor(floor int64) int64 {
	chain, ok := d.blockchain.(posvFinalityChain)
	if !ok {
		return floor
	}
	finalized := chain.PosvEnforcedFinalizedHeader()
	if finalized == nil {
		return floor
	}
	if finalizedFloor := int64(finalized.Number.Uint64()) - 1; finalizedFloor > floor {
		return finalizedFloor
	}
	return floor
}
