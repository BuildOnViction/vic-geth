// Copyright 2026 The Vic-geth Authors
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

package viction

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vrc25"
	"github.com/ethereum/go-ethereum/legacy/tomox/tradingstate"
	"github.com/ethereum/go-ethereum/legacy/tomoxlending/lendingstate"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// ErrBlacklistedAddress is returned when a transaction sender/receiver is
// blacklisted. Canonical definition; used by transaction pool admission,
// block import and the worker sealing lanes.
var ErrBlacklistedAddress = errors.New("blacklisted address")

// ChainReader is the subset of *core.BlockChain the Processor needs during
// block processing. Defined here to avoid an import cycle between core and
// core/viction; it embeds tradingstate.ChainContext so the chain can be passed
// straight into the TomoX/TomoZ engines.
type ChainReader interface {
	tradingstate.ChainContext

	// GetBlock retrieves a block from the chain by hash and number.
	GetBlock(hash common.Hash, number uint64) *types.Block
}

// Processor owns all Viction-specific block and transaction processing
// hooks. It is deliberately decoupled from StateProcessor so that block import,
// the tracing API and the miner can all drive the identical hook logic instead
// of re-implementing it. Create one instance per block execution: block import
// uses NewProcessor and calls the full hook cycle, while consumers that
// only re-execute transactions (tracing, eth_call) use NewTxProcessor,
// which leaves the chain, engine and order-matching paths inert.
type Processor struct {
	config        *params.ChainConfig // Chain configuration options
	chain         ChainReader         // Canonical block chain, nil for tx-only consumers
	engine        consensus.Engine    // Consensus engine, nil disables author-dependent paths
	tradingEngine TradingEngine       // legacy TomoX engine, nil when TomoX is not needed
	lendingEngine LendingEngine       // legacy TomoZ engine, nil when lending is not needed

	// Per-block state, reset by BeforeProcess (block import) or seeded once by
	// NewTxProcessor (transaction re-execution).
	currentBlockNumber *big.Int

	// tradingStateDB is the TomoX order-book trie. Non-nil only for pre-Atlas blocks.
	tradingStateDB *tradingstate.TradingStateDB

	// lendingStateDB is the TomoZ lending order-book trie. Non-nil only for pre-Atlas blocks.
	lendingStateDB *lendingstate.LendingStateDB

	// committedTradingRoot is the root returned by tradingStateDB.Commit() in
	// AfterProcess. It is already staged into the trie.Database dirty set and
	// ready to flush to LevelDB.
	committedTradingRoot common.Hash

	// committedLendingRoot is the root returned by lendingStateDB.Commit() in AfterProcess.
	committedLendingRoot common.Hash

	// feeProc owns the pre-Atlas VRC25 fee accounting for this block: the running
	// pool threaded into each transaction, the per-tx decrement, and the
	// end-of-block flush. See FeeProcessor.
	feeProc *FeeProcessor
}

// NewProcessor creates the full block-import processor. The TomoX and
// TomoZ engines are injected later via SetTradingEngine / SetLendingEngine;
// per-block state is initialised by BeforeProcess.
func NewProcessor(config *params.ChainConfig, chain ChainReader, engine consensus.Engine) *Processor {
	return &Processor{
		config: config,
		chain:  chain,
		engine: engine,
	}
}

// NewTxProcessor creates a transaction-only processor for consumers that
// re-execute transactions without importing a block (tracing API, eth_call).
// It seeds the VRC25 fee pool immediately from the given state; the chain,
// consensus engine and TomoX/TomoZ paths stay inert, so only
// BeforeApplyTransaction, HandleFee and FeePool are meaningful on the result.
func NewTxProcessor(config *params.ChainConfig, statedb vm.StateDB, blockNum *big.Int) *Processor {
	return &Processor{
		config:             config,
		currentBlockNumber: new(big.Int).Set(blockNum),
		feeProc:            NewFeeProcessor(config, statedb, blockNum),
	}
}

// SetTradingEngine injects the TomoX trading engine.
func (vp *Processor) SetTradingEngine(engine TradingEngine) {
	vp.tradingEngine = engine
}

// SetLendingEngine injects the TomoZ lending engine.
func (vp *Processor) SetLendingEngine(engine LendingEngine) {
	vp.lendingEngine = engine
}

// TradingEngine returns the injected TomoX engine, nil when not installed.
func (vp *Processor) TradingEngine() TradingEngine {
	return vp.tradingEngine
}

