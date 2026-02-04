package transactions

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// BlockSignerAddress is the target address for Viction block signing transactions (0x89)
	BlockSignerAddress = common.HexToAddress("0x0000000000000000000000000000000000000089")
)

// These transactions do not execute EVM code but emit a standard log to be indexed.
func ProcessSignTransaction(state *state.StateDB, tx *types.Transaction, header *types.Header) (bool, *types.Receipt, error) {
	if tx.To() == nil || *tx.To() != BlockSignerAddress {
		return false, nil, nil
	}

	log.Debug("Processing specific transaction: SignTransaction", "hash", tx.Hash().Hex())

	// Define the event signature hash for "Sign(address,bytes32)" or equivalent
	signerLog := &types.Log{
		Address: BlockSignerAddress,
		Topics:  []common.Hash{common.BytesToHash([]byte("SignTransaction"))}, // Placeholder topic
		// Data:    tx.Data(), // Usually contains the signature data
		BlockNumber: header.Number.Uint64(),
		TxHash:      tx.Hash(),
		TxIndex:     0, // Will be set by core
		BlockHash:   header.Hash(),
		Index:       0, // Will be set by core
	}

	receipt := types.NewReceipt(nil, false, 21000) // 21000 gas used (basic)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 21000
	receipt.Logs = []*types.Log{signerLog}
	receipt.Status = types.ReceiptStatusSuccessful

	return true, receipt, nil
}
