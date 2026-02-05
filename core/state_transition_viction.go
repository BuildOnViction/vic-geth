// Copyright 2026 The Vic-geth Authors
package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vrc25"
)

var slotTokensState = vrc25.SlotVRC25Contract["tokensState"]

// buyVRC25Gas checks sponsorship eligibility and deducts the gas fee from the sponsor's storage balance.
func (st *StateTransition) vrc25BuyGas() error {
	// Default payer is the sender
	st.payer = st.msg.From()

	// 1. Check if contract is sponsored (has fee capacity)
	feeCap := vrc25.GetFeeCapacity(st.state, st.evm.ChainConfig().Viction.VRC25Contract, st.msg.To())
	if feeCap == nil {
		return nil // Not sponsored, proceed with standard user payment
	}
	chainConfig := st.evm.ChainConfig().Viction

	// 2. Calculate Gas Cost with VRC25 Gas Price
	vrc25GasFee := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), (*big.Int)(chainConfig.VRC25GasPrice))

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
	st.gasPrice = (*big.Int)(chainConfig.VRC25GasPrice)
	st.payer = chainConfig.VRC25Contract

	return nil
}

func (st *StateTransition) isVRC25Transaction() bool {
	return st.payer != st.msg.From()
}

func (st *StateTransition) vrc25RefundGas(remaining *big.Int) {
	addr := st.msg.To()
	// Get current balance
	feeCap := vrc25.GetFeeCapacity(st.state, st.evm.ChainConfig().Viction.VRC25Contract, addr)
	if feeCap == nil {
		// Should not happen if isSponsoringTransaction is true, but handle safely
		return
	}

	// Refund to Contract's Storage Balance
	newFeeCap := new(big.Int).Add(feeCap, remaining)
	feeCapKey := state.GetStorageKeyForMapping(addr.Hash(), slotTokensState)
	st.state.SetState(st.evm.ChainConfig().Viction.VRC25Contract, feeCapKey, common.BigToHash(newFeeCap))
}
