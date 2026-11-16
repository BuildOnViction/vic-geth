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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// posvFinalizedBlockKey tracks the hash of the highest block finalized by PoSV BlockSigner votes.
var posvFinalizedBlockKey = []byte("PosvFinalizedBlock")

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
