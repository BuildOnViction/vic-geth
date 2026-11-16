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
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

const (
	finalityTestMainLength = 16 // Blocks of the main chain in the tests that need no pruned state
	finalityTestForkParent = 5  // Main chain block that the forks of those tests build on
)

func TestFinalityForkChoiceKeepsFinalizedBlock(t *testing.T) {
	tests := []struct {
		name        string
		finalized   uint64 // 0 for none
		forkParent  uint64
		wantAdopted bool
	}{
		{name: "fork below the finalized block is refused", finalized: 10, forkParent: 5},
		{name: "fork whose first block replaces the finalized block is refused", finalized: 10, forkParent: 9},
		{name: "fork on top of the finalized block is adopted", finalized: 10, forkParent: 10, wantAdopted: true},
		{name: "fork above the finalized block is adopted", finalized: 10, forkParent: 13, wantAdopted: true},
		{name: "fork is adopted while no block is finalized", forkParent: 5, wantAdopted: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newFinalityForkChoiceTest(t, big.NewInt(0), finalityTestMainLength)
			if tt.finalized != 0 {
				test.finalize(tt.finalized)
			}
			fork := test.heavierFork(tt.forkParent)

			test.insert(t, fork)

			test.requireHead(t, fork, tt.wantAdopted)
		})
	}
}

func TestFinalityForkChoicePrefersHigherJustifiedBlock(t *testing.T) {
	tests := []struct {
		name          string
		mainJustified *uint64 // nil when the heads of the chain know none
		forkJustified *uint64
		isForkHeavier bool
		wantAdopted   bool
	}{
		{name: "lighter fork with a higher justified block is adopted", mainJustified: justifiedAt(15), forkJustified: justifiedAt(30), wantAdopted: true},
		{name: "heavier fork with a lower justified block is refused", mainJustified: justifiedAt(30), forkJustified: justifiedAt(15), isForkHeavier: true},
		{name: "heavier fork with an equal justified block is adopted", mainJustified: justifiedAt(15), forkJustified: justifiedAt(15), isForkHeavier: true, wantAdopted: true},
		{name: "lighter fork with an equal justified block is refused", mainJustified: justifiedAt(15), forkJustified: justifiedAt(15)},
		{name: "heavier fork with an unknown justified block is adopted", mainJustified: justifiedAt(30), isForkHeavier: true, wantAdopted: true},
		{name: "lighter fork with an unknown justified block is refused", mainJustified: justifiedAt(30)},
		{name: "lighter fork is refused while the head's justified block is unknown", forkJustified: justifiedAt(30)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newFinalityForkChoiceTest(t, big.NewInt(0), finalityTestMainLength)
			fork := test.lighterFork(finalityTestForkParent)
			if tt.isForkHeavier {
				fork = test.heavierFork(finalityTestForkParent)
			}
			test.justify(test.main, tt.mainJustified)
			test.justify(fork, tt.forkJustified)

			test.insert(t, fork)

			test.requireHead(t, fork, tt.wantAdopted)
		})
	}
}

func TestFinalityForkChoiceFollowsDifficultyWhenJustifiedBlockIsUnreadable(t *testing.T) {
	// Readable justified block is above none at all, so reading a failed lookup as "none" would decide against total difficulty in both cases.
	tests := []struct {
		name             string
		isForkUnreadable bool // Otherwise the main chain is
		isForkHeavier    bool
	}{
		{name: "heavier fork with an unreadable justified block is adopted", isForkUnreadable: true, isForkHeavier: true},
		{name: "lighter fork is refused while the head's justified block is unreadable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newFinalityForkChoiceTest(t, big.NewInt(0), finalityTestMainLength)
			fork := test.lighterFork(finalityTestForkParent)
			if tt.isForkHeavier {
				fork = test.heavierFork(finalityTestForkParent)
			}
			readable, unreadable := fork, test.main
			if tt.isForkUnreadable {
				readable, unreadable = test.main, fork
			}
			test.justify(readable, justifiedAt(30))
			test.failJustified(unreadable)

			test.insert(t, fork)

			test.requireHead(t, fork, tt.isForkHeavier)
		})
	}
}

