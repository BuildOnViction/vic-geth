// Copyright 2015 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/legacy/tomox/tradingstate"
	"github.com/ethereum/go-ethereum/legacy/tomoxlending/lendingstate"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// VictionProcessor handles Viction-specific block and transaction logic.
type VictionProcessor struct {
	config *params.ChainConfig // Chain configuration
	chain  *BlockChain         // Canonical block chain, nil for tx-only consumers
	engine consensus.Engine    // Consensus engine, nil disables author-dependent paths
	fee    *FeeProcessor       // Zero-gas fee processor.

	blockNumber *big.Int // Cached block number for synchronizing between BeforeProcess and AfterProcess.

	lendingEngine        LendingEngine
	lendingStateDB       *lendingstate.LendingStateDB
	lendingCommittedRoot common.Hash

	tradingEngine        TradingEngine
	tradingStateDB       *tradingstate.TradingStateDB
	tradingCommittedRoot common.Hash
}

// Return new Processor instance for new block production.
func NewVictionProcessor(config *params.ChainConfig, chain *BlockChain, engine consensus.Engine) *VictionProcessor {
	return &VictionProcessor{
		config: config,
		chain:  chain,
		engine: engine,
	}
}

// Return new Processor instance for old block execution.
func NewTxVictionProcessor(config *params.ChainConfig, statedb *state.StateDB, blockNum *big.Int) *VictionProcessor {
	return &VictionProcessor{
		config: config,
		fee:    NewFeeProcessor(config, statedb, blockNum),

		blockNumber: new(big.Int).Set(blockNum),
	}
}

// Return a deep copy of Processor.
func (p *VictionProcessor) Copy() *VictionProcessor {
	if p == nil {
		return nil
	}
	cp := &VictionProcessor{
		config: p.config,
	}
	if p.blockNumber != nil {
		cp.blockNumber = new(big.Int).Set(p.blockNumber)
	}
	cp.fee = p.fee.Copy()
	return cp
}

// Return original balances snapshot.
func (p *VictionProcessor) FeePool() types.BalanceMap {
	if p == nil {
		return nil
	}
	return p.fee.FeePool()
}

// Pre block processing: - Prepare internal state for block processing - Active Viction-specific hard forks.
func (p *VictionProcessor) BeforeBlockProcess(block *types.Block, statedb *state.StateDB) error {
	if !p.isSupported() {
		return nil
	}

	header := block.Header()
	p.fee = NewFeeProcessor(p.config, statedb, header.Number)
	p.blockNumber = new(big.Int).Set(header.Number)

	p.lendingStateDB = nil
	p.lendingCommittedRoot = common.Hash{}
	p.tradingStateDB = nil
	p.tradingCommittedRoot = common.Hash{}

	misc.ApplyPosvHardForks(statedb, p.config, p.config.Viction, header.Number)

	if p.config.IsTomoXEnabled(header.Number) && header.Number.Uint64() > p.config.Posv.Epoch {
		parent := p.chain.GetBlock(header.ParentHash, header.Number.Uint64()-1)
		if parent != nil {
			parentAuthor, _ := p.engine.Author(parent.Header())

			if p.tradingEngine != nil {
				tradingState, err := p.tradingEngine.GetTradingState(parent, parentAuthor)
				if err != nil {
					return fmt.Errorf("native_trading: failed to open StateDB at block %d: %w", header.Number, err)
				}
				p.tradingStateDB = tradingState

				if header.Number.Uint64()%p.config.Posv.Epoch == 0 {
					if err := p.tradingEngine.UpdateMediumPriceBeforeEpoch(
						header.Number.Uint64()/p.config.Posv.Epoch,
						tradingState, statedb,
					); err != nil {
						return fmt.Errorf("native_trading: failed to exec UpdateMediumPriceBeforeEpoch at block %d: %w", header.Number, err)
					}
				}
			}

			if p.lendingEngine != nil {
				lendingState, err := p.lendingEngine.GetLendingState(parent, parentAuthor)
				if err != nil {
					return fmt.Errorf("native_lending: failed to open StateDB at block %d: %w", header.Number, err)
				}
				p.lendingStateDB = lendingState
			}
		}

		if header.Number.Uint64()%p.config.Posv.Epoch == p.config.Viction.LendingLiquidateTradeBlock && p.IsLendingInitialized() {
			_, _, _, _, _, err := p.lendingEngine.ProcessLiquidationData(header, p.chain, statedb, p.tradingStateDB, p.lendingStateDB)
			if err != nil {
				return fmt.Errorf("native_lending: failed to exec ProcessLiquidationData at block %d: %w", header.Number, err)
			}
			log.Info("[Processor][Native Lending] Epoch liquidation processed", "block", header.Number.Uint64())
		}
	}

	signer := types.MakeSigner(p.config, header.Number)
	types.CacheSigners(signer, block.Transactions())

	return nil
}

