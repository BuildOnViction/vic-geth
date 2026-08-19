// Copyright 2014 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/core/state"
)

// Try to process ZeroGas pre-transaction sequence.
func (st *StateTransition) buyGasZG() (bool, error) {
	if !st.isZeroGasSupported() {
		return false, nil
	}

	number := st.evm.Context.BlockNumber
	chainConfig := st.evm.ChainConfig()
	victionConfig := chainConfig.Viction

	// Atlas: ZeroGas supports buy/refund as normal transaction.
	if chainConfig.IsAtlas(number) {
		statedb := st.state.(*state.StateDB)
		zgcap := statedb.VicGetZeroGasCapacity(victionConfig.VRC25Contract, st.msg.To())
		if zgcap == nil || zgcap.Sign() == 0 {
			return false, nil
		}
		gasPrice := (*big.Int)(victionConfig.VRC25GasPrice)
		mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), gasPrice)
		if zgcap.Cmp(mgval) <= 0 {
			return false, nil
		}
		st.gasPrice = gasPrice
		st.payer = victionConfig.VRC25Contract
		statedb.SubBalance(st.payer, mgval)
		return true, nil
	}

	// Pre-Atlas: ZeroGas use block level gas pool.
	zp := st.zp
	if st.msg.To() == nil || zp == nil {
		return false, nil
	}
	zgcap, ok := zp[*st.msg.To()]
	if !ok || zgcap == nil {
		return false, nil
	}
	gasPrice := (*big.Int)(victionConfig.TRC21NewGasPrice)
	if !chainConfig.IsTIPGasPrice(number) {
		gasPrice = (*big.Int)(victionConfig.TRC21GasPrice)
	}
	mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), gasPrice)
	if zgcap.Cmp(mgval) < 0 {
		return false, nil
	}
	st.gasPrice = gasPrice
	st.payer = victionConfig.VRC25Contract
	return true, nil
}

// Try to process ZeroGas post-transaction sequence.
func (st *StateTransition) refundGasZG(remaining *big.Int) bool {
	if !st.isZeroGasSupported() {
		return false
	}

	number := st.evm.Context.BlockNumber
	chainConfig := st.evm.ChainConfig()
	victionConfig := chainConfig.Viction
	statedb := st.state.(*state.StateDB)

	if st.isZeroGasTransaction() {
		// Atlas: Refund remaining gas for the payer.
		if chainConfig.IsAtlas(number) {
			addr := st.msg.To()
			zgCap := statedb.VicGetZeroGasCapacity(victionConfig.VRC25Contract, addr)
			if zgCap != nil {
				gasFee := new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), (*big.Int)(victionConfig.VRC25GasPrice))
				statedb.VicSetVrc25Balance(victionConfig.VRC25Contract, *addr, new(big.Int).Sub(zgCap, gasFee))
			}
			statedb.AddBalance(st.payer, remaining)
		}
		// Pre-Atlas: ZeroGas use block level gas pool.
		return true
	} else {
		// Atlas: Regular transaction is not refunded. This is to keep compatibility with old data.
		// AtlasRefresh: Regular transaction is refunded to the coinbase.
		if !chainConfig.IsAtlasRefresh(number) && chainConfig.IsAtlas(number) {
			return true
		}
	}
	return false
}

// Try to reward validator owner.
func (st *StateTransition) rewardValidatorOwner() bool {
	if !st.isZeroGasSupported() {
		return false
	}

	number := st.evm.Context.BlockNumber
	chainConfig := st.evm.ChainConfig()
	victionConfig := chainConfig.Viction
	statedb := st.state.(*state.StateDB)

	gasFee := new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice)
	// Atlas: Fee is enforced with fixed price and go to validator owner as Atlas requires TIPGasPrice.
	if st.isZeroGasTransaction() && chainConfig.IsAtlas(number) {
		gasFee = new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), (*big.Int)(victionConfig.VRC25GasPrice))
	}

	// TIPGasPrice: Fee goes to validator-owner.
	if chainConfig.IsTIPGasPrice(number) {
		owner := statedb.VicGetValidatorOwner(victionConfig.ValidatorContract, st.evm.Context.Coinbase)
		if owner != (common.Address{}) {
			statedb.AddBalance(owner, gasFee)
		}
		return true
	}

	// Pre-TIPGasPrice: Fee goes to the coinbase.
	return false
}

// Return true if the chain supports ZeroGas.
func (st *StateTransition) isZeroGasSupported() bool {
	victionConfig := st.evm.ChainConfig().Viction
	return victionConfig != nil && victionConfig.VRC25Contract != (common.Address{})
}

// Return true if the transaction is sponsored.
func (st *StateTransition) isZeroGasTransaction() bool {
	return st.payer != common.Address{} && st.payer != st.msg.From()
}
