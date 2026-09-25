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
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
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

// Perform post transaction adjustment to match block import behavior.
func postTraceTx(statedb *state.StateDB, kind string, msg core.Message) {
	if kind == NativeTxNone {
		return
	}
	from := msg.From()
	statedb.SetNonce(from, statedb.GetNonce(from)-1)
}

// traceGasInfo is the minimal subset of a JS tracer's raw JSON result used to recover gas accounting.
type traceGasInfo struct {
	GasUsed *hexutil.Uint64 `json:"gasUsed"`
	Error   *string         `json:"error"`
}

// extractGasInfo derives (usedGas, failed) from a JS tracer's raw JSON.
func extractGasInfo(res json.RawMessage) (uint64, bool) {
	var info traceGasInfo
	if err := json.Unmarshal(res, &info); err != nil {
		return 0, false
	}
	var gas uint64
	if info.GasUsed != nil {
		gas = uint64(*info.GasUsed)
	}
	return gas, info.Error != nil && *info.Error != ""
}
