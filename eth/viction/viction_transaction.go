package viction

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// signMethodSelector is the 4-byte function selector for sign(uint256,bytes32).
var signMethodSelector = common.Hex2Bytes("e341eaa4")

// IsSigningTransaction returns true if the transaction is a block-signer
// registration transaction to the BlockSigner contract.
// blockSignAddr is the ValidatorBlockSignContract address from chain config.
func IsSigningTransaction(tx *types.Transaction, blockSignAddr common.Address) bool {
	if tx.To() == nil {
		return false
	}
	if *tx.To() != blockSignAddr {
		return false
	}
	data := tx.Data()
	if len(data) < 4 {
		return false
	}
	if !bytes.Equal(data[0:4], signMethodSelector) {
		return false
	}
	// sign(uint256 blockNumber, bytes32 blockHash) = 4 + 32 + 32 = 68 bytes
	if len(data) != 68 {
		return false
	}
	return true
}
