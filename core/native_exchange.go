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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/legacy/lending/lendingstate"
	"github.com/ethereum/go-ethereum/legacy/trading/tradingstate"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// LendingEngine is the interface of native decentralized lending platform.
type LendingEngine interface {
	// GetLendingStateRoot returns the lending state root embedded in the 0x92 system transaction of the given block.
	GetLendingStateRoot(block *types.Block, author common.Address) (common.Hash, error)

	// GetLendingState opens the LendingStateDB trie rooted at the given block's lending state root.
	GetLendingState(block *types.Block, author common.Address) (*lendingstate.LendingStateDB, error)

	// HasLendingState reports whether the given block has a lending state trie.
	HasLendingState(block *types.Block, author common.Address) bool

	// GetStateCache returns the trie-node cache backed by the LendingState LevelDB.
	GetStateCache() lendingstate.Database

	// CommitOrder applies a single lending order, reverting all state on error.
	CommitOrder(
		header *types.Header, coinbase common.Address,
		chain tradingstate.ChainContext, statedb *state.StateDB, lendingStateDB *lendingstate.LendingStateDB, tradingStateDB *tradingstate.TradingStateDB,
		lendingOrderBook common.Hash, order *lendingstate.LendingItem,
	) ([]*lendingstate.LendingTrade, []*lendingstate.LendingItem, error)

	// GetCollateralPrices returns the VIC-denominated prices for the lending token and collateral token.
	GetCollateralPrices(
		header *types.Header,
		chain tradingstate.ChainContext, statedb *state.StateDB, tradingStateDB *tradingstate.TradingStateDB,
		collateralToken common.Address, lendingToken common.Address,
	) (*big.Int, *big.Int, error)

	// GetMediumTradePriceBeforeEpoch returns the epoch-averaged trade price for the given token pair.
	GetMediumTradePriceBeforeEpoch(
		chain tradingstate.ChainContext, statedb *state.StateDB, tradingStateDB *tradingstate.TradingStateDB,
		baseToken common.Address, quoteToken common.Address,
	) (*big.Int, error)

	// ProcessLiquidationData computes which lending trades must be liquidated, auto-repaid, topped up, or recalled.
	ProcessLiquidationData(
		header *types.Header,
		chain tradingstate.ChainContext, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, lendingState *lendingstate.LendingStateDB,
	) (
		updatedTrades map[common.Hash]*lendingstate.LendingTrade,
		liquidatedTrades []*lendingstate.LendingTrade,
		autoRepayTrades []*lendingstate.LendingTrade,
		autoTopUpTrades []*lendingstate.LendingTrade,
		autoRecallTrades []*lendingstate.LendingTrade,
		err error,
	)
}

// TradingEngine is the interface of native decentralized exchange platform.
type TradingEngine interface {
	// CommitOrder replays a single order through the matching engine.
	CommitOrder(
		header *types.Header, coinbase common.Address,
		chain tradingstate.ChainContext, statedb *state.StateDB, tradingStateDB *tradingstate.TradingStateDB,
		orderBook common.Hash, order *tradingstate.OrderItem,
	) ([]map[string]string, []*tradingstate.OrderItem, error)

	// GetTradingState opens the TradingStateDB trie rooted at the given block.
	GetTradingState(block *types.Block, author common.Address) (*tradingstate.TradingStateDB, error)

	// GetTradingStateRoot returns the trading state root committed in the 0x92 tx of the given block.
	GetTradingStateRoot(block *types.Block, author common.Address) (common.Hash, error)

	// UpdateMediumPriceBeforeEpoch computes epoch-averaged prices.
	UpdateMediumPriceBeforeEpoch(epoch uint64, tradingStateDB *tradingstate.TradingStateDB, statedb *state.StateDB) error

	// GetStateCache returns the trie-node cache backed by the TradingState LevelDB.
	GetStateCache() tradingstate.Database
}

// Return LendingEngine instance.
func (p *VictionProcessor) LendingEngine() LendingEngine {
	return p.lendingEngine
}

// Set LendingEngine instance.
func (p *VictionProcessor) SetLendingEngine(engine LendingEngine) {
	p.lendingEngine = engine
}

// Return if Lending platform is ready to handle transactions.
func (p *VictionProcessor) IsLendingInitialized() bool {
	return p.lendingEngine != nil && p.lendingStateDB != nil && p.tradingStateDB != nil
}

func GetLendingStateRoot(block *types.Block, tradingStateAddr common.Address, author common.Address, config *params.ChainConfig) common.Hash {
	signer := types.MakeSigner(config, block.Number())
	for _, tx := range block.Transactions() {
		if tx.To() == nil || *tx.To() != tradingStateAddr {
			continue
		}
		from, err := types.Sender(signer, tx)
		if err != nil || from != author {
			continue
		}
		if len(tx.Data()) >= 64 {
			return common.BytesToHash(tx.Data()[32:])
		}
	}
	return lendingstate.EmptyRoot
}

func (p *VictionProcessor) CommittedLendingRoot() common.Hash {
	return p.lendingCommittedRoot
}

// Flush current block Lending State Trie to LevelDB.
func (p *StateProcessor) CommitLendingState(block *types.Block) error {
	if !p.viction.IsLendingInitialized() {
		return nil
	}
	lendingRoot := p.viction.CommittedLendingRoot()
	if lendingRoot == (common.Hash{}) {
		return nil
	}
	if err := p.viction.LendingEngine().GetStateCache().TrieDB().Commit(lendingRoot, false, nil); err != nil {
		return fmt.Errorf("native_lending: failed to commit Trie at block %d: %w", block.NumberU64(), err)
	}
	log.Debug("[NativeLending] Flushed Trie to disk", "block", block.NumberU64(), "root", lendingRoot.Hex())
	return nil
}