func TestFinalityForkChoiceStartsAtPrometheus(t *testing.T) {
	tests := []struct {
		name         string
		prometheus   *big.Int
		wantEnforced bool
	}{
		{name: "fork not scheduled"},
		{name: "head one block below the fork", prometheus: big.NewInt(finalityTestMainLength + 1)},
		{name: "head at the fork block", prometheus: big.NewInt(finalityTestMainLength), wantEnforced: true},
		{name: "head above the fork", prometheus: big.NewInt(1), wantEnforced: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newFinalityForkChoiceTest(t, tt.prometheus, finalityTestMainLength)
			test.finalize(10)
			fork := test.heavierFork(finalityTestForkParent)

			test.insert(t, fork)

			test.requireHead(t, fork, !tt.wantEnforced)
		})
	}
}

func TestFinalityForkChoiceLeavesChildOfHeadToDifficulty(t *testing.T) {
	test := newFinalityForkChoiceTest(t, big.NewInt(0), finalityTestMainLength)
	child := test.fork(finalityTestMainLength, 1, 0)
	// No real view of a child is below its parent's; the script shows that the child's view is not asked.
	test.justify(test.main, justifiedAt(30))
	test.justify(child, justifiedAt(15))

	test.insert(t, child)

	test.requireHead(t, child, true)
}

func TestFinalityForkChoiceLeavesBlocksOfHeadChainToDifficulty(t *testing.T) {
	test := newFinalityForkChoiceTest(t, big.NewInt(0), finalityTestMainLength)
	ancestors := test.main[:finalityTestMainLength-1]
	// No real view of an ancestor is above the head's; the script shows that the ancestors' views are not asked.
	test.justify(test.main, justifiedAt(15))
	test.justify(ancestors, justifiedAt(30))

	test.insert(t, ancestors)

	test.requireHead(t, test.main, true)
}

func TestFinalityForkChoiceIgnoresJustifiedBlockBeforePrometheus(t *testing.T) {
	test := newFinalityForkChoiceTest(t, nil, finalityTestMainLength)
	fork := test.lighterFork(finalityTestForkParent)
	test.justify(test.main, justifiedAt(15))
	test.justify(fork, justifiedAt(30))

	test.insert(t, fork)

	test.requireHead(t, fork, false)
}

func TestFinalityForkChoiceRefusesKnownChainWithoutFinalizedBlock(t *testing.T) {
	test := newFinalityForkChoiceTest(t, big.NewInt(0), finalityTestMainLength)
	test.finalize(10)
	fork := test.heavierFork(finalityTestForkParent)
	test.insert(t, fork)
	tip := fork[len(fork)-1]
	if !test.chain.HasBlockAndState(tip.Hash(), tip.NumberU64()) {
		t.Fatalf("fork block %d has no state, so a second import does not take the known-block path", tip.NumberU64())
	}

	test.insert(t, fork)

	test.requireHead(t, fork, false)
}

func TestFinalityForkChoiceAdoptsKnownChainOnceItsJustifiedBlockIsHigher(t *testing.T) {
	test := newFinalityForkChoiceTest(t, big.NewInt(0), finalityTestMainLength)
	fork := test.lighterFork(finalityTestForkParent)
	test.justify(test.main, justifiedAt(15))
	test.insert(t, fork)
	test.requireHead(t, fork, false)
	test.justify(fork, justifiedAt(30))

	test.insert(t, fork)

	test.requireHead(t, fork, true)
}

