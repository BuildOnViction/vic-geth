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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	defaultPosvWaitPeriod           = 10 * time.Second
	defaultPosvWaitPeriodCheckpoint = 20 * time.Second
)

// Attempt to commit new block for PoSV consensus. Returns true if it's our turn or we've waited long enough for out-of-turn fallback.
func (w *worker) preCommitNewWorkForPosv(parentHeader *types.Header) bool {
	// Pending block / snapshot updates must run even when the miner is stopped.
	if !w.isRunning() {
		return true
	}

	c := w.engine.(*posv.Posv)
	checkPointHeader := posv.GetCheckpointHeader(w.chainConfig.Posv, parentHeader, nil, w.chain)
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkPointHeader)
	// IsMyTurn returns (inTurn, currentIndex, parentIndex, validatorCount, err).
	ok, myIdx, parentIdx, nValidators, err := c.IsMyTurn(w.coinbase, parentHeader, validators)
	if err != nil {
		log.Warn("[Miner] Failed to commit new work", "err", err)
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
		gap := w.posvWaitPeriod * time.Duration(h)
		epoch := w.chainConfig.Posv.Epoch
		if epoch > 0 {
			nearest := epoch - (parentHeader.Number.Uint64() % epoch)
			if uint64(h) >= nearest {
				log.Debug("[Miner] Near-epoch out-of-turn wait", "parentNum", parentHeader.Number.Uint64(),
					"nearestBlocksToCheckpoint", nearest, "hops", h, "gap", w.posvWaitPeriodCheckpoint*time.Duration(h))
				gap = w.posvWaitPeriodCheckpoint * time.Duration(h)
			}
		}
		waited := time.Since(time.Unix(int64(parentHeader.Time), 0))
		if waited < 0 {
			waited = 0
		}
		log.Info("[Miner] Waiting for my turn", "gap", gap, "hops", h, "waited", waited)
		if gap > waited {
			return false
		}
		log.Info("[Miner] Waited until timeout. Committing new work", "waited", waited)
	}
	return true
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

// Extracts the PoSV special transactions of validator accounts from the pending transactions, so the miner can commit them ahead of regular ones.
// Regular transactions with lower nonces are extracted along with them, and everything extracted is removed from txs.
func ExtractSpecialTransactionsForPosv(
	signer types.Signer,
	txs map[common.Address]types.Transactions,
	signers map[common.Address]struct{},
) types.Transactions {
	if len(signers) == 0 {
		return types.Transactions{}
	}
	specialTxs := types.Transactions{}
	for from, accTxs := range txs {
		if len(accTxs) == 0 {
			delete(txs, from)
			continue
		}
		acc, _ := types.Sender(signer, accTxs[0])
		if _, ok := signers[acc]; !ok {
			continue
		}
		lastSpecialTx := -1
		for i, tx := range accTxs {
			if tx.IsPosvTransaction() {
				lastSpecialTx = i
			}
		}
		if lastSpecialTx < 0 {
			continue
		}

		specialTxs = append(specialTxs, accTxs[:lastSpecialTx+1]...)
		normalTxs := accTxs[lastSpecialTx+1:]
		if len(normalTxs) > 0 {
			txs[acc] = normalTxs
		} else {
			delete(txs, acc)
		}
		if from != acc {
			delete(txs, from)
		}
	}
	return specialTxs
}
