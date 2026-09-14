// Copyright 2017 The go-ethereum Authors
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

package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/legacy/lending/lendingstate"
	"github.com/ethereum/go-ethereum/legacy/trading/tradingstate"
	"github.com/ethereum/go-ethereum/params"
)

// Markers for Viction native system transactions handled without the EVM.
// NativeTxNone means the transaction is a regular EVM transaction.
const (
	NativeTxNone             = ""
	NativeTxTrading          = "trading"          // 0x91
	NativeTxTradingState     = "tradingState"     // 0x92
	NativeTxLending          = "lending"          // 0x93
	NativeTxLendingFinalized = "lendingFinalized" // 0x94
)

// callFrame is the Go equivalent of the call object built by call_tracer.js's
// result()/finalize(), used to report Viction native system transactions that
// execute without the EVM and therefore have no trace.
type callFrame struct {
	Type    string         `json:"type"`
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Value   *hexutil.Big   `json:"value"`
	Gas     hexutil.Uint64 `json:"gas"`
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Input   hexutil.Bytes  `json:"input"`
	Output  hexutil.Bytes  `json:"output"`
	Time    string         `json:"time"`
}

// Return a call frame for the native transactions.
func newNativeTxCallFrame(from common.Address, tx *types.Transaction) callFrame {
	return callFrame{
		Type:    "CALL",
		From:    from,
		To:      *tx.To(),
		Value:   (*hexutil.Big)(tx.Value()),
		Gas:     hexutil.Uint64(tx.Gas()),
		GasUsed: 0,
		Input:   tx.Data(),
		Output:  []byte{},
		Time:    "0s",
	}
}

// Try to dertermine Viction native transactions.
func nativeTransactionKind(config *params.ChainConfig, tx *types.Transaction, header *types.Header) string {
	if config == nil || config.Posv == nil || config.Viction == nil || tx.To() == nil {
		return NativeTxNone
	}
	vicConfig := config.Viction

	// 0x91 — Trading order-matching batch.
	if tx.IsTradingTransaction(vicConfig.TradingContract) && config.IsNativeTradingEnabled(header.Number) {
		if _, err := tradingstate.DecodeTxMatchesBatch(tx.Data()); err == nil {
			return NativeTxTrading
		}
	}
	// 0x92 — Trading state root commit.
	if *tx.To() == vicConfig.TradingStateContract && config.IsNativeTradingEnabled(header.Number) {
		return NativeTxTradingState
	}
	// 0x93 — Lending order-matching batch.
	if tx.IsLendingTransaction(vicConfig.LendingContract) && config.IsNativeTradingEnabled(header.Number) {
		if _, err := lendingstate.DecodeTxLendingBatch(tx.Data()); err == nil {
			return NativeTxLending
		}
	}
	// 0x94 — Lending finalized trade.
	if tx.IsLendingFinalizedTradeTransaction(vicConfig.LendingFinalizedContract) && config.IsNativeTradingEnabled(header.Number) {
		return NativeTxLendingFinalized
	}
	return NativeTxNone
}

// Apply a Viction native system transaction without the EVM.
func (api *PrivateDebugAPI) applyNativeTransaction(block *types.Block, tx *types.Transaction, index int, statedb *state.StateDB) (bool, string, error) {
	config := api.eth.blockchain.Config()
	header := block.Header()
	kind := nativeTransactionKind(config, tx, header)
	if kind == NativeTxNone {
		return false, NativeTxNone, nil
	}
	statedb.Prepare(tx.Hash(), block.Hash(), index)

	switch kind {
	case NativeTxLending, NativeTxLendingFinalized, NativeTxTrading, NativeTxTradingState:
		finalizeNativeTxState(config, statedb, header.Number)
		addNativeTxLog(statedb, *tx.To(), header)
	}
	return true, kind, nil
}

// Finalize StateDB state for Viction native transactions.
func finalizeNativeTxState(config *params.ChainConfig, statedb *state.StateDB, number *big.Int) {
	if config.IsByzantium(number) {
		statedb.Finalise(true)
	} else {
		statedb.IntermediateRoot(config.IsEIP158(number))
	}
}

// Create log entry for Viction native transactions.
func addNativeTxLog(statedb *state.StateDB, address common.Address, header *types.Header) {
	statedb.AddLog(&types.Log{
		Address:     address,
		BlockNumber: header.Number.Uint64(),
	})
}