func TestFinalityForkChoiceOnSideChainWithPrunedAncestor(t *testing.T) {
	// Fork parent lies below the blocks whose state is kept, so the fork is first written without state.
	const forkParent = TriesInMemory - 3
	tests := []struct {
		name        string
		finalized   uint64
		wantAdopted bool
	}{
		{name: "heavier side chain on top of the finalized block is imported", finalized: forkParent, wantAdopted: true},
		{name: "heavier side chain without the finalized block is not imported", finalized: forkParent + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newFinalityForkChoiceTest(t, big.NewInt(0), 2*TriesInMemory)
			test.finalize(tt.finalized)
			fork := test.fork(forkParent, 2*TriesInMemory, 0)
			tip := fork[len(fork)-1]

			test.insert(t, fork)

			test.requireHead(t, fork, tt.wantAdopted)
			if hasState := test.chain.HasState(tip.Root()); hasState != tt.wantAdopted {
				t.Fatalf("state of side chain block %d: have %t, want %t", tip.NumberU64(), hasState, tt.wantAdopted)
			}
		})
	}
}

func TestFinalityForkChoiceImportsLighterSideChainWithHigherJustifiedBlock(t *testing.T) {
	test := newFinalityForkChoiceTest(t, big.NewInt(0), 2*TriesInMemory)
	fork := test.fork(TriesInMemory-3, 10, 0)
	test.justify(test.main, justifiedAt(15))
	test.justify(fork, justifiedAt(30))

	test.insert(t, fork)

	test.requireHead(t, fork, true)
}

func TestFinalityForkChoiceOnKnownBlockInTheMiddleOfBatch(t *testing.T) {
	tests := []struct {
		name          string
		prometheus    *big.Int // nil when the fork is not scheduled
		finalized     uint64   // 0 for none
		mainJustified *uint64
		forkJustified *uint64
		wantAdopted   bool
	}{
		{name: "known block without the finalized block is refused", prometheus: big.NewInt(0), finalized: 10},
		{name: "known block with a lower justified block is refused", prometheus: big.NewInt(0), mainJustified: justifiedAt(30), forkJustified: justifiedAt(15)},
		{name: "known block with a higher justified block is adopted", prometheus: big.NewInt(0), mainJustified: justifiedAt(15), forkJustified: justifiedAt(30), wantAdopted: true},
		// Fork is lighter: upstream adopts a known block in the middle of a batch whatever its total difficulty.
		{name: "known block is adopted where finality has no say", prometheus: big.NewInt(0), wantAdopted: true},
		{name: "known block is adopted before Prometheus", finalized: 10, mainJustified: justifiedAt(30), forkJustified: justifiedAt(15), wantAdopted: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newFinalityForkChoiceTest(t, tt.prometheus, TriesInMemory)
			if tt.finalized != 0 {
				test.finalize(tt.finalized)
			}
			fork := test.fork(0, 2, 0)
			test.importWithPrunedFirstState(t, fork)
			test.justify(test.main, tt.mainJustified)
			test.justify(fork, tt.forkJustified)

			test.insert(t, fork)

			test.requireHead(t, fork, tt.wantAdopted)
		})
	}
}

func TestFinalityForkChoiceAdoptsBatchPastRefusedKnownBlock(t *testing.T) {
	test := newFinalityForkChoiceTest(t, big.NewInt(0), TriesInMemory)
	fork := test.fork(0, 3, 0)
	stored, tip := fork[:2], fork[2:]
	test.importWithPrunedFirstState(t, stored)
	test.justify(test.main, justifiedAt(30))
	test.justify(stored, justifiedAt(15))
	test.justify(tip, justifiedAt(45))

	test.insert(t, fork)

	test.requireHead(t, fork, true)
	known := stored[1]
	if canonical := test.chain.GetCanonicalHash(known.NumberU64()); canonical != known.Hash() {
		t.Fatalf("canonical block %d: have %s, want refused known block %s", known.NumberU64(), canonical, known.Hash())
	}
}

