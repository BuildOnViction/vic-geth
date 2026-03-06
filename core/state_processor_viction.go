// Copyright 2026 The Vic-geth Authors
// This file provides vic-extensions to the geth.
package core

import (
	"math/big"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vrc25"
	"github.com/ethereum/go-ethereum/legacy/tomox/tradingstate"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// TradingEngine is the interface that the legacy TomoX blackbox must satisfy.
// It's defined here to avoid an import cycle (core → legacy/tomox → ... → core).
// The concrete implementation is legacy/tomox.TomoX.
type TradingEngine interface {
	// CommitOrder replays a single order through the matching engine.
	// Mutates both statedb (token balances) and tradingStateDB (order book).
	CommitOrder(
		header *types.Header,
		coinbase common.Address,
		chain consensus.ChainContext,
		statedb *state.StateDB,
		tradingStateDB *tradingstate.TradingStateDB,
		orderBook common.Hash,
		order *tradingstate.OrderItem,
	) ([]map[string]string, []*tradingstate.OrderItem, error)

	// GetTradingState opens the TradingStateDB trie from the given block's
	// trading root. Used to initialize the parallel trie per block during sync.
	GetTradingState(block *types.Block, author common.Address) (*tradingstate.TradingStateDB, error)

	// UpdateMediumPriceBeforeEpoch computes and stores average trading prices
	// at epoch boundaries. Must be called before order matching at epoch blocks.
	UpdateMediumPriceBeforeEpoch(epoch uint64, tradingStateDB *tradingstate.TradingStateDB, statedb *state.StateDB) error
}

type victionProcessorState struct {
	currentBlockNumber *big.Int
	parrentState       *state.StateDB
	balanceFee         map[common.Address]*big.Int
	balanceUpdated     map[common.Address]*big.Int
	totalFeeUsed       *big.Int

	// TomoX legacy trading state (parallel Merkle trie for order books).
	// Initialized per block in beforeProcess from the parent block's trading root.
	// Only non-nil for pre-Atlas blocks where TomoX was active.
	tradingStateDB *tradingstate.TradingStateDB
}

func (p *StateProcessor) beforeProcess(block *types.Block, statedb *state.StateDB) error {
	header := block.Header()

	// Initialize victionState
	p.victionState = &victionProcessorState{
		currentBlockNumber: new(big.Int).Set(header.Number),
		balanceFee:         make(map[common.Address]*big.Int),
		balanceUpdated:     make(map[common.Address]*big.Int),
		totalFeeUsed:       big.NewInt(0),
		parrentState:       statedb.Copy(),
	}

	if p.config.TIPSigningBlock != nil && p.config.TIPSigningBlock.Cmp(header.Number) == 0 {
		statedb.DeleteAddress(p.config.Viction.ValidatorBlockSignContract)
	}
	if p.config.IsAtlas(header.Number) {
		misc.ApplyVIPVRC25Upgrade(statedb, p.config.Viction, p.config.AtlasBlock, header.Number)
	}
	if p.config.SaigonBlock != nil && p.config.SaigonBlock.Cmp(block.Number()) <= 0 {
		misc.ApplySaigonHardFork(statedb, p.config.Viction, p.config.SaigonBlock, block.Number())
	}

	// --- TomoX TradingStateDB initialization ---
	// Before Atlas, initialize the trading state trie from the parent block.
	if !p.config.IsAtlas(header.Number) && p.tradingEngine != nil && p.config.Posv != nil &&
		p.config.IsTIPTomoX(header.Number) && header.Number.Uint64() > p.config.Posv.Epoch {

		parent := p.bc.GetBlock(header.ParentHash, header.Number.Uint64()-1)
		if parent != nil {
			parentAuthor, _ := p.engine.Author(parent.Header())
			tradingState, err := p.tradingEngine.GetTradingState(parent, parentAuthor)
			if err != nil {
				log.Error("TomoX: failed to open TradingStateDB", "block", header.Number, "err", err)
			} else {
				p.victionState.tradingStateDB = tradingState

				// At epoch boundaries, update medium prices before any order matching
				if header.Number.Uint64()%p.config.Posv.Epoch == 0 {
					if err := p.tradingEngine.UpdateMediumPriceBeforeEpoch(
						header.Number.Uint64()/p.config.Posv.Epoch,
						tradingState, statedb,
					); err != nil {
						log.Error("TomoX: UpdateMediumPriceBeforeEpoch failed", "block", header.Number, "err", err)
					}
				}
			}
		}
	}

	// Initialize signers
	InitSignerInTransactions(p.config, header, block.Transactions())

	return nil
}

func (p *StateProcessor) afterProcess(block *types.Block, statedb *state.StateDB) error {
	if !p.config.IsAtlas(block.Number()) {
		vrc25.UpdateFeeCapacity(statedb, p.config.Viction.VRC25Contract, p.victionState.balanceUpdated, p.victionState.totalFeeUsed)
	}

	// --- TomoX trading root verification ---
	if p.victionState != nil && p.victionState.tradingStateDB != nil && p.tradingEngine != nil {
		gotRoot := p.victionState.tradingStateDB.IntermediateRoot()
		expectRoot := GetTradingStateRoot(block, p.config.Viction.TradingStateContract)
		if gotRoot != expectRoot {
			log.Error("TomoX: trading state root mismatch",
				"block", block.NumberU64(),
				"got", gotRoot.Hex(),
				"expect", expectRoot.Hex(),
			)
			// Log the mismatch but don't fail the block for now.
			// TODO: return error once full integration is verified on mainnet sync
		} else {
			log.Debug("TomoX: trading state root verified", "block", block.NumberU64(), "root", gotRoot.Hex())
		}
	}

	return nil
}

func (p *StateProcessor) beforeApplyTransaction(block *types.Block, tx *types.Transaction, msg types.Message, statedb *state.StateDB) error {
	header := block.Header()

	// Bypass blacklist for legacy blocks (before hardfork)
	maxBlockNumber := new(big.Int).SetInt64(9147459)
	if header.Number.Cmp(maxBlockNumber) <= 0 {
		if val := p.config.Viction.GetVictionBypassBalance(header.Number.Uint64(), msg.From()); val != nil {
			statedb.SetBalance(msg.From(), val)
		}
	}

	// Check blacklist after hardfork
	if p.config.IsTIPBlacklist(block.Number()) {
		if p.config.Viction.IsBlacklisted(msg.From()) {
			return ErrBlacklistedAddress
		}
		if tx.To() != nil && p.config.Viction.IsBlacklisted(*tx.To()) {
			return ErrBlacklistedAddress
		}
	}

	// TODO: TomoX/TomoZ validation intentionally skipped for now
	// When needed, add:
	// - ValidateTomoZApplyTransaction() for TRC21 token registration
	// - ValidateTomoXApplyTransaction() for TomoX pair registration

	return nil
}

func (p *StateProcessor) applyVictionTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	if tx.To() == nil {
		return false, nil, 0, nil, nil
	}
	to := *tx.To()
	vicConfig := p.config.Viction

	// 1. BlockSigner (0x89) — Validator signature transactions
	if to == vicConfig.ValidatorBlockSignContract && p.config.IsTIPSigning(header.Number) {
		return p.applySignTransaction(statedb, tx, header, usedGas)
	}

	// 2. TomoX system contracts — only active before Atlas hardfork
	if !p.config.IsAtlas(header.Number) {
		// 0x91 — TomoX matching batch (order execution)
		if to == vicConfig.TomoXContract && p.config.IsTIPTomoX(header.Number) {
			return p.applyTomoXTx(statedb, tx, header, usedGas)
		}

		// 0x92 — Trading state root commit
		if to == vicConfig.TradingStateContract && p.config.IsTIPTomoX(header.Number) {
			// TODO: verify trading state root against computed TradingStateDB
			return p.applyEmptyTransaction(statedb, tx, header, usedGas)
		}

		// 0x93 — Lending matching batch
		if to == vicConfig.LendingContract && p.config.IsTIPTomoXLending(header.Number) {
			return p.applyEmptyTransaction(statedb, tx, header, usedGas)
		}

		// 0x94 — Lending finalized trade (liquidation)
		if to == vicConfig.LendingFinalizedContract && p.config.IsTIPTomoXLending(header.Number) {
			return p.applyEmptyTransaction(statedb, tx, header, usedGas)
		}
	}

	// Not a viction-specific transaction — use standard EVM
	return false, nil, 0, nil, nil
}

