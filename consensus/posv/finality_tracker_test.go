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
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestUpdateFinalityAdvancesFinalizedBlock(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	tips := chain.tipsTo(t, chain.start(t), 46, quorum, isEveryTarget)

	chain.moveHead(t, engine, tips[30].header)
	requireTrackedFinalized(t, engine, nil)

	chain.moveHead(t, engine, tips[31].header)
	requireTrackedFinalized(t, engine, tips[15].header)

	chain.moveHead(t, engine, tips[46].header)
	requireTrackedFinalized(t, engine, tips[30].header)
}

func TestUpdateFinalityIgnoresHeadBelowRoot(t *testing.T) {
	chain, _ := newFinalityTrackerTestChain(t, 100)
	db := rawdb.NewMemoryDatabase()
	engine := New(chain.config.Posv, db)
	head := chain.addHeader(nil)
	for head.Number.Uint64() < 89 {
		head = chain.addHeader(head)
	}

	chain.moveHead(t, engine, head)

	requireTrackedFinalized(t, engine, nil)
	requireStoredViews(t, db)
}

func TestUpdateFinalityKeepsFinalizedBlockOnForkWithoutItsVotes(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)
	chain.fork = 1
	fork := chain.extendTo(t, tips[45], 50)
	requireFinalized(t, fork.view, tips[15].header)
	chain.moveHead(t, engine, tips[47].header)

	chain.moveHead(t, engine, fork.header)

	requireTrackedFinalized(t, engine, tips[30].header)
}

func TestUpdateFinalityForgetsFinalizedBlockOfAbandonedFork(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)
	chain.fork = 1
	fork := chain.extendTo(t, tips[20], 50)
	chain.moveHead(t, engine, tips[47].header)
	requireTrackedFinalized(t, engine, tips[30].header)

	chain.moveHead(t, engine, fork.header)

	requireTrackedFinalized(t, engine, nil)
}

func TestUpdateFinalityRewindsFinalizedBlock(t *testing.T) {
	tests := []struct {
		name          string
		prometheus    uint64
		rewindTo      uint64
		wantFinalized uint64 // 0 for none: the lowest signed block is the root at best, which is never finalized
	}{
		{name: "head above the finalized block keeps it", rewindTo: 50, wantFinalized: 45},
		{name: "head at the finalized block keeps it", rewindTo: 45, wantFinalized: 45},
		{name: "head below it falls back to the finalized block of the head's view", rewindTo: 35, wantFinalized: 15},
		{name: "head whose view finalized nothing leaves none", rewindTo: 20},
		{name: "head below the root leaves none", prometheus: 20, rewindTo: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, quorum := newFinalityTrackerTestChain(t, tt.prometheus)
			engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
			tips := chain.tipsTo(t, chain.start(t), 62, quorum, isEveryTarget)
			head := tips[62].header
			chain.moveHead(t, engine, head)
			requireTrackedFinalized(t, engine, chain.headerAt(t, head, 45))

			chain.moveHead(t, engine, chain.headerAt(t, head, tt.rewindTo))

			if tt.wantFinalized == 0 {
				requireTrackedFinalized(t, engine, nil)
				return
			}
			requireTrackedFinalized(t, engine, chain.headerAt(t, head, tt.wantFinalized))
		})
	}
}

func TestUpdateFinalityForgetsFinalizedBlockWhenViewIsUnavailable(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	db := rawdb.NewMemoryDatabase()
	tips := chain.tipsTo(t, chain.start(t), 62, quorum, isEveryTarget)
	chain.moveHead(t, New(chain.config.Posv, db), tips[62].header)
	restarted := New(chain.config.Posv, db)
	requireTrackedFinalized(t, restarted, tips[45].header)
	chain.dropBodies(tips[62].header, 35)

	chain.head = tips[35].header
	err := restarted.UpdateFinality(chain, chain.head)

	if !errors.Is(err, errUnknownBlock) {
		t.Fatalf("error: have %v, want %v", err, errUnknownBlock)
	}
	requireTrackedFinalized(t, restarted, nil)
}

