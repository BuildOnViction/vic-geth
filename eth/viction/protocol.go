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

package ethviction

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Constants to match up protocol versions and messages
const (
	eth63 = 63
	eth64 = 64
	eth65 = 65
)

// ProtocolVersions are the supported versions of the eth protocol (first is primary).
var ProtocolVersions = []uint{eth65, eth64, eth63}

type txPool interface {
	// Has returns an indicator whether txpool has a transaction
	// cached with the given hash.
	Has(hash common.Hash) bool

	// Get retrieves the transaction from local txpool with given
	// tx hash.
	Get(hash common.Hash) *types.Transaction

	// AddRemotes should add the given transactions to the pool.
	AddRemotes([]*types.Transaction) []error

	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending() (map[common.Address]types.Transactions, error)

	// SubscribeNewTxsEvent should return an event subscription of
	// NewTxsEvent and send events to the given channel.
	SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription
}

type orderPool interface {
	// AddRemotes should add the given transactions to the pool.
	AddRemotes([]*types.OrderTransaction) []error

	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending() (map[common.Address]types.OrderTransactions, error)

	// SubscribeTxPreEvent should return an event subscription of
	// TxPreEvent and send events to the given channel.
	SubscribeTxPreEvent(chan<- core.OrderTxPreEvent) event.Subscription
}

type lendingPool interface {
	// AddRemotes should add the given transactions to the pool.
	AddRemotes([]*types.LendingTransaction) []error

	// Pending should return pending transactions.
	// The slice should be modifiable by the caller.
	Pending() (map[common.Address]types.LendingTransactions, error)

	// SubscribeTxPreEvent should return an event subscription of
	// TxPreEvent and send events to the given channel.
	SubscribeTxPreEvent(chan<- core.LendingTxPreEvent) event.Subscription
}