// Process fee for given transaction.
func (p *VictionProcessor) ProcessFee(statedb *state.StateDB, tx *types.Transaction, from common.Address, usedGas uint64, failed bool) {
	if !p.isSupported() {
		return
	}
	p.fee.Process(statedb, p.blockNumber, tx, from, usedGas, failed)
}

// Post block processing: - Commit Trading/Lending/ZeroGas journal to database.
func (p *VictionProcessor) AfterProcess(block *types.Block, statedb *state.StateDB) error {
	if !p.isSupported() {
		return nil
	}

	p.fee.Flush(statedb)

	// IMPORTANT: tradingStateDB.Commit() must be called BEFORE IntermediateRoot().
	// IntermediateRoot calls trie.Hash() which collapses in-memory dirty nodes into hash nodes.
	// A subsequent trie.Commit() on a fully-hashed trie finds no dirty nodes and writes nothing to trie.Database.dirties.
	// If Commit() runs first, it flushes dirty nodes into trie.Database.dirties; trie.Database.Commit() can then persist them to LevelDB.
	if p.IsTradingInitialized() {
		tradingRoot, err := p.tradingStateDB.Commit()
		if err != nil {
			return fmt.Errorf("native_trading: failed to commit StateDB at block %d: %w", block.NumberU64(), err)
		}
		p.tradingCommittedRoot = tradingRoot

		blockAuthor, err := p.engine.Author(block.Header())
		if err != nil {
			return fmt.Errorf("native_trading: failed to resolve block author at block %d: %w", block.NumberU64(), err)
		}
		expectRoot := GetTradingStateRoot(block, p.config.Viction.TradingStateContract, blockAuthor, p.config)
		if tradingRoot != expectRoot {
			return fmt.Errorf("native_trading: state root mismatch at block %d: got %s, expected %s", block.NumberU64(), tradingRoot.Hex(), expectRoot.Hex())
		}
		log.Info("[Processor][Native Trading] State root verified", "block", block.NumberU64(), "root", tradingRoot.Hex())
	}

	if p.IsLendingInitialized() {
		lendingRoot, err := p.lendingStateDB.Commit()
		if err != nil {
			return fmt.Errorf("native_lending: failed to commit StateDB at block %d: %w", block.NumberU64(), err)
		}
		p.lendingCommittedRoot = lendingRoot

		blockAuthor, err := p.engine.Author(block.Header())
		if err != nil {
			return fmt.Errorf("native_lending: failed to resolve block author at block %d: %w", block.NumberU64(), err)
		}
		expectRoot := GetLendingStateRoot(block, p.config.Viction.TradingStateContract, blockAuthor, p.config)
		if lendingRoot != expectRoot {
			return fmt.Errorf("native_lending: state root mismatch at block %d: got %s, expected %s", block.NumberU64(), lendingRoot.Hex(), expectRoot.Hex())
		}
		log.Info("[Processor][Native Lending] State root verified", "block", block.NumberU64(), "root", lendingRoot.Hex())
	}

	return nil
}

// Pre transaction processing: - Fix incorrect balances for early blocks. - Prohibit blacklisted addresses.
func (p *VictionProcessor) BeforeApplyTransaction(block *types.Block, tx *types.Transaction, msg types.Message, statedb *state.StateDB) error {
	if !p.isSupported() {
		return nil
	}

	header := block.Header()
	if header.Number.BitLen() <= 64 && header.Number.Uint64() <= 9147459 {
		if val := p.config.Viction.GetVictionBypassBalance(header.Number.Uint64(), msg.From()); val != nil {
			statedb.SetBalance(msg.From(), val)
		}
	}
	if p.config.IsTIPBlacklist(block.Number()) {
		if sender, receiver := misc.ValidateVictionBlackList(p.config, msg.From(), tx.To()); sender || receiver {
			return ErrBlacklistedAddress
		}
	}

	return nil
}