func TestFinalizedBlockSurvivesRestart(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	db := rawdb.NewMemoryDatabase()
	tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)
	chain.moveHead(t, New(chain.config.Posv, db), tips[47].header)

	restarted := New(chain.config.Posv, db)

	requireTrackedFinalized(t, restarted, tips[30].header)
}

func TestForgottenFinalizedBlockStaysForgottenAfterRestart(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	db := rawdb.NewMemoryDatabase()
	engine := New(chain.config.Posv, db)
	tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)
	chain.moveHead(t, engine, tips[47].header)
	chain.moveHead(t, engine, tips[20].header)

	restarted := New(chain.config.Posv, db)

	requireTrackedFinalized(t, restarted, nil)
}

func TestFinalizedBlockReportsTrackedBlock(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)

	chain.moveHead(t, engine, tips[30].header)
	none, isKnownEarly := engine.FinalizedBlock()
	chain.moveHead(t, engine, tips[47].header)
	finalized, isKnown := engine.FinalizedBlock()

	if isKnownEarly || none != (BlockRef{}) {
		t.Fatalf("finalized block before any is finalized: have %d (known %t), want none", none.Number, isKnownEarly)
	}
	if !isKnown || finalized != blockRefOf(tips[30].header) {
		t.Fatalf("finalized block: have %d (%s, known %t), want 30 (%s)", finalized.Number, finalized.Hash, isKnown, tips[30].header.Hash())
	}
}

func TestJustifiedBlock(t *testing.T) {
	tests := []struct {
		name string
		head uint64
		want uint64
	}{
		{name: "root until a target is justified", head: 15, want: 0},
		{name: "target once its votes are included", head: 16, want: 15},
		{name: "highest of the justified targets", head: 47, want: 45},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, quorum := newFinalityTrackerTestChain(t, 0)
			engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
			tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)

			justified, err := engine.JustifiedBlock(chain, tips[tt.head].header)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if justified != blockRefOf(tips[tt.want].header) {
				t.Fatalf("justified block: have %d (%s), want %d (%s)", justified.Number, justified.Hash, tt.want, tips[tt.want].header.Hash())
			}
		})
	}
}

func TestJustifiedBlockOfSideChain(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)
	chain.fork = 1
	fork := chain.extendTo(t, tips[20], 50)
	chain.moveHead(t, engine, tips[47].header)

	justified, err := engine.JustifiedBlock(chain, fork.header)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if justified != blockRefOf(tips[15].header) {
		t.Fatalf("justified block: have %d, want 15, the last one justified before the fork", justified.Number)
	}
}

func TestJustifiedBlockUnknownBelowRoot(t *testing.T) {
	chain, _ := newFinalityTrackerTestChain(t, 100)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	head := chain.addHeader(nil)
	for head.Number.Uint64() < 89 {
		head = chain.addHeader(head)
	}

	justified, err := engine.JustifiedBlock(chain, head)

	if !errors.Is(err, ErrNoJustifiedBlock) {
		t.Fatalf("error: have %v, want %v", err, ErrNoJustifiedBlock)
	}
	if justified != (BlockRef{}) {
		t.Fatalf("justified block: have %d, want none", justified.Number)
	}
}

func TestJustifiedBlockUnknownInUnanchoredView(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	// Block 15 is justified more than one epoch below the head, out of reach of the unanchored start at 140.
	tips := chain.tipsTo(t, chain.start(t), 200, quorum, func(target uint64) bool { return target == 15 })
	requireJustified(t, tips[200].view, tips[15].header)

	justified, err := engine.JustifiedBlock(chain, tips[200].header)

	if !errors.Is(err, ErrNoJustifiedBlock) {
		t.Fatalf("error: have %v, want %v", err, ErrNoJustifiedBlock)
	}
	if justified != (BlockRef{}) {
		t.Fatalf("justified block: have %d, want none", justified.Number)
	}
}

