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

// buyVRC25Gas checks sponsorship eligibility and deducts the gas fee from the sponsor's storage balance.
func (st *StateTransition) buyVRC25Gas() error {
	// Default payer is the sender
	st.payer = st.msg.From()

	// 1. Check if contract is sponsored (has fee capacity)
	feeCap := st.getFeeCapacity(st.msg.To())
	if feeCap == nil {
		return nil // Not sponsored, proceed with standard user payment
	}
	chainConfig := st.evm.ChainConfig()

	// 2. Calculate Gas Cost with VRC25 Gas Price
	vrc25GasFee := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), chainConfig.VRC25GasPrice)

	// 3. Check sufficiency
	if feeCap.Cmp(vrc25GasFee) < 0 {
		return nil // Insufficient sponsor balance, fallback to user payment
	}

	// 4. Deduct from Contract's Storage Balance
	// Note: The native ETH deduction happens in state_transition.go via st.state.SubBalance(st.payer)
	newFeeCap := new(big.Int).Sub(feeCap, vrc25GasFee)
	feeCapKey := state.GetStorageKeyForMapping(st.msg.To().Hash(), slotTokensState)
	st.state.SetState(chainConfig.VRC25Contract, feeCapKey, common.BigToHash(newFeeCap))

	// 5. Set Payer to System Contract
	// This ensures buyGas() deducts native ETH from the system contract
	st.gasPrice = chainConfig.VRC25GasPrice
	st.payer = chainConfig.VRC25Contract

	return nil
}

func (st *StateTransition) isVRC25Transaction() bool {
	return st.payer != st.msg.From()
}

func (st *StateTransition) refundGasVRC25() {
	// Calculate value to refund
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)

	// Get current balance
	feeCap := st.getFeeCapacity(st.msg.To())
	if feeCap == nil {
		// Should not happen if isSponsoringTransaction is true, but handle safely
		return
	}

	// Refund to Contract's Storage Balance
	newFeeCap := new(big.Int).Add(feeCap, remaining)
	feeCapKey := state.GetStorageKeyForMapping(st.msg.To().Hash(), slotTokensState)
	st.state.SetState(st.evm.ChainConfig().VRC25Contract, feeCapKey, common.BigToHash(newFeeCap))
}
