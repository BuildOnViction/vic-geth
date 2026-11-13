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

package core

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// ErrDuplicateSpecialTransaction is returned when a special transaction for the same nonce already exists in pending.
var ErrDuplicateSpecialTransaction = errors.New("duplicate special transaction")

// posvValidateBlacklistTx rejects transactions whose sender or receiver is blacklisted.
func (pool *TxPool) posvValidateBlacklistTx(tx *types.Transaction, from common.Address) error {
	if sender, receiver := misc.ValidateVictionBlackList(pool.chainconfig, from, tx.To(), nil); sender || receiver {
		return ErrBlacklistedAddress
	}
	return nil
}

// Return true if the pool is on a POSV chain and the transaction is a special transaction.
func (pool *TxPool) isPosvSpecialTx(tx *types.Transaction) bool {
	return pool.chainconfig.Posv != nil && tx.IsPosvTransaction()
}

// posvExemptUnderpriced returns true if the transaction should be exempt from the minimum gas price check.
// Special transactions are exempt only when sent by a known signer; without a configured signer check no exemption is granted.
// ZeroGas transactions targeting registered tokens are exempt too.
func (pool *TxPool) posvExemptUnderpriced(tx *types.Transaction, from common.Address) bool {
	if pool.isPosvSpecialTx(tx) {
		return pool.IsSigner != nil && pool.IsSigner(from)
	}

	if tx.GasPrice().Sign() == 0 && tx.To() != nil &&
		pool.chainconfig.Viction != nil &&
		pool.chainconfig.Viction.VRC25RegistryContract != (common.Address{}) {

		feeCap := pool.currentState.VicGetZeroGasCapacity(pool.chainconfig.Viction.VRC25RegistryContract, tx.To())
		if feeCap != nil && feeCap.Sign() > 0 {
			if err := ValidateSponsoredTx(pool.currentState, pool.chainconfig.Viction.VRC25RegistryContract, from, *tx.To(), tx.Data()); err == nil {
				return true
			}
		}
	}

	return false
}

// Return true if Intrinsic Gas check should be skipped.
func (pool *TxPool) posvSkipIntrinsicGas(tx *types.Transaction) bool {
	return pool.isPosvSpecialTx(tx)
}

// Check and promote a special transaction directly to pending pool when it comes from a known signer at the account's pending nonce.
// Returns (handled, replaced, err):
//   - handled: the fast path took over the transaction; the caller in add() returns (replaced, err) as-is.
//     When false, the caller proceeds with its normal logic and the other two values carry no meaning.
//   - replaced: an already pending normal transaction at the same nonce was replaced.
//   - err: the error to fail the add with, e.g. ErrDuplicateSpecialTransaction.
//   - replaced and err only apply when handled is true.
func (pool *TxPool) posvTryPromoteSpecial(tx *types.Transaction, local bool) (handled bool, replaced bool, err error) {
	if !pool.isPosvSpecialTx(tx) {
		return false, false, nil
	}
	from, err := types.Sender(pool.signer, tx) // Duplicated sender recovery to keep geth logic intact.
	if err != nil {
		return true, false, ErrInvalidSender
	}
	if pool.IsSigner == nil || !pool.IsSigner(from) || pool.pendingNonces.get(from) != tx.Nonce() {
		// Use normal logic.
		return false, false, nil
	}

	replaced, err = pool.promoteSpecialTx(from, tx, local)
	return true, replaced, err
}

// Promote a special transaction directly to pending pool, bypassing the normal queue.
// Note: This method assumes the pool lock is held!
func (pool *TxPool) promoteSpecialTx(addr common.Address, tx *types.Transaction, local bool) (bool, error) {
	if pool.pending[addr] == nil {
		pool.pending[addr] = newTxList(true)
	}
	list := pool.pending[addr]
	old := list.txs.Get(tx.Nonce())
	if old != nil && old.IsPosvTransaction() {
		return false, ErrDuplicateSpecialTransaction
	}
	// Replace any existing normal tx at this nonce, or just insert
	if old != nil {
		pool.all.Remove(old.Hash())
		pool.priced.Removed(1)
		pendingReplaceMeter.Mark(1)
	} else {
		pendingGauge.Inc(1)
	}
	list.txs.Put(tx)

	if cost := tx.Cost(); list.costcap.Cmp(cost) < 0 {
		list.costcap = cost
	}
	if gas := tx.Gas(); list.gascap < gas {
		list.gascap = gas
	}
	pool.all.Add(tx)
	pool.priced.Put(tx)
	pool.beats[addr] = time.Now()
	pool.pendingNonces.set(addr, tx.Nonce()+1)

	if local {
		if !pool.locals.contains(addr) {
			log.Info("[TxPool] Setting new local account", "address", addr)
			pool.locals.add(addr)
		}
	}
	if local || pool.locals.contains(addr) {
		localGauge.Inc(1)
	}
	pool.journalTx(addr, tx)

	log.Trace("[TxPool] Promoted special transaction to pending", "hash", tx.Hash(), "from", addr, "to", tx.To(), "nonce", tx.Nonce())
	pool.queueTxEvent(tx)
	return old != nil, nil
}