func TestJustifiedBlockFailsOnMissingChainData(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	tips := chain.tipsTo(t, chain.start(t), 10, quorum, isEveryTarget)
	delete(chain.blocks, tips[5].header.Hash())

	_, err := engine.JustifiedBlock(chain, tips[10].header)

	if !errors.Is(err, errUnknownBlock) {
		t.Fatalf("error: have %v, want %v", err, errUnknownBlock)
	}
}

func TestFinalityViewEqualsStepwiseView(t *testing.T) {
	tests := []struct {
		name string
		head uint64
	}{
		{name: "root", head: 0},
		{name: "block before the first target", head: 14},
		{name: "first target", head: 15},
		{name: "block that justifies the first target", head: 16},
		{name: "block with an open vote window", head: 40},
		{name: "block that finalizes the second target", head: 46},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, quorum := newFinalityTrackerTestChain(t, 0)
			engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
			tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)

			view, err := engine.finalityView(chain, tips[tt.head].header)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			requireSameView(t, view, tips[tt.head].view)
		})
	}
}

func TestFinalityViewStoredOnlyForSignedBlocks(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	db := rawdb.NewMemoryDatabase()
	engine := New(chain.config.Posv, db)
	tips := chain.tipsTo(t, chain.start(t), 20, quorum, isEveryTarget)

	chain.moveHead(t, engine, tips[20].header)

	requireStoredViews(t, db, tips[0].header, tips[15].header)
}

func TestFinalityViewRestartsFromStoredView(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	db := rawdb.NewMemoryDatabase()
	tips := chain.tipsTo(t, chain.start(t), 50, quorum, isEveryTarget)
	chain.moveHead(t, New(chain.config.Posv, db), tips[50].header)
	restarted := New(chain.config.Posv, db)
	// Without the bodies up to the last signed block, only its stored view can lead to the head's view.
	chain.dropBodies(tips[50].header, 45)

	view, err := restarted.finalityView(chain, tips[50].header)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	requireSameView(t, view, tips[50].view)
}

func TestFinalityViewRederivesUnusableStoredView(t *testing.T) {
	tests := []struct {
		name   string
		stored func(t *testing.T, other *FinalityView) []byte
	}{
		{
			name:   "unreadable view",
			stored: func(t *testing.T, other *FinalityView) []byte { return []byte("{") },
		},
		{
			name: "view of another block",
			stored: func(t *testing.T, other *FinalityView) []byte {
				blob, err := json.Marshal(other)
				if err != nil {
					t.Fatalf("encode view: %v", err)
				}
				return blob
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, quorum := newFinalityTrackerTestChain(t, 0)
			db := rawdb.NewMemoryDatabase()
			tips := chain.tipsTo(t, chain.start(t), 35, quorum, isEveryTarget)
			chain.moveHead(t, New(chain.config.Posv, db), tips[20].header)
			signed := blockRefOf(tips[15].header)
			if err := db.Put(finalityViewKey(signed), tt.stored(t, tips[30].view)); err != nil {
				t.Fatalf("overwrite stored view: %v", err)
			}
			restarted := New(chain.config.Posv, db)

			view, err := restarted.finalityView(chain, tips[20].header)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			requireSameView(t, view, tips[20].view)
			stored := loadFinalityView(db, signed)
			if stored == nil {
				t.Fatalf("stored view of block %d: have none, want the rederived view", signed.Number)
			}
			requireSameView(t, stored, tips[15].view)
		})
	}
}

func TestFinalityViewFollowsReorg(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)
	chain.fork = 1
	fork := chain.tipsTo(t, tips[20], 50, quorum, func(target uint64) bool { return target == 45 })
	chain.moveHead(t, engine, tips[47].header)

	chain.moveHead(t, engine, fork[50].header)
	forkView, forkErr := engine.finalityView(chain, fork[50].header)
	view, err := engine.finalityView(chain, tips[47].header)

	if forkErr != nil || err != nil {
		t.Fatalf("unexpected errors: fork %v, first chain %v", forkErr, err)
	}
	requireSameView(t, forkView, fork[50].view)
	requireSameView(t, view, tips[47].view)
	requireJustified(t, forkView, fork[45].header)
	requireFinalized(t, forkView, nil)
}

