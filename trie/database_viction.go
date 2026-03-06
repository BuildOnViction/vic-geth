package trie

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// Lock exposes the internal RWMutex for legacy TomoX trie operations.
// This is intentionally exposed for backward compatibility with the legacy
// trading state trie that needs direct lock access during preimage writes.
var _ = (*Database)(nil)

// LockAccessor provides access to the internal RWMutex of the trie database.
type LockAccessor struct {
	db *Database
}

// Lock is a public accessor for the trie database's internal RWMutex.
// Used by legacy/tomox/tradingstate/tomox_trie.go for preimage operations.
type DatabaseLock struct {
	sync.RWMutex
}

// GetLock returns a reference to the Database's internal lock.
func (db *Database) GetLock() *sync.RWMutex {
	return &db.lock
}

// InsertPreimage is a public wrapper around the internal insertPreimage method.
// Used by legacy/tomox/tradingstate/tomox_trie.go to write preimages.
func (db *Database) InsertPreimage(hash common.Hash, preimage []byte) {
	db.insertPreimage(hash, preimage)
}

// Preimage is a public wrapper around the internal preimage method.
// Used by legacy/tomox/tradingstate/tomox_trie.go to read preimages.
func (db *Database) Preimage(hash common.Hash) ([]byte, error) {
	p := db.preimage(hash)
	return p, nil
}
