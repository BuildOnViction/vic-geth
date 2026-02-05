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
	transferFuncHex     = common.Hex2Bytes("0xa9059cbb")
	transferFromFuncHex = common.Hex2Bytes("0x23b872dd")
)

var (
	ErrInvalidParams   = errors.New("invalid parameters")
	ErrInsufficientFee = errors.New("insufficient VRC25 token fee")
)

// we use vm.StateDB interface instead of *StateDB
func GetFeeCapacity(statedb vm.StateDB, vrc25Contract common.Address, addr *common.Address) *big.Int {
	if addr == nil {
		return nil
	}
	feeCapKey := state.GetStorageKeyForMapping(addr.Hash(), SlotVRC25Contract["tokensState"])
	feeCapHash := statedb.GetState(vrc25Contract, feeCapKey)
	return feeCapHash.Big()
}

// This function validate VRC25 transaction
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

	funcHex := data[:4]

	if balanceHash == (common.Hash{}) {
		if minFeeHash != (common.Hash{}) {
			return ErrInsufficientFee
		}
	} else {
		balance := balanceHash.Big()
		minFee := minFeeHash.Big()
		value := big.NewInt(0)

		if bytes.Equal(funcHex, transferFuncHex) && len(data) == 68 {
			value = common.BytesToHash(data[36:]).Big()
		} else {
			if bytes.Equal(funcHex, transferFromFuncHex) && len(data) == 80 {
				// Small fix here: only consider the value if 'from' matches
				if from.Hex() == common.BytesToAddress(data[4:36]).Hex() {
					value = common.BytesToHash(data[68:]).Big()
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
