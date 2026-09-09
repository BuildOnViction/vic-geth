// Copyright 2018 The go-ethereum Authors
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

package trie

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// Lock exposes the internal RWMutex for legacy Trading trie operations.
// This is intentionally exposed for backward compatibility with the legacy
// trading state trie that needs direct lock access during preimage writes.
var _ = (*Database)(nil)

// LockAccessor provides access to the internal RWMutex of the trie database.
type LockAccessor struct {
	db *Database
}

// Lock is a public accessor for the trie database's internal RWMutex.
// Used by legacy/trading/tradingstate/trading_trie.go for preimage operations.
type DatabaseLock struct {
	sync.RWMutex
}

// GetLock returns a reference to the Database's internal lock.
func (db *Database) GetLock() *sync.RWMutex {
	return &db.lock
}

// InsertPreimage is a public wrapper around the internal insertPreimage method.
// Used by legacy/trading/tradingstate/trading_trie.go to write preimages.
func (db *Database) InsertPreimage(hash common.Hash, preimage []byte) {
	db.insertPreimage(hash, preimage)
}

// Preimage is a public wrapper around the internal preimage method.
// Used by legacy/trading/tradingstate/trading_trie.go to read preimages.
func (db *Database) Preimage(hash common.Hash) ([]byte, error) {
	p := db.preimage(hash)
	return p, nil
}

// Database returns the underlying trie database.
func (t *Trie) Database() *Database {
	return t.db
}
