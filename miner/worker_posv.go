// Copyright 2026 The Vic-geth Authors
// POSV-specific miner helper functions.
// Extracted from worker.go to isolate consensus-specific logic.

package miner

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	posvWaitPeriod           = 10 * time.Second
	posvWaitPeriodCheckpoint = 20 * time.Second // longer wait per hop near epoch boundary (matches victionchain)
)

// turnCommitNewWorkWithPosv checks whether this node should seal the next block
// based on the round-robin validator schedule. Returns true if it's our turn
// (or we've waited long enough for out-of-turn fallback).
func (w *worker) turnCommitNewWorkWithPosv(parentHeader *types.Header) bool {
	// Pending block / snapshot updates must run even when the miner is stopped.
	if !w.isRunning() || w.chainConfig.Posv == nil {
		return true
	}

	c, engineOk := w.engine.(*posv.Posv)
	if !engineOk {
		log.Error("Chain has POSV config but consensus engine is not *posv.Posv")
		return false
	}
	checkPointHeader := posv.GetCheckpointHeader(w.chainConfig.Posv, parentHeader, w.chain, nil)
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkPointHeader)
	// IsMyTurn returns (inTurn, currentIndex, parentIndex, validatorCount, err).
	ok, myIdx, parentIdx, nValidators, err := c.IsMyTurn(w.coinbase, parentHeader, validators)
	if err != nil {
		log.Warn("[Posv] Failed to trying to commit new work", "err", err)
		return false
	}
	if !ok {
		log.Debug("Not in turn to commit new block, waiting")
		// Parent block author not in checkpoint list: only validators[0] is in-turn per IsMyTurn.
		if parentIdx == -1 {
			return false
		}
		// Our etherbase is not in the validator set.
		if myIdx == -1 {
			return false
		}
		// Hop counts forward steps from parent author to us on the validator ring.
		h := Hop(nValidators, parentIdx, myIdx)
		gap := posvWaitPeriod * time.Duration(h)
		// Near the next checkpoint, out-of-turn nodes wait longer so the in-turn validator
		// can seal the epoch block first (same rule as victionchain worker).
		epoch := w.chainConfig.Posv.Epoch
		if epoch > 0 {
			nearest := epoch - (parentHeader.Number.Uint64() % epoch)
			if uint64(h) >= nearest {
				log.Debug("Near-epoch out-of-turn wait", "parentNum", parentHeader.Number.Uint64(),
					"nearestBlocksToCheckpoint", nearest, "hops", h, "gap", posvWaitPeriodCheckpoint*time.Duration(h))
				gap = posvWaitPeriodCheckpoint * time.Duration(h)
			}
		}
		waited := time.Since(time.Unix(int64(parentHeader.Time), 0))
		if waited < 0 {
			waited = 0
		}
		log.Debug("Waiting for turn", "gap", gap, "hops", h, "waited", waited)
		if gap > waited {
			return false
		}
		log.Debug("Wait enough, sealing now", "waited", waited)
	}
	return true
}

// Hop returns how many validator slots to wait after parentIdx before myIdx may seal,
// for the round-robin order used with posvWaitPeriod / posvWaitPeriodCheckpoint.
// pre is parent block author's index, cur is ours.
func Hop(len, pre, cur int) int {
	switch {
	case pre < cur:
		return cur - (pre + 1)
	case pre > cur:
		return (len - pre) + (cur - 1)
	default:
		return len - 1
	}
}