func TestFinalityTrackingFollowsNewHead(t *testing.T) {
	tests := []struct {
		name  string
		adopt func(t *testing.T, test *finalityForkChoiceTest) []*types.Block // Make the returned blocks the head chain through the path under test
	}{
		{name: "imported block", adopt: func(t *testing.T, test *finalityForkChoiceTest) []*types.Block {
			blocks := test.fork(uint64(len(test.main)), 1, 0)
			test.insert(t, blocks)
			return blocks
		}},
		{name: "known chain at the start of a batch", adopt: func(t *testing.T, test *finalityForkChoiceTest) []*types.Block {
			fork := test.lighterFork(finalityTestForkParent)
			test.justify(test.main, justifiedAt(15))
			test.insert(t, fork)
			test.requireHead(t, fork, false)
			test.justify(fork, justifiedAt(30))
			test.insert(t, fork)
			return fork
		}},
		{name: "known block in the middle of a batch", adopt: func(t *testing.T, test *finalityForkChoiceTest) []*types.Block {
			fork := test.fork(0, 2, 0)
			test.importWithPrunedFirstState(t, fork)
			test.justify(test.main, justifiedAt(15))
			test.justify(fork, justifiedAt(30))
			test.insert(t, fork)
			return fork
		}},
		{name: "block sealed by the node", adopt: func(t *testing.T, test *finalityForkChoiceTest) []*types.Block {
			blocks := test.fork(uint64(len(test.main)), 1, 0)
			test.writeSealed(t, blocks[0])
			return blocks
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newFinalityForkChoiceTest(t, big.NewInt(0), TriesInMemory)

			blocks := tt.adopt(t, test)

			test.requireHead(t, blocks, true)
			head := blocks[len(blocks)-1]
			tracked := test.engine.tracked
			if tracked == nil {
				t.Fatalf("finality never tracked, want head block %d (%s)", head.NumberU64(), head.Hash())
			}
			if tracked.Hash() != head.Hash() {
				t.Fatalf("finality tracked up to block %d (%s), want head block %d (%s)", tracked.Number, tracked.Hash(), head.NumberU64(), head.Hash())
			}
		})
	}
}

func TestHasAncestor(t *testing.T) {
	test := newFinalityForkChoiceTest(t, nil, finalityTestMainLength)
	fork := test.lighterFork(finalityTestForkParent)
	test.insert(t, fork)
	test.requireHead(t, fork, false)
	main, sideTip := test.main, fork[len(fork)-1]
	tests := []struct {
		name     string
		block    *types.Block
		ancestor *types.Block
		want     bool
	}{
		{name: "block itself", block: main[9], ancestor: main[9], want: true},
		{name: "canonical ancestor of a canonical block", block: main[15], ancestor: main[2], want: true},
		{name: "side ancestor of a side block", block: sideTip, ancestor: fork[0], want: true},
		{name: "fork point of a side block", block: sideTip, ancestor: main[finalityTestForkParent-1], want: true},
		{name: "canonical block below the fork point of a side block", block: sideTip, ancestor: main[2], want: true},
		{name: "canonical block above the fork point of a side block", block: sideTip, ancestor: main[finalityTestForkParent], want: false},
		{name: "canonical block at the height of a side block", block: sideTip, ancestor: main[sideTip.NumberU64()-1], want: false},
		{name: "side block below a canonical block", block: main[15], ancestor: fork[0], want: false},
		{name: "descendant of the block", block: main[2], ancestor: main[9], want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ancestor := posv.BlockRef{Number: tt.ancestor.NumberU64(), Hash: tt.ancestor.Hash()}

			has := test.chain.hasAncestor(tt.block.Header(), ancestor)

			if has != tt.want {
				t.Fatalf("block %d has ancestor %d (%s): have %t, want %t", tt.block.NumberU64(), ancestor.Number, ancestor.Hash, has, tt.want)
			}
		})
	}
}

// finalityStubEngine is a consensus engine whose finality knowledge is scripted by the test.
type finalityStubEngine struct {
	consensus.Engine

	finalized  *posv.BlockRef
	justified  map[common.Hash]uint64 // Justified block number by head; a head without entry knows none
	unreadable map[common.Hash]bool   // Heads whose justified block cannot be read
	tracked    *types.Header          // Head of the last update of finality tracking, nil before the first
}

