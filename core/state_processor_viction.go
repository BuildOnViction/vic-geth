// Copyright 2026 The Vic-geth Authors
// This file provides vic-extensions to the geth.
package core

import (
	"math/big"
	"runtime"
	"sync"


	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type victionProcessorState struct {
	currentBlockNumber *big.Int
	parrentState   		*state.StateDB
	balanceFee         map[common.Address]*big.Int
	balanceUpdated     map[common.Address]*big.Int
	totalFeeUsed       *big.Int
}


func (p *StateProcessor) beforeProcess(block *types.Block, statedb *state.StateDB) error {
	header := block.Header()

	// Clear previous state and initialize new
	p.victionState = &victionProcessorState{
		currentBlockNumber: new(big.Int).Set(header.Number),
		balanceFee:         make(map[common.Address]*big.Int),
		balanceUpdated:     make(map[common.Address]*big.Int),
		totalFeeUsed:       big.NewInt(0),
	}

	// 1. Hardfork: TIPSigning
	if p.config.TIPSigningBlock.Cmp(header.Number) == 0 {
		statedb.DeleteAddress(p.config.Viction.ValidatorBlockSignContract)
	}

	// 2. Hardfork: Atlas
	if p.config.IsAtlas(header.Number) {
		//[to-do] implement Atlas hardfork logic
		// misc.ApplyVIPVRC25Upgarde(statedb, p.config.AtlasBlock, header.Number)
	}

	// 3. Hardfork: Saigon
	if p.config.SaigonBlock != nil && p.config.SaigonBlock.Cmp(block.Number()) == 0 {
		//[to-do] implement Saigon hardfork logic
		// misc.ApplySaigonHardFork(statedb, p.config.SaigonBlock, block.Number())
	}

	p.victionState.parrentState = statedb.Copy()
	InitSignerInTransactions(p.config, block.Header(), block.Transactions())

	p.victionState.balanceUpdated = make(map[common.Address]*big.Int)
	p.victionState.totalFeeUsed = big.NewInt(0)
	
	return nil
}

func (p *StateProcessor) afterProcess(block *types.Block, statedb *state.StateDB) error {
	return nil
}

func (p *StateProcessor) beforeApplyTransaction(tx *types.Transaction, msg Message, statedb *state.StateDB) error {
	return nil
}

func (p *StateProcessor) afterApplyTransaction(tx *types.Transaction, msg Message, statedb *state.StateDB, receipt *types.Receipt, usedGas uint64, err error) error {
	return nil
}

// --- Helpers ---
func InitSignerInTransactions(config *params.ChainConfig, header *types.Header, txs types.Transactions) {
	nWorker := runtime.NumCPU()
	signer := types.MakeSigner(config, header.Number)
	chunkSize := txs.Len() / nWorker
	if txs.Len()%nWorker != 0 {
		chunkSize++
	}
	wg := sync.WaitGroup{}
	wg.Add(nWorker)
	for i := 0; i < nWorker; i++ {
		from := i * chunkSize
		to := from + chunkSize
		if to > txs.Len() {
			to = txs.Len()
		}
		go func(from int, to int) {
			for j := from; j < to; j++ {
				types.CacheSigner(signer, txs[j])
				txs[j].CacheHash()
			}
			wg.Done()
		}(from, to)
	}
	wg.Wait()
}
