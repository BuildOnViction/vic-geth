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

package rawdb

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// posvFinalizedBlockKey tracks the hash of the highest block finalized by PoSV BlockSigner votes.
	posvFinalizedBlockKey = []byte("PosvFinalizedBlock")

	// posvSignedBlockPrefix + signer address + block number (uint64 big endian) -> hash of the block the signer signed at that number
	posvSignedBlockPrefix = []byte("PosvSigned")
)

// Return the hash of the highest PoSV-finalized block, or the zero hash when none is stored.
func ReadPosvFinalizedBlockHash(db ethdb.KeyValueReader) common.Hash {
	data, _ := db.Get(posvFinalizedBlockKey)
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// Store the hash of the highest PoSV-finalized block.
func WritePosvFinalizedBlockHash(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Put(posvFinalizedBlockKey, hash.Bytes()); err != nil {
		log.Crit("Failed to store PoSV finalized block hash", "err", err)
	}
}

// Remove the stored PoSV-finalized block hash.
func DeletePosvFinalizedBlockHash(db ethdb.KeyValueWriter) {
	if err := db.Delete(posvFinalizedBlockKey); err != nil {
		log.Crit("Failed to delete PoSV finalized block hash", "err", err)
	}
}

// Return the hash of the block signer signed at number, or the zero hash when it signed none.
func ReadPosvSignedBlockHash(db ethdb.KeyValueReader, signer common.Address, number uint64) common.Hash {
	data, _ := db.Get(posvSignedBlockKey(signer, number))
	if len(data) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(data)
}

// Store the hash of the block signer signed at number.
func WritePosvSignedBlockHash(db ethdb.KeyValueWriter, signer common.Address, number uint64, hash common.Hash) {
	if err := db.Put(posvSignedBlockKey(signer, number), hash.Bytes()); err != nil {
		log.Crit("Failed to store PoSV signed block hash", "err", err)
	}
}

// Remove the signed-block records of signer below number.
func DeletePosvSignedBlockHashesBelow(db ethdb.KeyValueStore, signer common.Address, number uint64) {
	prefix := append(append([]byte{}, posvSignedBlockPrefix...), signer.Bytes()...)
	it := db.NewIterator(prefix, nil)
	defer it.Release()

	batch := db.NewBatch()
	for it.Next() {
		key := it.Key()
		if len(key) != len(prefix)+8 {
			continue
		}
		// Keys sort by big-endian block number, so the first one at or above number ends the stale range.
		if binary.BigEndian.Uint64(key[len(prefix):]) >= number {
			break
		}
		if err := batch.Delete(key); err != nil {
			log.Crit("Failed to delete PoSV signed block hash", "err", err)
		}
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to delete PoSV signed block hashes", "err", err)
	}
}

// posvSignedBlockKey = posvSignedBlockPrefix + signer address + num (uint64 big endian)
func posvSignedBlockKey(signer common.Address, number uint64) []byte {
	key := append(append([]byte{}, posvSignedBlockPrefix...), signer.Bytes()...)
	return append(key, encodeBlockNumber(number)...)
}
