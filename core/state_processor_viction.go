// Copyright 2026 The Vic-geth Authors
package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/legacy/tomox/tradingstate"
	"github.com/ethereum/go-ethereum/legacy/tomoxlending/lendingstate"
	"github.com/ethereum/go-ethereum/params"
)

// TradingEngine is the interface the TomoX engine must satisfy.
// Defined here to avoid an import cycle between core and legacy/tomox.
type TradingEngine interface {
	// CommitOrder replays a single order through the matching engine,
	// mutating statedb and tradingStateDB.
	CommitOrder(
		header *types.Header,
		coinbase common.Address,
		chain tradingstate.ChainContext,
		statedb *state.StateDB,
		tradingStateDB *tradingstate.TradingStateDB,
		orderBook common.Hash,
		order *tradingstate.OrderItem,
	) ([]map[string]string, []*tradingstate.OrderItem, error)

	// GetTradingState opens the TradingStateDB trie rooted at the given block.
	GetTradingState(block *types.Block, author common.Address) (*tradingstate.TradingStateDB, error)

	// GetTradingStateRoot returns the trading state root committed in the 0x92 tx
	// of the given block.
	GetTradingStateRoot(block *types.Block, author common.Address) (common.Hash, error)

	// UpdateMediumPriceBeforeEpoch computes epoch-averaged prices; must be called
	// before order matching at epoch boundaries.
	UpdateMediumPriceBeforeEpoch(epoch uint64, tradingStateDB *tradingstate.TradingStateDB, statedb *state.StateDB) error

	// GetStateCache returns the trie-node cache backed by the tomox LevelDB.
	// Used to flush trie nodes to disk after each block.
	GetStateCache() tradingstate.Database
}

// LendingEngine is the interface the TomoZ lending engine must satisfy.
// Defined here to avoid an import cycle between core and legacy/tomoxlending.
type LendingEngine interface {
	// GetLendingStateRoot returns the lending state root embedded in the 0x92
	// system transaction of the given block.
	GetLendingStateRoot(block *types.Block, author common.Address) (common.Hash, error)

	// GetLendingState opens the LendingStateDB trie rooted at the given block's
	// lending state root.
	GetLendingState(block *types.Block, author common.Address) (*lendingstate.LendingStateDB, error)

	// HasLendingState reports whether the given block has a lending state trie.
	HasLendingState(block *types.Block, author common.Address) bool

	// GetStateCache returns the trie-node cache backed by the tomoxlending LevelDB.
	// Used to flush trie nodes to disk after each block.
	GetStateCache() lendingstate.Database

	// CommitOrder applies a single lending order, reverting all state on error.
	CommitOrder(
		header *types.Header,
		coinbase common.Address,
		chain tradingstate.ChainContext,
		statedb *state.StateDB,
		lendingStateDB *lendingstate.LendingStateDB,
		tradingStateDB *tradingstate.TradingStateDB,
		lendingOrderBook common.Hash,
		order *lendingstate.LendingItem,
	) ([]*lendingstate.LendingTrade, []*lendingstate.LendingItem, error)

	// GetCollateralPrices returns the VIC-denominated prices for the lending
	// token and collateral token.
	GetCollateralPrices(
		header *types.Header,
		chain tradingstate.ChainContext,
		statedb *state.StateDB,
		tradingStateDB *tradingstate.TradingStateDB,
		collateralToken common.Address,
		lendingToken common.Address,
	) (*big.Int, *big.Int, error)

	// GetMediumTradePriceBeforeEpoch returns the epoch-averaged trade price for
	// the given token pair from the trading state.
	GetMediumTradePriceBeforeEpoch(
		chain tradingstate.ChainContext,
		statedb *state.StateDB,
		tradingStateDB *tradingstate.TradingStateDB,
		baseToken common.Address,
		quoteToken common.Address,
	) (*big.Int, error)

	// ProcessLiquidationData computes which lending trades must be liquidated,
	// auto-repaid, topped up, or recalled at epoch boundaries.
	ProcessLiquidationData(
		header *types.Header,
		chain tradingstate.ChainContext,
		statedb *state.StateDB,
		tradingState *tradingstate.TradingStateDB,
		lendingState *lendingstate.LendingStateDB,
	) (
		updatedTrades map[common.Hash]*lendingstate.LendingTrade,
		liquidatedTrades []*lendingstate.LendingTrade,
		autoRepayTrades []*lendingstate.LendingTrade,
		autoTopUpTrades []*lendingstate.LendingTrade,
		autoRecallTrades []*lendingstate.LendingTrade,
		err error,
	)
}

// ApplySignTransaction processes a BlockSigner special transaction (0x89)
// without the EVM: increments the sender nonce, adds a log entry, and returns a
// zero-gas receipt. Used by the VictionProcessor (block import), the standalone
// ApplyTransaction path, and the miner (block creation).
func ApplySignTransaction(config *params.ChainConfig, statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	// Validate nonce BEFORE Finalise to avoid invalidating the snapshot
	// on error (the caller may need to RevertToSnapshot).
	from, err := types.Sender(types.MakeSigner(config, header.Number), tx)
	if err != nil {
		return true, nil, 0, err, nil
	}
	nonce := statedb.GetNonce(from)
	if nonce < tx.Nonce() {
		return true, nil, 0, ErrNonceTooHigh, nil
	} else if nonce > tx.Nonce() {
		return true, nil, 0, ErrNonceTooLow, nil
	}

	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}

	statedb.SetNonce(from, nonce+1)

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 0
	// Set the receipt logs and create a bloom for filtering
	logEntry := &types.Log{}
	logEntry.Address = config.Viction.ValidatorBlockSignContract
	logEntry.BlockNumber = header.Number.Uint64()
	statedb.AddLog(logEntry)
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return true, receipt, 0, nil, nil
}

// SetTradingEngine injects the TomoX trading engine into the state processor.
func (p *StateProcessor) SetTradingEngine(engine TradingEngine) {
	p.viction.SetTradingEngine(engine)
}

// SetLendingEngine injects the TomoZ lending engine into the state processor.
func (p *StateProcessor) SetLendingEngine(engine LendingEngine) {
	p.viction.SetLendingEngine(engine)
}
