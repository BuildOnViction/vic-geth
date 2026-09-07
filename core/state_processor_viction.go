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
	"github.com/ethereum/go-ethereum/legacy/lending/lendingstate"
	"github.com/ethereum/go-ethereum/legacy/trading/tradingstate"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// VictionProcessor handles Viction-specific block and transaction logic.
type VictionProcessor struct {
	config *params.ChainConfig // Chain configuration
	chain  *BlockChain         // Canonical block chain, nil for tx-only consumers
	engine consensus.Engine    // Consensus engine, nil disables author-dependent paths

	blockNumber                *big.Int         // Current processing block number.
	zeroGasCapacities          types.BalanceMap // Remaining ZeroGas capacities.
	updatedZeroGasCapacities   types.BalanceMap // Updated ZeroGas capacities to be flushed to database.
	totalUsedZeroGasCapacities *big.Int         // Sum of capacities used during block processing.

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
	p := &VictionProcessor{
		config:      config,
		blockNumber: new(big.Int).Set(blockNum),
	}
	p.snapshotCapacities(statedb, blockNum)
	return p
}

// Return a deep copy of Processor.
func (p *VictionProcessor) Copy() *VictionProcessor {
	if p == nil {
		return nil
	}
	cp := &VictionProcessor{
		config:                     p.config,
		updatedZeroGasCapacities:   make(types.BalanceMap),
		totalUsedZeroGasCapacities: new(big.Int),
	}
	if p.blockNumber != nil {
		cp.blockNumber = new(big.Int).Set(p.blockNumber)
	}
	if p.zeroGasCapacities != nil {
		cp.zeroGasCapacities = make(types.BalanceMap, len(p.zeroGasCapacities))
		for k, v := range p.zeroGasCapacities {
			if v != nil {
				cp.zeroGasCapacities[k] = new(big.Int).Set(v)
			}
		}
	}
	return cp
}

// Return original balances snapshot.
func (p *VictionProcessor) ZeroGasPool() types.BalanceMap {
	if p == nil {
		return nil
	}
	return p.zeroGasCapacities
}

// Pre block processing: - Prepare internal state for block processing - Active Viction-specific hard forks.
func (p *VictionProcessor) PreBlockProcess(block *types.Block, statedb *state.StateDB) error {
	if !p.isSupported() {
		return nil
	}

	header := block.Header()
	p.blockNumber = new(big.Int).Set(header.Number)
	p.snapshotCapacities(statedb, header.Number)

	p.lendingStateDB = nil
	p.lendingCommittedRoot = common.Hash{}
	p.tradingStateDB = nil
	p.tradingCommittedRoot = common.Hash{}

	misc.ApplyPosvHardForks(statedb, p.config, p.config.Viction, header.Number)

	if p.config.IsNativeTradingEnabled(header.Number) && header.Number.Uint64() > p.config.Posv.Epoch {
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
			log.Info("[NativeLending] Epoch liquidation processed", "block", header.Number.Uint64())
		}
	}

	signer := types.MakeSigner(p.config, header.Number)
	types.CacheSigners(signer, block.Transactions())

	return nil
}

// Post block processing: - Commit Trading/Lending/ZeroGas journal to database.
func (p *VictionProcessor) PostBlockProcess(block *types.Block, statedb *state.StateDB) error {
	if !p.isSupported() {
		return nil
	}

	p.flushRemainCapacities(statedb)

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
		log.Debug("[NativeTrading] State root verified", "block", block.NumberU64(), "root", tradingRoot.Hex())
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
		log.Debug("[NativeLending] State root verified", "block", block.NumberU64(), "root", lendingRoot.Hex())
	}

	return nil
}

