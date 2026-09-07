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
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/sortlgc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Commit native trading/lending trie nodes for the given block to their LevelDB backing stores.
func (bc *BlockChain) commitNativeExchangeState(block *types.Block) error {
	p, ok := bc.processor.(*StateProcessor)
	if !ok || p == nil {
		return nil
	}

	if bc.cacheConfig.TrieDirtyDisabled {
		if err := p.CommitTradingState(block); err != nil {
			return err
		}
		return p.CommitLendingState(block)
	}
	if err := p.CommitTradingStateDeferred(block); err != nil {
		return err
	}
	return p.CommitLendingStateDeferred(block)
}

// Flush any in-memory trading/lending trie roots not yet committed to LevelDB.
func (bc *BlockChain) stopViction() {
	p, ok := bc.processor.(*StateProcessor)
	if !ok || p == nil {
		return
	}
	p.FlushTradingStateGCCache()
	p.FlushLendingStateGCCache()
}

// Inject the Native Trading Engine into the Processor.
func (bc *BlockChain) SetTradingEngine(engine TradingEngine) {
	p, ok := bc.processor.(*StateProcessor)
	if !ok {
		log.Error("[NativeTrading] Engine not installed: Processor is not a *StateProcessor")
		return
	}
	p.viction.SetTradingEngine(engine)
	log.Info("[NativeTrading] Engine installed on state processor")
}

// Inject the Native Lending Engine into the Processor.
func (bc *BlockChain) SetLendingEngine(engine LendingEngine) {
	p, ok := bc.processor.(*StateProcessor)
	if !ok {
		log.Error("[NativeLending] Engine not installed: Processor is not a *StateProcessor")
		return
	}
	p.viction.SetLendingEngine(engine)
	log.Info("[NativeLending] Engine installed on state processor")
}

func (bc *BlockChain) UpdateValidators() error {
	engine, ok := bc.Engine().(*posv.Posv)
	if bc.Config().Posv == nil || !ok {
		return ErrPosvRequired
	}
	log.Info("[Blockchain] Preparing new validators list for next epoch.")

	contractAddress := bc.chainConfig.Viction.ValidatorContract
	if contractAddress == (common.Address{}) {
		return ErrNoValidatorContract
	}

	var candidates []common.Address
	stateDB, err := bc.State()
	if err != nil {
		return fmt.Errorf("failed to get state at block #%v: %v", bc.CurrentHeader().Number, err)
	}
	candidates = stateDB.VicGetCandidates(contractAddress)

	var validators []posv.ValidatorInfo
	for _, candidate := range candidates {
		_, cap := stateDB.VicGetValidatorInfo(contractAddress, candidate)
		if candidate.String() != "0x0000000000000000000000000000000000000000" {
			validators = append(validators, posv.ValidatorInfo{Address: candidate, Capacity: cap})
		}
	}
	if len(validators) == 0 {
		return ErrNoValidators
	}

	header := bc.CurrentHeader()
	if bc.Config().IsAtlas(header.Number) {
		sort.SliceStable(validators, func(i, j int) bool {
			return validators[i].Capacity.Cmp(validators[j].Capacity) >= 0
		})
	} else {
		sortlgc.Slice(validators, func(i, j int) bool {
			return validators[i].Capacity.Cmp(validators[j].Capacity) >= 0
		})
	}

	vs := make([]common.Address, 0)
	if len(validators) > int(bc.chainConfig.Viction.ValidatorMaxCount) {
		for _, v := range validators[:bc.chainConfig.Viction.ValidatorMaxCount] {
			vs = append(vs, v.Address)
		}
	} else {
		for _, v := range validators {
			vs = append(vs, v.Address)
		}
	}
	err = engine.SetCheckpointSigners(bc, header, vs)
	if err != nil {
		return err
	}

	log.Info("[Blockchain] Updated validators list for next epoch", "signers", len(vs))
	return nil
}

// Check if two blocks are same path. Assume block 1 is ahead block 2.
func (bc *BlockChain) AreTwoBlockSamePath(bh1 common.Hash, bh2 common.Hash) bool {
	h1 := bc.GetHeaderByHash(bh1)
	h2 := bc.GetHeaderByHash(bh2)
	if h1 == nil || h2 == nil {
		return false
	}
	toLevel := h2.Number.Uint64()
	hash1 := bh1

	for h1.Number.Uint64() > toLevel {
		hash1 = h1.ParentHash
		h1 = bc.GetHeaderByHash(hash1)
		if h1 == nil {
			return false
		}
	}

	return hash1 == bh2
}
