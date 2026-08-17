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

// BalanceMap holds sponsoring capacities of VRC25 tokens.
type BalanceMap = map[common.Address]*big.Int

// FeeProcessor applies zero-gas logic for sponsoring transactions.
type FeeProcessor struct {
	config           *params.ChainConfig // Chain configuration
	originalBalances BalanceMap          // Capacities snapshot during initialization of FeeProcessor.
	updatedBalances  BalanceMap          // Capacities changed during block processing.
	totalChange      *big.Int            // Sum of fee incurred during block processing.
}

// Return new FeeProcessor instance.
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

// Return a deep copy of FeeProcessor.
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

// Return original balances snapshot.
func (p *FeeProcessor) FeePool() BalanceMap {
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
