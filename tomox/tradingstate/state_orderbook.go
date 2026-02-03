// Copyright 2014 The go-ethereum Authors
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

package tradingstate

import "github.com/ethereum/go-ethereum/common"

// stateObject represents an Ethereum orderId which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// tradingExchangeObject values can be accessed and modified through the object.
// Finally, call CommitAskTrie to write the modified storage trie into a database.
type tradingExchanges struct {
	orderBookHash common.Hash
	data          tradingExchangeObject
	db            *TradingStateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by TradingStateDB.Commit.
	dbErr error

	// Write caches.
	asksTrie             Trie // storage trie, which becomes non-nil on first access
	bidsTrie             Trie // storage trie, which becomes non-nil on first access
	ordersTrie           Trie // storage trie, which becomes non-nil on first access
	liquidationPriceTrie Trie

	stateAskObjects      map[common.Hash]*stateOrderList
	stateAskObjectsDirty map[common.Hash]struct{}

	stateBidObjects      map[common.Hash]*stateOrderList
	stateBidObjectsDirty map[common.Hash]struct{}

	stateOrderObjects      map[common.Hash]*stateOrderItem
	stateOrderObjectsDirty map[common.Hash]struct{}

	liquidationPriceStates      map[common.Hash]*liquidationPriceState
	liquidationPriceStatesDirty map[common.Hash]struct{}

	onDirty func(hash common.Hash) // Callback method to mark a state object newly dirty
}
