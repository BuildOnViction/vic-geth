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

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/viction"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards

	// viction owns all Viction-specific processing hooks (hardfork activation,
	// system transactions, VRC25 fees, TomoX/TomoZ replay). See viction.Processor.
	viction *viction.Processor

	// Deferred trie GC fields for TomoX/TomoZ (full-node path).
	// These are managed entirely by blockchain_viction.go / commitVictionState.
	tradingTriegc *prque.Prque // deferred GC queue for TomoX trading trie roots
	lendingTriegc *prque.Prque // deferred GC queue for TomoZ lending trie roots
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config:        config,
		bc:            bc,
		engine:        engine,
		viction:       viction.NewProcessor(config, bc, engine),
		tradingTriegc: prque.New(nil),
		lendingTriegc: prque.New(nil),
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	// Viction hooks
	if err := p.viction.BeforeProcess(block, statedb); err != nil {
		return nil, nil, 0, err
	}
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	blockContext := NewEVMBlockContext(header, p.bc, nil)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, p.config, cfg)
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		msg, err := tx.AsMessage(types.MakeSigner(p.config, header.Number))
		if err != nil {
			return nil, nil, 0, err
		}
		if err := p.viction.BeforeApplyTransaction(block, tx, msg, statedb); err != nil {
			return nil, nil, 0, err
		}
		statedb.Prepare(tx.Hash(), block.Hash(), i)

		// Apply Viction-specific system transactions (BlockSigner, TomoX).
		handled, receipt, _, err, _ := p.applyVictionTransaction(statedb, tx, header, usedGas)
		if err != nil {
			return nil, nil, 0, err
		}

		if !handled {
			receipt, err = applyTransaction(msg, p.config, p.bc, nil, gp, statedb, header, tx, usedGas, vmenv, p.viction.FeePool())
			if err != nil {
				return nil, nil, 0, err
			}
		}

		// Execute Viction-specific post-transaction logic.
		if err := p.viction.AfterApplyTransaction(tx, msg, statedb, receipt, receipt.GasUsed, err); err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles())

	// Execute Viction-specific post-processing logic.
	if err := p.viction.AfterProcess(block, statedb); err != nil {
		return nil, nil, 0, err
	}

	return receipts, allLogs, *usedGas, nil
}

func applyTransaction(msg types.Message, config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, evm *vm.EVM, feePool viction.FeePool) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment
	txContext := NewEVMTxContext(msg)
	// Add addresses to access list if applicable
	if config.IsYoloV2(header.Number) {
		statedb.AddAddressToAccessList(msg.From())
		if dst := msg.To(); dst != nil {
			statedb.AddAddressToAccessList(*dst)
			// If it's a create-tx, the destination will be added inside evm.create
		}
		for _, addr := range evm.ActivePrecompiles() {
			statedb.AddAddressToAccessList(addr)
		}
	}

	// Update the evm with the new transaction context.
	evm.Reset(txContext, statedb)
	// Apply the transaction to the current state (included in the env)
	result, err := ApplyMessage(evm, msg, gp, feePool)
	if err != nil {
		return nil, err
	}
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing whether the root touch-delete accounts.
	receipt := types.NewReceipt(root, result.Failed(), *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = statedb.BlockHash()
	receipt.BlockNumber = header.Number
	receipt.TransactionIndex = uint(statedb.TxIndex())

	return receipt, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
//
// On POSV chains, BlockSigner transactions (0x89) are handled without the EVM
// when past the TIPSigning fork — they only increment the sender nonce and
// produce a zero-gas receipt. This reuses the same ApplySignTransaction function
// called by StateProcessor.applyVictionTransaction during block import.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	// POSV: BlockSigner (0x89) bypass — same logic as applyVictionTransaction.
	if tx.To() != nil && config.Viction != nil &&
		tx.IsSigningTransaction(config.Viction.ValidatorBlockSignContract) &&
		config.IsTIPSigning(header.Number) {
		handled, receipt, _, err, _ := ApplySignTransaction(config, statedb, tx, header, usedGas)
		if handled {
			return receipt, err
		}
	}

	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, config, cfg)
	// Standalone path (miner / chain_makers): no per-block fee pool is threaded,
	// so sponsored VRC25 accounting is not applied here. Live blocks are all
	// post-Atlas, where capacity is read directly from state in vrc25BuyGas.
	return applyTransaction(msg, config, bc, author, gp, statedb, header, tx, usedGas, vmenv, nil)
}

// applyVictionTransaction dispatches Viction system transactions during block
// import: the 0x89 BlockSigner tx is handled here in core so that
// ApplySignTransaction returns the core-native nonce errors; all other system
// transactions (0x91–0x94) delegate to the viction.Processor hook.
func (p *StateProcessor) applyVictionTransaction(statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	if tx.To() != nil && p.config.Viction != nil &&
		tx.IsSigningTransaction(p.config.Viction.ValidatorBlockSignContract) &&
		p.config.IsTIPSigning(header.Number) {
		return ApplySignTransaction(p.config, statedb, tx, header, usedGas)
	}
	return p.viction.ApplyVictionTransaction(statedb, tx, header, usedGas)
}

// ApplySignTransaction processes a BlockSigner special transaction (0x89)
// without the EVM: increments the sender nonce, adds a log entry, and returns a
// zero-gas receipt. Used during block import (applyVictionTransaction), the
// standalone ApplyTransaction path, and the miner (block creation).
func ApplySignTransaction(config *params.ChainConfig, statedb *state.StateDB, tx *types.Transaction, header *types.Header, usedGas *uint64) (bool, *types.Receipt, uint64, error, *big.Int) {
	// Validate nonce BEFORE Finalise to avoid invalidating the snapshot
	// on error (the caller may need to RevertToSnapshot).
	from, err := types.Sender(types.MakeSigner(config, header.Number), tx)
	if err != nil {
		return true, nil, 0, err, nil
	}
	nonce := statedb.GetNonce(from)
	if nonce < tx.Nonce() {
		return true, nil, 0, ErrNonceTooHigh, nil
	} else if nonce > tx.Nonce() {
		return true, nil, 0, ErrNonceTooLow, nil
	}

	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}

	statedb.SetNonce(from, nonce+1)

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 0
	// Set the receipt logs and create a bloom for filtering
	logEntry := &types.Log{}
	logEntry.Address = config.Viction.ValidatorBlockSignContract
	logEntry.BlockNumber = header.Number.Uint64()
	statedb.AddLog(logEntry)
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})

	return true, receipt, 0, nil, nil
}
