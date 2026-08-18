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

// Try to buyGas with gas sponsoring on supported chain.
func (st *StateTransition) buyGasVrc25() error {
	st.payer = st.msg.From()

	chainConfig := st.evm.ChainConfig()
	victionConfig := chainConfig.Viction
	if victionConfig == nil || victionConfig.VRC25Contract == (common.Address{}) {
		return nil
	}

	number := st.evm.Context.BlockNumber
	if chainConfig.IsAtlas(number) {
		stateDb := st.state.(*state.StateDB)
		zgCap := stateDb.VicGetZeroGasCapacity(victionConfig.VRC25Contract, st.msg.To())
		if zgCap == nil || zgCap.Sign() == 0 {
			return nil
		}

		vrc25GasPrice := (*big.Int)(victionConfig.VRC25GasPrice)
		vrc25GasAmount := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), vrc25GasPrice)
		if zgCap.Cmp(vrc25GasAmount) <= 0 {
			return nil
		}

		st.gasPrice = vrc25GasPrice
		st.payer = victionConfig.VRC25Contract
		return nil
	}

	zgPool := st.zgPool
	if st.msg.To() == nil || zgPool == nil {
		return nil
	}
	zgCap, ok := zgPool[*st.msg.To()]
	if !ok || zgCap == nil {
		return nil
	}

	gasPrice := (*big.Int)(victionConfig.TRC21NewGasPrice)
	if !chainConfig.IsTIPGasPrice(number) {
		gasPrice = (*big.Int)(victionConfig.TRC21GasPrice)
	}

	gasAmount := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), gasPrice)
	if zgCap.Cmp(gasAmount) < 0 {
		return nil
	}
	st.gasPrice = gasPrice
	st.payer = victionConfig.VRC25Contract
	return nil
}

// refundGasVrc25 handles gas refund for sponsored transactions.
func (st *StateTransition) refundGasVrc25(remaining *big.Int) {
	number := st.evm.Context.BlockNumber
	chainConfig := st.evm.ChainConfig()
	victionConfig := chainConfig.Viction
	statedb := st.state.(*state.StateDB)

	if st.isZeroGasTransaction() {
		if !chainConfig.IsAtlas(number) {
			// PreAtlas ZG: buyGas was skipped, nothing to refund.
			return
		}

		// Atlas ZG: deduct exactly gasUsed * price from the token's storage slot.
		addr := st.msg.To()
		zgCap := statedb.VicGetZeroGasCapacity(victionConfig.VRC25Contract, addr)
		if zgCap != nil {
			gasFee := new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), (*big.Int)(victionConfig.VRC25GasPrice))
			statedb.VicSetVrc25Balance(victionConfig.VRC25Contract, *addr, new(big.Int).Sub(zgCap, gasFee))
		}
		// Refund remaining native balance to the VRC25 issuer contract.
		st.state.AddBalance(st.payer, remaining)
	} else if chainConfig.IsPostAtlas(number) {
		// PostAtlas: refund remaining gas to sender.
		st.state.AddBalance(st.msg.From(), remaining)
	}
	// Between Atlas and PostAtlas: no refund (gas burned).
}

func (st *StateTransition) isZeroGasTransaction() bool {
	return st.payer != st.msg.From()
}

// distributeFee distributes the transaction fee to the correct recipient.
func (st *StateTransition) distributeFee() {
	number := st.evm.Context.BlockNumber
	chainConfig := st.evm.ChainConfig()
	victionConfig := chainConfig.Viction

	gasFee := new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice)

	if victionConfig == nil {
		// Non-Viction blockchain: fee always goes to the coinbase.
		st.state.AddBalance(st.evm.Context.Coinbase, gasFee)
		return
	}

	// Atlas: Fee is enforced with fixed price and go to validator-owner as Atlas require TIPGasPrice.
	if st.isZeroGasTransaction() && chainConfig.IsAtlas(number) && victionConfig.VRC25GasPrice != nil {
		gasFee = new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), (*big.Int)(victionConfig.VRC25GasPrice))
	}

	// TIPGasPrice: fee goes to validator-owner.
	if chainConfig.IsTIPGasPrice(number) {
		slot := state.StorageLocationOfValidatorOwner(st.evm.Context.Coinbase)
		ownerHash := st.state.GetState(victionConfig.ValidatorContract, slot)
		owner := common.BytesToAddress(ownerHash.Bytes())
		if owner != (common.Address{}) {
			st.state.AddBalance(owner, gasFee)
		}
		return
	}

	// Pre-TIPGasPrice: fee goes to the coinbase.
	st.state.AddBalance(st.evm.Context.Coinbase, gasFee)
}
