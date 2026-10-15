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
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

// customTracer is a well-formed JavaScript tracer object that is not bundled.
const customTracer = "{step:function(){},fault:function(){},result:function(){return 1}}"

// rawCustomTracerConfigs are trace configs sent verbatim over JSON-RPC. Raw
// strings keep the JSON escapes intact so the server-side decoder resolves them.
var rawCustomTracerConfigs = []string{
	`{"tracer":""}`,
	`{"tracer":"call\u0054racer;1"}`,
	`{"tracer":"callTracer\u0000"}`,
	`{"tracer":"` + customTracer + `"}`,
}

func TestValidateTracerConfigAllowsDefaultAndBuiltin(t *testing.T) {
	str := func(s string) *string { return &s }

	allowed := []struct {
		name   string
		config *TraceConfig
	}{
		{"nil config", nil},
		{"nil tracer", &TraceConfig{}},
		{"call tracer", &TraceConfig{Tracer: str("callTracer")}},
		{"prestate tracer", &TraceConfig{Tracer: str("prestateTracer")}},
	}
	for _, tt := range allowed {
		if err := validateTracerConfig(tt.config); err != nil {
			t.Errorf("%s: have %v, want nil", tt.name, err)
		}
	}
}

func TestValidateTracerConfigRejectsCustomTracer(t *testing.T) {
	rejected := []string{
		"",
		"CallTracer",
		" callTracer",
		"callTracer;1",
		customTracer,
	}
	for _, tracer := range rejected {
		tracer := tracer
		if err := validateTracerConfig(&TraceConfig{Tracer: &tracer}); err != errCustomTracer {
			t.Errorf("tracer %q: have %v, want %v", tracer, err, errCustomTracer)
		}
	}
}

// TestTraceEntryPointsRejectCustomTracerWithoutBackend runs each entry point on
// an API with no backend, so any chain access before validation panics.
func TestTraceEntryPointsRejectCustomTracerWithoutBackend(t *testing.T) {
	api := &PrivateDebugAPI{}
	ctx := context.Background()
	block := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(1)})
	blob, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatalf("failed to encode block: %v", err)
	}
	latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)

	calls := []struct {
		name string
		call func(config *TraceConfig) error
	}{
		{"traceTx with nil message", func(config *TraceConfig) error {
			_, err := api.traceTx(ctx, nil, vm.BlockContext{}, nil, nil, config)
			return err
		}},
		{"TraceChain", func(config *TraceConfig) error {
			_, err := api.TraceChain(ctx, rpc.BlockNumber(0), rpc.BlockNumber(1), config)
			return err
		}},
		{"TraceTransaction", func(config *TraceConfig) error {
			_, err := api.TraceTransaction(ctx, common.Hash{}, config)
			return err
		}},
		{"TraceCall", func(config *TraceConfig) error {
			_, err := api.TraceCall(ctx, ethapi.CallArgs{}, latest, config)
			return err
		}},
		{"traceBlock", func(config *TraceConfig) error {
			_, err := api.traceBlock(ctx, block, config)
			return err
		}},
		{"TraceBlock", func(config *TraceConfig) error {
			_, err := api.TraceBlock(ctx, blob, config)
			return err
		}},
	}
	for _, c := range calls {
		c := c
		t.Run(c.name, func(t *testing.T) {
			tracer := customTracer
			expectCustomTracerRejected(t, func() error { return c.call(&TraceConfig{Tracer: &tracer}) })
		})
	}
}

func TestTraceRPCMethodsRejectCustomTracer(t *testing.T) {
	client := newDebugTestClient(t)
	block := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(1)})
	blob, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatalf("failed to encode block: %v", err)
	}

	calls := []struct {
		method string
		args   []interface{}
	}{
		{"debug_traceBlock", []interface{}{blob}},
		{"debug_traceTransaction", []interface{}{common.Hash{}}},
		{"debug_traceCall", []interface{}{map[string]interface{}{}, "latest"}},
	}
	for _, c := range calls {
		for _, config := range rawCustomTracerConfigs {
			var result interface{}
			args := append(append([]interface{}{}, c.args...), json.RawMessage(config))
			err := client.Call(&result, c.method, args...)
			if err == nil || !strings.Contains(err.Error(), errCustomTracer.Error()) {
				t.Errorf("%s %s: have %v, want %v", c.method, config, err, errCustomTracer)
			}
		}
	}
}

func TestTraceChainSubscriptionRejectsCustomTracer(t *testing.T) {
	client := newDebugTestClient(t)

	for _, config := range rawCustomTracerConfigs {
		ch := make(chan interface{})
		sub, err := client.Subscribe(context.Background(), "debug", ch, "traceChain", "0x0", "0x1", json.RawMessage(config))
		if sub != nil {
			sub.Unsubscribe()
		}
		if err == nil || !strings.Contains(err.Error(), errCustomTracer.Error()) {
			t.Errorf("traceChain %s: have %v, want %v", config, err, errCustomTracer)
		}
	}
}

// expectCustomTracerRejected fails the test unless call returns errCustomTracer.
// A panic means the call touched the missing backend before validating.
func expectCustomTracerRejected(t *testing.T, call func() error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panicked before tracer rejection: %v", r)
		}
	}()
	if err := call(); err != errCustomTracer {
		t.Errorf("have %v, want %v", err, errCustomTracer)
	}
}

// newDebugTestClient serves a backend-free debug API over an in-process RPC
// connection.
func newDebugTestClient(t *testing.T) *rpc.Client {
	t.Helper()
	server := rpc.NewServer()
	if err := server.RegisterName("debug", &PrivateDebugAPI{}); err != nil {
		t.Fatalf("failed to register debug API: %v", err)
	}
	client := rpc.DialInProc(server)
	t.Cleanup(func() {
		client.Close()
		server.Stop()
	})
	return client
}