// LendingEngine returns the injected TomoZ engine, nil when not installed.
func (vp *Processor) LendingEngine() LendingEngine {
	return vp.lendingEngine
}

// HasTradingState reports whether a TomoX trading trie is open for the current block.
func (vp *Processor) HasTradingState() bool {
	return vp.tradingStateDB != nil
}

// HasLendingState reports whether a TomoZ lending trie is open for the current block.
func (vp *Processor) HasLendingState() bool {
	return vp.lendingStateDB != nil
}

// CommittedTradingRoot returns the trading trie root committed by AfterProcess,
// or the zero hash when nothing was committed. Used by the blockchain to flush
// the staged trie nodes to LevelDB.
func (vp *Processor) CommittedTradingRoot() common.Hash {
	return vp.committedTradingRoot
}

// CommittedLendingRoot returns the lending trie root committed by AfterProcess,
// or the zero hash when nothing was committed.
func (vp *Processor) CommittedLendingRoot() common.Hash {
	return vp.committedLendingRoot
}

// FeePool returns the running fee-capacity map to thread into ApplyMessage.
// Safe on a nil receiver (returns nil).
func (vp *Processor) FeePool() FeePool {
	if vp == nil {
		return nil
	}
	return vp.feeProc.FeePool()
}

// HandleFee applies the pre-Atlas per-transaction fee accounting for one
// re-executed transaction. It delegates to the owned FeeProcessor; see
// FeeProcessor.HandleFee for the exact semantics. Safe on a nil receiver.
func (vp *Processor) HandleFee(statedb vm.StateDB, tx *types.Transaction, from common.Address, usedGas uint64, failed bool) {
	if vp == nil {
		return
	}
	vp.feeProc.HandleFee(statedb, vp.currentBlockNumber, tx, from, usedGas, failed)
}

// Copy returns an independent transaction-only copy of vp for use by a parallel
// worker: the fee pool is deep-copied so goroutines never mutate a shared map,
// while the flush accumulators start fresh (tracing never flushes). The chain,
// engines and TomoX/TomoZ trie state are deliberately not carried over.
func (vp *Processor) Copy() *Processor {
	if vp == nil {
		return nil
	}
	cp := &Processor{
		config: vp.config,
	}
	if vp.currentBlockNumber != nil {
		cp.currentBlockNumber = new(big.Int).Set(vp.currentBlockNumber)
	}
	cp.feeProc = vp.feeProc.Copy()
	return cp
}

