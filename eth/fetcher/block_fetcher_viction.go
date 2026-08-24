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

package fetcher

import (
	"time"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type PosvBackend interface {
	// Create a new block with attestor's signature. Only accept non-attested block.
	PosvAttestBlock(block *types.Block) (*types.Block, error)

	// Create new Randomize transaction and submit to TxPool.
	PosvRandomNumber(block *types.Block) error

	// Create new BlockSign transaction and submit to TxPool.
	PosvSignBlock(block *types.Block) error
}

// Reverify a PoSV block to see it's still eligible for further processing.
func (f *BlockFetcher) retryPosvBlock(peer string, block *types.Block) (requireAttest, dropPeer bool) {
	maxRetries := 3
	for retries := 0; retries <= maxRetries; retries++ {
		switch err := f.verifyHeader(block.Header()); err {
		case nil:

		case consensus.ErrFutureBlock:
			if retries == maxRetries {
				log.Debug("Propagated block is still from the future, timed out.", "peer", peer, "number", block.Number(), "hash", block.Hash())
				return false, true
			}
			until := time.Unix(int64(block.Time()), 0)
			if duration := time.Until(until); duration > 0 {
				log.Debug("Propagated block is from the future, waiting", "peer", peer, "number", block.Number(), "hash", block.Hash(), "duration", duration)
				time.Sleep(duration)
			}
			time.Sleep(50 * time.Millisecond)

		case posv.ErrNoAttestorSignature:
			return true, false

		default:
			log.Warn("Propagated block verification failed", "peer", peer, "number", block.Number(), "hash", block.Hash(), "err", err)
			return false, true
		}
	}

	return false, false
}