func (e *finalityStubEngine) UpdateFinality(chain consensus.ChainReader, head *types.Header) error {
	e.tracked = head
	return nil
}

func (e *finalityStubEngine) FinalizedBlock() (posv.BlockRef, bool) {
	if e.finalized == nil {
		return posv.BlockRef{}, false
	}
	return *e.finalized, true
}

func (e *finalityStubEngine) JustifiedBlock(chain consensus.ChainReader, header *types.Header) (posv.BlockRef, error) {
	if e.unreadable[header.Hash()] {
		return posv.BlockRef{}, errors.New("view cannot be derived")
	}
	number, ok := e.justified[header.Hash()]
	if !ok {
		return posv.BlockRef{}, posv.ErrNoJustifiedBlock
	}
	return posv.BlockRef{Number: number}, nil
}

// finalityForkChoiceTest is a chain that imported a main chain, and the means to build forks of it.
type finalityForkChoiceTest struct {
	config  *params.ChainConfig
	engine  *finalityStubEngine
	makerDb ethdb.Database // Holds the state that forks are generated on
	genesis *types.Block
	chain   *BlockChain
	main    []*types.Block // Block at number n is main[n-1]
}

// newFinalityForkChoiceTest returns a chain with the Prometheus fork at the given block, nil for never, that imported a main chain of length blocks.
func newFinalityForkChoiceTest(t *testing.T, prometheus *big.Int, length int) *finalityForkChoiceTest {
	t.Helper()
	config := *params.TestChainConfig
	config.PrometheusBlock = prometheus
	spec := &Genesis{Config: &config, BaseFee: big.NewInt(params.InitialBaseFee)}
	test := &finalityForkChoiceTest{
		config: &config,
		engine: &finalityStubEngine{
			Engine:     ethash.NewFaker(),
			justified:  make(map[common.Hash]uint64),
			unreadable: make(map[common.Hash]bool),
		},
		makerDb: rawdb.NewMemoryDatabase(),
	}
	test.genesis = spec.MustCommit(test.makerDb)

	chainDb := rawdb.NewMemoryDatabase()
	spec.MustCommit(chainDb)
	chain, err := NewBlockChain(chainDb, nil, test.config, test.engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("create chain: %v", err)
	}
	t.Cleanup(chain.Stop)
	test.chain = chain

	test.main, _ = GenerateChain(test.config, test.genesis, test.engine, test.makerDb, length, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{1})
	})
	test.insert(t, test.main)
	test.requireHead(t, test.main, true)
	return test
}

// Return length blocks on top of the main chain block at parent, genesis for 0. Each block's time is moved by timeOffset seconds.
func (f *finalityForkChoiceTest) fork(parent uint64, length int, timeOffset int64) []*types.Block {
	parentBlock := f.genesis
	if parent > 0 {
		parentBlock = f.main[parent-1]
	}
	blocks, _ := GenerateChain(f.config, parentBlock, f.engine, f.makerDb, length, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{2})
		b.OffsetTime(timeOffset)
	})
	return blocks
}

// heavierFork returns a fork from the main chain block at parent that ends at the height of the main chain with a higher total difficulty.
func (f *finalityForkChoiceTest) heavierFork(parent uint64) []*types.Block {
	// Blocks that follow their parent sooner are more difficult.
	return f.fork(parent, len(f.main)-int(parent), -9)
}

// lighterFork returns a fork from the main chain block at parent that ends below the main chain, so with a lower total difficulty.
func (f *finalityForkChoiceTest) lighterFork(parent uint64) []*types.Block {
	return f.fork(parent, len(f.main)-int(parent)-3, 0)
}

// finalize scripts the main chain block at number as the finalized block.
func (f *finalityForkChoiceTest) finalize(number uint64) {
	block := f.main[number-1]
	f.engine.finalized = &posv.BlockRef{Number: block.NumberU64(), Hash: block.Hash()}
}

