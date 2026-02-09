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
	slotBalances := SlotVRC25Token["balances"]
	balanceKey := state.GetStorageKeyForMapping(from.Hash(), slotBalances)
	balanceHash := statedb.GetState(token, balanceKey)

	if balanceHash != (common.Hash{}) {
		// 3. Check if balance is positive
		balance := balanceHash.Big()
		if balance.Sign() <= 0 {
			return nil
		}
		feeUsed := big.NewInt(0)

		// 4. Retrieve the issuer address of the token
		issuerKey := state.GetStorageKeyForSlot(SlotVRC25Token["issuer"])
		issuerHash := statedb.GetState(token, issuerKey)
		if issuerHash == (common.Hash{}) {
			return nil
		}
		issuerAddr := common.BytesToAddress(issuerHash.Bytes())

		// 5. Retrieve the minimum fee required by the token
		minFeeKey := state.GetStorageKeyForSlot(SlotVRC25Token["minFee"])
		minFeeHash := statedb.GetState(token, minFeeKey)
		minFee := minFeeHash.Big()

		// 6. Determine the actual fee to charge (lesser of balance or minFee)
		if balance.Cmp(minFee) < 0 {
			feeUsed = balance
		} else {
			feeUsed = minFee
		}

		// 7. Deduct the fee from the user's balance and update state
		balance.Sub(balance, feeUsed)
		statedb.SetState(token, balanceKey, common.BigToHash(balance))

		// 8. Add the fee to the issuer's balance and update state
		issuerBalanceKey := state.GetStorageKeyForMapping(issuerAddr.Hash(), slotBalances)
		issuerBalanceHash := statedb.GetState(token, issuerBalanceKey)
		issuerBalance := issuerBalanceHash.Big()
		issuerBalance.Add(issuerBalance, feeUsed)
		statedb.SetState(token, issuerBalanceKey, common.BigToHash(issuerBalance))
	}
	return nil
}

// we use vm.StateDB interface instead of *StateDB
func GetFeeCapacity(statedb vm.StateDB, vrc25Contract common.Address, addr *common.Address) *big.Int {
	if addr == nil {
		return nil
	}
	feeCapKey := state.GetStorageKeyForMapping(addr.Hash(), SlotVRC25Contract["tokensState"])
	feeCapHash := statedb.GetState(vrc25Contract, feeCapKey)
	return feeCapHash.Big()
}

// This function validates VRC25 transactions
// User's balance must be greater than or equal to the required fee
func ValidateVRC25Transaction(statedb vm.StateDB, vrc25Contract common.Address, from common.Address, to common.Address, data []byte) error {
	if data == nil || statedb == nil {
		return ErrInvalidParams
	}

	slotBalances := SlotVRC25Token["balances"]
	balanceKey := state.GetStorageKeyForMapping(from.Hash(), slotBalances)
	balanceHash := statedb.GetState(to, balanceKey)
	minFeeSlot := SlotVRC25Token["minFee"]
	minFeeKey := state.GetStorageKeyForSlot(minFeeSlot)
	minFeeHash := statedb.GetState(to, minFeeKey)

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
