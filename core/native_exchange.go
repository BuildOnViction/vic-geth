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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/legacy/lending/lendingstate"
	"github.com/ethereum/go-ethereum/legacy/trading/tradingstate"
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
