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

package posv

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

const (
	finalityNumberLength  = 8   // Length of a block number in a finality view key
	inmemoryFinalityViews = 128 // Number of recent finality views to keep in memory
)

var (
	// finalizedBlockKey -> last finalized block of the canonical chain.
	finalizedBlockKey = []byte("posv-finality-finalized")

	// finalityViewPrefix + block number (big endian) + block hash -> finality view of a signed block.
	// The number comes first, so the views below a finalized block are pruned in key order whatever their fork.
	finalityViewPrefix = []byte("posv-finality-view-")
)

// finalityBlockChain is the chain access needed to derive the view of a block from an earlier one: headers to walk back, bodies to count the votes.
type finalityBlockChain interface {
	finalityChain
	GetBlock(hash common.Hash, number uint64) *types.Block
}

// Bring finality tracking up to head, which must be the head of the canonical chain.
// The view of head is derived and stored, and the finalized block moves up when that view finalized a higher one.
// A finalized block that left the canonical chain, as after a rewind, is forgotten first, so only the one of head's view can replace it.
// While finality is not active at head, nothing else happens.
func (c *Posv) UpdateFinality(chain consensus.ChainReader, head *types.Header) error {
	if err := c.forgetDroppedFinalizedBlock(chain, head); err != nil {
		return err
	}
	view, err := c.finalityView(chain, head)
	if errors.Is(err, errFinalityInactive) {
		return nil
	}
	if err != nil {
		return err
	}
	return c.advanceFinalizedBlock(view)
}

// Move the finalized block up to the one of view when that one is higher, and prune the stored views below it.
// A view with a lower finalized block, as on a fork that lacks some votes, changes nothing.
func (c *Posv) advanceFinalizedBlock(view *FinalityView) error {
	c.finalityLock.Lock()
	defer c.finalityLock.Unlock()

	if view.Finalized == nil || (c.finalized != nil && view.Finalized.Number <= c.finalized.Number) {
		return nil
	}
	if err := storeFinalizedBlock(c.db, view.Finalized); err != nil {
		return err
	}
	c.finalized = view.Finalized
	log.Debug("[PoSV] Finalized block", "number", c.finalized.Number, "hash", c.finalized.Hash, "justified", view.Justified.Number, "head", view.Number)
	return c.pruneFinalityViews(c.finalized.Number)
}

// Return the finality view at header, derived from the nearest ancestor whose view is known.
// Views of signed blocks are stored on the way, so later calls and restarts start close to the head.
// When no view is known within one epoch of blocks, the walk stops at an unanchored view instead of replaying the chain.
// Return errFinalityInactive for a block below the root.
func (c *Posv) finalityView(chain finalityBlockChain, header *types.Header) (*FinalityView, error) {
	p, err := newFinalityParams(chain.Config())
	if err != nil {
		return nil, err
	}
	if header.Number.Uint64() < p.rootNumber() {
		return nil, errFinalityInactive
	}

	// Walk back to nearest block whose view is known, collecting the blocks to replay.
	var (
		view    *FinalityView
		pending []*types.Header
	)
	for cursor := header; ; {
		if view, err = c.knownFinalityView(chain, p, cursor, uint64(len(pending))); err != nil {
			return nil, err
		}
		if view != nil {
			break
		}
		pending = append(pending, cursor)
		parent := chain.GetHeader(cursor.ParentHash, cursor.Number.Uint64()-1)
		if parent == nil {
			return nil, fmt.Errorf("%w: parent of block %d (%s)", consensus.ErrUnknownAncestor, cursor.Number, cursor.Hash())
		}
		cursor = parent
	}

	for i := len(pending) - 1; i >= 0; i-- {
		number, hash := pending[i].Number.Uint64(), pending[i].Hash()
		block := chain.GetBlock(hash, number)
		if block == nil {
			return nil, fmt.Errorf("%w: body of block %d (%s)", errUnknownBlock, number, hash)
		}
		if view, err = view.apply(chain, block); err != nil {
			return nil, err
		}
		if err := c.keepFinalityView(p, view); err != nil {
			return nil, err
		}
	}
	return view, nil
}

// Forget the finalized block when it is no longer on the canonical chain that ends at head.
// It runs before the view of head is derived, so a view that cannot be derived never leaves a finalized block the chain does not contain.
func (c *Posv) forgetDroppedFinalizedBlock(chain consensus.ChainHeaderReader, head *types.Header) error {
	c.finalityLock.Lock()
	defer c.finalityLock.Unlock()

	if c.finalized == nil || isCanonical(chain, *c.finalized) {
		return nil
	}
	log.Warn("[PoSV] Finalized block left the canonical chain", "number", c.finalized.Number, "hash", c.finalized.Hash, "head", head.Number)
	if err := storeFinalizedBlock(c.db, nil); err != nil {
		return err
	}
	c.finalized = nil
	return nil
}

// Cache a derived view, and store it when it belongs to a signed block.
func (c *Posv) keepFinalityView(p *finalityParams, view *FinalityView) error {
	c.finalityViews.Add(view.Hash, view)
	if view.Number%p.interval != 0 {
		return nil
	}
	blob, err := json.Marshal(view)
	if err != nil {
		return err
	}
	return c.db.Put(finalityViewKey(BlockRef{Number: view.Number, Hash: view.Hash}), blob)
}

