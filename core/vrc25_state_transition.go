// Copyright 2026 The Vic-geth Authors
package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

func (st *StateTransition) getFeeCapacity(contract *common.Address) *big.Int {
	if contract == nil {
		return nil
	}
	feeCapKey := state.GetStorageKeyForMapping(contract.Hash(), state.SlotVRC25Contract["tokensState"])
	feeCap := st.state.GetState(st.evm.ChainConfig().VRC25Contract, feeCapKey)

	if feeCap == (common.Hash{}) {
		return nil
	}
	
	return feeCap.Big()
}

func (st *StateTransition) applySponsoringTransaction() error {
	feeCap := st.getFeeCapacity(st.msg.To())

	if feeCap == nil {
		st.payer = st.msg.From()
		return nil
	}
	// TODO: implement sponsoring logic
	return nil
}

func (st *StateTransition) isSponsoringTransaction() bool {
	return st.payer != st.msg.From()
}

func (st *StateTransition) refundGasSponsoringTransaction() {
}
