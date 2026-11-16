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

package core

import (
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// posvFinalityChain is the chain access the PoSV finality tracker reads blocks, receipts and the canonical mapping through.
type posvFinalityChain interface {
	Config() *params.ChainConfig
	GetBlock(hash common.Hash, number uint64) *types.Block
	GetCanonicalHash(number uint64) common.Hash
	GetHeader(hash common.Hash, number uint64) *types.Header
	GetHeaderByHash(hash common.Hash) *types.Header
	GetReceiptsByHash(hash common.Hash) types.Receipts
}

// posvFinality tracks the highest block finalized by the BlockSigner votes recorded on the canonical chain.
//
// A block is finalized once its creator, its attestor and the senders of successful BlockSigner transactions for it,
// included in the ConsensusLimitTimeFinality blocks after it on the same chain, cover posv.FinalityThresholdPercent
// of the validators in its epoch checkpoint header. Every ancestor of a finalized block is final too.
//
// The tallies are read and written under the chain insertion lock only; the finalized header is safe to read without it.
type posvFinality struct {
	chain        posvFinalityChain
	db           ethdb.KeyValueStore
	engine       *posv.Posv
	signContract common.Address
	epoch        uint64
	window       uint64 // blocks after a signed block whose votes count toward its finality

	finalized   atomic.Value // *types.Header of the highest finalized block, nil when none
	lastApplied common.Hash  // canonical block whose votes were tallied last
	tallies     map[common.Hash]*posv.FinalityTally
}

// Create the finality tracker of a PoSV chain, or return nil when engine is not PoSV or the chain config has no PoSV epoch or finality window.
func newPosvFinality(chain posvFinalityChain, db ethdb.KeyValueStore, engine consensus.Engine) *posvFinality {
	posvEngine, ok := engine.(*posv.Posv)
	config := chain.Config()
	if !ok || config.Posv == nil || config.Posv.Epoch == 0 || config.Viction == nil || config.Viction.ConsensusLimitTimeFinality == 0 {
		return nil
	}
	f := &posvFinality{
		chain:        chain,
		db:           db,
		engine:       posvEngine,
		signContract: config.Viction.ValidatorBlockSignContract,
		epoch:        config.Posv.Epoch,
		window:       config.Viction.ConsensusLimitTimeFinality,
		tallies:      make(map[common.Hash]*posv.FinalityTally),
	}
	f.finalized.Store((*types.Header)(nil))
	return f
}

// Return the highest finalized block, or nil when none is known.
func (f *posvFinality) finalizedHeader() *types.Header {
	return f.finalized.Load().(*types.Header)
}

// Reload the finalized block stored in the database for the given head and restart tallying from the next canonical block.
// A stored block that is not a canonical ancestor of head, e.g. after a rewind below it, is dropped.
func (f *posvFinality) load(head *types.Header) {
	f.lastApplied = common.Hash{}
	f.tallies = make(map[common.Hash]*posv.FinalityTally)
	f.finalized.Store((*types.Header)(nil))

	hash := rawdb.ReadPosvFinalizedBlockHash(f.db)
	if hash == (common.Hash{}) {
		return
	}
	header := f.chain.GetHeaderByHash(hash)
	if header == nil || !f.isCanonicalAncestor(header, head.Number.Uint64()+1) {
		log.Warn("[PoSV] Dropped finalized block that is not on the canonical chain", "hash", hash, "head", head.Number)
		rawdb.DeletePosvFinalizedBlockHash(f.db)
		return
	}
	f.finalized.Store(header)
	log.Info("[PoSV] Loaded finalized block", "number", header.Number, "hash", hash)
}

// Tally the votes of a block that just became the canonical head, advancing the finalized block when one reaches quorum.
// All ancestors of head must already be canonical. A head that does not extend the last tallied block, e.g. after a reorg
// or a restart, first has the votes of the blocks before it tallied again.
func (f *posvFinality) applyHead(head *types.Block) {
	if head.ParentHash() != f.lastApplied {
		f.retally(head)
	}
	f.tallyVotes(head)
	f.lastApplied = head.Hash()
}

// Discard the tallies and tally again the votes of the window before head, dropping a finalized block that head does not descend from.
func (f *posvFinality) retally(head *types.Block) {
	f.tallies = make(map[common.Hash]*posv.FinalityTally)
	number := head.NumberU64()
	if finalized := f.finalizedHeader(); finalized != nil && !f.isCanonicalAncestor(finalized, number) {
		log.Warn("[PoSV] Dropped finalized block reorganised out of the canonical chain", "number", finalized.Number, "hash", finalized.Hash(), "head", number)
		f.setFinalized(nil)
	}
	// Only votes for blocks from number-window on can still change a tally, and they are included from number-window+1 on.
	first := uint64(1)
	if number > f.window {
		first = number - f.window + 1
	}
	if number <= first {
		return
	}
	ancestors := make([]*types.Block, 0, number-first)
	for block := f.chain.GetBlock(head.ParentHash(), number-1); block != nil && block.NumberU64() >= first; block = f.chain.GetBlock(block.ParentHash(), block.NumberU64()-1) {
		ancestors = append(ancestors, block)
	}
	for i := len(ancestors) - 1; i >= 0; i-- {
		f.tallyVotes(ancestors[i])
	}
}

// Add the votes in block to the tallies of the blocks they sign, then advance the finalized block and drop tallies that can no longer change.
func (f *posvFinality) tallyVotes(block *types.Block) {
	var (
		number       = block.NumberU64()
		signer       = types.MakeSigner(f.chain.Config(), block.Number())
		receipts     types.Receipts
		receiptsRead bool
		ignored      map[common.Hash]struct{}
	)
	for i, tx := range block.Transactions() {
		target, ok := tx.SignedBlockHash(f.signContract)
		if !ok {
			continue
		}
		if _, skip := ignored[target]; skip {
			continue
		}
		tally := f.tallyOf(target, number)
		if tally == nil {
			if ignored == nil {
				ignored = make(map[common.Hash]struct{})
			}
			ignored[target] = struct{}{}
			continue
		}
		if !receiptsRead {
			receipts, receiptsRead = f.chain.GetReceiptsByHash(block.Hash()), true
		}
		if i >= len(receipts) || !receipts[i].IsSuccessful() {
			continue
		}
		sender, err := types.Sender(signer, tx)
		if err != nil {
			continue
		}
		tally.AddSigner(sender)
	}
	f.advance()
	f.prune(number)
}

// Return the tally of the signed block target for a vote included in the block at includedIn, opening it on the first vote.
// Return nil when the vote cannot count: the signed block is unknown, not a canonical ancestor, already final,
// or the vote is outside the finality window.
func (f *posvFinality) tallyOf(target common.Hash, includedIn uint64) *posv.FinalityTally {
	if tally, ok := f.tallies[target]; ok {
		if includedIn <= tally.Number || includedIn-tally.Number > f.window {
			return nil
		}
		return tally
	}
	header := f.chain.GetHeaderByHash(target)
	if header == nil {
		return nil
	}
	number := header.Number.Uint64()
	if number >= includedIn || includedIn-number > f.window || !f.isCanonicalAncestor(header, includedIn) {
		return nil
	}
	if finalized := f.finalizedHeader(); finalized != nil && number <= finalized.Number.Uint64() {
		return nil
	}
	validators := f.checkpointValidators(number)
	if len(validators) == 0 {
		return nil
	}
	tally := posv.NewFinalityTally(number, validators)
	if creator, err := f.engine.Author(header); err == nil {
		tally.AddSigner(creator)
	}
	if attestor, err := f.engine.Attestor(header); err == nil {
		tally.AddSigner(attestor)
	}
	f.tallies[target] = tally
	return tally
}

// Return the validators recorded in the checkpoint header of the epoch containing the block at number, or nil when it is unavailable.
func (f *posvFinality) checkpointValidators(number uint64) []common.Address {
	checkpointNumber := number - number%f.epoch
	checkpoint := f.chain.GetHeader(f.chain.GetCanonicalHash(checkpointNumber), checkpointNumber)
	if checkpoint == nil || len(checkpoint.Extra) < posv.ExtraVanity+posv.ExtraSeal {
		return nil
	}
	return posv.ExtractValidatorsFromCheckpointHeader(checkpoint)
}

// Report whether header is on the canonical chain below the block at number below.
func (f *posvFinality) isCanonicalAncestor(header *types.Header, below uint64) bool {
	number := header.Number.Uint64()
	return number < below && f.chain.GetCanonicalHash(number) == header.Hash()
}

// Finalize the highest tallied block that reached quorum, when it is above the finalized block.
func (f *posvFinality) advance() {
	var (
		best       common.Hash
		bestNumber uint64
		found      bool
	)
	for hash, tally := range f.tallies {
		if tally.HasQuorum() && (!found || tally.Number > bestNumber) {
			best, bestNumber, found = hash, tally.Number, true
		}
	}
	if !found {
		return
	}
	if finalized := f.finalizedHeader(); finalized != nil && bestNumber <= finalized.Number.Uint64() {
		return
	}
	header := f.chain.GetHeader(best, bestNumber)
	if header == nil {
		return
	}
	f.setFinalized(header)
	log.Info("[PoSV] Finalized block", "number", bestNumber, "hash", best)
}

// Drop the tallies of blocks that are final or can no longer receive votes after the block at number.
func (f *posvFinality) prune(number uint64) {
	finalized := f.finalizedHeader()
	for hash, tally := range f.tallies {
		if tally.Number+f.window <= number || (finalized != nil && tally.Number <= finalized.Number.Uint64()) {
			delete(f.tallies, hash)
		}
	}
}

// Set and persist the finalized block; nil clears it.
func (f *posvFinality) setFinalized(header *types.Header) {
	f.finalized.Store(header)
	if header == nil {
		rawdb.DeletePosvFinalizedBlockHash(f.db)
		return
	}
	rawdb.WritePosvFinalizedBlockHash(f.db, header.Hash())
}
