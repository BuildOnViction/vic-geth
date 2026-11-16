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

package victionapi

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru"
)

var (
	errSignNonCanonical = errors.New("block is not on the canonical chain")
	errSignConflict     = errors.New("another block at this height was already signed")
)

// canonicalHashReader reads the canonical block hash at a height.
type canonicalHashReader interface {
	GetCanonicalHash(number uint64) common.Hash
}

// Serialize `BlockSign.sign` function call data.
func CreateBlockSignData(blockNumber *big.Int, blockHash common.Hash) []byte {
	data := common.Hex2Bytes("e341eaa4") // sign(uint256,bytes32)
	data = append(data, common.LeftPadBytes(blockNumber.Bytes(), 32)...)
	data = append(data, blockHash.Bytes()...)
	return data
}

// Create `BlockSign.sign` transaction.
func CreateBlockSignTransaction(nonce uint64, contractAddr common.Address, blockNumber *big.Int, blockHash common.Hash) *types.Transaction {
	data := CreateBlockSignData(blockNumber, blockHash)
	return types.NewTransaction(nonce, contractAddr, big.NewInt(0), 200000, big.NewInt(0), data)
}

// Get block signers for a given block from the state. Results are cached by block hash.
func GetBlockSignData(
	header *types.Header,
	config *params.ChainConfig, victionConfig *params.VictionConfig,
	chainReader consensus.ChainReader, blockchain *core.BlockChain, blksigCache *lru.ARCCache,
) ([]types.Transaction, error) {
	if header == nil {
		return nil, nil
	}
	blockHash := header.Hash()
	if cachedTxs, ok := blksigCache.Get(blockHash); ok {
		if signTxs, ok := cachedTxs.([]types.Transaction); ok && signTxs != nil {
			return signTxs, nil
		}
	}
	number := header.Number
	block := chainReader.GetBlock(blockHash, number.Uint64())
	if block == nil {
		return nil, fmt.Errorf("block %d (%s) body not found", number, blockHash)
	}

	signTxs := []types.Transaction{}
	// Successful signing transactions count toward rewards, penalties and finality, failed transactions don't.
	transactions := block.Transactions()
	receipts := blockchain.GetReceiptsByHash(blockHash)
	if len(transactions) != len(receipts) {
		return nil, fmt.Errorf("block %d (%s) has mismatched number of transactions and receipts", number.Uint64(), blockHash)
	}

	for i, tx := range transactions {
		if !tx.IsSigningTransaction(victionConfig.ValidatorBlockSignContract) {
			continue
		}
		if receipts != nil && i < len(receipts) && !receipts[i].IsSuccessful() {
			continue
		}
		signTxs = append(signTxs, *tx)
	}
	blksigCache.Add(blockHash, signTxs)
	return signTxs, nil
}

// Create new BlockSign transaction and submit it to TxPool.
func SubmitBlockSignTransaction(
	block *types.Block,
	snapshot *posv.Snapshot,
	config *params.ChainConfig,
	victionConfig *params.VictionConfig,
	blockchain *core.BlockChain,
	chainDb ethdb.Database,
	etherbase common.Address,
	wallet accounts.Wallet,
	txPool *core.TxPool,
) error {
	number := block.NumberU64()
	// Pre-TIP2019: Emit SignBlock transaction every block.
	// TIP2019: Emit SignBlock transaction every *victionConfig.ValidatorSignInterval* blocks.
	if config.IsTIP2019(block.Number()) && number%victionConfig.ValidatorSignInterval != 0 {
		return nil
	}
	if etherbase.IsZero() {
		return nil
	}
	if !IsEligibleValidator(etherbase, snapshot) {
		return nil
	}
	if err := checkBlockSignable(chainDb, blockchain, etherbase, block); err != nil {
		if errors.Is(err, errSignConflict) {
			log.Warn("[SignBlock] Refused to sign a block competing with a signed one", "number", number, "blockHash", block.Hash(), "etherbase", etherbase, "err", err)
		} else {
			log.Debug("[SignBlock] Skipped block off the canonical chain", "number", number, "blockHash", block.Hash())
		}
		return nil
	}

	nonce := NextNonce(etherbase, blockchain, txPool)
	pendingTxs, _ := txPool.Pending(false)
	expectedData := CreateBlockSignData(block.Number(), block.Hash())
	contractAddr := victionConfig.ValidatorBlockSignContract
	for _, transaction := range pendingTxs[etherbase] {
		if transaction.To() != nil && *transaction.To() == contractAddr && bytes.Equal(transaction.Data(), expectedData) {
			log.Info("[SignBlock] Skipped. BlockSign transaction already exists in pool", number, "blockHash", block.Hash(), "etherbase", etherbase, "nonce", transaction.Nonce())
			return nil
		}
	}

	// Record before submitting, so a failed or crashed submission can never be followed by a vote for a competing block.
	// Votes for blocks older than two epochs are not mined, so older records cannot matter.
	recordBlockSign(chainDb, etherbase, block, 2*config.Posv.Epoch)
	transaction := CreateBlockSignTransaction(nonce, victionConfig.ValidatorBlockSignContract, block.Number(), block.Hash())
	if err := SignAndSubmitTransaction(etherbase, wallet, transaction, config.ChainID, txPool, "SignBlock"); err != nil {
		return err
	}
	return nil
}

// Check that etherbase may sign block: the block is canonical and etherbase signed no other block at its height,
// so a validator never votes for two competing blocks.
func checkBlockSignable(db ethdb.KeyValueReader, chain canonicalHashReader, etherbase common.Address, block *types.Block) error {
	number := block.NumberU64()
	if chain.GetCanonicalHash(number) != block.Hash() {
		return errSignNonCanonical
	}
	if signed := rawdb.ReadPosvSignedBlockHash(db, etherbase, number); signed != (common.Hash{}) && signed != block.Hash() {
		return fmt.Errorf("%w: signed %x", errSignConflict, signed)
	}
	return nil
}

// Record that etherbase signs block, dropping its records more than retention blocks older.
func recordBlockSign(db ethdb.KeyValueStore, etherbase common.Address, block *types.Block, retention uint64) {
	number := block.NumberU64()
	rawdb.WritePosvSignedBlockHash(db, etherbase, number, block.Hash())
	if number > retention {
		rawdb.DeletePosvSignedBlockHashesBelow(db, etherbase, number-retention)
	}
}
