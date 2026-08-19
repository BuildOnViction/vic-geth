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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
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

	misc.ApplyPosvHardForks(statedb, p.config, p.config.Viction, header.Number)

	signer := types.MakeSigner(p.config, header.Number)
	types.CacheSigners(signer, block.Transactions())

	return nil
}

// Post block processing: - Commit ZeroGas journal to database.
func (p *VictionProcessor) PostBlockProcess(block *types.Block, statedb *state.StateDB) error {
	if !p.isSupported() {
		return nil
	}

	p.flushRemainCapacities(statedb)

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
	if p.config.IsTIPBlacklist(block.Number()) {
		if sender, receiver := misc.ValidateVictionBlackList(p.config, msg.From(), tx.To()); sender || receiver {
			return ErrBlacklistedAddress
		}
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
	receipt.Logs = statedb.GetLogs(tx.Hash(), header.Hash())
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
	if !p.config.IsAtlas(blockNum) && p.config.Viction != nil && p.config.Viction.VRC25Contract != (common.Address{}) {
		p.zeroGasCapacities = statedb.VicGetZeroGasCapacities(p.config.Viction.VRC25Contract)
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
		zgcap := statedb.VicGetZeroGasCapacity(vicConfig.VRC25Contract, tx.To())
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
	statedb.VicSetZeroGasCapacities(p.config.Viction.VRC25Contract, p.updatedZeroGasCapacities, p.totalUsedZeroGasCapacities)
}
