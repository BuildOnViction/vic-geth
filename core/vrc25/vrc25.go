// Copyright 2026 The Vic-geth Authors
package vrc25

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
)

var (
	SlotVRC25Contract = map[string]uint64{
		"minCap":      0,
		"tokens":      1,
		"tokensState": 2,
	}
	SlotVRC25Token = map[string]uint64{
		"balances": 0,
		"minFee":   1,
		"issuer":   2,
	}
	transferFunctionSelector     = common.Hex2Bytes("0xa9059cbb")
	transferFromFunctionSelector = common.Hex2Bytes("0x23b872dd")
)

var (
	ErrInvalidParams   = errors.New("invalid parameters")
	ErrInsufficientFee = errors.New("insufficient VRC25 token fee")
)

func PayFeeWithVRC25(statedb vm.StateDB, from common.Address, token common.Address) error {
	// 1. Check for valid statedb
	if statedb == nil {
		return ErrInvalidParams
	}
	// 2. Retrieve the balance of the from address for the VRC25 token
	slotBalances := SlotVRC25Token["balances"]                 // uint64 slot index
	balanceSlot := state.StorageLocationFromSlot(slotBalances) // StorageLocation (32-byte slot)
	balanceKey := state.StorageLocationOfMappingElement(balanceSlot, from.Hash().Bytes())
	balanceHash := statedb.GetState(token, balanceKey.Hash())

	if balanceHash != (common.Hash{}) {
		// 3. Check if balance is positive
		balance := balanceHash.Big()
		if balance.Sign() <= 0 {
			return nil
		}
		// 4. Retrieve the issuer address of the token
		issuerKey := state.StorageLocationFromSlot(SlotVRC25Token["issuer"])
		issuerHash := statedb.GetState(token, issuerKey.Hash())
		if issuerHash == (common.Hash{}) {
			return nil
		}
		issuerAddr := common.BytesToAddress(issuerHash.Bytes())

		// 5. Retrieve the minimum fee required by the token
		minFeeKey := state.StorageLocationFromSlot(SlotVRC25Token["minFee"])
		minFeeHash := statedb.GetState(token, minFeeKey.Hash())
		minFee := minFeeHash.Big()

		// 6. Determine the actual fee to charge (lesser of balance or minFee)
		feeUsed := new(big.Int).Set(minFee)
		if balance.Cmp(minFee) < 0 {
			feeUsed = balance
		}

		// 7. Deduct the fee from the user's balance and update state
		balance.Sub(balance, feeUsed)
		statedb.SetState(token, balanceKey.Hash(), common.BigToHash(balance))

		// 8. Add the fee to the issuer's balance and update state
		issuerBalanceKey := state.StorageLocationOfMappingElement(balanceSlot, issuerAddr.Hash().Bytes())
		issuerBalanceHash := statedb.GetState(token, issuerBalanceKey.Hash())
		issuerBalance := issuerBalanceHash.Big()
		newIssuerBalance := new(big.Int).Add(issuerBalance, feeUsed)
		statedb.SetState(token, issuerBalanceKey.Hash(), common.BigToHash(newIssuerBalance))
	}
	return nil
}

// UpdateFeeCapacity updates the fee capacity for VRC25 tokens in the VRC25 contract state.
// This is used to batch update fees at the end of block processing.
func UpdateFeeCapacity(statedb vm.StateDB, vrc25Contract common.Address, newBalance map[common.Address]*big.Int, totalFeeUsed *big.Int) {
	if statedb == nil || len(newBalance) == 0 {
		return
	}
	slotTokensState := SlotVRC25Contract["tokensState"]
	for token, value := range newBalance {
		balanceKey := state.StorageLocationOfMappingElement(state.StorageLocationFromSlot(slotTokensState), token.Hash().Bytes())
		statedb.SetState(vrc25Contract, balanceKey.Hash(), common.BigToHash(value))
	}
	statedb.SubBalance(vrc25Contract, totalFeeUsed)
}

