// Copyright 2026 The go-ethereum Authors
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
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
)

func TestCallTracerHighGasLoop(t *testing.T) {
	tracer, err := New("callTracer")
	if err != nil {
		t.Fatal(err)
	}
	// Match the RPC trace timeout on a loop using 13 million gas.
	timer := time.AfterFunc(5*time.Second, func() {
		tracer.Stop(errors.New("execution timeout"))
	})
	t.Cleanup(func() { timer.Stop() })
	_, _, err = runtime.Execute(common.FromHex("0x6207a1205b60019003806004575000"), nil, &runtime.Config{
		GasLimit:  15000000,
		EVMConfig: vm.Config{Debug: true, Tracer: tracer},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := tracer.GetResult()
	if err != nil {
		t.Fatal(err)
	}
	var trace callTrace
	if err := json.Unmarshal(result, &trace); err != nil {
		t.Fatal(err)
	}
	if trace.Error != "" || len(trace.Calls) != 0 || trace.GasUsed == nil || uint64(*trace.GasUsed) != 13000005 {
		t.Fatalf("unexpected loop trace: %s", result)
	}
}