// BeforeProcess initialises Viction-specific per-block state and runs hardfork
// activation logic (TIPSigning, Atlas, Saigon, VRC25 fee snapshots, TomoX/TomoZ
// trie opening, and epoch liquidation). Called once at the start of Process
// before any transactions are applied.
func (vp *Processor) BeforeProcess(block *types.Block, statedb *state.StateDB) error {
	header := block.Header()

	vp.currentBlockNumber = new(big.Int).Set(header.Number)
	vp.tradingStateDB = nil
	vp.lendingStateDB = nil
	vp.committedTradingRoot = common.Hash{}
	vp.committedLendingRoot = common.Hash{}
	vp.feeProc = NewFeeProcessor(vp.config, statedb, header.Number)

	if vp.config.TIPSigningBlock != nil && vp.config.TIPSigningBlock.Cmp(header.Number) == 0 {
		if vp.config.Viction != nil {
			statedb.DeleteAddress(vp.config.Viction.ValidatorBlockSignContract)
		}
	}
	if vp.config.Viction != nil {
		if vp.config.IsAtlas(header.Number) {
			misc.ApplyVIPVRC25Upgrade(statedb, vp.config.Viction, vp.config.AtlasBlock, header.Number)
		}
		if vp.config.IsSaigon(block.Number()) {
			misc.ApplySaigonHardFork(statedb, vp.config.Viction, vp.config.SaigonBlock, block.Number())
		}
	}

	// Open TomoX and TomoZ tries from the parent block.
	// Both engines share the same parent fetch; Author() is called once per engine
	// since the signature scheme may differ in future (currently both use PoSV).
	if vp.config.Posv != nil && vp.config.IsTomoXEnabled(header.Number) &&
		header.Number.Uint64() > vp.config.Posv.Epoch {

		parent := vp.chain.GetBlock(header.ParentHash, header.Number.Uint64()-1)
		if parent != nil {
			parentAuthor, _ := vp.engine.Author(parent.Header())

			if vp.tradingEngine != nil {
				tradingState, err := vp.tradingEngine.GetTradingState(parent, parentAuthor)
				if err != nil {
					return fmt.Errorf("TomoX: failed to open TradingStateDB at block %d: %w", header.Number, err)
				}
				vp.tradingStateDB = tradingState

				if header.Number.Uint64()%vp.config.Posv.Epoch == 0 {
					if err := vp.tradingEngine.UpdateMediumPriceBeforeEpoch(
						header.Number.Uint64()/vp.config.Posv.Epoch,
						tradingState, statedb,
					); err != nil {
						return fmt.Errorf("TomoX: UpdateMediumPriceBeforeEpoch failed at block %d: %w", header.Number, err)
					}
				}
			}

			if vp.lendingEngine != nil {
				lendingState, err := vp.lendingEngine.GetLendingState(parent, parentAuthor)
				if err != nil {
					return fmt.Errorf("TomoZ: failed to open LendingStateDB at block %d: %w", header.Number, err)
				}
				vp.lendingStateDB = lendingState
			}
		}
	}

	// At epoch+LendingLiquidateTradeBlock boundaries, run liquidation before the tx
	// loop so that AutoTopUp changes land on the same trie objects that will be used
	// during applyLendingTx and committed in AfterProcess.
	if vp.lendingEngine != nil && vp.config.Posv != nil &&
		vp.config.IsTomoXEnabled(header.Number) &&
		vp.config.Viction != nil &&
		header.Number.Uint64()%vp.config.Posv.Epoch == uint64(vp.config.Viction.LendingLiquidateTradeBlock) &&
		vp.tradingStateDB != nil && vp.lendingStateDB != nil {

		_, _, _, _, _, err := vp.lendingEngine.ProcessLiquidationData(
			header, vp.chain, statedb, vp.tradingStateDB, vp.lendingStateDB,
		)
		if err != nil {
			return fmt.Errorf("TomoZ: ProcessLiquidationData failed at block %d: %w", header.Number, err)
		}
		log.Debug("TomoZ: epoch liquidation processed", "block", header.Number.Uint64())
	}

	InitSignerInTransactions(vp.config, header, block.Transactions())

	return nil
}

// AfterProcess flushes accumulated VRC25 fee updates to state and commits the
// TomoX and TomoZ trie roots, verifying them against the 0x92 system
// transaction embedded in the block. Called once after all transactions have
// been applied.
func (vp *Processor) AfterProcess(block *types.Block, statedb *state.StateDB) error {
	// Pre-Atlas: flush accumulated VRC25 fee updates to state.
	vp.feeProc.Flush(statedb)

	// Commit and verify the TomoX trading state root.
	//
	// IMPORTANT: tradingStateDB.Commit() must be called BEFORE IntermediateRoot().
	// IntermediateRoot calls trie.Hash() which collapses in-memory dirty nodes into
	// hash nodes.  A subsequent trie.Commit() on a fully-hashed trie finds no dirty
	// nodes and writes nothing to trie.Database.dirties.  If Commit() runs first, it
	// flushes dirty nodes into trie.Database.dirties; trie.Database.Commit() can then
	// persist them to LevelDB.
	if vp.tradingStateDB != nil && vp.tradingEngine != nil {
		tradingRoot, err := vp.tradingStateDB.Commit()
		if err != nil {
			return fmt.Errorf("TomoX: TradingStateDB.Commit failed at block %d: %w", block.NumberU64(), err)
		}
		vp.committedTradingRoot = tradingRoot

		blockAuthor, err := vp.engine.Author(block.Header())
		if err != nil {
			return fmt.Errorf("TomoX: failed to resolve block author at block %d: %w", block.NumberU64(), err)
		}
		expectRoot := GetTradingStateRoot(block, vp.config.Viction.TradingStateContract, blockAuthor, vp.config)
		if tradingRoot != expectRoot {
			return fmt.Errorf("TomoX: trading state root mismatch at block %d: got %s, expected %s",
				block.NumberU64(), tradingRoot.Hex(), expectRoot.Hex())
		}
		log.Debug("TomoX: trading state root verified", "block", block.NumberU64(), "root", tradingRoot.Hex())
	}

	// Commit and verify the TomoZ lending state root (same logic as above).
	if vp.lendingStateDB != nil && vp.tradingStateDB != nil {
		lendingRoot, err := vp.lendingStateDB.Commit()
		if err != nil {
			return fmt.Errorf("TomoZ: LendingStateDB.Commit failed at block %d: %w", block.NumberU64(), err)
		}
		vp.committedLendingRoot = lendingRoot

		blockAuthor, err := vp.engine.Author(block.Header())
		if err != nil {
			return fmt.Errorf("TomoZ: failed to resolve block author at block %d: %w", block.NumberU64(), err)
		}
		expectRoot := GetLendingStateRoot(block, vp.config.Viction.TradingStateContract, blockAuthor, vp.config)
		if lendingRoot != expectRoot {
			return fmt.Errorf("TomoZ: lending state root mismatch at block %d: got %s, expected %s",
				block.NumberU64(), lendingRoot.Hex(), expectRoot.Hex())
		}
		log.Debug("TomoZ: lending state root verified", "block", block.NumberU64(), "root", lendingRoot.Hex())
	}

	return nil
}

