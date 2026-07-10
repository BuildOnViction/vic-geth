// Copyright 2026 The Vic-geth Authors
package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vrc25"
	"github.com/ethereum/go-ethereum/params"
)

// VictionFeePool is the per-execution running map of VRC25 token fee capacities,
// keyed by token (transaction recipient). It is threaded explicitly through
// ApplyMessage as the StateTransition's feePool, so block import and each tracing
// re-execution own an independent instance and never share fee state. A nil pool
// means "no sponsorship" — every transaction is treated as a regular VIC tx.
type VictionFeePool = map[common.Address]*big.Int

// CopyVictionFeePool returns an independent deep copy of src (keys and *big.Int
// values cloned), or nil if src is nil. The parallel block tracer gives each task
// its own copy so worker goroutines never mutate a shared map.
func CopyVictionFeePool(src VictionFeePool) VictionFeePool {
	if src == nil {
		return nil
	}
	dst := make(VictionFeePool, len(src))
	for k, v := range src {
		if v != nil {
			dst[k] = new(big.Int).Set(v)
		}
	}
	return dst
}

// VictionProcessor owns the pre-Atlas VRC25 fee accounting for a single
// execution over one block. It is a plain value type deliberately decoupled from
// StateProcessor (which needs the blockchain and consensus engine), so that the
// tracing API — which re-executes transactions with only a StateDB — can create
// and drive its own instance with the exact same seed/decrement/flush logic as
// block import. This is the single source of truth for the fee formula; block
// import and tracing share it and therefore cannot diverge.
type VictionProcessor struct {
	config     *params.ChainConfig
	feePool    VictionFeePool // running per-token capacity; nil = no sponsorship
	feeUpdated VictionFeePool // tokens charged this block -> final capacity (flush)
	totalFee   *big.Int       // total fee charged this block (flush)
}

// NewVictionProcessor builds the fee processor for a block. On pre-Atlas Viction
// blocks with a VRC25 issuer configured it snapshots all token fee capacities;
// otherwise feePool stays nil and every transaction is a regular VIC tx.
func NewVictionProcessor(config *params.ChainConfig, statedb vm.StateDB, blockNum *big.Int) *VictionProcessor {
	vp := &VictionProcessor{
		config:     config,
		feeUpdated: make(VictionFeePool),
		totalFee:   new(big.Int),
	}
	if !config.IsAtlas(blockNum) && config.Viction != nil &&
		config.Viction.VRC25Contract != (common.Address{}) {
		vp.feePool = vrc25.GetAllFeeCapacities(statedb, config.Viction.VRC25Contract)
	}
	return vp
}

// FeePool returns the running fee-capacity map to thread into ApplyMessage.
// Safe on a nil receiver (returns nil).
func (vp *VictionProcessor) FeePool() VictionFeePool {
	if vp == nil {
		return nil
	}
	return vp.feePool
}

// HandleFee applies the pre-Atlas per-transaction fee accounting: it decrements
// the running pool for tx's token by the fee charged, records the update for the
// end-of-block flush, and for a failed sponsored transaction applies the same
// PayFeeWithVRC25 balance mutation as block import.
//
// No-op on a nil receiver, post-Atlas blocks, non-VRC25 transactions, contract
// creations, or when the pool has no entry for the token.
func (vp *VictionProcessor) HandleFee(statedb vm.StateDB, blockNum *big.Int, tx *types.Transaction, from common.Address, usedGas uint64, failed bool) {
	if vp == nil || vp.feePool == nil || tx.To() == nil || vp.config.IsAtlas(blockNum) {
		return
	}
	token := *tx.To()
	runningCap, ok := vp.feePool[token]
	if !ok || runningCap == nil {
		return
	}
	vicCfg := vp.config.Viction
	fee := new(big.Int).SetUint64(usedGas)
	if vp.config.TIPTRC21FeeBlock != nil && blockNum.Cmp(vp.config.TIPTRC21FeeBlock) > 0 &&
		vicCfg != nil && vicCfg.VRC25GasPrice != nil {
		fee = new(big.Int).Mul(fee, (*big.Int)(vicCfg.VRC25GasPrice))
	}
	if runningCap.Cmp(fee) > 0 {
		newCap := new(big.Int).Sub(runningCap, fee)
		vp.feePool[token] = newCap
		vp.feeUpdated[token] = newCap
		vp.totalFee.Add(vp.totalFee, fee)
		if failed {
			vrc25.PayFeeWithVRC25(statedb, from, token)
		}
	}
}

// Flush writes the accumulated pre-Atlas fee updates back to state. Called once
// per block after all transactions (block import only; tracing never flushes).
// No-op when nothing was charged.
func (vp *VictionProcessor) Flush(statedb vm.StateDB) {
	if vp == nil || vp.feePool == nil || len(vp.feeUpdated) == 0 || vp.config.Viction == nil {
		return
	}
	vrc25.UpdateFeeCapacity(statedb, vp.config.Viction.VRC25Contract, vp.feeUpdated, vp.totalFee)
}