// Handle Viction own transactions without the EVM.
// Return (false, nil, 0, nil, nil) for regular transactions.
func (p *VictionProcessor) ApplyNativeTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	if !p.isSupported() || tx.To() == nil {
		return false, nil, 0, nil, nil
	}
	vicConfig := p.config.Viction

	// 0x89 — Block signing.
	if tx.IsSigningTransaction(vicConfig.ValidatorBlockSignContract) && p.config.IsTIPSigning(header.Number) {
		return p.applyBlockSigningTransaction(statedb, tx, header, usedGas)
	}

	// 0x91 — Trading order-matching batch.
	if tx.IsTradingTransaction(vicConfig.TradingContract) && p.config.IsTomoXEnabled(header.Number) {
		if batch, err := tradingstate.DecodeTxMatchesBatch(tx.Data()); err == nil {
			return p.applyTradingTransaction(statedb, tx, header, usedGas, batch)
		}
	}

	// 0x92 — Trading state root commit, verified in AfterBlockProcess.
	if *tx.To() == vicConfig.TradingStateContract && p.config.IsTomoXEnabled(header.Number) {
		return p.applyEmptyTransaction(statedb, tx, header, usedGas)
	}

	// 0x93 — Lending order-matching batch.
	if tx.IsLendingTransaction(vicConfig.LendingContract) && p.config.IsTomoXEnabled(header.Number) {
		if batch, err := lendingstate.DecodeTxLendingBatch(tx.Data()); err == nil {
			return p.applyLendingTransaction(statedb, tx, header, usedGas, batch)
		}
	}

	// 0x94 — Lending finalized trade.
	if tx.IsLendingFinalizedTradeTransaction(vicConfig.LendingFinalizedContract) && p.config.IsTomoXEnabled(header.Number) {
		return p.applyEmptyTransaction(statedb, tx, header, usedGas)
	}

	return false, nil, 0, nil, nil
}

// Post transaction processing: - Apply zero-gas.
func (p *VictionProcessor) AfterApplyTransaction(tx *types.Transaction, msg types.Message, statedb *state.StateDB, receipt *types.Receipt, usedGas uint64, err error) error {
	if !p.isSupported() || tx.To() == nil || p.blockNumber == nil {
		return nil
	}

	token := *tx.To()
	vicCfg := p.config.Viction

	if p.config.IsAtlas(p.blockNumber) {
		if receipt.Status == types.ReceiptStatusFailed && vicCfg.VRC25GasPrice != nil {
			feeCap := statedb.VicGetZeroGasCapacity(vicCfg.VRC25Contract, tx.To())
			fee := new(big.Int).Mul(
				new(big.Int).SetUint64(usedGas),
				(*big.Int)(vicCfg.VRC25GasPrice),
			)
			if feeCap != nil && feeCap.Cmp(fee) > 0 {
				PayTxFeeUsingToken(statedb, msg.From(), token)
			}
		}
		return nil
	}

	failed := receipt.Status == types.ReceiptStatusFailed
	p.fee.Process(statedb, p.blockNumber, tx, msg.From(), usedGas, failed)

	return nil
}

// applyBlockSigningTransaction processes a BlockSigner special transaction (0x89)
// without the EVM: increments the sender nonce, adds a log entry, and returns a
// zero-gas receipt. Used during block import (ApplyNativeTransaction), the
// standalone ApplyTransaction path, and the miner (block creation).
func (p *VictionProcessor) applyBlockSigningTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	// Validate nonce BEFORE Finalise to avoid invalidating the snapshot
	// on error (the caller may need to RevertToSnapshot).
	from, err := types.Sender(types.MakeSigner(p.config, header.Number), tx)
	if err != nil {
		return true, nil, 0, err, nil
	}
	nonce := statedb.GetNonce(from)
	if nonce < tx.Nonce() {
		return true, nil, 0, ErrNonceTooHigh, nil
	} else if nonce > tx.Nonce() {
		return true, nil, 0, ErrNonceTooLow, nil
	}
	var root []byte
	if p.config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(p.config.IsEIP158(header.Number)).Bytes()
	}
	statedb.SetNonce(from, nonce+1)
	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 0

	log := &types.Log{}
	log.Address = p.config.Viction.ValidatorBlockSignContract
	log.BlockNumber = header.Number.Uint64()
	statedb.AddLog(log)
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return true, receipt, 0, nil, nil
}