// BeforeApplyTransaction runs per-transaction Viction pre-checks: optional
// balance override for specific accounts in early blocks, and blacklist
// enforcement after TIPBlacklist.
func (vp *Processor) BeforeApplyTransaction(block *types.Block, tx *types.Transaction, msg types.Message, statedb *state.StateDB) error {
	if vp.config.Viction == nil {
		return nil
	}
	header := block.Header()

	if header.Number.BitLen() <= 64 && header.Number.Uint64() <= 9147459 {
		if val := vp.config.Viction.GetVictionBypassBalance(header.Number.Uint64(), msg.From()); val != nil {
			statedb.SetBalance(msg.From(), val)
		}
	}

	if vp.config.IsTIPBlacklist(block.Number()) {
		if sender, receiver := BlacklistedTxParty(vp.config, msg.From(), tx.To()); sender || receiver {
			return ErrBlacklistedAddress
		}
	}

	return nil
}

// BlacklistedTxParty reports whether the transaction sender or recipient is
// blacklisted. It is not fork-gated: callers (block import, transaction pool
// admission, worker sealing lanes) apply their own gating and their own
// action on a hit.
func BlacklistedTxParty(config *params.ChainConfig, from common.Address, to *common.Address) (sender, receiver bool) {
	if config.Viction == nil {
		return false, false
	}
	return config.Viction.IsBlacklisted(from), to != nil && config.Viction.IsBlacklisted(*to)
}

// ApplyVictionTransaction checks whether tx is a Viction system transaction
// (0x91 TomoX batch, 0x92 state-root commit, 0x93 lending batch, 0x94
// finalized trade) and handles it without the EVM.
// Returns (false, nil, 0, nil, nil) for regular transactions.
//
// The 0x89 block-signer transaction is NOT handled here: it is dispatched by
// core (see core.ApplySignTransaction) so its nonce errors stay core-native.
func (vp *Processor) ApplyVictionTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	if tx.To() == nil || vp.config.Viction == nil {
		return false, nil, 0, nil, nil
	}
	vicConfig := vp.config.Viction

	// 0x91 — TomoX order-matching batch (active only in TIPTomoX..Atlas window).
	if tx.IsTradingTransaction(vicConfig.TomoXContract) && vp.config.IsTomoXEnabled(header.Number) {
		if batch, err := tradingstate.DecodeTxMatchesBatch(tx.Data()); err == nil {
			return vp.applyTomoXTx(statedb, tx, header, usedGas, batch)
		}
		// Decode failed — let the normal EVM path handle it.
	}

	// 0x92 — trading state root commit; verified in AfterProcess
	if tx.To() != nil && *tx.To() == vicConfig.TradingStateContract && vp.config.IsTomoXEnabled(header.Number) {
		return vp.applyEmptyTransaction(statedb, tx, header, usedGas)
	}

	// 0x93 — TomoZ lending order-matching batch.
	// The outer gate is IsTomoXEnabled (TIPTomoX..Atlas); the inner matching
	// only runs once TIPTomoXLending is also active.
	if tx.IsLendingTransaction(vicConfig.LendingContract) && vp.config.IsTomoXEnabled(header.Number) {
		if batch, err := lendingstate.DecodeTxLendingBatch(tx.Data()); err == nil {
			return vp.applyLendingTx(statedb, tx, header, usedGas, batch)
		}
		// Decode failed — let the normal EVM path handle it.
	}

	// 0x94 — lending finalized trade; outer gate is IsTomoXEnabled.
	if tx.IsLendingFinalizedTradeTransaction(vicConfig.LendingFinalizedContract) && vp.config.IsTomoXEnabled(header.Number) {
		return vp.applyEmptyTransaction(statedb, tx, header, usedGas)
	}

	return false, nil, 0, nil, nil
}

