// Copyright 2014 The go-ethereum Authors
// (original work)
// Copyright 2025 The Viction Authors
// (modifications)// This file is part of the go-ethereum library.
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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// BalanceMap is the per-execution running map of VRC25 token fee capacities,
// keyed by token (transaction recipient). It is threaded explicitly through
// ApplyMessage as the StateTransition's feePool, so block import and each tracing
// re-execution own an independent instance and never share fee state. A nil pool
// means "no sponsorship" — every transaction is treated as a regular VIC tx.
type BalanceMap = map[common.Address]*big.Int

// FeeProcessor owns the pre-Atlas VRC25 fee accounting for a single
// execution over one block. It is a plain value type deliberately decoupled from
// StateProcessor (which needs the blockchain and consensus engine), so that the
// tracing API — which re-executes transactions with only a StateDB — can create
// and drive its own instance with the exact same seed/decrement/flush logic as
// block import. This is the single source of truth for the fee formula; block
// import and tracing share it and therefore cannot diverge.
type FeeProcessor struct {
	config           *params.ChainConfig
	originalBalances BalanceMap // running per-token capacity; nil = no sponsorship
	updatedBalances  BalanceMap // tokens charged this block -> final capacity (flush)
	totalChange      *big.Int   // total fee charged this block (flush)
}

// NewFeeProcessor builds the fee processor for a block. On pre-Atlas Viction
// blocks with a VRC25 issuer configured it snapshots all token fee capacities;
// otherwise feePool stays nil and every transaction is a regular VIC tx.
func NewFeeProcessor(config *params.ChainConfig, statedb *state.StateDB, blockNum *big.Int) *FeeProcessor {
	p := &FeeProcessor{
		config:          config,
		updatedBalances: make(BalanceMap),
		totalChange:     new(big.Int),
	}
	if !config.IsAtlas(blockNum) && config.Viction != nil &&
		config.Viction.VRC25Contract != (common.Address{}) {
		p.originalBalances = statedb.VicGetZeroGasCapacities(config.Viction.VRC25Contract)
	}
	return p
}

// Copy returns an independent copy of vp with the running fee pool deep-copied
// (keys and *big.Int values cloned) and fresh flush accumulators, or nil on a
// nil receiver. The parallel block tracer gives each task its own copy so worker
// goroutines never mutate a shared map; copies are never flushed.
func (p *FeeProcessor) Copy() *FeeProcessor {
	if p == nil {
		return nil
	}
	cp := &FeeProcessor{
		config:          p.config,
		updatedBalances: make(BalanceMap),
		totalChange:     new(big.Int),
	}
	if p.originalBalances != nil {
		cp.originalBalances = make(BalanceMap, len(p.originalBalances))
		for k, v := range p.originalBalances {
			if v != nil {
				cp.originalBalances[k] = new(big.Int).Set(v)
			}
		}
	}
	return cp
}

// FeePool returns the running fee-capacity map to thread into ApplyMessage.
// Safe on a nil receiver (returns nil).
func (p *FeeProcessor) FeePool() BalanceMap {
	if p == nil {
		return nil
	}
	return p.originalBalances
}

// HandleFee applies the pre-Atlas per-transaction fee accounting: it decrements
// the running pool for tx's token by the fee charged, records the update for the
// end-of-block flush, and for a failed sponsored transaction applies the same
// PayFeeWithVRC25 balance mutation as block import.
//
// No-op on a nil receiver, post-Atlas blocks, non-VRC25 transactions, contract
// creations, or when the pool has no entry for the token.
func (p *FeeProcessor) HandleFee(statedb *state.StateDB, blockNum *big.Int, tx *types.Transaction, from common.Address, usedGas uint64, failed bool) {
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

// Flush writes the accumulated pre-Atlas fee updates back to state. Called once
// per block after all transactions (block import only; tracing never flushes).
// No-op when nothing was charged.
func (p *FeeProcessor) Flush(statedb *state.StateDB) {
	if p == nil || p.originalBalances == nil || len(p.updatedBalances) == 0 || p.config.Viction == nil {
		return
	}
	statedb.VicSetZeroGasCapacities(p.config.Viction.VRC25Contract, p.updatedBalances, p.totalChange)
}
