// Copyright 2014 The go-ethereum Authors
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

package core

import (
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/sortlgc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// Return underlying VictionProcessor instance in the Proccesor.
func (bc *BlockChain) VictionProcessor() *VictionProcessor {
	p, ok := bc.processor.(*StateProcessor)
	if !ok || p == nil {
		return nil
	}
	return p.viction
}

// Flush current block Lending State Trie to LevelDB.
func (bc *BlockChain) CommitLendingState(block *types.Block) error {
	p := bc.VictionProcessor()
	if p == nil || !p.IsLendingInitialized() {
		return nil
	}
	lendingRoot := p.CommittedLendingRoot()
	if lendingRoot == (common.Hash{}) {
		return nil
	}
	if err := p.LendingEngine().GetStateCache().TrieDB().Commit(lendingRoot, false, nil); err != nil {
		return fmt.Errorf("native_lending: failed to commit Trie at block %d: %w", block.NumberU64(), err)
	}
	log.Debug("[NativeLending] Flushed Trie to disk", "block", block.NumberU64(), "root", lendingRoot.Hex())
	return nil
}

// Flush current block Lending State Trie in GC cache to LevelDB.
func (bc *BlockChain) CommitLendingStateDeferred(block *types.Block) error {
	p := bc.VictionProcessor()
	if p == nil || !p.IsLendingInitialized() {
		return nil
	}
	current := block.NumberU64()
	lendingRoot := p.CommittedLendingRoot()
	if lendingRoot == (common.Hash{}) {
		return nil
	}
	lendingTrieDB := p.LendingEngine().GetStateCache().TrieDB()
	lendingTrieDB.Reference(lendingRoot, common.Hash{})
	bc.lendingTriegc.Push(lendingRoot, -int64(current))

	if err := lendingTrieDB.Commit(lendingRoot, true, nil); err != nil {
		return fmt.Errorf("native_lending: failed to commit Trie at block %d: %w", current, err)
	}
	log.Debug("[NativeLending] Flushed Trie to disk", "block", current, "root", lendingRoot.Hex())

	if current > TriesInMemory {
		// If we exceeded our memory allowance, flush matured singleton nodes to disk
		var (
			nodes, imgs = lendingTrieDB.Size()
			limit       = common.StorageSize(bc.cacheConfig.TrieDirtyLimit) * 1024 * 1024
		)
		if nodes > limit || imgs > 4*1024*1024 {
			lendingTrieDB.Cap(limit - ethdb.IdealBatchSize)
		}
		chosen := current - TriesInMemory
		for !bc.lendingTriegc.Empty() {
			root, number := bc.lendingTriegc.Pop()
			if uint64(-number) > chosen {
				bc.lendingTriegc.Push(root, number)
				break
			}
			lendingTrieDB.Dereference(root.(common.Hash))
		}
	}
	return nil
}

// Flush all Lending State Trie entries in GC cache to LevelDB.
func (bc *BlockChain) FlushLendingStateGCCache() {
	p := bc.VictionProcessor()
	if bc.cacheConfig.TrieDirtyDisabled || p == nil || p.LendingEngine() == nil {
		return
	}

	lendingTrieDB := p.LendingEngine().GetStateCache().TrieDB()
	for !bc.lendingTriegc.Empty() {
		root := bc.lendingTriegc.PopItem()
		if err := lendingTrieDB.Commit(root.(common.Hash), true, nil); err != nil {
			log.Error("[NativeLending] Failed to commit Trie on shutdown", "root", root, "err", err)
		}
		lendingTrieDB.Dereference(root.(common.Hash))
	}
}

// Flush current block Trading State Trie to LevelDB.
func (bc *BlockChain) CommitTradingState(block *types.Block) error {
	p := bc.VictionProcessor()
	if p == nil || !p.IsTradingInitialized() {
		return nil
	}
	tradingRoot := p.CommittedTradingRoot()
	if tradingRoot == (common.Hash{}) {
		return nil
	}
	if err := p.TradingEngine().GetStateCache().TrieDB().Commit(tradingRoot, false, nil); err != nil {
		return fmt.Errorf("native_trading: failed to commit Trie at block %d: %w", block.NumberU64(), err)
	}
	log.Debug("[NativeTrading] Flushed Trie to disk", "block", block.NumberU64(), "root", tradingRoot.Hex())
	return nil
}

// Flush current block Trading State Trie in GC cache to LevelDB.
func (bc *BlockChain) CommitTradingStateDeferred(block *types.Block) error {
	p := bc.VictionProcessor()
	if p == nil || !p.IsTradingInitialized() {
		return nil
	}
	current := block.NumberU64()
	tradingRoot := p.CommittedTradingRoot()
	if tradingRoot == (common.Hash{}) {
		return nil
	}
	tradingTrieDB := p.TradingEngine().GetStateCache().TrieDB()
	tradingTrieDB.Reference(tradingRoot, common.Hash{})
	bc.tradingTriegc.Push(tradingRoot, -int64(current))

	if err := tradingTrieDB.Commit(tradingRoot, true, nil); err != nil {
		return fmt.Errorf("native_trading: failed to commit Trie at block %d: %w", current, err)
	}
	log.Debug("[NativeTrading] Flushed Trie to disk", "block", current, "root", tradingRoot.Hex())

	if current > TriesInMemory {
		// If we exceeded our memory allowance, flush matured singleton nodes to disk
		var (
			nodes, imgs = tradingTrieDB.Size()
			limit       = common.StorageSize(bc.cacheConfig.TrieDirtyLimit) * 1024 * 1024
		)
		if nodes > limit || imgs > 4*1024*1024 {
			tradingTrieDB.Cap(limit - ethdb.IdealBatchSize)
		}
		chosen := current - TriesInMemory
		for !bc.tradingTriegc.Empty() {
			root, number := bc.tradingTriegc.Pop()
			if uint64(-number) > chosen {
				bc.tradingTriegc.Push(root, number)
				break
			}
			tradingTrieDB.Dereference(root.(common.Hash))
		}
	}
	return nil
}

// Flush all Trading State Trie entries in GC cache to LevelDB.
func (bc *BlockChain) FlushTradingStateGCCache() {
	p := bc.VictionProcessor()
	if bc.cacheConfig.TrieDirtyDisabled || p == nil || p.TradingEngine() == nil {
		return
	}

	tradingTrieDB := p.TradingEngine().GetStateCache().TrieDB()
	for !bc.tradingTriegc.Empty() {
		root := bc.tradingTriegc.PopItem()
		if err := tradingTrieDB.Commit(root.(common.Hash), true, nil); err != nil {
			log.Error("[NativeTrading] Failed to commit Trie on shutdown", "root", root, "err", err)
		}
		tradingTrieDB.Dereference(root.(common.Hash))
	}
}

// Inject the Native Trading Engine into the Processor.
func (bc *BlockChain) SetTradingEngine(engine TradingEngine) {
	p, ok := bc.processor.(*StateProcessor)
	if !ok {
		log.Error("[NativeTrading] Engine not installed: Processor is not a *StateProcessor")
		return
	}
	p.viction.SetTradingEngine(engine)
	log.Info("[NativeTrading] Engine installed on state processor")
}

// Inject the Native Lending Engine into the Processor.
func (bc *BlockChain) SetLendingEngine(engine LendingEngine) {
	p, ok := bc.processor.(*StateProcessor)
	if !ok {
		log.Error("[NativeLending] Engine not installed: Processor is not a *StateProcessor")
		return
	}
	p.viction.SetLendingEngine(engine)
	log.Info("[NativeLending] Engine installed on state processor")
}

func (bc *BlockChain) UpdateValidators() error {
	engine, ok := bc.Engine().(*posv.Posv)
	if bc.Config().Posv == nil || !ok {
		return ErrPosvRequired
	}
	log.Info("[Blockchain] Preparing new validators list for next epoch.")

	contractAddress := bc.chainConfig.Viction.ValidatorContract
	if contractAddress == (common.Address{}) {
		return ErrNoValidatorContract
	}

	var candidates []common.Address
	stateDB, err := bc.State()
	if err != nil {
		return fmt.Errorf("failed to get state at block #%v: %v", bc.CurrentHeader().Number, err)
	}
	candidates = stateDB.VicGetCandidates(contractAddress)

	var validators []posv.ValidatorInfo
	for _, candidate := range candidates {
		if candidate.IsZero() {
			continue
		}
		_, cap := stateDB.VicGetValidatorInfo(contractAddress, candidate)
		validators = append(validators, posv.ValidatorInfo{Address: candidate, Capacity: cap})
	}
	if len(validators) == 0 {
		return ErrNoValidators
	}

	header := bc.CurrentHeader()
	if bc.Config().IsAtlas(header.Number) {
		sort.SliceStable(validators, func(i, j int) bool {
			return validators[i].Capacity.Cmp(validators[j].Capacity) >= 0
		})
	} else {
		sortlgc.Slice(validators, func(i, j int) bool {
			return validators[i].Capacity.Cmp(validators[j].Capacity) >= 0
		})
	}

	count := len(validators)
	if max := int(bc.chainConfig.Viction.ValidatorMaxCount); count > max {
		count = max
	}
	vs := make([]common.Address, 0, count)
	for _, v := range validators[:count] {
		vs = append(vs, v.Address)
	}
	err = engine.SetCheckpointSigners(bc, header, vs)
	if err != nil {
		return err
	}

	log.Info("[Blockchain] Updated validators list for next epoch", "signers", len(vs))
	return nil
}

// Check if two blocks are same path. Assume block 1 is ahead block 2.
func (bc *BlockChain) AreTwoBlockSamePath(bh1 common.Hash, bh2 common.Hash) bool {
	h1 := bc.GetHeaderByHash(bh1)
	h2 := bc.GetHeaderByHash(bh2)
	if h1 == nil || h2 == nil {
		return false
	}
	toLevel := h2.Number.Uint64()
	hash1 := bh1

	for h1.Number.Uint64() > toLevel {
		hash1 = h1.ParentHash
		h1 = bc.GetHeaderByHash(hash1)
		if h1 == nil {
			return false
		}
	}

	return hash1 == bh2
}

// Commit native trading/lending trie nodes for the given block to their LevelDB backing stores.
func (bc *BlockChain) commitNativeExchangeState(block *types.Block) error {
	if bc.cacheConfig.TrieDirtyDisabled {
		if err := bc.CommitTradingState(block); err != nil {
			return err
		}
		return bc.CommitLendingState(block)
	}
	if err := bc.CommitTradingStateDeferred(block); err != nil {
		return err
	}
	return bc.CommitLendingStateDeferred(block)
}

// Flush any in-memory trading/lending trie roots not yet committed to LevelDB.
func (bc *BlockChain) stopViction() {
	bc.FlushTradingStateGCCache()
	bc.FlushLendingStateGCCache()
}