func TestUpdateFinalityPrunesViewsBelowFinalizedBlock(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	db := rawdb.NewMemoryDatabase()
	engine := New(chain.config.Posv, db)
	tips := chain.tipsTo(t, chain.start(t), 47, quorum, isEveryTarget)
	chain.fork = 1
	fork := chain.extendTo(t, tips[10], 20)
	chain.moveHead(t, engine, fork.header)
	chain.moveHead(t, engine, tips[20].header)
	requireStoredViews(t, db, tips[0].header, tips[15].header, chain.headerAt(t, fork.header, 15))

	chain.moveHead(t, engine, tips[47].header)

	requireTrackedFinalized(t, engine, tips[30].header)
	requireStoredViews(t, db, tips[30].header, tips[45].header)
}

func TestFinalityViewStartsUnanchoredBeyondOneEpoch(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	tips := chain.tipsTo(t, chain.start(t), 200, quorum, isEveryTarget)
	// Walk gives up one epoch below the head; without the bodies up to there, a replay from the root cannot be what derives the view.
	chain.dropBodies(tips[200].header, 200-finalityTestEpoch)

	view, err := engine.finalityView(chain, tips[200].header)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	requireSameView(t, view, tips[200].view)
	requireFinalized(t, view, tips[180].header)
}

func TestFinalityViewUnanchoredClaimsNoMoreThanVotesProve(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	// Block 120 is justified below the unanchored start at 130, so only the full history shows it as the source that block 135 finalizes.
	tips := chain.tipsTo(t, chain.start(t), 190, quorum, func(target uint64) bool { return target == 120 || target == 135 })
	requireFinalized(t, tips[190].view, tips[120].header)

	view, err := engine.finalityView(chain, tips[190].header)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	requireJustified(t, view, tips[135].header)
	requireFinalized(t, view, nil)
}