// AfterApplyTransaction accumulates the VRC25 token fee deducted for this
// transaction into the per-block feeUpdated map. Runs only on pre-Atlas blocks.
func (vp *Processor) AfterApplyTransaction(tx *types.Transaction, msg types.Message, statedb *state.StateDB, receipt *types.Receipt, usedGas uint64, err error) error {
	if vp.currentBlockNumber == nil {
		return nil
	}

	blockNum := vp.currentBlockNumber

	if tx.To() == nil {
		return nil
	}

	token := *tx.To()
	vicCfg := vp.config.Viction

	if vp.config.IsAtlas(blockNum) {
		// Post-Atlas: charge VRC25 token fee for failed sponsored txs.
		if receipt.Status == types.ReceiptStatusFailed && vicCfg != nil && vicCfg.VRC25GasPrice != nil {
			feeCap := vrc25.GetFeeCapacity(statedb, vicCfg.VRC25Contract, tx.To())
			fee := new(big.Int).Mul(
				new(big.Int).SetUint64(usedGas),
				(*big.Int)(vicCfg.VRC25GasPrice),
			)
			if feeCap != nil && feeCap.Cmp(fee) > 0 {
				vrc25.PayFeeWithVRC25(statedb, msg.From(), token)
			}
		}
		return nil
	}

	// Pre-Atlas: decrement the running fee pool and record the update for the
	// AfterProcess flush. Shared with the tracing API via FeeProcessor, so the
	// fee formula cannot drift.
	failed := receipt.Status == types.ReceiptStatusFailed
	vp.feeProc.HandleFee(statedb, blockNum, tx, msg.From(), usedGas, failed)

	return nil
}

// applyEmptyTransaction produces a zero-gas receipt without running the EVM.
// Used for system transactions whose effects are handled outside the EVM (0x91–0x94).
func (vp *Processor) applyEmptyTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	var root []byte
	if vp.config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(vp.config.IsEIP158(header.Number)).Bytes()
	}
	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 0

	log := &types.Log{}
	log.Address = *tx.To()
	log.BlockNumber = header.Number.Uint64()
	statedb.AddLog(log)
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return true, receipt, 0, nil, nil
}

// applyTomoXTx replays a TomoX order-matching batch (0x91 transaction).
//
// batch is pre-decoded by the caller in ApplyVictionTransaction; no second
// RLP decode is performed here.
//
// On epoch-boundary blocks (block % Epoch == 0) skips order execution
// entirely — UpdateMediumPriceBeforeEpoch already ran in BeforeProcess.
func (vp *Processor) applyTomoXTx(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64, batch tradingstate.TxMatchBatch) (bool, *types.Receipt, uint64, error, *big.Int) {
	var root []byte
	if vp.config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(vp.config.IsEIP158(header.Number)).Bytes()
	}

	isEpochBlock := vp.config.Posv != nil && header.Number.Uint64()%vp.config.Posv.Epoch == 0

	if !isEpochBlock && vp.tradingStateDB != nil && vp.tradingEngine != nil {
		// Use the block author (recovered from header signature) as coinbase,
		// not header.Coinbase which is zeroed in PoSV blocks. The author is
		// the address passed to ValidateTradingOrder -> DoSettleBalance for
		// masternode fee accounting.
		coinbase, err := vp.engine.Author(header)
		if err != nil {
			log.Warn("TomoX: failed to recover block author, using zero address", "err", err)
		}
		tradingEngine := vp.tradingEngine
		tradingStateDB := vp.tradingStateDB

		for i, txDataMatch := range batch.Data {
			order, err := txDataMatch.DecodeOrder()
			if err != nil {
				log.Warn("TomoX: failed to decode order, skipping", "index", i, "err", err)
				continue
			}

			orderBook := tradingstate.GetTradingOrderBookHash(order.BaseToken, order.QuoteToken)

			_, rejects, err := tradingEngine.CommitOrder(header, coinbase, vp.chain, statedb, tradingStateDB, orderBook, order)
			if err != nil {
				return true, nil, 0, fmt.Errorf("TomoX: CommitOrder failed index=%d order=%s: %w", i, order.Hash.Hex(), err), nil
			}

			if len(rejects) > 0 {
				log.Debug("TomoX: orders rejected", "count", len(rejects))
			}
		}
	}

	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 0

	txLog := &types.Log{}
	txLog.Address = *tx.To()
	txLog.BlockNumber = header.Number.Uint64()
	statedb.AddLog(txLog)
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return true, receipt, 0, nil, nil
}

