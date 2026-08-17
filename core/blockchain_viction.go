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
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sortlgc"
)

// commitVictionState manages persistence of native trading and native lending trie nodes using the
// same deferred GC strategy as the main EVM state trie (Reference + priority
// queue; Commit to LevelDB every TriesInMemory blocks; Dereference old roots).
//
// viction.Processor.AfterProcess calls TradingStateDB.Commit() / LendingStateDB.Commit() which
// stage dirty trie nodes into the respective trie.Database dirty sets.
//
// Here we:
//  1. Reference the new root to keep it in memory.
//  2. Push it onto the deferred GC queue (tradingTriegc / lendingTriegc).
//  3. Every TriesInMemory blocks: commit the root that is TriesInMemory behind
//     HEAD to LevelDB, then Dereference older roots that are no longer needed.
//
// This mirrors the deferred-commit strategy used for the main EVM state trie
// and avoids a LevelDB write on every single block (which would cause
// excessive write amplification).
func (bc *BlockChain) commitVictionState(block *types.Block) error {
	if !bc.chainConfig.IsTomoXEnabled(block.Number()) {
		return nil
	}
	if bc.cacheConfig.TrieDirtyDisabled {
		return bc.commitVictionStateDirect(block)
	}
	return bc.commitVictionStateDeferred(block)
}

// commitVictionStateDirect is used in archive mode (TrieDirtyDisabled): flush
// every block immediately to LevelDB, no deferred GC needed.
func (bc *BlockChain) commitVictionStateDirect(block *types.Block) error {
	p, ok := bc.processor.(*StateProcessor)
	if !ok || p.viction == nil {
		return nil
	}
	if p.viction.IsTradingInitialized() {
		tradingRoot := p.viction.CommittedTradingRoot()
		if tradingRoot != (common.Hash{}) {
			if err := p.viction.TradingEngine().GetStateCache().TrieDB().Commit(tradingRoot, false, nil); err != nil {
				return fmt.Errorf("native_trading: failed to commit Trie at block %d: %w", block.NumberU64(), err)
			}
			log.Info("[Processor][Native Trading] Flushed Trie to disk", "block", block.NumberU64(), "root", tradingRoot.Hex())
		}
	}
	if p.viction.IsLendingInitialized() {
		lendingRoot := p.viction.CommittedLendingRoot()
		if lendingRoot != (common.Hash{}) {
			if err := p.viction.LendingEngine().GetStateCache().TrieDB().Commit(lendingRoot, false, nil); err != nil {
				return fmt.Errorf("native_lending: failed to commit Trie at block %d: %w", block.NumberU64(), err)
			}
			log.Info("[Processor][Native Lending] Flushed Trie to disk", "block", block.NumberU64(), "root", lendingRoot.Hex())
		}
	}
	return nil
}

// commitVictionStateDeferred is the full-node path for trading/lending trie persistence.
//
// Unlike the EVM trie, we commit the current block's root to LevelDB on every
// block rather than deferring the commit TriesInMemory blocks. This is
// necessary because GetTradingStateRoot returns EmptyRoot for any block that
// has no 0x92 system tx, causing dirty nodes to accumulate in the
// trie.Database dirty cache until Dereference removes them without ever being
// written — the "nodes=0" bug.
//
// We still push every root onto the GC queue and defer the Dereference by
// TriesInMemory blocks, which is sufficient for reorg safety (same as EVM).
// The extra write overhead per block is small compared to order matching.
func (bc *BlockChain) commitVictionStateDeferred(block *types.Block) error {
	p, ok := bc.processor.(*StateProcessor)
	if !ok || p.viction == nil {
		return nil
	}
	current := block.NumberU64()

	// Trading trie: commit the current block's dirty root immediately.
	if p.viction.IsTradingInitialized() {
		tradingRoot := p.viction.CommittedTradingRoot()
		if tradingRoot != (common.Hash{}) {
			tradingTrieDB := p.viction.TradingEngine().GetStateCache().TrieDB()
			tradingTrieDB.Reference(tradingRoot, common.Hash{})
			p.tradingTriegc.Push(tradingRoot, -int64(current))

			if err := tradingTrieDB.Commit(tradingRoot, true, nil); err != nil {
				return fmt.Errorf("native_trading: failed to commit Trie at block %d: %w", current, err)
			}
			log.Info("[Processor][Native Trading] Flushed Trie to disk", "block", current, "root", tradingRoot.Hex())

			// Dereference roots old enough to no longer need keeping in memory.
			if current > TriesInMemory {
				chosen := current - TriesInMemory
				for !p.tradingTriegc.Empty() {
					root, number := p.tradingTriegc.Pop()
					if uint64(-number) > chosen {
						p.tradingTriegc.Push(root, number)
						break
					}
					tradingTrieDB.Dereference(root.(common.Hash))
				}
			}
		}
	}

	// Lending trie: same strategy as trading trie.
	if p.viction.IsLendingInitialized() {
		lendingRoot := p.viction.CommittedLendingRoot()
		if lendingRoot != (common.Hash{}) {
			lendingTrieDB := p.viction.LendingEngine().GetStateCache().TrieDB()
			lendingTrieDB.Reference(lendingRoot, common.Hash{})
			p.lendingTriegc.Push(lendingRoot, -int64(current))

			if err := lendingTrieDB.Commit(lendingRoot, true, nil); err != nil {
				return fmt.Errorf("native_lending: failed to commit Trie at block %d: %w", current, err)
			}
			log.Info("[Processor][Native Lending] Flushed Trie to disk", "block", current, "root", lendingRoot.Hex())

			if current > TriesInMemory {
				chosen := current - TriesInMemory
				for !p.lendingTriegc.Empty() {
					root, number := p.lendingTriegc.Pop()
					if uint64(-number) > chosen {
						p.lendingTriegc.Push(root, number)
						break
					}
					lendingTrieDB.Dereference(root.(common.Hash))
				}
			}
		}
	}

	return nil
}

