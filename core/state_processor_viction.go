// Copyright 2026 The Vic-geth Authors
// This file provides vic-extensions to the geth.
package core

import (
	"math/big"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vrc25"
	"github.com/ethereum/go-ethereum/params"
)

type victionProcessorState struct {
	currentBlockNumber *big.Int
	parrentState       *state.StateDB
	balanceFee         map[common.Address]*big.Int
	balanceUpdated     map[common.Address]*big.Int
	totalFeeUsed       *big.Int
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

	// Initialize signers
	InitSignerInTransactions(p.config, header, block.Transactions())

	return nil
}

func (p *StateProcessor) afterProcess(block *types.Block, statedb *state.StateDB) error {
	if !p.config.IsAtlas(block.Number()) {
		vrc25.UpdateFeeCapacity(statedb, p.config.Viction.VRC25Contract, p.victionState.balanceUpdated, p.victionState.totalFeeUsed)
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
	// 1. BlockSigner (0x89) - Validator signature transactions
	if tx.To() != nil && *tx.To() == p.config.Viction.ValidatorBlockSignContract && p.config.IsTIPSigning(header.Number) {
		return p.applySignTransaction(statedb, tx, header, usedGas)
	}

	// TODO: TomoX/TomoZ/Lending transactions intentionally skipped for now
	// When needed, add checks for:
	// 2. TradingStateAddr (0x92) - TomoX state synchronization
	// 3. TomoXLendingAddress (0x93) - Lending protocol transactions
	// 4. TomoXLendingFinalizedTradeAddress (0x94) - Lending finalization
	// 5. TomoXContract (0x91) - Trading transactions
	// All would call applyEmptyTransaction()

	// Not a victionchain-specific transaction, use standard EVM
	return false, nil, 0, nil, nil
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
