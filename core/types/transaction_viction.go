package types

import "github.com/ethereum/go-ethereum/common"

// IsTomoXApplyTransaction returns true if the transaction is directed to the given TomoXContract address.
func (tx *Transaction) IsTomoXApplyTransaction(contractAddress common.Address) bool {
	if tx.To() == nil {
		return false
	}
	return *tx.To() == contractAddress
}

// IsSkipNonceTransaction returns true if the transaction is a system transaction that skips nonce checking.
func (tx *Transaction) IsSkipNonceTransaction(contractAddress common.Address) bool {
	return tx.IsTomoXApplyTransaction(contractAddress)
}

// IsSigningTransaction returns true if the transaction is a block signer registration tx.
func (tx *Transaction) IsSigningTransaction() bool {
	if tx.To() == nil {
		return false
	}
	// Signer contract address is 0x88
	return *tx.To() == common.HexToAddress("0x0000000000000000000000000000000000000088")
}

// IsTradingTransaction checks if a target address is one of the designated system exchange contracts
func IsTradingTransaction(to *common.Address) bool {
	if to == nil {
		return false
	}

	addr := *to
	return addr == common.HexToAddress("0x0000000000000000000000000000000000000091") || // TomoXContract
		addr == common.HexToAddress("0x0000000000000000000000000000000000000092") || // TradingState
		addr == common.HexToAddress("0x0000000000000000000000000000000000000093") || // Lending
		addr == common.HexToAddress("0x0000000000000000000000000000000000000094") // LendingFinalized
}