// commitSpecialTransactions applies POSV special transactions (BlockSigner,
// Randomize) in input order. Returns true if processing was interrupted by a
// new head event.
func (w *worker) commitSpecialTransactions(txs types.Transactions, coinbase common.Address, interrupt *int32) bool {
	if !w.ensureGasPool() {
		log.Warn("[POSV commitSpecialTxs] gas pool not ready")
		return true
	}
	if len(txs) == 0 {
		return false
	}

	for _, tx := range txs {
		if stop, isNewHead := w.checkInterrupt(interrupt); stop {
			return isNewHead
		}
		if w.current.gasPool.Gas() < params.TxGas {
			log.Warn("Not enough gas for further special transactions", "have", w.current.gasPool, "want", params.TxGas)
			break
		}
		if tx == nil {
			continue
		}
		if tx.Protected() && !w.chainConfig.IsEIP155(w.current.header.Number) {
			log.Warn("Ignoring replay protected special transaction", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)
			continue
		}
		// Validate BlockSigner special tx payload and target block range.
		if tx.To() != nil && w.chainConfig.Viction != nil {
			if *tx.To() == w.chainConfig.Viction.ValidatorBlockSignContract {
				if len(tx.Data()) < 68 {
					log.Warn("Skipping special transaction with invalid BlockSigner payload length", "hash", tx.Hash(), "len", len(tx.Data()))
					continue
				}
				// ABI layout: selector(4) + uint256 blockNumber(32) + bytes32 blockHash(32)
				blkNumber := binary.BigEndian.Uint64(tx.Data()[28:36])
				curr := w.current.header.Number.Uint64()
				epochRange := w.chainConfig.Posv.Epoch * 2
				if w.chainConfig.Posv != nil && (blkNumber >= curr || (curr > epochRange && blkNumber <= curr-epochRange)) {
					log.Debug("Skipping special tx with invalid signed block number", "hash", tx.Hash(), "blkNumber", blkNumber, "current", curr, "epoch", w.chainConfig.Posv.Epoch)
					continue
				}
			}
		}
		from, _ := types.Sender(w.current.signer, tx)
		// Never seal a special transaction touching a blacklisted address
		// after the TIPBlacklist hardfork: importing nodes reject such a
		// block.
		if skip, _ := w.blacklistTxAction(tx, from); skip {
			log.Debug("Skipping special transaction with blacklisted party", "hash", tx.Hash(), "sender", from, "to", tx.To())
			continue
		}
		nonce := w.current.state.GetNonce(from)
		if nonce != tx.Nonce() {
			continue
		}
		w.current.state.Prepare(tx.Hash(), common.Hash{}, w.current.tcount)
		// All special txs go through commitTransaction which delegates to
		// core.ApplyTransaction — BlockSigner txs are handled without the
		// EVM, Randomize txs go through the normal EVM path.
		_, err := w.commitTransaction(tx, coinbase)
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			log.Warn("[POSV commitSpecialTxs] gas limit exceeded", "sender", from, "txHash", tx.Hash())
			return false
		case errors.Is(err, core.ErrNonceTooLow):
		case errors.Is(err, core.ErrNonceTooHigh):
		case errors.Is(err, nil):
			w.current.tcount++
		default:
			log.Warn("[POSV commitSpecialTxs] special tx failed", "hash", tx.Hash(), "sender", from, "err", err)
		}
	}

	if interrupt != nil {
		w.resubmitAdjustCh <- &intervalAdjust{inc: false}
	}
	return false
}

// blacklistTxAction reports whether tx must be excluded from the sealing
// block because its sender or receiver is blacklisted after the TIPBlacklist
// hardfork. When skip is true, pop tells the caller to drop the whole sending
// account — a blacklisted sender taints every queued transaction from that
// account — while pop=false means only this transaction is bad (blacklisted
// receiver) and the sender's next transaction may still be minable.
//
// It uses the same params API as the block-import check in
// VictionProcessor.BeforeApplyTransaction, so the miner can never seal a
// transaction the importing nodes would reject as blacklisted.
func (w *worker) blacklistTxAction(tx *types.Transaction, from common.Address) (skip, pop bool) {
	if !w.chainConfig.IsTIPBlacklist(w.current.header.Number) {
		return false, false
	}
	sender, receiver := core.BlacklistedTxParty(w.chainConfig, from, tx.To())
	if sender {
		return true, true
	}
	if receiver {
		return true, false
	}
	return false, false
}
