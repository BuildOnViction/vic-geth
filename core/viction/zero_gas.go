// Copyright 2014 The go-ethereum Authors
// (original work)
// Copyright 2025 The Viction Authors
// (modifications)// This file is part of the go-ethereum library.
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

package viction

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

var (
	ErrInvalidParams   = errors.New("invalid parameters")
	ErrInsufficientFee = errors.New("insufficient VRC25 token fee")
)

// Pay Transaction fee using VRC25 token.
func PayTxFeeUsingToken(statedb *state.StateDB, from common.Address, token common.Address) error {
	if statedb == nil {
		return ErrInvalidParams
	}
	issuer := statedb.VicGetVrc25Issuer(token)
	if issuer == (common.Address{}) {
		return nil
	}
	minFee := statedb.VicGetVrc25MinFee(token)

	fromBalance := statedb.VicGetVrc25Balance(token, from)
	if fromBalance.Sign() <= 0 {
		return nil
	}
	feeUsed := new(big.Int).Set(minFee)
	if fromBalance.Cmp(minFee) < 0 {
		feeUsed = fromBalance
	}

	fromBalance.Sub(fromBalance, feeUsed)
	statedb.VicSetVrc25Balance(token, from, fromBalance)

	issuerBalance := statedb.VicGetVrc25Balance(token, issuer)
	issuerBalance.Add(issuerBalance, feeUsed)
	statedb.VicSetVrc25Balance(token, issuer, issuerBalance)

	return nil
}

// Check given transaction is valid for sponsoring fee.
func ValidateSponsoredTx(statedb *state.StateDB, token common.Address, from common.Address, to common.Address, data []byte) error {
	if statedb == nil || data == nil {
		return ErrInvalidParams
	}

	balance := statedb.VicGetVrc25Balance(token, from)
	minFee := statedb.VicGetVrc25MinFee(token)

	if balance.Cmp(common.Big0) == 0 {
		if minFee.Cmp(common.Big0) > 0 {
			return ErrInsufficientFee
		}
	} else {
		value := big.NewInt(0)
		if len(data) > 4 {
			funcHex := data[:4]
			if bytes.Equal(funcHex, common.Hex2Bytes(state.TransferMethodHex)) && len(data) == 68 {
				value = common.BytesToHash(data[36:]).Big()
			} else {
				if bytes.Equal(funcHex, common.Hex2Bytes(state.TransferFromMethodHex)) && len(data) == 80 {
					if from.Hex() == common.BytesToAddress(data[4:36]).Hex() {
						value = common.BytesToHash(data[68:]).Big()
					}
				}
			}

		}
		totalFee := new(big.Int).Add(minFee, value)
		if balance.Cmp(totalFee) < 0 {
			return ErrInsufficientFee
		}
	}

	return nil
}
