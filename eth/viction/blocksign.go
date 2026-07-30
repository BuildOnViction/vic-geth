// Copyright 2014 The go-ethereum Authors
// (original work)
// Copyright 2025 The Viction Authors
// (modifications)
// This file is part of the go-ethereum library.
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
	"fmt"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Get block signers from the state.
func GetBlockSignData(
	config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader, bc *core.BlockChain,
) ([]types.Transaction, error) {
	if header == nil {
		return nil, fmt.Errorf("getblocksigndata: header is nil")
	}
	blockHash := header.Hash()
	blockNumber := header.Number
	block := chain.GetBlock(blockHash, blockNumber.Uint64())
	if block == nil {
		return nil, fmt.Errorf("getblocksigndata: block body not found (number=%d hash=%s)", blockNumber, blockHash)
	}
	data := []types.Transaction{}

	// Block-sign txs are EVM-executed and may fail.
	// Only successful signing txs count toward rewards and penalties.
	//
	// On post-Byzantium receipt format, `Receipt.Status` is the correct source
	// of success/failure. Using `len(PostState)` is unreliable and can misclassify.
	var receipts types.Receipts
	if config != nil {
		receipts = bc.GetReceiptsByHash(blockHash)
	}
	txs := block.Transactions()
	if receipts != nil && len(receipts) != len(txs) {
		return nil, fmt.Errorf(
			"getblocksigndata: receipts/tx count mismatch (number=%d hash=%s txs=%d receipts=%d)",
			blockNumber.Uint64(), blockHash, len(txs), len(receipts),
		)
	}

	for i, tx := range txs {
		if !tx.IsSigningTransaction(vicConfig.ValidatorBlockSignContract) {
			continue
		}
		if receipts != nil && i < len(receipts) {
			r := receipts[i]
			var status uint64
			if len(r.PostState) > 0 {
				status = types.ReceiptStatusSuccessful
			} else {
				status = r.Status
			}
			if status == types.ReceiptStatusFailed {
				continue
			}
		}
		data = append(data, *tx)
	}
	return data, nil
}