// Pre transaction processing: - Fix incorrect balances for early blocks. - Prohibit blacklisted addresses.
func (p *VictionProcessor) PreApplyTransaction(block *types.Block, tx *types.Transaction, msg types.Message, statedb *state.StateDB) error {
	if !p.isSupported() {
		return nil
	}

	header := block.Header()
	if header.Number.BitLen() <= 64 && header.Number.Uint64() <= 9147459 {
		if val := p.config.Viction.GetBypassBalance(header.Number.Uint64(), msg.From()); val != nil {
			statedb.SetBalance(msg.From(), val)
		}
	}
	if sender, receiver := misc.ValidateVictionBlackList(p.config, msg.From(), tx.To(), header.Number); sender || receiver {
		return ErrBlacklistedAddress
	}

	return nil
}

// Handle Viction own transactions without the EVM.
// Return (false, nil, 0, nil, nil) for regular transactions.
func (p *VictionProcessor) ApplyNativeTransaction(tx *types.Transaction, header *types.Header, statedb *state.StateDB, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	if !p.isSupported() || tx.To() == nil {
		return false, nil, 0, nil, nil
	}
	vicConfig := p.config.Viction

	// 0x89 — Block signing.
	if tx.IsSigningTransaction(vicConfig.ValidatorBlockSignContract) && p.config.IsTIPSigning(header.Number) {
		return p.applyBlockSigningTransaction(tx, header, statedb, usedGas)
	}

	// 0x91 — Trading order-matching batch.
	if tx.IsTradingTransaction(vicConfig.TradingContract) && p.config.IsNativeTradingEnabled(header.Number) {
		if batch, err := tradingstate.DecodeTxMatchesBatch(tx.Data()); err == nil {
			return p.applyTradingTransaction(tx, header, statedb, usedGas, batch)
		}
	}

	// 0x92 — Trading state root commit, verified in AfterBlockProcess.
	if *tx.To() == vicConfig.TradingStateContract && p.config.IsNativeTradingEnabled(header.Number) {
		return p.applyEmptyTransaction(tx, header, statedb, usedGas)
	}

	// 0x93 — Lending order-matching batch.
	if tx.IsLendingTransaction(vicConfig.LendingContract) && p.config.IsNativeTradingEnabled(header.Number) {
		if batch, err := lendingstate.DecodeTxLendingBatch(tx.Data()); err == nil {
			return p.applyLendingTransaction(tx, header, statedb, usedGas, batch)
		}
	}

	// 0x94 — Lending finalized trade.
	if tx.IsLendingFinalizedTradeTransaction(vicConfig.LendingFinalizedContract) && p.config.IsNativeTradingEnabled(header.Number) {
		return p.applyEmptyTransaction(tx, header, statedb, usedGas)
	}

	return false, nil, 0, nil, nil
}

// Post transaction processing: - Apply zero-gas.
func (p *VictionProcessor) PostApplyTransaction(tx *types.Transaction, msg types.Message, statedb *state.StateDB, usedGas uint64, failed bool) error {
	if !p.isSupported() {
		return nil
	}
	p.processZeroGas(statedb, tx, msg.From(), usedGas, failed)
	return nil
}

