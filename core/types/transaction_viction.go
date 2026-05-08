// Copyright 2026 The Vic-geth Authors
package types

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
)

// signMethodSelector is the 4-byte function selector for sign(uint256,bytes32).
var signMethodSelector = common.Hex2Bytes("e341eaa4")

// IsTradingTransaction returns true if the tx is a TomoX order-matching batch (0x91).
// tomoXContract must come from ChainConfig.Viction.TomoXContract.
func (tx *Transaction) IsTradingTransaction(tomoXContract common.Address) bool {
	if tx.To() == nil {
		return false
	}
	return *tx.To() == tomoXContract
}

// IsLendingTransaction returns true if the tx is a TomoZ lending order-matching batch (0x93).
// lendingContract must come from ChainConfig.Viction.LendingContract.
func (tx *Transaction) IsLendingTransaction(lendingContract common.Address) bool {
	if tx.To() == nil {
		return false
	}
	return *tx.To() == lendingContract
}

// IsLendingFinalizedTradeTransaction returns true if the tx is a TomoZ finalized-trade commit (0x94).
// lendingFinalizedContract must come from ChainConfig.Viction.LendingFinalizedContract.
func (tx *Transaction) IsLendingFinalizedTradeTransaction(lendingFinalizedContract common.Address) bool {
	if tx.To() == nil {
		return false
	}
	return *tx.To() == lendingFinalizedContract
}

// IsSigningTransaction returns true if the transaction is a block-signer
// registration transaction to the BlockSigner contract.
// blockSignAddr is the ValidatorBlockSignContract address from chain config.
func (tx *Transaction) IsSigningTransaction(blockSignAddr common.Address) bool {
	if tx == nil || tx.To() == nil {
		return false
	}
	if *tx.To() != blockSignAddr {
		return false
	}
	data := tx.Data()
	// sign(uint256 blockNumber, bytes32 blockHash) = 4 + 32 + 32 = 68 bytes
	if len(data) != 68 {
		return false
	}
	return bytes.Equal(data[0:4], signMethodSelector)
}