// justify scripts the justified block that every block of blocks knows as head; nil makes them know none.
func (f *finalityForkChoiceTest) justify(blocks []*types.Block, justified *uint64) {
	for _, block := range blocks {
		if justified == nil {
			delete(f.engine.justified, block.Hash())
			continue
		}
		f.engine.justified[block.Hash()] = *justified
	}
}

// failJustified scripts a failure for the justified block that every block of blocks knows as head.
func (f *finalityForkChoiceTest) failJustified(blocks []*types.Block) {
	for _, block := range blocks {
		f.engine.unreadable[block.Hash()] = true
	}
}

// Import blocks, a fork from genesis that fork choice must refuse, then extend the main chain by the one block
// whose write garbage collects the state of the first of blocks and keeps the state of the next one.
// Importing blocks again then executes their first block, on genesis state that is never pruned, and finds the next one known in the middle of the batch.
// The main chain must hold TriesInMemory blocks.
func (f *finalityForkChoiceTest) importWithPrunedFirstState(t *testing.T, blocks []*types.Block) {
	t.Helper()
	f.insert(t, blocks)
	f.requireHead(t, blocks, false)
	f.main = append(f.main, f.fork(uint64(len(f.main)), 1, 0)...)
	f.insert(t, f.main[len(f.main)-1:])
	f.requireHead(t, f.main, true)

	first, next := blocks[0], blocks[1]
	if hasFirst, hasNext := f.chain.HasState(first.Root()), f.chain.HasBlockAndState(next.Hash(), next.NumberU64()); hasFirst || !hasNext {
		t.Fatalf("state of fork blocks %d and %d: have %t and %t, want false and true", first.NumberU64(), next.NumberU64(), hasFirst, hasNext)
	}
}

// Execute block on its parent's state and write it the way the miner writes a block it sealed.
func (f *finalityForkChoiceTest) writeSealed(t *testing.T, block *types.Block) {
	t.Helper()
	statedb, err := state.New(f.chain.GetBlockByHash(block.ParentHash()).Root(), f.chain.stateCache, nil)
	if err != nil {
		t.Fatalf("open state of parent of block %d: %v", block.NumberU64(), err)
	}
	receipts, logs, _, err := f.chain.processor.Process(block, statedb, nil, vm.Config{})
	if err != nil {
		t.Fatalf("execute block %d: %v", block.NumberU64(), err)
	}
	if _, err := f.chain.WriteBlockWithState(block, receipts, logs, statedb, false); err != nil {
		t.Fatalf("write block %d: %v", block.NumberU64(), err)
	}
}

// insert imports blocks, which must be valid: fork choice refuses a chain without failing its import.
func (f *finalityForkChoiceTest) insert(t *testing.T, blocks []*types.Block) {
	t.Helper()
	if n, err := f.chain.InsertChain(blocks); err != nil {
		t.Fatalf("insert block %d: %v", blocks[n].NumberU64(), err)
	}
}

// requireHead fails unless the chain adopted blocks up to their last one, or, when wantAdopted is false, kept the main chain as it was and holds blocks beside it.
func (f *finalityForkChoiceTest) requireHead(t *testing.T, blocks []*types.Block, wantAdopted bool) {
	t.Helper()
	tip, want := blocks[len(blocks)-1], f.main[len(f.main)-1]
	if wantAdopted {
		want = tip
	}
	if head := f.chain.CurrentBlock(); head.Hash() != want.Hash() {
		t.Fatalf("head: have block %d (%s), want block %d (%s)", head.NumberU64(), head.Hash(), want.NumberU64(), want.Hash())
	}
	if canonical := f.chain.GetCanonicalHash(want.NumberU64()); canonical != want.Hash() {
		t.Fatalf("canonical block %d: have %s, want %s", want.NumberU64(), canonical, want.Hash())
	}
	if !f.chain.HasBlock(tip.Hash(), tip.NumberU64()) {
		t.Fatalf("block %d (%s) is not stored", tip.NumberU64(), tip.Hash())
	}
}

// justifiedAt returns the number of a justified block for a test table, where nil stands for unknown.
func justifiedAt(number uint64) *uint64 {
	return &number
}
