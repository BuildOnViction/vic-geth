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

package tracers

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

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