// stopViction flushes any in-memory trading/lending trie roots that were not yet
// committed to LevelDB (the tail of the deferred GC queues).  Called from
// BlockChain.Stop() before the node exits.
func (bc *BlockChain) stopViction() {
	if bc.cacheConfig.TrieDirtyDisabled {
		return // archive mode commits every block; nothing to flush here
	}
	p, ok := bc.processor.(*StateProcessor)
	if !ok || p == nil {
		return
	}

	// Flush all remaining trading trie roots to LevelDB.
	if p.viction.TradingEngine() != nil {
		tradingTrieDB := p.viction.TradingEngine().GetStateCache().TrieDB()
		for !p.tradingTriegc.Empty() {
			root := p.tradingTriegc.PopItem().(common.Hash)
			if err := tradingTrieDB.Commit(root, true, nil); err != nil {
				log.Error("[Processor][Native Trading] Failed to commit Trie on shutdown", "root", root, "err", err)
			}
			tradingTrieDB.Dereference(root)
		}
	}

	// Flush all remaining lending trie roots to LevelDB.
	if p.viction.LendingEngine() != nil {
		lendingTrieDB := p.viction.LendingEngine().GetStateCache().TrieDB()
		for !p.lendingTriegc.Empty() {
			root := p.lendingTriegc.PopItem().(common.Hash)
			if err := lendingTrieDB.Commit(root, true, nil); err != nil {
				log.Error("[Processor][Native Lending] Failed to commit Trie on shutdown", "root", root, "err", err)
			}
			lendingTrieDB.Dereference(root)
		}
	}
}

// SetTradingEngine injects the native trading engine into the block processor.
func (bc *BlockChain) SetTradingEngine(engine TradingEngine) {
	p, ok := bc.processor.(*StateProcessor)
	if !ok {
		log.Error("[Processor][Native Trading] Engine not installed: Processor is not a *StateProcessor")
		return
	}
	p.viction.SetTradingEngine(engine)
	log.Info("[Processor][Native Trading] Engine installed on state processor")
}

// SetLendingEngine injects the native lending engine into the block processor.
func (bc *BlockChain) SetLendingEngine(engine LendingEngine) {
	p, ok := bc.processor.(*StateProcessor)
	if !ok {
		log.Error("[Processor][Native Lending] Engine not installed: Processor is not a *StateProcessor")
		return
	}
	p.viction.SetLendingEngine(engine)
	log.Info("[Processor][Native Lending] Engine installed on state processor")
}

func (bc *BlockChain) UpdateM1() error {
	engine, ok := bc.Engine().(*posv.Posv)
	if bc.Config().Posv == nil || !ok {
		return fmt.Errorf("PoSV engine is not enabled")
	}
	log.Info("It's time to update new set of masternodes for the next epoch...")

	contractAddress := bc.chainConfig.Viction.ValidatorContract
	if contractAddress == (common.Address{}) {
		return fmt.Errorf("validator contract address is not set in chain config")
	}

	var candidates []common.Address

	// get candidates from slot of stateDB
	// if can't get anything, request from contracts
	stateDB, err := bc.State()
	if err != nil {
		return fmt.Errorf("failed to get state at current root (block %v): %v", bc.CurrentHeader().Number, err)
	}
	candidates = stateDB.VicGetCandidates(contractAddress)

	var ms []posv.ValidatorInfo
	for _, candidate := range candidates {
		_, cap := stateDB.VicGetValidatorInfo(contractAddress, candidate)

		//TODO: smart contract shouldn't return "0x0000000000000000000000000000000000000000"
		if candidate.String() != "0x0000000000000000000000000000000000000000" {
			ms = append(ms, posv.ValidatorInfo{Address: candidate, Capacity: cap})
		}
	}
	if len(ms) == 0 {
		log.Error("No masternode found. Stopping node")
		return fmt.Errorf("no masternode found")
	} else {
		header := bc.CurrentHeader()
		if bc.Config().IsAtlas(header.Number) {
			sort.SliceStable(ms, func(i, j int) bool {
				return ms[i].Capacity.Cmp(ms[j].Capacity) >= 0
			})
		} else {
			// Must sort `ms`, not `candidates`: indices i,j are in [0, len(slice));
			// len(candidates) can exceed len(ms) when zero-address entries are skipped.
			sortlgc.Slice(ms, func(i, j int) bool {
				return ms[i].Capacity.Cmp(ms[j].Capacity) >= 0
			})
		}
		log.Info("Ordered list of masternode candidates")
		for _, m := range ms {
			log.Info("", "address", m.Address.String(), "stake", m.Capacity)
		}
		// update masternodes
		log.Info("Updating new set of masternodes")
		hv := make([]common.Address, 0)
		if len(ms) > int(bc.chainConfig.Viction.ValidatorMaxCount) {
			for _, v := range ms[:bc.chainConfig.Viction.ValidatorMaxCount] {
				hv = append(hv, v.Address)
			}
		} else {
			for _, v := range ms {
				hv = append(hv, v.Address)
			}
		}
		err = engine.SetCheckpointSigners(bc, header, hv)
		if err != nil {
			return err
		}
		log.Info("Masternodes are ready for the next epoch")
	}
	return nil
}

// AreTwoBlockSamePath check if two blocks are same path
// Assume block 1 is ahead block 2 so we need to check parentHash
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