// Flush current block Lending State Trie in GC cache to LevelDB.
func (p *StateProcessor) CommitLendingStateDeferred(block *types.Block) error {
	if !p.viction.IsLendingInitialized() {
		return nil
	}
	current := block.NumberU64()
	lendingRoot := p.viction.CommittedLendingRoot()
	if lendingRoot == (common.Hash{}) {
		return nil
	}
	lendingTrieDB := p.viction.LendingEngine().GetStateCache().TrieDB()
	lendingTrieDB.Reference(lendingRoot, common.Hash{})
	p.lendingTriegc.Push(lendingRoot, -int64(current))

	if err := lendingTrieDB.Commit(lendingRoot, true, nil); err != nil {
		return fmt.Errorf("native_lending: failed to commit Trie at block %d: %w", current, err)
	}
	log.Debug("[NativeLending] Flushed Trie to disk", "block", current, "root", lendingRoot.Hex())

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
	return nil
}

// Flush all Lending State Trie entires in GC cache to LevelDB.
func (p *StateProcessor) FlushLendingStateGCCache() {
	if p.bc.cacheConfig.TrieDirtyDisabled || p.viction.LendingEngine() == nil {
		return
	}

	lendingTrieDB := p.viction.LendingEngine().GetStateCache().TrieDB()
	for !p.lendingTriegc.Empty() {
		root := p.lendingTriegc.PopItem()
		if err := lendingTrieDB.Commit(root.(common.Hash), true, nil); err != nil {
			log.Error("[NativeLending] Failed to commit Trie on shutdown", "root", root, "err", err)
		}
		lendingTrieDB.Dereference(root.(common.Hash))
	}
}

// Return TradingEngine instance.
func (p *VictionProcessor) TradingEngine() TradingEngine {
	return p.tradingEngine
}

// Set TradingEngine instance.
func (p *VictionProcessor) SetTradingEngine(engine TradingEngine) {
	p.tradingEngine = engine
}

// Return if Trading platform is ready to handle transactions.
func (p *VictionProcessor) IsTradingInitialized() bool {
	return p.tradingEngine != nil && p.tradingStateDB != nil
}

func GetTradingStateRoot(block *types.Block, tradingStateAddr common.Address, author common.Address, config *params.ChainConfig) common.Hash {
	signer := types.MakeSigner(config, block.Number())
	for _, tx := range block.Transactions() {
		if tx.To() == nil || *tx.To() != tradingStateAddr {
			continue
		}
		from, err := types.Sender(signer, tx)
		if err != nil || from != author {
			continue
		}
		if len(tx.Data()) >= 32 {
			return common.BytesToHash(tx.Data()[:32])
		}
	}
	return tradingstate.EmptyRoot
}

func (p *VictionProcessor) CommittedTradingRoot() common.Hash {
	return p.tradingCommittedRoot
}

// Flush current block Trading State Trie to LevelDB.
func (p *StateProcessor) CommitTradingState(block *types.Block) error {
	if !p.viction.IsTradingInitialized() {
		return nil
	}
	tradingRoot := p.viction.CommittedTradingRoot()
	if tradingRoot == (common.Hash{}) {
		return nil
	}
	if err := p.viction.TradingEngine().GetStateCache().TrieDB().Commit(tradingRoot, false, nil); err != nil {
		return fmt.Errorf("native_trading: failed to commit Trie at block %d: %w", block.NumberU64(), err)
	}
	log.Debug("[NativeTrading] Flushed Trie to disk", "block", block.NumberU64(), "root", tradingRoot.Hex())
	return nil
}

// Flush current block Trading State Trie in GC cache to LevelDB.
func (p *StateProcessor) CommitTradingStateDeferred(block *types.Block) error {
	if !p.viction.IsTradingInitialized() {
		return nil
	}
	current := block.NumberU64()
	tradingRoot := p.viction.CommittedTradingRoot()
	if tradingRoot == (common.Hash{}) {
		return nil
	}
	tradingTrieDB := p.viction.TradingEngine().GetStateCache().TrieDB()
	tradingTrieDB.Reference(tradingRoot, common.Hash{})
	p.tradingTriegc.Push(tradingRoot, -int64(current))

	if err := tradingTrieDB.Commit(tradingRoot, true, nil); err != nil {
		return fmt.Errorf("native_trading: failed to commit Trie at block %d: %w", current, err)
	}
	log.Debug("[NativeTrading] Flushed Trie to disk", "block", current, "root", tradingRoot.Hex())

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
	return nil
}

// Flush all Trading State Trie entires in GC cache to LevelDB.
func (p *StateProcessor) FlushTradingStateGCCache() {
	if p.bc.cacheConfig.TrieDirtyDisabled || p.viction.TradingEngine() == nil {
		return
	}

	tradingTrieDB := p.viction.TradingEngine().GetStateCache().TrieDB()
	for !p.tradingTriegc.Empty() {
		root := p.tradingTriegc.PopItem()
		if err := tradingTrieDB.Commit(root.(common.Hash), true, nil); err != nil {
			log.Error("[NativeTrading] Failed to commit Trie on shutdown", "root", root, "err", err)
		}
		tradingTrieDB.Dereference(root.(common.Hash))
	}
}