// Process block signing transaction (0x89).
func (p *VictionProcessor) applyBlockSigningTransaction(tx *types.Transaction, header *types.Header, statedb *state.StateDB, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
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
func (p *VictionProcessor) applyEmptyTransaction(tx *types.Transaction, header *types.Header, statedb *state.StateDB, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
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
func (p *VictionProcessor) applyTradingTransaction(tx *types.Transaction, header *types.Header, statedb *state.StateDB, usedGas *uint64, batch tradingstate.TxMatchBatch) (bool, *types.Receipt, uint64, error, *big.Int) {
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
			log.Warn("[NativeTrading] Failed to recover block author, using zero address", "err", err)
		}
		tradingEngine := p.tradingEngine
		tradingStateDB := p.tradingStateDB

		for i, txDataMatch := range batch.Data {
			order, err := txDataMatch.DecodeOrder()
			if err != nil {
				log.Warn("[NativeTrading] Failed to decode order, skipping", "index", i, "err", err)
				continue
			}

			orderBook := tradingstate.GetTradingOrderBookHash(order.BaseToken, order.QuoteToken)

			_, rejects, err := tradingEngine.CommitOrder(header, coinbase, p.chain, statedb, tradingStateDB, orderBook, order)
			if err != nil {
				return true, nil, 0, fmt.Errorf("native_trading: failed to commit order index=%d order=%s: %w", i, order.Hash.Hex(), err), nil
			}

			if len(rejects) > 0 {
				log.Info("[NativeTrading] Orders rejected", "count", len(rejects))
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
func (p *VictionProcessor) applyLendingTransaction(tx *types.Transaction, header *types.Header, statedb *state.StateDB, usedGas *uint64, batch lendingstate.TxLendingBatch) (bool, *types.Receipt, uint64, error, *big.Int) {
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
			log.Warn("[NativeLending] Failed to recover block author, using zero address", "err", err)
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
				log.Info("[NativeLending] Orders rejected", "count", len(rejects))
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

// Create a snapshot of ZeroGas capacities.
func (p *VictionProcessor) snapshotCapacities(statedb *state.StateDB, blockNum *big.Int) {
	p.zeroGasCapacities = nil
	// Pre-Atlas: ZeroGas use block level gas pool.
	if !p.config.IsAtlas(blockNum) && p.config.Viction != nil && p.config.Viction.VRC25RegistryContract != (common.Address{}) {
		p.zeroGasCapacities = statedb.VicGetZeroGasCapacities(p.config.Viction.VRC25RegistryContract)
	}
	p.updatedZeroGasCapacities = make(types.BalanceMap)
	p.totalUsedZeroGasCapacities = new(big.Int)
}

// Handle ZeroGas fee for sponsored transactions.
func (p *VictionProcessor) processZeroGas(statedb *state.StateDB, tx *types.Transaction, from common.Address, usedGas uint64, failed bool) {
	if tx.To() == nil || p.blockNumber == nil {
		return
	}
	token := *tx.To()
	vicConfig := p.config.Viction

	// Atlas: ZeroGas use transaction level gas pool.
	if p.config.IsAtlas(p.blockNumber) && failed {
		zgcap := statedb.VicGetZeroGasCapacity(vicConfig.VRC25RegistryContract, tx.To())
		gasFee := new(big.Int).Mul(new(big.Int).SetUint64(usedGas), (*big.Int)(vicConfig.VRC25GasPrice))
		if zgcap != nil && zgcap.Cmp(gasFee) > 0 {
			PayTxFeeUsingToken(statedb, from, token)
		}
		return
	}

	// Pre-Atlas: ZeroGas use block level gas pool.
	if p.zeroGasCapacities == nil {
		return
	}
	zhcap, ok := p.zeroGasCapacities[token]
	if !ok || zhcap == nil {
		return
	}
	gasFee := new(big.Int).SetUint64(usedGas)
	if p.config.TIPGasPriceBlock != nil && p.blockNumber.Cmp(p.config.TIPGasPriceBlock) > 0 && vicConfig != nil && vicConfig.TRC21NewGasPrice != nil {
		gasFee = new(big.Int).Mul(gasFee, (*big.Int)(vicConfig.TRC21NewGasPrice))
	}
	if zhcap.Cmp(gasFee) > 0 {
		newCap := new(big.Int).Sub(zhcap, gasFee)
		p.zeroGasCapacities[token] = newCap
		p.updatedZeroGasCapacities[token] = newCap
		p.totalUsedZeroGasCapacities.Add(p.totalUsedZeroGasCapacities, gasFee)
		if failed {
			PayTxFeeUsingToken(statedb, from, token)
		}
	}
}

// Persist remaining ZeroGas capacities to database.
func (p *VictionProcessor) flushRemainCapacities(statedb *state.StateDB) {
	if p.zeroGasCapacities == nil || len(p.updatedZeroGasCapacities) == 0 || p.config.Viction == nil {
		return
	}
	statedb.VicSetZeroGasCapacities(p.config.Viction.VRC25RegistryContract, p.updatedZeroGasCapacities, p.totalUsedZeroGasCapacities)
}