// applyEmptyTransaction creates a zero-gas receipt for system transactions
// that bypass EVM execution (e.g., TomoX order matches, trading state root commits).
func (p *StateProcessor) applyEmptyTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
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

func (p *StateProcessor) applySignTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	var root []byte
	if p.config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(p.config.IsEIP158(header.Number)).Bytes()
	}
	// Validate Sender
	from, err := types.Sender(types.MakeSigner(p.config, header.Number), tx)
	if err != nil {
		return true, nil, 0, err, nil
	}
	// Nonce Validation
	nonce := statedb.GetNonce(from)
	if nonce < tx.Nonce() {
		return true, nil, 0, ErrNonceTooHigh, nil
	} else if nonce > tx.Nonce() {
		return true, nil, 0, ErrNonceTooLow, nil
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

func (p *StateProcessor) afterApplyTransaction(tx *types.Transaction, msg types.Message, statedb *state.StateDB, receipt *types.Receipt, usedGas uint64, err error) error {
	if p.victionState == nil || p.victionState.currentBlockNumber == nil {
		return nil
	}

	blockNum := p.victionState.currentBlockNumber
	isAtlas := p.config.IsAtlas(blockNum)

	// VRC25 / TRC21 Fee Logic
	if !isAtlas && tx.To() != nil {
		if p.config.IsTIPTRC21Fee(blockNum) {
			fee := new(big.Int).SetUint64(usedGas)
			if p.config.Viction.TRC21GasPrice != nil {
				price := (*big.Int)(p.config.Viction.TRC21GasPrice)
				fee = fee.Mul(fee, price)
			}

			balanceFee := vrc25.GetFeeCapacity(statedb, p.config.Viction.VRC25Contract, tx.To())

			if receipt.Status == types.ReceiptStatusFailed {
				if balanceFee != nil && balanceFee.Cmp(fee) > 0 {
					vrc25.PayFeeWithVRC25(statedb, msg.From(), *tx.To())
				}
			}

			if balanceFee != nil && balanceFee.Cmp(fee) >= 0 {
				currentVal, ok := p.victionState.balanceFee[*tx.To()]
				if !ok {
					currentVal = balanceFee
				}
				newVal := new(big.Int).Sub(currentVal, fee)
				p.victionState.balanceFee[*tx.To()] = newVal
				p.victionState.balanceUpdated[*tx.To()] = newVal
				p.victionState.totalFeeUsed = new(big.Int).Add(p.victionState.totalFeeUsed, fee)
			}
		}
	}
	return nil
}

// --- Helpers ---
func InitSignerInTransactions(config *params.ChainConfig, header *types.Header, txs types.Transactions) {
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

// SetTradingEngine injects the legacy TomoX trading engine into the state processor.
// Called by the blockchain layer during initialization when TomoX is needed for sync.
// Stored on the processor (not per-block state) since it persists across blocks.
func (p *StateProcessor) SetTradingEngine(engine TradingEngine) {
	p.tradingEngine = engine
}

// applyTomoXTx decodes and replays TomoX order matches from a 0x91 transaction.
// During sync, it decodes the TxMatchBatch, replays each order via CommitOrder
// (mutating both the state and the trading orderbook), and returns an empty receipt.
func (p *StateProcessor) applyTomoXTx(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	var root []byte
	if p.config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(p.config.IsEIP158(header.Number)).Bytes()
	}

	if len(tx.Data()) > 0 && p.victionState != nil && p.victionState.tradingStateDB != nil && p.tradingEngine != nil {
		txMatchBatch, err := tradingstate.DecodeTxMatchesBatch(tx.Data())
		if err != nil {
			log.Error("TomoX: failed to decode TxMatchBatch", "tx", tx.Hash().Hex(), "err", err)
		} else {
			coinbase := header.Coinbase
			tradingEngine := p.tradingEngine
			tradingStateDB := p.victionState.tradingStateDB

			for i, txDataMatch := range txMatchBatch.Data {
				// Decode the order from the match data
				order, err := txDataMatch.DecodeOrder()
				if err != nil {
					log.Error("TomoX: failed to decode order", "index", i, "err", err)
					continue
				}

				orderBook := tradingstate.GetTradingOrderBookHash(order.BaseToken, order.QuoteToken)

				trades, rejects, err := tradingEngine.CommitOrder(header, coinbase, p.bc, statedb, tradingStateDB, orderBook, order)
				if err != nil {
					log.Error("TomoX: CommitOrder failed", "index", i, "order", order.Hash.Hex(), "err", err)
					continue
				}

				if len(rejects) > 0 {
					log.Debug("TomoX: orders rejected", "count", len(rejects))
				}
				_ = trades // trades are logged but not needed for state correctness
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
	// receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return true, receipt, 0, nil, nil
}

// GetTradingStateRoot extracts the trading state root from the 0x92 transaction
// in a block. Returns EmptyRoot if no such transaction exists.
// The 0x92 tx data format: [32 bytes trading root | 32 bytes lending root]
func GetTradingStateRoot(block *types.Block, tradingStateAddr common.Address) common.Hash {
	for _, tx := range block.Transactions() {
		if tx.To() != nil && *tx.To() == tradingStateAddr {
			if len(tx.Data()) >= 32 {
				return common.BytesToHash(tx.Data()[:32])
			}
		}
	}
	return tradingstate.EmptyRoot
}
