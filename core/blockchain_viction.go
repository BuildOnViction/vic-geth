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
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/common/sortlgc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

// Return underlying VictionProcessor instance in the Proccesor.
func (bc *BlockChain) VictionProcessor() *VictionProcessor {
	p, ok := bc.processor.(*StateProcessor)
	if !ok || p == nil {
		return nil
	}
	return p.viction
}

// nativeTrieAccess groups the per-platform accessors needed to flush a native state trie.
type nativeTrieAccess struct {
	logTag string                                 // log prefix, e.g. "[NativeLending]"
	errTag string                                 // error prefix, e.g. "native_lending"
	ready  func(*VictionProcessor) bool           // reports whether the platform is initialized
	root   func(*VictionProcessor) common.Hash    // returns the last committed state root
	trieDB func(*VictionProcessor) *trie.Database // returns the platform trie DB, or nil if the engine is unset
	triegc *prque.Prque                           // deferred GC queue for the platform trie roots
}

// Return the per-blockchain accessors for the native lending trie.
func (bc *BlockChain) lendingTrieAccess() nativeTrieAccess {
	return nativeTrieAccess{
		logTag: "[NativeLending]",
		errTag: "native_lending",
		ready:  (*VictionProcessor).IsLendingInitialized,
		root:   (*VictionProcessor).CommittedLendingRoot,
		trieDB: func(p *VictionProcessor) *trie.Database {
			engine := p.LendingEngine()
			if engine == nil {
				return nil
			}
			return engine.GetStateCache().TrieDB()
		},
		triegc: bc.lendingTriegc,
	}
}

// Return the per-blockchain accessors for the native trading trie.
func (bc *BlockChain) tradingTrieAccess() nativeTrieAccess {
	return nativeTrieAccess{
		logTag: "[NativeTrading]",
		errTag: "native_trading",
		ready:  (*VictionProcessor).IsTradingInitialized,
		root:   (*VictionProcessor).CommittedTradingRoot,
		trieDB: func(p *VictionProcessor) *trie.Database {
			engine := p.TradingEngine()
			if engine == nil {
				return nil
			}
			return engine.GetStateCache().TrieDB()
		},
		triegc: bc.tradingTriegc,
	}
}

// Flush current block native state trie to LevelDB.
func (bc *BlockChain) commitNativeTrie(a nativeTrieAccess, block *types.Block) error {
	p := bc.VictionProcessor()
	if p == nil || !a.ready(p) {
		return nil
	}
	root := a.root(p)
	if root == (common.Hash{}) {
		return nil
	}
	if err := a.trieDB(p).Commit(root, false, nil); err != nil {
		return fmt.Errorf("%s: failed to commit Trie at block %d: %w", a.errTag, block.NumberU64(), err)
	}
	log.Debug(a.logTag+" Flushed Trie to disk", "block", block.NumberU64(), "root", root.Hex())
	return nil
}

// Flush current block native state trie in GC cache to LevelDB.
func (bc *BlockChain) commitNativeTrieDeferred(a nativeTrieAccess, block *types.Block) error {
	p := bc.VictionProcessor()
	if p == nil || !a.ready(p) {
		return nil
	}
	current := block.NumberU64()
	root := a.root(p)
	if root == (common.Hash{}) {
		return nil
	}
	trieDB := a.trieDB(p)
	trieDB.Reference(root, common.Hash{})
	a.triegc.Push(root, -int64(current))

	if err := trieDB.Commit(root, true, nil); err != nil {
		return fmt.Errorf("%s: failed to commit Trie at block %d: %w", a.errTag, current, err)
	}
	log.Debug(a.logTag+" Flushed Trie to disk", "block", current, "root", root.Hex())

	if current > TriesInMemory {
		// If we exceeded our memory allowance, flush matured singleton nodes to disk
		var (
			nodes, imgs = trieDB.Size()
			limit       = common.StorageSize(bc.cacheConfig.TrieDirtyLimit) * 1024 * 1024
		)
		if nodes > limit || imgs > 4*1024*1024 {
			trieDB.Cap(limit - ethdb.IdealBatchSize)
		}
		chosen := current - TriesInMemory
		for !a.triegc.Empty() {
			root, number := a.triegc.Pop()
			if uint64(-number) > chosen {
				a.triegc.Push(root, number)
				break
			}
			trieDB.Dereference(root.(common.Hash))
		}
	}
	return nil
}

// Flush all native state trie entries in GC cache to LevelDB.
func (bc *BlockChain) flushNativeTrieGCCache(a nativeTrieAccess) {
	p := bc.VictionProcessor()
	if bc.cacheConfig.TrieDirtyDisabled || p == nil {
		return
	}
	trieDB := a.trieDB(p)
	if trieDB == nil {
		return
	}
	for !a.triegc.Empty() {
		root := a.triegc.PopItem()
		if err := trieDB.Commit(root.(common.Hash), true, nil); err != nil {
			log.Error(a.logTag+" Failed to commit Trie on shutdown", "root", root, "err", err)
		}
		trieDB.Dereference(root.(common.Hash))
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
		if err := bc.commitNativeTrie(bc.tradingTrieAccess(), block); err != nil {
			return err
		}
		return bc.commitNativeTrie(bc.lendingTrieAccess(), block)
	}
	if err := bc.commitNativeTrieDeferred(bc.tradingTrieAccess(), block); err != nil {
		return err
	}
	return bc.commitNativeTrieDeferred(bc.lendingTrieAccess(), block)
}

// Flush any in-memory trading/lending trie roots not yet committed to LevelDB.
func (bc *BlockChain) stopViction() {
	bc.flushNativeTrieGCCache(bc.tradingTrieAccess())
	bc.flushNativeTrieGCCache(bc.lendingTrieAccess())
}