// applyLendingTx replays a TomoZ lending order-matching batch (0x93 transaction).
//
// batch is pre-decoded by the caller in ApplyVictionTransaction; no second
// RLP decode is performed here.
//
// Mirrors the epoch-skip rule from applyTomoXTx: on epoch-boundary blocks
// skips all order execution — UpdateMediumPriceBeforeEpoch already ran in BeforeProcess.
func (vp *Processor) applyLendingTx(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64, batch lendingstate.TxLendingBatch) (bool, *types.Receipt, uint64, error, *big.Int) {
	var root []byte
	if vp.config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(vp.config.IsEIP158(header.Number)).Bytes()
	}

	isEpochBlock := vp.config.Posv != nil && header.Number.Uint64()%vp.config.Posv.Epoch == 0

	if !isEpochBlock &&
		vp.lendingStateDB != nil &&
		vp.tradingStateDB != nil &&
		vp.lendingEngine != nil {

		// Use the block author (recovered from header signature) as coinbase,
		// not header.Coinbase which is zeroed in PoSV blocks. The author is
		// the address passed to ValidateLendingOrder -> DoSettleBalance for
		// masternode fee accounting.
		coinbase, err := vp.engine.Author(header)
		if err != nil {
			log.Warn("TomoZ: failed to recover block author, using zero address", "err", err)
		}
		lendingStateDB := vp.lendingStateDB
		tradingStateDB := vp.tradingStateDB

		for i, order := range batch.Data {
			if order == nil {
				continue
			}
			lendingOrderBook := lendingstate.GetLendingOrderBookHash(order.LendingToken, order.Term)
			_, rejects, err := vp.lendingEngine.CommitOrder(
				header, coinbase, vp.chain, statedb,
				lendingStateDB, tradingStateDB, lendingOrderBook, order,
			)
			if err != nil {
				return true, nil, 0, fmt.Errorf("TomoZ: CommitOrder failed index=%d: %w", i, err), nil
			}
			if len(rejects) > 0 {
				log.Debug("TomoZ: lending orders rejected", "count", len(rejects))
			}
		}
	}

	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 0
	txLog := &types.Log{}
	txLog.Address = *tx.To()
	txLog.BlockNumber = header.Number.Uint64()
	statedb.AddLog(txLog)
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return true, receipt, 0, nil, nil
}

// InitSignerInTransactions pre-caches sender addresses for all transactions in
// parallel, amortizing ECDSA recovery before the sequential transaction loop.
func InitSignerInTransactions(config *params.ChainConfig, header *types.Header, txs types.Transactions) {
	if txs.Len() == 0 {
		return
	}
	nWorker := runtime.NumCPU()
	signer := types.MakeSigner(config, header.Number)
	chunkSize := txs.Len() / nWorker
	if txs.Len()%nWorker != 0 {
		chunkSize++
	}
	wg := sync.WaitGroup{}
	wg.Add(nWorker)
	for i := 0; i < nWorker; i++ {
		from := i * chunkSize
		to := from + chunkSize
		if to > txs.Len() {
			to = txs.Len()
		}
		go func(from int, to int) {
			defer wg.Done()
			for j := from; j < to; j++ {
				types.CacheSigner(signer, txs[j])
				txs[j].CacheHash()
			}
		}(from, to)
	}
	wg.Wait()
}

// GetLendingStateRoot extracts the TomoZ lending state root from the 0x92 tx.
// The tx payload is [32 bytes trading root | 32 bytes lending root]; this returns bytes [32:64].
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

// GetTradingStateRoot extracts the TomoX trading state root from the 0x92 tx.
// The tx payload is [32 bytes trading root | 32 bytes lending root]; this returns bytes [0:32].
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

// TradingEngine is the interface the TomoX engine must satisfy.
// Defined here to avoid an import cycle between core/viction and legacy/tomox.
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
// Defined here to avoid an import cycle between core/viction and legacy/tomoxlending.
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
