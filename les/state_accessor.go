// Copyright 2021 The go-ethereum Authors
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

package les

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/light"
)

// stateAtBlock retrieves the state database associated with a certain block.
func (leth *LightEthereum) stateAtBlock(ctx context.Context, block *types.Block, reexec uint64) (*state.StateDB, error) {
	return light.NewState(ctx, block.Header(), leth.odr), nil
}

// stateAtTransaction returns the execution environment of a certain transaction.
func (leth *LightEthereum) stateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, types.BalanceMap, string, error) {
	// Short circuit if it's genesis block.
	if block.NumberU64() == 0 {
		return nil, vm.BlockContext{}, nil, nil, core.NativeTxNone, errors.New("no transaction in genesis")
	}
	// Create the parent state database
	parent, err := leth.blockchain.GetBlock(ctx, block.ParentHash(), block.NumberU64()-1)
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, core.NativeTxNone, err
	}
	statedb, err := leth.stateAtBlock(ctx, parent, reexec)
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, core.NativeTxNone, err
	}
	if txIndex == 0 && len(block.Transactions()) == 0 {
		return nil, vm.BlockContext{}, statedb, nil, core.NativeTxNone, nil
	}
	// Recompute transactions up to the target index.
	signer := types.MakeSigner(leth.blockchain.Config(), block.Number())
	vp := core.NewVictionProcessor(leth.blockchain.Config(), nil, nil).ForkAtBlock(statedb, block.Number())
	zp := vp.ZeroGasPool()
	for idx, tx := range block.Transactions() {
		// Assemble the transaction call message and return if the requested offset
		msg, _ := tx.AsMessage(signer, block.BaseFee())
		// Reproduce Viction per-tx pre-checks (balance override + blacklist) before execution, mirroring block import.
		if err := vp.PreApplyTransaction(block, tx, msg, statedb); err != nil {
			return nil, vm.BlockContext{}, nil, nil, core.NativeTxNone, err
		}
		txContext := core.NewEVMTxContext(msg)
		context := core.NewEVMBlockContext(block.Header(), leth.blockchain, nil)
		statedb.Prepare(tx.Hash(), idx)
		if idx == txIndex {
			kind := core.NativeTransactionKind(leth.blockchain.Config(), tx, block.Header())
			return msg, context, statedb, zp, kind, nil
		}
		// Not yet the searched for transaction, execute on top of the current state
		vmenv := vm.NewEVM(context, txContext, statedb, leth.blockchain.Config(), vm.Config{})
		res, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas()), zp)
		if err != nil {
			return nil, vm.BlockContext{}, nil, nil, core.NativeTxNone, fmt.Errorf("transaction %#x failed: %v", tx.Hash(), err)
		}
		core.PostTraceTx(statedb, core.NativeTransactionKind(leth.blockchain.Config(), tx, block.Header()), msg)
		if err := vp.PostApplyTransaction(tx, msg, statedb, res.UsedGas, res.Failed()); err != nil {
			return nil, vm.BlockContext{}, nil, nil, core.NativeTxNone, err
		}
		// Ensure any modifications are committed to the state
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		statedb.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
	}
	return nil, vm.BlockContext{}, nil, nil, core.NativeTxNone, fmt.Errorf("transaction index %d out of range for block %#x", txIndex, block.Hash())
}
