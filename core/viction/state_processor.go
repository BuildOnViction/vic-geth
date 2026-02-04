// Copyright 2015 The go-ethereum Authors
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

package coreviction

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tomox/tradingstate"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config       *params.ChainConfig // Chain configuration options
	bc           *BlockChain         // Canonical block chain
	engine       posv.Engine         // Consensus engine used for block rewards
	victionState *victionProcessorState
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine posv.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, cfg vm.Config, balanceFee map[common.Address]*big.Int) (types.Receipts, []*types.Log, uint64, error) {
	// Viction hooks
	if err := p.beforeProcess(block, statedb); err != nil {
		return nil, nil, 0, err
	}
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(core.GasPool).AddGas(block.GasLimit())
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	if common.TIPSigningBlock.Cmp(header.Number) == 0 {
		statedb.DeleteAddress(common.HexToAddress(common.BlockSigners))
	}
	if p.config.IsAtlas(header.Number) {
		misc.ApplyVIPVRC25Upgarde(statedb, p.config.AtlasBlock, header.Number)
	}

	if p.config.SaigonBlock != nil && p.config.SaigonBlock.Cmp(block.Number()) <= 0 {
		if common.IsTestnet {
			misc.ApplySaigonHardForkTestnet(statedb, p.config.SaigonBlock, block.Number(), p.config.Posv)
		} else {
			misc.ApplySaigonHardFork(statedb, p.config.SaigonBlock, block.Number())
		}
	}
	parentState := statedb.Copy()
	InitSignerInTransactions(p.config, header, block.Transactions())

	balanceUpdated := map[common.Address]*big.Int{}
	totalFeeUsed := big.NewInt(0)

	// Check if we're past the Atlas block
	isAtlas := p.config.IsAtlas(block.Number())

	for i, tx := range block.Transactions() {
		// check black-list txs after hf
		if (block.Number().Uint64() >= common.BlackListHFBlock) && !common.IsTestnet {
			// check if sender is in black list
			if tx.From() != nil && common.Blacklist[*tx.From()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with sender in black-list: %v", tx.From().Hex())
			}
			// check if receiver is in black list
			if tx.To() != nil && common.Blacklist[*tx.To()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with receiver in black-list: %v", tx.To().Hex())
			}
		}
		// validate balance slot, minFee slot for TomoZ
		if p.config.IsTomoZEnabled(block.Number()) && tx.IsTomoZApplyTransaction() {
			copyState := statedb.Copy()
			if err := ValidateTomoZApplyTransaction(p.bc, copyState, common.BytesToAddress(tx.Data()[4:])); err != nil {
				return nil, nil, 0, err
			}
		}
		// validate balance slot, token decimal for TomoX
		if p.config.IsTomoXEnabled(block.Number()) && tx.IsTomoXApplyTransaction() {
			copyState := statedb.Copy()
			if err := ValidateTomoXApplyTransaction(p.bc, copyState, common.BytesToAddress(tx.Data()[4:])); err != nil {
				return nil, nil, 0, err
			}
		}

		statedb.Prepare(tx.Hash(), block.Hash(), i)
		receipt, gas, err, tokenFeeUsed := ApplyTransaction(p.config, balanceFee, p.bc, nil, gp, statedb, tradingState, header, tx, usedGas, cfg)
		if err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
		if tokenFeeUsed && !isAtlas {
			fee := new(big.Int).SetUint64(gas)
			if block.Header().Number.Cmp(common.TIPTRC21FeeBlock) > 0 {
				fee = fee.Mul(fee, common.TRC21GasPrice)
			}
			balanceFee[*tx.To()] = new(big.Int).Sub(balanceFee[*tx.To()], fee)
			balanceUpdated[*tx.To()] = balanceFee[*tx.To()]
			totalFeeUsed = totalFeeUsed.Add(totalFeeUsed, fee)
		}
	}

	if !isAtlas {
		state.UpdateTRC21Fee(statedb, balanceUpdated, totalFeeUsed)
	}

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, parentState, block.Transactions(), block.Uncles(), receipts)
	return receipts, allLogs, *usedGas, nil
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func applyTransaction(msg types.Message, config *params.ChainConfig, bc core.ChainContext, author *common.Address, gp *core.GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (*types.Receipt, error) {
	return nil, nil
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, tokensFee map[common.Address]*big.Int, bc *BlockChain, author *common.Address, gp *core.GasPool, statedb *state.StateDB, tomoxState *tradingstate.TradingStateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error, bool) {
	return nil, 0, nil, false
}

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
