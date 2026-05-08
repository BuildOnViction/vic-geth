// Copyright 2026 The Vic-geth Authors
package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vrc25"
)

// vrc25BuyGas checks VRC25 sponsorship eligibility and adjusts payer/gasPrice.
func (st *StateTransition) vrc25BuyGas() error {
	st.payer = st.msg.From()

	victionConfig := st.evm.ChainConfig().Viction
	if victionConfig == nil || victionConfig.VRC25Contract == (common.Address{}) {
		return nil
	}

	blockNum := st.evm.Context.BlockNumber

	if !st.evm.ChainConfig().IsAtlas(blockNum) {
		// Pre-Atlas path: use the running activeFeeBalance map for eligibility.
		// This map is loaded from state at block start (beforeProcess) and
		// decremented after each VRC25 tx (afterApplyTransaction), ensuring
		// correct capacity tracking across multiple txs to the same token.
		if st.msg.To() == nil || activeFeeBalance == nil {
			return nil
		}
		feeCap, ok := activeFeeBalance[*st.msg.To()]
		if !ok || feeCap == nil {
			// Token not in the registered list — treat as regular VIC tx.
			return nil
		}

		var effectiveGasPrice *big.Int
		if st.evm.ChainConfig().TIPTRC21FeeBlock != nil && blockNum.Cmp(st.evm.ChainConfig().TIPTRC21FeeBlock) > 0 {
			effectiveGasPrice = (*big.Int)(victionConfig.VRC25GasPrice) // 250,000,000
		} else {
			effectiveGasPrice = (*big.Int)(victionConfig.TRC21GasPrice) // 2,500
		}

		mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), effectiveGasPrice)
		if feeCap.Cmp(mgval) < 0 {
			// Token is registered but capacity is insufficient — fall through
			// to regular VIC path.  buyGas will then check the sender's VIC
			// balance at the original gasPrice (which for VRC25 txs is
			// typically 0, so this effectively rejects the tx with
			// ErrInsufficientFunds
			return nil
		}
		// Set payer = VRC25Contract so isVRC25Transaction() returns true.
		// buyGas will skip the balance check and SubBalance for pre-Atlas sponsored txs.
		st.gasPrice = effectiveGasPrice
		st.payer = victionConfig.VRC25Contract
		return nil
	}

	// Post-Atlas: read capacity from statedb (each tx writes back via vrc25RefundGas).
	feeCap := vrc25.GetFeeCapacity(st.state, victionConfig.VRC25Contract, st.msg.To())
	if feeCap == nil || feeCap.Sign() == 0 {
		return nil
	}

	vrc25GasPrice := (*big.Int)(victionConfig.VRC25GasPrice)
	vrc25GasFee := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), vrc25GasPrice)
	if feeCap.Cmp(vrc25GasFee) <= 0 {
		return nil
	}

	st.gasPrice = vrc25GasPrice
	st.payer = victionConfig.VRC25Contract
	return nil
}

func (st *StateTransition) isVRC25Transaction() bool {
	return st.payer != st.msg.From()
}

// vrc25RefundGas handles gas refund for sponsored transactions.
func (st *StateTransition) vrc25RefundGas(remaining *big.Int) {
	if st.isVRC25Transaction() {
		blockNum := st.evm.Context.BlockNumber
		if !st.evm.ChainConfig().IsAtlas(blockNum) {
			// Pre-Atlas VRC25: buyGas was skipped entirely, nothing to refund.
			return
		}

		// Post-Atlas VRC25: deduct exactly gasUsed * price from the token's storage slot.
		addr := st.msg.To()
		victionConfig := st.evm.ChainConfig().Viction
		vrc25Contract := victionConfig.VRC25Contract
		feeCap := vrc25.GetFeeCapacity(st.state, vrc25Contract, addr)
		if feeCap != nil {
			gasUsedFee := new(big.Int).Mul(
				new(big.Int).SetUint64(st.gasUsed()),
				(*big.Int)(victionConfig.VRC25GasPrice),
			)
			vrc25.SetFeeCapacity(st.state, vrc25Contract, *addr, new(big.Int).Sub(feeCap, gasUsedFee))
		}
		// Refund remaining native balance to the VRC25 issuer contract.
		st.state.AddBalance(st.payer, remaining)
	}
	// Post-Atlas non-VRC25: no refund - remaining gas is burned.
}

// applyTransactionFee distributes the transaction fee to the correct recipient.
//
// After the TIPTRC21Fee fork the fee goes to the validator-owner stored on-chain
// inside VictionConfig.ValidatorContract. Before that fork, or when no owner is
// registered, the fee falls back to the block coinbase.
//
// When the Atlas fork is active and this is a VRC25-sponsored transaction the fee
// amount is re-derived using VictionConfig.VRC25GasPrice (which matches the price
// already used in buyGas / refundGas) instead of the regular gasPrice.
func (st *StateTransition) applyTransactionFee() {
	victionCfg := st.evm.ChainConfig().Viction
	blockNum := st.evm.Context.BlockNumber

	txFee := new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice)

	if victionCfg == nil {
		// Non-Viction chain: fee always goes to the coinbase.
		st.state.AddBalance(st.evm.Context.Coinbase, txFee)
		return
	}

	// After Atlas HF, VRC25-sponsored transactions carry a different gas price that
	// was set on st.gasPrice in vrc25BuyGas. However, if IsAtlas and we are a VRC25
	// transaction the gasPrice was already overridden to VRC25GasPrice, so txFee is
	// already correct. Explicitly recalculate only when VRC25GasPrice is set and the
	// current gasPrice could have been overridden (i.e., IsAtlas is active).
	if st.evm.ChainConfig().IsAtlas(blockNum) && st.isVRC25Transaction() && victionCfg.VRC25GasPrice != nil {
		txFee = new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), (*big.Int)(victionCfg.VRC25GasPrice))
	}

	// Before TIPTRC21Fee fork: fee goes to the block coinbase.
	chainCfg := st.evm.ChainConfig()
	if chainCfg.TIPTRC21FeeBlock == nil || blockNum.Cmp(chainCfg.TIPTRC21FeeBlock) <= 0 {
		st.state.AddBalance(st.evm.Context.Coinbase, txFee)
		return
	}

	// After TIPTRC21Fee fork: route fee to the registered owner of the validator.
	slot := state.StorageLocationOfValidatorOwner(st.evm.Context.Coinbase)
	ownerHash := st.state.GetState(victionCfg.ValidatorContract, slot)
	owner := common.BytesToAddress(ownerHash.Bytes())
	if owner != (common.Address{}) {
		st.state.AddBalance(owner, txFee)
	}
}