// Process transaction as null transcation for system transactions (0x92,0x94).
func (p *VictionProcessor) applyEmptyTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	var root []byte
	if p.config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(p.config.IsEIP158(header.Number)).Bytes()
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

// Process Trading order-matching batch transaction (0x91).
func (p *VictionProcessor) applyTradingTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64, batch tradingstate.TxMatchBatch) (bool, *types.Receipt, uint64, error, *big.Int) {
	var root []byte
	if p.config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(p.config.IsEIP158(header.Number)).Bytes()
	}

	if !p.config.Posv.IsCheckpointBlock(header.Number.Uint64()) && p.IsTradingInitialized() {
		// Use the block author (recovered from header signature) as coinbase, not header.Coinbase which is zeroed in PoSV blocks.
		// The author is the address passed to ValidateTradingOrder -> DoSettleBalance for validator fee accounting.
		coinbase, err := p.engine.Author(header)
		if err != nil {
			log.Warn("[Processor][Native Trading] Failed to recover block author, using zero address", "err", err)
		}
		tradingEngine := p.tradingEngine
		tradingStateDB := p.tradingStateDB

		for i, txDataMatch := range batch.Data {
			order, err := txDataMatch.DecodeOrder()
			if err != nil {
				log.Warn("[Processor][Native Trading] Failed to decode order, skipping", "index", i, "err", err)
				continue
			}

			orderBook := tradingstate.GetTradingOrderBookHash(order.BaseToken, order.QuoteToken)

			_, rejects, err := tradingEngine.CommitOrder(header, coinbase, p.chain, statedb, tradingStateDB, orderBook, order)
			if err != nil {
				return true, nil, 0, fmt.Errorf("native_trading: failed to commit order index=%d order=%s: %w", i, order.Hash.Hex(), err), nil
			}

			if len(rejects) > 0 {
				log.Info("[Processor][Native Trading] Orders rejected", "count", len(rejects))
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

// Process Lending order-matching batch transaction (0x93).
func (p *VictionProcessor) applyLendingTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64, batch lendingstate.TxLendingBatch) (bool, *types.Receipt, uint64, error, *big.Int) {
	var root []byte
	if p.config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(p.config.IsEIP158(header.Number)).Bytes()
	}

	if !p.config.Posv.IsCheckpointBlock(header.Number.Uint64()) && p.IsLendingInitialized() {
		// Use the block author (recovered from header signature) as coinbase, not header.Coinbase which is zeroed in PoSV blocks.
		// The author is the address passed to ValidateLendingOrder -> DoSettleBalance for validator fee accounting.
		coinbase, err := p.engine.Author(header)
		if err != nil {
			log.Warn("[Processor][Native Lending] Failed to recover block author, using zero address", "err", err)
		}
		lendingStateDB := p.lendingStateDB
		tradingStateDB := p.tradingStateDB

		for i, order := range batch.Data {
			if order == nil {
				continue
			}
			lendingOrderBook := lendingstate.GetLendingOrderBookHash(order.LendingToken, order.Term)
			_, rejects, err := p.lendingEngine.CommitOrder(
				header, coinbase, p.chain, statedb,
				lendingStateDB, tradingStateDB, lendingOrderBook, order,
			)
			if err != nil {
				return true, nil, 0, fmt.Errorf("native_lending: failed to commit order index=%d order=%s: %w", i, order.Hash.Hex(), err), nil
			}
			if len(rejects) > 0 {
				log.Info("[Processor][Native Lending] Orders rejected", "count", len(rejects))
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

func (p *VictionProcessor) isSupported() bool {
	if p == nil || p.config.Posv == nil || p.config.Viction == nil {
		return false
	}
	return true
}

// ----------------------------- Native Lending --------------------------------

func (p *VictionProcessor) LendingEngine() LendingEngine {
	return p.lendingEngine
}

func (p *VictionProcessor) SetLendingEngine(engine LendingEngine) {
	p.lendingEngine = engine
}

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

// ----------------------------- Native Trading --------------------------------

func (p *VictionProcessor) TradingEngine() TradingEngine {
	return p.tradingEngine
}

func (p *VictionProcessor) SetTradingEngine(engine TradingEngine) {
	p.tradingEngine = engine
}

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

// FeeProcessor applies zero-gas logic for sponsoring transactions.
type FeeProcessor struct {
	config           *params.ChainConfig // Chain configuration
	originalBalances types.BalanceMap    // Capacities snapshot during initialization of FeeProcessor.
	updatedBalances  types.BalanceMap    // Capacities changed during block processing.
	totalChange      *big.Int            // Sum of fee incurred during block processing.
}

// Return new FeeProcessor instance.
func NewFeeProcessor(config *params.ChainConfig, statedb *state.StateDB, blockNum *big.Int) *FeeProcessor {
	p := &FeeProcessor{
		config:          config,
		updatedBalances: make(types.BalanceMap),
		totalChange:     new(big.Int),
	}
	if !config.IsAtlas(blockNum) && config.Viction != nil &&
		config.Viction.VRC25Contract != (common.Address{}) {
		p.originalBalances = statedb.VicGetZeroGasCapacities(config.Viction.VRC25Contract)
	}
	return p
}

// Return a deep copy of FeeProcessor.
func (p *FeeProcessor) Copy() *FeeProcessor {
	if p == nil {
		return nil
	}
	cp := &FeeProcessor{
		config:          p.config,
		updatedBalances: make(types.BalanceMap),
		totalChange:     new(big.Int),
	}
	if p.originalBalances != nil {
		cp.originalBalances = make(types.BalanceMap, len(p.originalBalances))
		for k, v := range p.originalBalances {
			if v != nil {
				cp.originalBalances[k] = new(big.Int).Set(v)
			}
		}
	}
	return cp
}

// Return original balances snapshot.
func (p *FeeProcessor) FeePool() types.BalanceMap {
	if p == nil {
		return nil
	}
	return p.originalBalances
}

// Process fee for given transaction.
func (p *FeeProcessor) Process(statedb *state.StateDB, blockNum *big.Int, tx *types.Transaction, from common.Address, usedGas uint64, failed bool) {
	if p == nil || p.originalBalances == nil || tx.To() == nil || p.config.IsAtlas(blockNum) {
		return
	}
	token := *tx.To()
	runningCap, ok := p.originalBalances[token]
	if !ok || runningCap == nil {
		return
	}
	vicCfg := p.config.Viction
	fee := new(big.Int).SetUint64(usedGas)
	if p.config.TIPTRC21FeeBlock != nil && blockNum.Cmp(p.config.TIPTRC21FeeBlock) > 0 &&
		vicCfg != nil && vicCfg.VRC25GasPrice != nil {
		fee = new(big.Int).Mul(fee, (*big.Int)(vicCfg.VRC25GasPrice))
	}
	if runningCap.Cmp(fee) > 0 {
		newCap := new(big.Int).Sub(runningCap, fee)
		p.originalBalances[token] = newCap
		p.updatedBalances[token] = newCap
		p.totalChange.Add(p.totalChange, fee)
		if failed {
			PayTxFeeUsingToken(statedb, from, token)
		}
	}
}

// Persist the `updatedBalances` and `totalChange` to database.
func (p *FeeProcessor) Flush(statedb *state.StateDB) {
	if p == nil || p.originalBalances == nil || len(p.updatedBalances) == 0 || p.config.Viction == nil {
		return
	}
	statedb.VicSetZeroGasCapacities(p.config.Viction.VRC25Contract, p.updatedBalances, p.totalChange)
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
	log.Info("[Processor][Native Lending] Flushed Trie to disk", "block", block.NumberU64(), "root", lendingRoot.Hex())
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
			log.Error("[Processor][Native Lending] Failed to commit Trie on shutdown", "root", root, "err", err)
		}
		lendingTrieDB.Dereference(root.(common.Hash))
	}
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
	log.Info("[Processor][Native Trading] Flushed Trie to disk", "block", block.NumberU64(), "root", tradingRoot.Hex())
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
	log.Info("[Processor][Native Trading] Flushed Trie to disk", "block", current, "root", tradingRoot.Hex())

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
			log.Error("[Processor][Native Trading] Failed to commit Trie on shutdown", "root", root, "err", err)
		}
		tradingTrieDB.Dereference(root.(common.Hash))
	}
}
