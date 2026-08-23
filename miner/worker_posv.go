// Copyright 2015 The go-ethereum Authors
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

package miner

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	posvWaitPeriod           = 10 * time.Second
	posvWaitPeriodCheckpoint = 20 * time.Second
)

// Attempt to commit new block for PoSV consensus. Returns true if it's our turn or we've waited long enough for out-of-turn fallback.
func (w *worker) commitNewWorkForPoSV(parentHeader *types.Header) bool {
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
		log.Warn("[Miner][PoSV] Failed to commit new work", "err", err)
		return false
	}
	if !ok {
		if parentIdx == -1 {
			return false
		}
		if myIdx == -1 {
			return false
		}
		h := common.CircularDistance(myIdx, parentIdx, nValidators) - 1
		gap := posvWaitPeriod * time.Duration(h)
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
		log.Info("[Miner][PoSV] Waiting for my turn", "gap", gap, "hops", h, "waited", waited)
		if gap > waited {
			return false
		}
		log.Info("[Miner][PoSV] Waited until timeout. Committing new work", "waited", waited)
	}
	return true
}

// Commit transactions required for PoSV consensus (Block Signing, Randomize). Returns true if the process is interrupted.
func (w *worker) commitPosvTransactions(txs types.Transactions, coinbase common.Address, interrupt *int32) bool {
	if !w.ensureGasPool() {
		log.Warn("[Miner][PoSV] Gas pool not ready")
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
			log.Warn("[Miner][PoSV] Not enough gas for further special transactions", "have", w.current.gasPool, "want", params.TxGas)
			break
		}
		if tx == nil {
			continue
		}
		if tx.Protected() && !w.chainConfig.IsEIP155(w.current.header.Number) {
			log.Warn("[Miner][PoSV] Ignoring replay protected special transaction", "hash", tx.Hash(), "eip155", w.chainConfig.EIP155Block)
			continue
		}
		if tx.To() != nil && w.chainConfig.Viction != nil {
			if *tx.To() == w.chainConfig.Viction.ValidatorBlockSignContract {
				if len(tx.Data()) < 68 {
					log.Warn("[Miner][PoSV] Invalid BlockSigner transaction: payload length is incorrect", "hash", tx.Hash(), "len", len(tx.Data()))
					continue
				}
				blkNumber := binary.BigEndian.Uint64(tx.Data()[28:36])
				curr := w.current.header.Number.Uint64()
				epochRange := w.chainConfig.Posv.Epoch * 2
				if w.chainConfig.Posv != nil && (blkNumber >= curr || (curr > epochRange && blkNumber <= curr-epochRange)) {
					log.Warn("[Miner][PoSV] Invalid BlockSigner transaction: block number is incorrect", "hash", tx.Hash(), "blkNumber", blkNumber, "current", curr, "epoch", w.chainConfig.Posv.Epoch)
					continue
				}
			}
		}
		from, _ := types.Sender(w.current.signer, tx)
		if skip, _ := w.EnforceBlacklist(tx, from); skip {
			log.Debug("[Miner][PoSV] Ignored blacklisted address", "hash", tx.Hash(), "sender", from, "to", tx.To())
			continue
		}
		nonce := w.current.state.GetNonce(from)
		if nonce != tx.Nonce() {
			continue
		}
		w.current.state.Prepare(tx.Hash(), common.Hash{}, w.current.tcount)
		_, err := w.commitTransaction(tx, coinbase)
		switch {
		case errors.Is(err, core.ErrGasLimitReached):
			log.Warn("[Miner][PoSV] Exceed gas limit", "sender", from, "txHash", tx.Hash())
			return false
		case errors.Is(err, core.ErrNonceTooLow):
		case errors.Is(err, core.ErrNonceTooHigh):
		case errors.Is(err, nil):
			w.current.tcount++
		default:
			log.Warn("[Miner][PoSV] Failed to process transaction", "hash", tx.Hash(), "sender", from, "err", err)
		}
	}

	if interrupt != nil {
		w.resubmitAdjustCh <- &intervalAdjust{inc: false}
	}
	return false
}

// Prevent blacklisted addresses to be used in the chain.
func (w *worker) EnforceBlacklist(tx *types.Transaction, from common.Address) (skip, pop bool) {
	sender, receiver := misc.ValidateVictionBlackList(w.chainConfig, from, tx.To(), nil)
	if sender {
		return true, true
	}
	if receiver {
		return true, false
	}
	return false, false
}