// Return the view at header when it needs no replay: it is cached, stored, or one of the two views a chain can start from.
// Return nil when the view has to be derived from the parent's. walked is the number of blocks already collected for replay.
func (c *Posv) knownFinalityView(chain finalityChain, p *finalityParams, header *types.Header, walked uint64) (*FinalityView, error) {
	number, hash := header.Number.Uint64(), header.Hash()
	if cached, ok := c.finalityViews.Get(hash); ok {
		return cached.(*FinalityView), nil
	}
	if number%p.interval == 0 {
		if view := loadFinalityView(c.db, BlockRef{Number: number, Hash: hash}); view != nil {
			c.finalityViews.Add(hash, view)
			return view, nil
		}
	}
	isRoot := number == p.rootNumber()
	if !isRoot && walked < p.epoch {
		return nil, nil
	}

	epoch, err := epochHeaderOf(chain, header, p.epoch)
	if err != nil {
		return nil, err
	}
	var view *FinalityView
	if isRoot {
		view, err = newRootFinalityView(chain.Config(), header, epoch)
	} else {
		// History this deep is not replayed: one epoch of votes is enough for the view to catch up with the chain.
		view, err = newUnanchoredFinalityView(chain.Config(), header, epoch)
	}
	if err != nil {
		return nil, err
	}
	return view, c.keepFinalityView(p, view)
}

// Delete the stored views of all blocks below the given number, on every fork.
// A view below the finalized block is never the start of a replay again: the view of the finalized block is nearer.
func (c *Posv) pruneFinalityViews(below uint64) error {
	it := c.db.NewIterator(finalityViewPrefix, nil)
	defer it.Release()

	batch := c.db.NewBatch()
	for it.Next() {
		key := it.Key()
		if len(key) != len(finalityViewPrefix)+finalityNumberLength+common.HashLength {
			continue
		}
		if binary.BigEndian.Uint64(key[len(finalityViewPrefix):]) >= below {
			break
		}
		if err := batch.Delete(common.CopyBytes(key)); err != nil {
			return err
		}
	}
	if err := it.Error(); err != nil {
		return err
	}
	return batch.Write()
}

// Return the latest epoch header at or below header, found through header's own ancestry.
func epochHeaderOf(chain finalityChain, header *types.Header, epoch uint64) (*types.Header, error) {
	for header.Number.Uint64()%epoch != 0 {
		parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
		if parent == nil {
			return nil, fmt.Errorf("%w: parent of block %d (%s)", consensus.ErrUnknownAncestor, header.Number, header.Hash())
		}
		header = parent
	}
	return header, nil
}

// Return the database key of the finality view of a signed block.
func finalityViewKey(block BlockRef) []byte {
	key := make([]byte, len(finalityViewPrefix)+finalityNumberLength+common.HashLength)
	n := copy(key, finalityViewPrefix)
	binary.BigEndian.PutUint64(key[n:], block.Number)
	copy(key[n+finalityNumberLength:], block.Hash.Bytes())
	return key
}

// Report whether the referenced block is on the canonical chain.
func isCanonical(chain consensus.ChainHeaderReader, block BlockRef) bool {
	header := chain.GetHeaderByNumber(block.Number)
	return header != nil && header.Hash() == block.Hash
}

// Return the stored finalized block, or nil when none is stored or the stored one cannot be read.
// An unreadable block is replaced on the next update of finality.
func loadFinalizedBlock(db ethdb.KeyValueReader) *BlockRef {
	blob, err := db.Get(finalizedBlockKey)
	if err != nil {
		return nil
	}
	block := new(BlockRef)
	if err := json.Unmarshal(blob, block); err != nil {
		log.Warn("[PoSV] Stored finalized block is unreadable", "err", err)
		return nil
	}
	return block
}

// Return the stored finality view of a signed block, or nil when none is stored or the stored one cannot be read.
// An unreadable view is derived again by the caller, which overwrites it.
func loadFinalityView(db ethdb.KeyValueReader, block BlockRef) *FinalityView {
	blob, err := db.Get(finalityViewKey(block))
	if err != nil {
		return nil
	}
	view := new(FinalityView)
	if err := json.Unmarshal(blob, view); err != nil {
		log.Warn("[PoSV] Stored finality view is unreadable", "number", block.Number, "hash", block.Hash, "err", err)
		return nil
	}
	if view.Number != block.Number || view.Hash != block.Hash {
		log.Warn("[PoSV] Stored finality view belongs to another block", "number", block.Number, "hash", block.Hash)
		return nil
	}
	return view
}

// Store the finalized block, or delete the stored one when block is nil.
func storeFinalizedBlock(db ethdb.KeyValueWriter, block *BlockRef) error {
	if block == nil {
		return db.Delete(finalizedBlockKey)
	}
	blob, err := json.Marshal(block)
	if err != nil {
		return err
	}
	return db.Put(finalizedBlockKey, blob)
}