func TestFinalityViewFailsOnMissingChainData(t *testing.T) {
	tests := []struct {
		name    string
		remove  func(chain *finalityTestChain, tips map[uint64]finalityTestTip)
		wantErr error
	}{
		{
			name: "body of a block to replay",
			remove: func(chain *finalityTestChain, tips map[uint64]finalityTestTip) {
				delete(chain.blocks, tips[5].header.Hash())
			},
			wantErr: errUnknownBlock,
		},
		{
			name: "header of an ancestor",
			remove: func(chain *finalityTestChain, tips map[uint64]finalityTestTip) {
				delete(chain.headers, tips[5].header.Hash())
			},
			wantErr: consensus.ErrUnknownAncestor,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, quorum := newFinalityTrackerTestChain(t, 0)
			engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
			tips := chain.tipsTo(t, chain.start(t), 10, quorum, isEveryTarget)
			tt.remove(chain, tips)

			_, err := engine.finalityView(chain, tips[10].header)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: have %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewUnanchoredFinalityView(t *testing.T) {
	chain, _ := newFinalityTrackerTestChain(t, 20)
	byNumber := map[uint64]*types.Header{0: chain.addHeader(nil)}
	for number := uint64(1); number <= 70; number++ {
		byNumber[number] = chain.addHeader(byNumber[number-1])
	}

	view, err := newUnanchoredFinalityView(chain.config, byNumber[70], byNumber[60])

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Number != 70 || view.Hash != byNumber[70].Hash() {
		t.Fatalf("view block: have %d (%s), want 70 (%s)", view.Number, view.Hash, byNumber[70].Hash())
	}
	if view.Epoch != blockRefOf(byNumber[60]) {
		t.Fatalf("epoch header: have %d, want 60", view.Epoch.Number)
	}
	if view.Justified != (BlockRef{}) {
		t.Fatalf("justified block: have %d, want none", view.Justified.Number)
	}
	requireFinalized(t, view, nil)
	if len(view.Tallies) != 0 {
		t.Fatalf("open tallies: have %d, want 0", len(view.Tallies))
	}
}

func TestNewUnanchoredFinalityViewRejectsWrongHeaders(t *testing.T) {
	tests := []struct {
		name    string
		block   uint64
		epoch   uint64
		wantErr error
	}{
		{name: "block below the root", block: 14, epoch: 0, wantErr: errFinalityInactive},
		{name: "header that is not the epoch header", block: 70, epoch: 0, wantErr: errInvalidFinalityRoot},
		{name: "epoch header above the block", block: 59, epoch: 60, wantErr: errInvalidFinalityRoot},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, _ := newFinalityTrackerTestChain(t, 20)
			byNumber := map[uint64]*types.Header{0: chain.addHeader(nil)}
			for number := uint64(1); number <= 70; number++ {
				byNumber[number] = chain.addHeader(byNumber[number-1])
			}

			_, err := newUnanchoredFinalityView(chain.config, byNumber[tt.block], byNumber[tt.epoch])

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: have %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// newFinalityTrackerTestChain returns a chain of four validators and the keys of three of them, the fewest that justify a block.
func newFinalityTrackerTestChain(t *testing.T, prometheus uint64) (*finalityTestChain, []*ecdsa.PrivateKey) {
	t.Helper()
	keys := finalityTestKeys(t, 0, 4)
	return newFinalityTestChain(prometheus, finalityTestAddresses(keys)), keys[:3]
}

func isEveryTarget(target uint64) bool {
	return true
}

func (c *finalityTestChain) CurrentHeader() *types.Header {
	return c.head
}

func (c *finalityTestChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return c.headers[hash]
}

// GetHeaderByNumber resolves number on the chain that ends at the head, as the canonical index of a node does.
func (c *finalityTestChain) GetHeaderByNumber(number uint64) *types.Header {
	if c.head == nil || number > c.head.Number.Uint64() {
		return nil
	}
	header := c.head
	for header != nil && header.Number.Uint64() > number {
		header = c.headers[header.ParentHash]
	}
	return header
}

func (c *finalityTestChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	block := c.blocks[hash]
	if block == nil || block.NumberU64() != number {
		return nil
	}
	return block
}

// tipsTo extends from to the block at number and returns every tip on the way, from included, by block number.
// The block after each signed block for which isVoted reports true includes the votes of keys for that block.
func (c *finalityTestChain) tipsTo(t *testing.T, from finalityTestTip, number uint64, keys []*ecdsa.PrivateKey, isVoted func(target uint64) bool) map[uint64]finalityTestTip {
	t.Helper()
	tips := map[uint64]finalityTestTip{from.header.Number.Uint64(): from}
	for tip := from; tip.header.Number.Uint64() < number; {
		var txs []*types.Transaction
		if parent := tip.header.Number.Uint64(); parent%finalityTestInterval == 0 && isVoted(parent) {
			txs = c.votes(t, keys, tip.header)
		}
		tip = c.extend(t, tip, txs...)
		tips[tip.header.Number.Uint64()] = tip
	}
	return tips
}

// headerAt returns the ancestor of head at number.
func (c *finalityTestChain) headerAt(t *testing.T, head *types.Header, number uint64) *types.Header {
	t.Helper()
	header := head
	for header != nil && header.Number.Uint64() > number {
		header = c.headers[header.ParentHash]
	}
	if header == nil || header.Number.Uint64() != number {
		t.Fatalf("block %d has no ancestor at %d", head.Number, number)
	}
	return header
}

// dropBodies removes the bodies of the ancestors of head at or below number.
func (c *finalityTestChain) dropBodies(head *types.Header, number uint64) {
	for header := head; header != nil; header = c.headers[header.ParentHash] {
		if header.Number.Uint64() <= number {
			delete(c.blocks, header.Hash())
		}
	}
}

// moveHead makes head the head of the canonical chain and brings the engine's finality tracking up to it.
func (c *finalityTestChain) moveHead(t *testing.T, engine *Posv, head *types.Header) {
	t.Helper()
	c.head = head
	if err := engine.UpdateFinality(c, head); err != nil {
		t.Fatalf("update finality at block %d: %v", head.Number, err)
	}
}

// requireTrackedFinalized fails unless the engine tracks want as finalized block; a nil want expects none.
func requireTrackedFinalized(t *testing.T, engine *Posv, want *types.Header) {
	t.Helper()
	engine.finalityLock.Lock()
	finalized := engine.finalized
	engine.finalityLock.Unlock()

	switch {
	case want == nil && finalized != nil:
		t.Fatalf("tracked finalized block: have %d, want none", finalized.Number)
	case want != nil && finalized == nil:
		t.Fatalf("tracked finalized block: have none, want %d", want.Number)
	case want != nil && *finalized != blockRefOf(want):
		t.Fatalf("tracked finalized block: have %d (%s), want %d (%s)", finalized.Number, finalized.Hash, want.Number, want.Hash())
	}
}

// requireStoredViews fails unless the database stores the views of exactly the given blocks.
func requireStoredViews(t *testing.T, db ethdb.Database, want ...*types.Header) {
	t.Helper()
	stored := make(map[BlockRef]bool)
	it := db.NewIterator(finalityViewPrefix, nil)
	defer it.Release()
	for it.Next() {
		key := it.Key()[len(finalityViewPrefix):]
		stored[BlockRef{Number: binary.BigEndian.Uint64(key), Hash: common.BytesToHash(key[finalityNumberLength:])}] = true
	}
	if err := it.Error(); err != nil {
		t.Fatalf("iterate stored views: %v", err)
	}

	for _, header := range want {
		if !stored[blockRefOf(header)] {
			t.Fatalf("stored views: block %d (%s) is missing", header.Number, header.Hash())
		}
	}
	if len(stored) != len(want) {
		t.Fatalf("stored views: have %d, want %d", len(stored), len(want))
	}
}

// requireSameView fails unless both views hold the same finality state; a nil list equals an empty one.
func requireSameView(t *testing.T, have, want *FinalityView) {
	t.Helper()
	if have.Number != want.Number || have.Hash != want.Hash {
		t.Fatalf("view block: have %d (%s), want %d (%s)", have.Number, have.Hash, want.Number, want.Hash)
	}
	if have.Epoch != want.Epoch {
		t.Fatalf("epoch header at %d: have %d, want %d", want.Number, have.Epoch.Number, want.Epoch.Number)
	}
	if have.Justified != want.Justified {
		t.Fatalf("justified block at %d: have %d, want %d", want.Number, have.Justified.Number, want.Justified.Number)
	}
	if (have.Finalized == nil) != (want.Finalized == nil) || (have.Finalized != nil && *have.Finalized != *want.Finalized) {
		t.Fatalf("finalized block at %d: have %v, want %v", want.Number, have.Finalized, want.Finalized)
	}
	if len(have.Tallies) != len(want.Tallies) {
		t.Fatalf("open tallies at %d: have %d, want %d", want.Number, len(have.Tallies), len(want.Tallies))
	}
	for i, tally := range want.Tallies {
		other := have.Tallies[i]
		if other.Target != tally.Target || other.Source != tally.Source || other.Epoch != tally.Epoch {
			t.Fatalf("tally %d at %d: have %+v, want %+v", i, want.Number, other, tally)
		}
		if len(other.Voters) != len(tally.Voters) {
			t.Fatalf("voters of block %d at %d: have %d, want %d", tally.Target.Number, want.Number, len(other.Voters), len(tally.Voters))
		}
		for j, voter := range tally.Voters {
			if other.Voters[j] != voter {
				t.Fatalf("voter %d of block %d at %d: have %s, want %s", j, tally.Target.Number, want.Number, other.Voters[j], voter)
			}
		}
	}
}
