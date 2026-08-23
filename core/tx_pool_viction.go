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

// --- Errors ---

// ErrDuplicateSpecialTransaction is returned when a special transaction
// (BlockSigner / Randomize) for the same nonce already exists in pending.
var ErrDuplicateSpecialTransaction = errors.New("duplicate special transaction")

// --- POSV pool fields ---
// These are set on TxPool after construction by the eth backend (eth/backend.go).
//
//   pool.IsSigner = func(addr) bool { ... }
//
// The IsSigner field is declared in the TxPool struct in tx_pool.go with the comment:
//   IsSigner func(common.Address) bool // [POSV] checks if address is a current validator

// --- Validation helpers ---

// validateBlacklistTx rejects transactions whose sender or receiver is
// blacklisted. Pool admission is local policy, and the legacy pool
// rejected these unconditionally.
func (pool *TxPool) validateBlacklistTx(tx *types.Transaction, from common.Address) error {
	if sender, receiver := misc.ValidateVictionBlackList(pool.chainconfig, from, tx.To()); sender || receiver {
		return ErrBlacklistedAddress
	}
	return nil
}

// validateSufficientTransaction checks that the sender has enough native balance
// to cover the transaction cost (value + gas*gasPrice).
// VRC25 fee sponsorship is NOT handled at txpool level for now.
// Called from validateTx in tx_pool.go.
func (pool *TxPool) validateSufficientTransaction(tx *types.Transaction, from common.Address) error {
	balance := pool.currentState.GetBalance(from)
	if balance.Cmp(tx.Cost()) < 0 {
		return ErrInsufficientFunds
	}
	return nil
}

// isPosvSpecialTx returns true if the pool is on a POSV chain and the
// transaction is a special transaction (BlockSigner or Randomize).
func (pool *TxPool) isPosvSpecialTx(tx *types.Transaction) bool {
	return pool.chainconfig.Posv != nil && tx.IsPosvTransaction()
}

// posvValidateGasPrice returns true if the transaction should be exempt from
// the minimum gas price check. Only special txs from known signers are exempt,
// as well as VRC25-sponsored zero-gasPrice transactions targeting registered tokens.
func (pool *TxPool) posvValidateGasPrice(tx *types.Transaction, from common.Address) bool {
	// Special POSV transactions (BlockSigner/Randomize) from signers.
	if pool.isPosvSpecialTx(tx) {
		if pool.IsSigner == nil {
			// If IsSigner is not configured, allow all special txs through
			// (they are validated at block-level anyway).
			return true
		}
		return pool.IsSigner(from)
	}

	// VRC25: zero-gasPrice tx to a registered token with sufficient capacity.
	if tx.GasPrice().Sign() == 0 && tx.To() != nil &&
		pool.chainconfig.Viction != nil &&
		pool.chainconfig.Viction.VRC25RegistryContract != (common.Address{}) {

		cap := pool.currentState.VicGetZeroGasCapacity(pool.chainconfig.Viction.VRC25RegistryContract, tx.To())
		if cap != nil && cap.Sign() > 0 {
			if err := ValidateSponsoredTx(pool.currentState, pool.chainconfig.Viction.VRC25RegistryContract, from, *tx.To(), tx.Data()); err == nil {
				return true
			}
		}
	}

	return false
}

// posvSkipIntrinsicGas returns true if intrinsic gas check should be skipped.
func (pool *TxPool) posvSkipIntrinsicGas(tx *types.Transaction) bool {
	return pool.isPosvSpecialTx(tx)
}

// posvTryPromoteSpecial attempts to fast-path promote a special transaction
// directly to pending. Returns (handled, replaced, err).
// If handled is false, the caller should proceed with normal add logic.
func (pool *TxPool) posvTryPromoteSpecial(tx *types.Transaction, from common.Address, local bool) (handled bool, replaced bool, err error) {
	if !pool.isPosvSpecialTx(tx) {
		return false, false, nil
	}
	log.Debug("[POSV TxPool] special tx detected", "hash", tx.Hash(), "from", from, "nonce", tx.Nonce(), "to", tx.To(), "isSignerNil", pool.IsSigner == nil)

	if pool.IsSigner == nil {
		return false, false, nil // fall through to normal path
	}
	isSigner := pool.IsSigner(from)
	pendingNonce := pool.pendingNonces.get(from)
	log.Debug("[POSV TxPool] special tx signer check", "hash", tx.Hash(), "from", from, "isSigner", isSigner, "pendingNonce", pendingNonce, "txNonce", tx.Nonce())

	if !isSigner || pendingNonce != tx.Nonce() {
		return false, false, nil // fall through to normal path
	}
	r, e := pool.promoteSpecialTx(from, tx, local)
	return true, r, e
}

// promoteSpecialTx directly inserts a special transaction (BlockSigner/Randomize)
// into the pending pool, bypassing the normal queue. This matches victionchain
// behavior where special txs from signers are fast-tracked.
//
// Note, this method assumes the pool lock is held!
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
	pool.beats[addr] = time.Now()
	pool.pendingNonces.set(addr, tx.Nonce()+1)

	if local {
		if !pool.locals.contains(addr) {
			log.Info("Setting new local account", "address", addr)
			pool.locals.add(addr)
		}
	}
	if local || pool.locals.contains(addr) {
		localGauge.Inc(1)
	}
	pool.journalTx(addr, tx)

	log.Debug("[POSV TxPool] promoted special tx to pending", "hash", tx.Hash(), "from", addr, "to", tx.To(), "nonce", tx.Nonce())
	go pool.txFeed.Send(NewTxsEvent{Txs: []*types.Transaction{tx}})
	return old != nil, nil
}
