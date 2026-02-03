// Copyright 2026 The Vic-geth Authors
package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

var slotTokensState = state.SlotVRC25Contract["tokensState"]

func (st *StateTransition) getFeeCapacity(contract *common.Address) *big.Int {
	if contract == nil {
		return nil
	}
	feeCapKey := state.GetStorageKeyForMapping(contract.Hash(), slotTokensState)
	feeCap := st.state.GetState(st.evm.ChainConfig().VRC25Contract, feeCapKey)

	if feeCap == (common.Hash{}) {
		return nil
	}

	return feeCap.Big()
}

func (st *StateTransition) setFeeCapacity(contract *common.Address, newFeeCap *big.Int) {
	if !st.isVRC25Transaction() || contract == nil {
		return
	}
	feeCapKey := state.GetStorageKeyForMapping(contract.Hash(), slotTokensState)
	st.state.SetState(st.evm.ChainConfig().VRC25Contract, feeCapKey, common.BigToHash(
		newFeeCap,
	))
}

// buyVRC25Gas checks sponsorship eligibility and deducts the gas fee from the sponsor's storage balance.
func (st *StateTransition) buyVRC25Gas() error {
	feeCap := st.getFeeCapacity(st.msg.To())

	if feeCap == nil {
		st.payer = st.msg.From()
		return nil
	}

	vrc25GasFee := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.evm.ChainConfig().VRC25GasPrice) // use VRC25 gas price to calculate fee

	if vrc25GasFee.Cmp(feeCap) > 0 {
		st.payer = st.msg.From()
		return nil
	}

	// vrc25 tx
	st.gasPrice = st.evm.ChainConfig().VRC25GasPrice // override gas price
	st.payer = st.evm.ChainConfig().VRC25Contract

	newFeeCap := new(big.Int).Sub(feeCap, vrc25GasFee)
	st.setFeeCapacity(st.msg.To(), newFeeCap)

	return nil
}

func (st *StateTransition) isVRC25Transaction() bool {
	return st.payer != st.msg.From()
}

func (st *StateTransition) refundVRC25Gas() {
}
