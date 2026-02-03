// Copyright 2026 The Vic-geth Authors
package state

import (
	"github.com/ethereum/go-ethereum/common"
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