// SetFeeCapacity writes a token's fee capacity into the VRC25 issuer contract's
// tokensState mapping. Called during post-Atlas gas purchase to deduct the
// pre-committed fee from the slot before EVM execution.
func SetFeeCapacity(statedb vm.StateDB, vrc25Contract common.Address, token common.Address, value *big.Int) {
	key := state.StorageLocationOfMappingElement(state.StorageLocationFromSlot(SlotVRC25Contract["tokensState"]), token.Hash().Bytes())
	statedb.SetState(vrc25Contract, key.Hash(), common.BigToHash(value))
}

// GetAllFeeCapacities reads the entire tokensState mapping from the VRC25 issuer
// contract and returns a snapshot of every registered token's fee capacity.
func GetAllFeeCapacities(statedb vm.StateDB, vrc25Contract common.Address) map[common.Address]*big.Int {
	result := map[common.Address]*big.Int{}

	// tokens is a dynamic array at slot 1; read its length first.
	tokensSlot := state.StorageLocationFromSlot(SlotVRC25Contract["tokens"])
	tokenCount := statedb.GetState(vrc25Contract, tokensSlot.Hash()).Big().Uint64()

	slotTokensState := SlotVRC25Contract["tokensState"]
	for i := uint64(0); i < tokenCount; i++ {
		// Each element of the dynamic array is stored at keccak256(slot) + i.
		elemKey := state.StorageLocationOfDynamicArrayElement(tokensSlot, i, 1)
		tokenHash := statedb.GetState(vrc25Contract, elemKey.Hash())
		if tokenHash == (common.Hash{}) {
			continue
		}
		token := common.BytesToAddress(tokenHash.Bytes())
		balanceKey := state.StorageLocationOfMappingElement(state.StorageLocationFromSlot(slotTokensState), token.Hash().Bytes())
		cap := statedb.GetState(vrc25Contract, balanceKey.Hash()).Big()
		result[token] = cap
	}
	return result
}

// we use vm.StateDB interface instead of *StateDB
func GetFeeCapacity(statedb vm.StateDB, vrc25Contract common.Address, addr *common.Address) *big.Int {
	if addr == nil {
		return nil
	}
	feeCapKey := state.StorageLocationOfMappingElement(state.StorageLocationFromSlot(SlotVRC25Contract["tokensState"]), addr.Hash().Bytes())
	feeCapHash := statedb.GetState(vrc25Contract, feeCapKey.Hash())
	return feeCapHash.Big()
}

// This function validates VRC25 transactions
// User's balance must be greater than or equal to the required fee
func ValidateVRC25Transaction(statedb vm.StateDB, vrc25Contract common.Address, from common.Address, to common.Address, data []byte) error {
	if data == nil || statedb == nil {
		return ErrInvalidParams
	}

	slotBalances := SlotVRC25Token["balances"]
	balanceKey := state.StorageLocationOfMappingElement(state.StorageLocationFromSlot(slotBalances), from.Hash().Bytes())
	balanceHash := statedb.GetState(to, balanceKey.Hash())
	minFeeSlot := SlotVRC25Token["minFee"]
	minFeeKey := state.StorageLocationFromSlot(minFeeSlot)
	minFeeHash := statedb.GetState(to, minFeeKey.Hash())

	if balanceHash == (common.Hash{}) {
		if minFeeHash != (common.Hash{}) {
			return ErrInsufficientFee
		}
	} else {
		balance := balanceHash.Big()
		minFee := minFeeHash.Big()
		value := big.NewInt(0)

		if len(data) > 4 {
			funcHex := data[:4]
			if bytes.Equal(funcHex, transferFunctionSelector) && len(data) == 68 {
				value = common.BytesToHash(data[36:]).Big()
			} else {
				if bytes.Equal(funcHex, transferFromFunctionSelector) && len(data) == 80 {
					// Small fix here: only consider the value if 'from' matches
					if from.Hex() == common.BytesToAddress(data[4:36]).Hex() {
						value = common.BytesToHash(data[68:]).Big()
					}
				}
			}

		}
		requiredFee := new(big.Int).Add(minFee, value)
		if balance.Cmp(requiredFee) < 0 {
			return ErrInsufficientFee
		}
	}

	return nil
}
