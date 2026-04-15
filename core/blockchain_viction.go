// Copyright 2026 The Vic-geth Authors
package core

import (
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sortlgc"
)

// SetTradingEngine injects the TomoX trading engine into the block processor.
func (bc *BlockChain) SetTradingEngine(engine TradingEngine) {
	sp, ok := bc.processor.(*StateProcessor)
	if !ok {
		log.Error("SetTradingEngine: processor is not a *StateProcessor, trading engine not installed")
		return
	}
	sp.SetTradingEngine(engine)
	log.Info("TomoX trading engine installed on state processor")
}

// SetLendingEngine injects the TomoZ lending engine into the block processor.
func (bc *BlockChain) SetLendingEngine(engine LendingEngine) {
	sp, ok := bc.processor.(*StateProcessor)
	if !ok {
		log.Error("SetLendingEngine: processor is not a *StateProcessor, lending engine not installed")
		return
	}
	sp.SetLendingEngine(engine)
	log.Info("TomoZ lending engine installed on state processor")
}

// beforeProcessViction runs TomoZ liquidation data at epoch boundaries before
// the main transaction loop. Only active for pre-Atlas lending-enabled blocks.
func (bc *BlockChain) beforeProcessViction(block *types.Block, statedb *state.StateDB) error {
	if bc.chainConfig.Posv == nil {
		return nil
	}
	sp, ok := bc.processor.(*StateProcessor)
	if !ok || sp.lendingEngine == nil || sp.tradingEngine == nil {
		return nil
	}
	if !bc.chainConfig.IsTIPTomoXLending(block.Number()) || bc.chainConfig.IsAtlas(block.Number()) {
		return nil
	}
	if block.NumberU64()%bc.chainConfig.Posv.Epoch != uint64(bc.chainConfig.Viction.LendingLiquidateTradeBlock) {
		return nil
	}

	parent := bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil
	}
	parentAuthor, err := bc.Engine().Author(parent.Header())
	if err != nil {
		return fmt.Errorf("TomoZ: liquidation: failed to resolve parent author: %w", err)
	}
	tradingState, err := sp.tradingEngine.GetTradingState(parent, parentAuthor)
	if err != nil {
		return fmt.Errorf("TomoZ: liquidation: failed to open TradingStateDB: %w", err)
	}
	lendingState, err := sp.lendingEngine.GetLendingState(parent, parentAuthor)
	if err != nil {
		return fmt.Errorf("TomoZ: liquidation: failed to open LendingStateDB: %w", err)
	}

	_, _, _, _, _, err = sp.lendingEngine.ProcessLiquidationData(
		block.Header(), bc, statedb, tradingState, lendingState,
	)
	if err != nil {
		return fmt.Errorf("TomoZ: ProcessLiquidationData failed at block %d: %w", block.NumberU64(), err)
	}
	log.Debug("TomoZ: epoch liquidation processed", "block", block.NumberU64())
	return nil
}

func (bc *BlockChain) UpdateM1() error {
	engine, ok := bc.Engine().(*posv.Posv)
	if bc.Config().Posv == nil || !ok {
		return fmt.Errorf("PoSV engine is not enabled")
	}
	log.Info("It's time to update new set of masternodes for the next epoch...")

	contractAddress := bc.chainConfig.Viction.ValidatorContract
	if contractAddress == (common.Address{}) {
		return fmt.Errorf("validator contract address is not set in chain config")
	}

	var candidates []common.Address

	// get candidates from slot of stateDB
	// if can't get anything, request from contracts
	stateDB, err := bc.State()
	if err != nil {
		return fmt.Errorf("failed to get state at current root (block %v): %v", bc.CurrentHeader().Number, err)
	}
	candidates = stateDB.VicGetCandidates(contractAddress)

	var ms []posv.Masternode
	for _, candidate := range candidates {
		_, cap := stateDB.VicGetValidatorInfo(contractAddress, candidate)

		//TODO: smart contract shouldn't return "0x0000000000000000000000000000000000000000"
		if candidate.String() != "0x0000000000000000000000000000000000000000" {
			ms = append(ms, posv.Masternode{Address: candidate, Stake: cap})
		}
	}
	if len(ms) == 0 {
		log.Error("No masternode found. Stopping node")
		return fmt.Errorf("no masternode found")
	} else {
		header := bc.CurrentHeader()
		if bc.Config().IsAtlas(header.Number) {
			sort.SliceStable(ms, func(i, j int) bool {
				return ms[i].Stake.Cmp(ms[j].Stake) >= 0
			})
		} else {
			// Must sort `ms`, not `candidates`: indices i,j are in [0, len(slice));
			// len(candidates) can exceed len(ms) when zero-address entries are skipped.
			sortlgc.Slice(ms, func(i, j int) bool {
				return ms[i].Stake.Cmp(ms[j].Stake) >= 0
			})
		}
		log.Info("Ordered list of masternode candidates")
		for _, m := range ms {
			log.Info("", "address", m.Address.String(), "stake", m.Stake)
		}
		// update masternodes
		log.Info("Updating new set of masternodes")
		if len(ms) > int(bc.chainConfig.Viction.ValidatorMaxCount) {
			err = engine.UpdateMasternodes(bc, header, ms[:bc.chainConfig.Viction.ValidatorMaxCount])
		} else {
			err = engine.UpdateMasternodes(bc, header, ms)
		}
		if err != nil {
			return err
		}
		log.Info("Masternodes are ready for the next epoch")
	}
	return nil
}
